package main

// gerbillo build (ADR 0012): genera la Gallery pubblica statica in dist/ dalla Foto
// pubblicata (pubblica.go). dist/ contiene solo HTML, CSS e l'Indice delle Foto: le
// Varianti stanno nella loro cartella e nginx le serve a /gallery/f/<pid>/<variante>.webp.
// Il build scrive una cartella nuova e la sostituisce alla vecchia in un colpo solo.

import (
	"embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

//go:embed modelli
var modelli embed.FS

var pagine = template.Must(template.ParseFS(modelli, "modelli/*.html"))

const perPagina = 240

type opzioniBuild struct {
	base     string // prefisso degli URL: /gallery/ in produzione, / nell'anteprima
	media    string // "" = Varianti; altrimenti URL di albums/ di Coppermine (anteprima)
	pubblico string // URL assoluto della Gallery, per la Copia per il forum
	sito     string // URL del Sito principale
	striscia string // testo della striscia gialla dell'anteprima
}

func build(args []string) error {
	fs := flag.NewFlagSet("build", flag.ExitOnError)
	conf := fs.String("config", "gerbillo.toml", "file di configurazione TOML")
	indice := fs.String("indice", "", "Indice delle Entità (JSON)")
	dist := fs.String("dist", "dist", "cartella di uscita")
	var o opzioniBuild
	fs.StringVar(&o.base, "base", "/gallery/", "prefisso degli URL")
	fs.StringVar(&o.media, "media", "", "URL di albums/ di Coppermine al posto delle Varianti (anteprima)")
	fs.StringVar(&o.pubblico, "pubblico", "https://progettodighe.it/gallery/", "URL pubblico della Gallery")
	fs.StringVar(&o.sito, "sito", "https://progettodighe.it/", "URL del Sito principale")
	fs.StringVar(&o.striscia, "striscia", "", "striscia dell'anteprima")
	fs.Parse(args)
	var c configImporta
	if _, err := toml.DecodeFile(*conf, &c); err != nil {
		return err
	}
	if c.Contenuti == "" || *indice == "" {
		return errors.New("servono contenuti nella configurazione e -indice")
	}
	inizio := time.Now()
	cont, err := leggiContenuti(c.Contenuti, *indice)
	if err != nil {
		return err
	}
	nuova := strings.TrimSuffix(*dist, "/") + ".nuova"
	os.RemoveAll(nuova)
	b := &builder{opzioniBuild: o, c: cont, dir: nuova}
	if err := b.tutto(); err != nil {
		return err
	}
	vecchia := strings.TrimSuffix(*dist, "/") + ".vecchia"
	os.RemoveAll(vecchia)
	if err := os.Rename(*dist, vecchia); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.Rename(nuova, *dist); err != nil {
		return err
	}
	os.RemoveAll(vecchia)
	for _, s := range cont.ignoti {
		fmt.Fprintln(os.Stderr, "slug sconosciuto:", s)
	}
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	fmt.Printf("build: %d Foto, %d pagine in %s; memoria Go massima %d MiB\n",
		len(cont.foto), b.scritte, time.Since(inizio).Round(time.Millisecond), m.Sys>>20)
	return nil
}

type builder struct {
	opzioniBuild
	c       *contenuti
	dir     string
	scritte int
	// per ogni Foto: l'elenco in cui sta (per precedente e successiva) e la sua posizione
	contesto map[int]contestoFoto
}

type contestoFoto struct {
	elenco []*fotoPubblicata
	i      int
	su     voce // la paginaSito dell'elenco
}

type paginaSito struct {
	Titolo, Base, Striscia, Indietro, Avanti string
	Briciole                                 []voce
	Corpo                                    any
}

type voce struct {
	URL, Testo, Mini, Sito string
	N                      int
}

type miniatura struct{ URL, Mini, Titolo string }

type corpoGriglia struct {
	Sottotitolo      string
	Link             []voce
	Foto             []miniatura
	Pagina, Pagine   int
	Indietro, Avanti string
}

type corpoElenco struct {
	Sottotitolo string
	Gruppi      []gruppo
}

type gruppo struct {
	Nome        string
	Sottogruppi []sottogruppo
}

type sottogruppo struct {
	Nome string
	Voci []voce
}

type corpoFoto struct {
	Schermo, Mini, Originale, Alt, Formato           string
	Description, Authors, Datazione, Luogo, Caricata string
	BBCode                                           string
	Larghezza, Altezza                               int
	Slug                                             []voce
	Contributore                                     *voce
	Tags                                             []string
}

func (b *builder) url(parti ...string) string {
	if len(parti) == 0 {
		return b.base
	}
	return b.base + path.Join(parti...) + "/"
}

func (b *builder) scrivi(modello string, p paginaSito, parti ...string) error {
	p.Base, p.Striscia = b.base, b.striscia
	dir := filepath.Join(append([]string{b.dir}, parti...)...)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	f, err := os.Create(filepath.Join(dir, "index.html"))
	if err != nil {
		return err
	}
	defer f.Close()
	b.scritte++
	return pagine.ExecuteTemplate(f, modello, p)
}

func (b *builder) mini(f *fotoPubblicata) string {
	switch {
	case f.Formato != "immagine":
		return b.base + "statico/documento.svg"
	case b.media != "":
		return b.media + f.Coppermine.Filepath + "thumb_" + f.Coppermine.Filename
	}
	return b.base + "f/" + strconv.Itoa(f.Pid) + "/mini.webp"
}

func (b *builder) schermo(f *fotoPubblicata) string {
	switch {
	case f.Formato != "immagine":
		return ""
	case b.media != "":
		return b.media + f.Coppermine.Filepath + f.Coppermine.Filename
	}
	return b.base + "f/" + strconv.Itoa(f.Pid) + "/schermo.webp"
}

// titolo ripiega sul nome della prima Entità: un terzo delle Foto non ha titolo.
func (b *builder) titolo(f *fotoPubblicata) string {
	switch {
	case f.Title != "":
		return f.Title
	case len(f.Slug) > 0:
		return b.c.entita[f.Slug[0]].Title + " · foto " + strconv.Itoa(f.Pid)
	}
	return "Foto " + strconv.Itoa(f.Pid)
}

func (b *builder) miniature(fs []*fotoPubblicata) []miniatura {
	m := make([]miniatura, len(fs))
	for i, f := range fs {
		m[i] = miniatura{b.url("f", strconv.Itoa(f.Pid)), b.mini(f), f.Title}
	}
	return m
}

func (b *builder) tutto() error {
	perEntita := map[string][]*fotoPubblicata{}
	perContributore := map[int][]*fotoPubblicata{}
	nomi := map[int]string{}
	altre := map[int][]*fotoPubblicata{} // Categoria Coppermine radice → Foto senza slug
	for _, f := range b.c.foto {
		for _, s := range f.Slug {
			perEntita[s] = append(perEntita[s], f)
		}
		if f.Contributore != nil {
			perContributore[f.Contributore.ID] = append(perContributore[f.Contributore.ID], f)
			nomi[f.Contributore.ID] = f.Contributore.Nome
		}
		if len(f.Slug) == 0 {
			r := b.radice(f.Coppermine.Aid)
			altre[r] = append(altre[r], f)
		}
	}
	b.contesto = map[int]contestoFoto{}
	casa := voce{URL: b.url(), Testo: "Gallery"}

	// Entità: una griglia per paginaSito del sito, che è anche il contesto delle sue Foto.
	slugs := chiaviOrdinate(perEntita)
	for _, s := range slugs {
		e, fs := b.c.entita[s], perEntita[s]
		su := voce{URL: b.url(s), Testo: e.Title}
		b.registraContesto(fs, su)
		var link []voce
		if !e.Draft {
			link = append(link, voce{URL: b.sito + s + "/", Testo: "Pagina di " + e.Title + " sul sito"})
		}
		sotto := luogoEntita(e)
		if err := b.griglie(fs, e.Title, sotto, link, []voce{casa}, s); err != nil {
			return err
		}
	}
	// Da classificare: Foto senza slug, per Categoria Coppermine.
	gruppiAltre := chiaviOrdinate(altre)
	daClassificare := voce{URL: b.url("altre"), Testo: "Da classificare"}
	for _, cid := range gruppiAltre {
		nome := b.c.album.Categorie[cid].Nome
		su := voce{URL: b.url("altre", strconv.Itoa(cid)), Testo: "Da classificare: " + nome}
		b.registraContesto(altre[cid], su)
		if err := b.griglie(altre[cid], nome, "Foto non ancora collegate a una paginaSito del sito.", nil, []voce{casa, daClassificare}, "altre", strconv.Itoa(cid)); err != nil {
			return err
		}
	}
	// Contributori: la Vista per Contributore (#2). Non è il contesto delle Foto.
	contributori := voce{URL: b.url("contributori"), Testo: "Contributori"}
	ids := chiaviOrdinate(perContributore)
	for _, id := range ids {
		if err := b.griglie(perContributore[id], "Foto di "+nomi[id], "", nil, []voce{casa, contributori}, "contributori", strconv.Itoa(id)); err != nil {
			return err
		}
	}
	for _, f := range b.c.foto {
		if err := b.paginaFoto(f, casa); err != nil {
			return err
		}
	}
	if err := b.home(perEntita, slugs); err != nil {
		return err
	}
	if err := b.elencoContributori(ids, nomi, perContributore, casa); err != nil {
		return err
	}
	if err := b.elencoAltre(gruppiAltre, altre, casa); err != nil {
		return err
	}
	return b.statici(perEntita)
}

// registraContesto assegna a ogni Foto l'elenco in cui navigare con le frecce.
// Vale il primo elenco che la contiene: la prima Entità, altrimenti la sua categoria.
func (b *builder) registraContesto(fs []*fotoPubblicata, su voce) {
	for i, f := range fs {
		if _, ok := b.contesto[f.Pid]; !ok {
			b.contesto[f.Pid] = contestoFoto{fs, i, su}
		}
	}
}

func (b *builder) radice(aid int) int {
	cid := b.c.album.Album[aid].Categoria
	if cid >= cidPersonali {
		return 1 // User galleries
	}
	for i := 0; i < 10; i++ {
		c, ok := b.c.album.Categorie[cid]
		if !ok || c.Parent == 0 {
			return cid
		}
		cid = c.Parent
	}
	return cid
}

// griglie scrive un elenco di Foto in pagine da perPagina: la prima in parti/, le altre in parti/<n>/.
func (b *builder) griglie(fs []*fotoPubblicata, tit, sotto string, link, briciole []voce, parti ...string) error {
	n := (len(fs) + perPagina - 1) / perPagina
	pag := func(k int) string {
		if k == 1 {
			return b.url(parti...)
		}
		return b.url(append(parti, strconv.Itoa(k))...)
	}
	for k := 1; k <= n; k++ {
		fine := min(k*perPagina, len(fs))
		corpo := corpoGriglia{Foto: b.miniature(fs[(k-1)*perPagina : fine]), Pagina: k, Pagine: n}
		if k == 1 {
			corpo.Sottotitolo, corpo.Link = conteggio(len(fs), sotto), link
		}
		if k > 1 {
			corpo.Indietro = pag(k - 1)
		}
		if k < n {
			corpo.Avanti = pag(k + 1)
		}
		dove := parti
		if k > 1 {
			dove = append(slices.Clone(parti), strconv.Itoa(k))
		}
		p := paginaSito{Titolo: tit, Briciole: briciole, Corpo: corpo, Indietro: corpo.Indietro, Avanti: corpo.Avanti}
		if err := b.scrivi("griglia", p, dove...); err != nil {
			return err
		}
	}
	return nil
}

func conteggio(n int, dopo string) string {
	s := strconv.Itoa(n) + " foto"
	if n == 1 {
		s = "1 foto"
	}
	if dopo != "" {
		s += " · " + dopo
	}
	return s
}

func luogoEntita(e *entita) string {
	var p []string
	for _, s := range []string{e.Geolocation.Provincia, e.Geolocation.Regione, e.Geolocation.Stato} {
		if s != "" {
			p = append(p, s)
		}
	}
	return strings.Join(p, ", ")
}

func (b *builder) paginaFoto(f *fotoPubblicata, casa voce) error {
	pid := strconv.Itoa(f.Pid)
	c := corpoFoto{
		Schermo: b.schermo(f), Mini: b.mini(f), Formato: f.Formato,
		Alt: f.Alt, Description: f.Description, Authors: strings.Join(f.Authors, ", "),
		Datazione: f.Datazione, Luogo: f.Coppermine.Luogo, Tags: f.tagPubblici(),
		BBCode: fmt.Sprintf("[url=%sf/%s/][img]%sf/%s/mini.webp[/img][/url]", b.pubblico, pid, b.pubblico, pid),
	}
	if c.Alt == "" {
		c.Alt = f.Title
	}
	if b.media != "" {
		c.Originale = b.media + f.Coppermine.Filepath + f.Coppermine.Filename
	}
	if t, err := time.Parse(time.RFC3339, f.Date); err == nil {
		c.Caricata = t.Format("02/01/2006")
	}
	// Dimensioni della Variante schermo, per riservare lo spazio prima del caricamento.
	c.Larghezza, c.Altezza = f.Larghezza, f.Altezza
	if l := max(f.Larghezza, f.Altezza); b.media == "" && l > 1600 {
		c.Larghezza, c.Altezza = f.Larghezza*1600/l, f.Altezza*1600/l
	}
	for _, s := range f.Slug {
		e := b.c.entita[s]
		v := voce{URL: b.url(s), Testo: e.Title}
		if !e.Draft {
			v.Sito = b.sito + s + "/"
		}
		c.Slug = append(c.Slug, v)
	}
	if f.Contributore != nil {
		c.Contributore = &voce{URL: b.url("contributori", strconv.Itoa(f.Contributore.ID)), Testo: f.Contributore.Nome}
	}
	p := paginaSito{Titolo: b.titolo(f), Corpo: c, Briciole: []voce{casa}}
	if ctx, ok := b.contesto[f.Pid]; ok {
		p.Briciole = append(p.Briciole, ctx.su)
		if ctx.i > 0 {
			p.Indietro = b.url("f", strconv.Itoa(ctx.elenco[ctx.i-1].Pid))
		}
		if ctx.i+1 < len(ctx.elenco) {
			p.Avanti = b.url("f", strconv.Itoa(ctx.elenco[ctx.i+1].Pid))
		}
	}
	return b.scrivi("foto", p, "f", pid)
}

// home è la Vista ad albero: stato → regione → Entità.
func (b *builder) home(perEntita map[string][]*fotoPubblicata, slugs []string) error {
	albero := map[string]map[string][]voce{}
	for _, s := range slugs {
		e, fs := b.c.entita[s], perEntita[s]
		st, reg := e.Geolocation.Stato, e.Geolocation.Regione
		if st == "" {
			st = "Senza stato"
		}
		if albero[st] == nil {
			albero[st] = map[string][]voce{}
		}
		albero[st][reg] = append(albero[st][reg], voce{URL: b.url(s), Testo: e.Title, Mini: b.mini(fs[0]), N: len(fs)})
	}
	stati := chiaviOrdinate(albero)
	if i := slices.Index(stati, "Italia"); i > 0 {
		stati = append([]string{"Italia"}, slices.Delete(stati, i, i+1)...)
	}
	var corpo corpoElenco
	collegate := 0
	for _, f := range b.c.foto {
		if len(f.Slug) > 0 {
			collegate++
		}
	}
	corpo.Sottotitolo = fmt.Sprintf("%d foto, di cui %d collegate a %d dighe.", len(b.c.foto), collegate, len(slugs))
	for _, st := range stati {
		g := gruppo{Nome: st}
		for _, reg := range chiaviOrdinate(albero[st]) {
			v := albero[st][reg]
			slices.SortFunc(v, func(a, b voce) int { return strings.Compare(strings.ToLower(a.Testo), strings.ToLower(b.Testo)) })
			g.Sottogruppi = append(g.Sottogruppi, sottogruppo{reg, v})
		}
		corpo.Gruppi = append(corpo.Gruppi, g)
	}
	return b.scrivi("elenco", paginaSito{Titolo: "Dighe", Corpo: corpo})
}

func (b *builder) elencoContributori(ids []int, nomi map[int]string, per map[int][]*fotoPubblicata, casa voce) error {
	var v []voce
	for _, id := range ids {
		v = append(v, voce{URL: b.url("contributori", strconv.Itoa(id)), Testo: nomi[id], Mini: b.mini(per[id][0]), N: len(per[id])})
	}
	slices.SortFunc(v, func(a, b voce) int { return strings.Compare(strings.ToLower(a.Testo), strings.ToLower(b.Testo)) })
	corpo := corpoElenco{Sottotitolo: fmt.Sprintf("%d contributori.", len(v)), Gruppi: []gruppo{{Sottogruppi: []sottogruppo{{Voci: v}}}}}
	return b.scrivi("elenco", paginaSito{Titolo: "Contributori", Corpo: corpo, Briciole: []voce{casa}}, "contributori")
}

func (b *builder) elencoAltre(cids []int, altre map[int][]*fotoPubblicata, casa voce) error {
	var v []voce
	n := 0
	for _, cid := range cids {
		v = append(v, voce{URL: b.url("altre", strconv.Itoa(cid)), Testo: b.c.album.Categorie[cid].Nome, Mini: b.mini(altre[cid][0]), N: len(altre[cid])})
		n += len(altre[cid])
	}
	corpo := corpoElenco{
		Sottotitolo: fmt.Sprintf("%d foto non ancora collegate a una paginaSito del sito, divise per categoria di Coppermine.", n),
		Gruppi:      []gruppo{{Sottogruppi: []sottogruppo{{Voci: v}}}},
	}
	return b.scrivi("elenco", paginaSito{Titolo: "Da classificare", Corpo: corpo, Briciole: []voce{casa}}, "altre")
}

// statici scrive CSS, icona e l'Indice delle Foto per il Sito principale (ADR 0002).
func (b *builder) statici(perEntita map[string][]*fotoPubblicata) error {
	dir := filepath.Join(b.dir, "statico")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	for _, n := range []string{"stile.css", "documento.svg"} {
		d, err := modelli.ReadFile("modelli/" + n)
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(dir, n), d, 0o644); err != nil {
			return err
		}
	}
	indice := map[string][]int{}
	for s, fs := range perEntita {
		for _, f := range fs {
			indice[s] = append(indice[s], f.Pid)
		}
	}
	d, err := json.Marshal(indice) // le chiavi delle mappe escono ordinate
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(b.dir, "indice-foto.json"), d, 0o644)
}

func chiaviOrdinate[K int | string, V any](m map[K]V) []K {
	k := make([]K, 0, len(m))
	for x := range m {
		k = append(k, x)
	}
	slices.Sort(k)
	return k
}
