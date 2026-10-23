package main

// Mappatura degli Album di Coppermine verso gli Slug (G2, #10). Le proposte automatiche
// vanno in arricchimento/mappature.yaml nel Repository dei contenuti, accanto alle righe
// scritte a mano, che non vengono mai toccate. Per ora solo la categoria Dighe.

import (
	"bytes"
	"cmp"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/BurntSushi/toml"
	"gopkg.in/yaml.v3"
)

const cidDighe = 2

const (
	automatica = "automatica" // proposta dal confronto dei nomi
	manuale    = "manuale"    // scritta o confermata a mano: mai riscritta
	daFare     = "da-fare"    // nessun abbinamento certo
)

// Le chiavi sono in ordine alfabetico, come nei YAML delle Foto.
type mappatura struct {
	Candidati  []string `yaml:"candidati,omitempty"`
	Categories []string `yaml:"categories,omitempty"`
	Coppermine string   `yaml:"coppermine"` // percorso dell'album, solo per orientarsi
	Origine    string   `yaml:"origine"`
	Slug       []string `yaml:"slug,omitempty"`
	Tags       []string `yaml:"tags,omitempty"`
}

type fileMappature struct {
	Album     map[int]*mappatura `yaml:"album"`
	Categorie map[int]*mappatura `yaml:"categorie"`
}

const intestazioneMappature = `# Mappature degli Album di Coppermine (G2, #10; ADR 0009, 0013).
# Le righe "origine: automatica" e "da-fare" le riscrive gerbillo mappa.
# Per correggerne o completarne una, scrivi "origine: manuale": da lì non viene più toccata.
# Un album senza riga eredita la Mappatura della sua categoria. I commenti non si conservano.

`

func mappa(args []string) error {
	fs := flag.NewFlagSet("mappa", flag.ExitOnError)
	conf := fs.String("config", "gerbillo.toml", "file di configurazione TOML")
	indice := fs.String("indice", "", "Indice delle Entità (JSON)")
	manuali := fs.String("manuali", "", "righe manuali da aggiungere (es. dati/mappature-manuali.yaml): vincono su quelle esistenti")
	fs.Parse(args)
	var c configImporta
	if _, err := toml.DecodeFile(*conf, &c); err != nil {
		return err
	}
	if c.Contenuti == "" || *indice == "" {
		return errors.New("servono contenuti nella configurazione e -indice")
	}
	idx, err := leggiIndice(*indice)
	if err != nil {
		return err
	}
	b, err := os.ReadFile(filepath.Join(c.Contenuti, "importazione", "album.yaml"))
	if err != nil {
		return fmt.Errorf("manca l'Importazione: %w", err)
	}
	var cpg albumCoppermine
	if err := yaml.Unmarshal(b, &cpg); err != nil {
		return err
	}
	p := filepath.Join(c.Contenuti, "arricchimento", "mappature.yaml")
	var vecchie fileMappature
	if b, err := os.ReadFile(p); err == nil {
		if err := yaml.Unmarshal(b, &vecchie); err != nil {
			return fmt.Errorf("%s: %w", p, err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if *manuali != "" {
		b, err := os.ReadFile(*manuali)
		if err != nil {
			return err
		}
		var m fileMappature
		if err := yaml.Unmarshal(b, &m); err != nil {
			return fmt.Errorf("%s: %w", *manuali, err)
		}
		vecchie.Album, vecchie.Categorie = unisciManuali(vecchie.Album, m.Album), unisciManuali(vecchie.Categorie, m.Categorie)
	}
	nuove, rap := abbina(cpg, idx, vecchie)
	out, err := yaml.Marshal(nuove)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(p, append([]byte(intestazioneMappature), out...), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(c.Contenuti, "arricchimento", "rapporto-mappature.md"), rap.markdown(), 0o644); err != nil {
		return err
	}
	fmt.Printf("album Dighe: %d, abbinati in automatico %.1f%% (foto %.1f%%)\n",
		rap.album, perc(rap.autoAlbum, rap.album), perc(rap.autoFoto, rap.foto))
	return registra(c.Contenuti, "arricchimento", fmt.Sprintf("mappature: %d album Dighe, %d da fare", rap.album, len(rap.daFare)))
}

func unisciManuali(dst, src map[int]*mappatura) map[int]*mappatura {
	if dst == nil {
		dst = map[int]*mappatura{}
	}
	for k, v := range src {
		if v.Origine == manuale {
			dst[k] = v
		}
	}
	return dst
}

func perc(a, b int) float64 {
	if b == 0 {
		return 0
	}
	return 100 * float64(a) / float64(b)
}

type voceRapporto struct {
	aid, foto int
	percorso  string
	candidati []string
}

type rapportoMappature struct {
	album, foto                              int
	autoAlbum, autoFoto, eredAlbum, eredFoto int
	manAlbum, manFoto                        int
	daFare, ambigui                          []voceRapporto
	sconosciuti                              []string // "album 12: dighe/xxx"
}

// abbina produce le Mappature: le righe manuali restano, le altre si rigenerano.
func abbina(cpg albumCoppermine, idx []entita, vecchie fileMappature) (fileMappature, *rapportoMappature) {
	chiavi := chiaviEntita(idx)
	noti := map[string]bool{}
	for _, e := range idx {
		noti[e.Slug] = true
	}
	percorso := func(cid int) []string {
		var p []string
		for cid != 0 && len(p) < 10 {
			c, ok := cpg.Categorie[cid]
			if !ok {
				break
			}
			p = append([]string{c.Nome}, p...)
			cid = c.Parent
		}
		return p
	}
	radice := func(cid int) int {
		for i := 0; i < 10; i++ {
			c, ok := cpg.Categorie[cid]
			if !ok || c.Parent == 0 {
				return cid
			}
			cid = c.Parent
		}
		return cid
	}
	out := fileMappature{Album: map[int]*mappatura{}, Categorie: map[int]*mappatura{}}
	rap := &rapportoMappature{}
	controlla := func(dove string, m *mappatura) {
		for _, s := range m.Slug {
			if !noti[s] {
				rap.sconosciuti = append(rap.sconosciuti, dove+": "+s)
			}
		}
	}

	// Categorie: solo dal terzo livello sotto Dighe (Dighe > Italia > regione > diga),
	// perché stati e regioni non sono dighe.
	for cid, c := range cpg.Categorie {
		p := percorso(cid)
		if radice(cid) != cidDighe || len(p) < 4 {
			continue
		}
		desc := strings.Join(p, " / ")
		if v := vecchie.Categorie[cid]; v != nil && v.Origine == manuale {
			v.Coppermine = desc
			out.Categorie[cid] = v
			controlla(fmt.Sprintf("categoria %d", cid), v)
			continue
		}
		if s, certo := trova(c.Nome, p, chiavi); certo && len(s) == 1 {
			out.Categorie[cid] = &mappatura{Coppermine: desc, Origine: automatica, Slug: s}
		}
	}

	// ereditata è la Mappatura della categoria antenata più vicina che ha uno Slug.
	ereditata := func(cid int) *mappatura {
		for i := 0; cid != 0 && i < 10; i++ {
			if m := out.Categorie[cid]; m != nil && len(m.Slug) > 0 {
				return m
			}
			cid = cpg.Categorie[cid].Parent
		}
		return nil
	}
	for aid, a := range cpg.Album {
		if radice(a.Categoria) != cidDighe {
			continue
		}
		p := percorso(a.Categoria)
		desc := strings.Join(append(p, a.Title), " / ")
		rap.album++
		rap.foto += a.Foto
		if v := vecchie.Album[aid]; v != nil && v.Origine == manuale {
			v.Coppermine = desc
			out.Album[aid] = v
			rap.manAlbum++
			rap.manFoto += a.Foto
			controlla(fmt.Sprintf("album %d", aid), v)
			continue
		}
		s, certo := trova(a.Title, p, chiavi)
		ered := ereditata(a.Categoria)
		switch {
		case certo && len(s) == 1:
			out.Album[aid] = &mappatura{Coppermine: desc, Origine: automatica, Slug: s}
			rap.autoAlbum++
			rap.autoFoto += a.Foto
		case ered != nil:
			// Sottoalbum di una diga ("La Forra" sotto "Vajont"): eredita, nessuna riga.
			rap.eredAlbum++
			rap.eredFoto += a.Foto
			if ered.Origine == automatica {
				rap.autoAlbum++
				rap.autoFoto += a.Foto
			}
		default:
			out.Album[aid] = &mappatura{Coppermine: desc, Origine: daFare, Candidati: s}
			v := voceRapporto{aid, a.Foto, desc, s}
			if certo && len(s) > 1 {
				rap.ambigui = append(rap.ambigui, v)
			} else {
				rap.daFare = append(rap.daFare, v)
			}
		}
	}
	ordina := func(v []voceRapporto) {
		slices.SortFunc(v, func(a, b voceRapporto) int { return cmp.Or(b.foto-a.foto, a.aid-b.aid) })
	}
	ordina(rap.daFare)
	ordina(rap.ambigui)
	slices.Sort(rap.sconosciuti)
	return out, rap
}

// chiaviEntita indicizza le Entità per nome normalizzato: titolo e nome dello slug,
// senza il suffisso numerico degli omonimi (montagna-spaccata-1).
func chiaviEntita(idx []entita) map[string][]*entita {
	k := map[string][]*entita{}
	agg := func(chiave string, e *entita) {
		if chiave != "" && !slices.Contains(k[chiave], e) {
			k[chiave] = append(k[chiave], e)
		}
	}
	for i := range idx {
		e := &idx[i]
		nome := e.Slug[strings.LastIndex(e.Slug, "/")+1:]
		agg(normNome(e.Title), e)
		for _, alt := range reAlternativa.Split(e.Title, -1) { // "Lumiei o di La Maina di Sauris"
			agg(normNome(alt), e)
		}
		agg(normNome(nome), e)
		agg(reOmonimo.ReplaceAllString(nome, ""), e)
	}
	return k
}

var (
	reParentesi   = regexp.MustCompile(`\([^)]*\)`)
	reSeparati    = regexp.MustCompile(`\s+[-–/]\s+|,\s*`)
	reNonAlnum    = regexp.MustCompile(`[^a-z0-9]+`)
	reOmonimo     = regexp.MustCompile(`-\d+$`)
	reAlternativa = regexp.MustCompile(`\s+o\s+(di\s+)?`)
	reMultipla    = regexp.MustCompile(`[-.]|\s+e\s+`) // "Arno-Campellio-Isola", "Pian Sapeio e Giacopiane"
	reLocalita    = regexp.MustCompile(`-(di|del|della|in|sul|sulla|presso)-.*$`)
	reArticolo    = regexp.MustCompile(`^(diga|lago|invaso|serbatoio|traversa|sbarramento)(-(di|del|della|dello|dei|delle|degli|d|dell))?-`)
	accenti       = strings.NewReplacer("à", "a", "á", "a", "â", "a", "ä", "a", "è", "e", "é", "e", "ê", "e", "ë", "e",
		"ì", "i", "í", "i", "î", "i", "ï", "i", "ò", "o", "ó", "o", "ô", "o", "ö", "o",
		"ù", "u", "ú", "u", "û", "u", "ü", "u", "ç", "c", "ñ", "n", "ß", "ss", "'", "", "’", "")
)

// normNome riduce un nome a una chiave confrontabile: "Diga di Pont'Ventoux (TO)" → "pontventoux".
func normNome(s string) string {
	s = reParentesi.ReplaceAllString(s, " ")
	s = accenti.Replace(strings.ToLower(s))
	s = strings.Trim(reNonAlnum.ReplaceAllString(s, "-"), "-")
	return reArticolo.ReplaceAllString(s, "")
}

// trova cerca le dighe che corrispondono a un nome Coppermine: prima il nome intero,
// poi le sue parti ("Barcis - Ponte Antoi"). Più candidati si separano con stato e regione.
// Se nessuno corrisponde, prova il nome accorciato ("S. Caterina di Auronzo" → santa-caterina):
// quei candidati non sono certi e vanno confermati a mano.
func trova(nome string, percorso []string, chiavi map[string][]*entita) ([]string, bool) {
	var trovate []*entita
	aggiungi := func(k string) {
		for _, e := range chiavi[k] {
			if !slices.Contains(trovate, e) {
				trovate = append(trovate, e)
			}
		}
	}
	parti := reSeparati.Split(reParentesi.ReplaceAllString(nome, " "), -1)
	certo := true
	if aggiungi(normNome(nome)); len(trovate) == 0 {
		for _, parte := range parti {
			aggiungi(normNome(parte))
		}
	}
	if len(trovate) == 0 {
		certo = false
		for _, parte := range append(reMultipla.Split(nome, -1), append(parti, nome)...) {
			k := reLocalita.ReplaceAllString(normNome(parte), "")
			aggiungi(k)
			if r, ok := strings.CutPrefix(k, "s-"); ok {
				aggiungi("santa-" + r)
				aggiungi("san-" + r)
				aggiungi("santo-" + r)
			}
		}
	}
	if len(trovate) > 1 {
		stato, regione := luogo(percorso)
		var stessa []*entita
		for _, e := range trovate {
			if normLuogo(e.Geolocation.Stato) == stato && (regione == "" || normLuogo(e.Geolocation.Regione) == regione) {
				stessa = append(stessa, e)
			}
		}
		if len(stessa) > 0 {
			trovate = stessa
		}
	}
	s := make([]string, len(trovate))
	for i, e := range trovate {
		s[i] = e.Slug
	}
	slices.Sort(s)
	return s, certo
}

// luogo ricava stato e regione dal percorso Coppermine: Dighe > Italia > Piemonte > …
func luogo(p []string) (stato, regione string) {
	if len(p) > 1 {
		stato = normLuogo(strings.SplitN(p[1], " - ", 2)[0])
	}
	if stato == "italia" && len(p) > 2 {
		regione = normLuogo(p[2])
	}
	return
}

// normLuogo rende confrontabili "Friuli Venezia-Giulia" e "Friuli-Venezia Giulia".
func normLuogo(s string) string {
	return reNonAlnum.ReplaceAllString(accenti.Replace(strings.ToLower(s)), "")
}

func (r *rapportoMappature) markdown() []byte {
	var b bytes.Buffer
	p := func(f string, a ...any) { fmt.Fprintf(&b, f, a...) }
	p("# Mappature: Dighe\n\nAlbum sotto Dighe: %d, con %d Foto.\n\n", r.album, r.foto)
	p("| | Album | Foto |\n|---|---:|---:|\n")
	p("| abbinati in automatico | %d (%.1f%%) | %d (%.1f%%) |\n", r.autoAlbum, perc(r.autoAlbum, r.album), r.autoFoto, perc(r.autoFoto, r.foto))
	p("| di cui ereditati da una categoria | %d | %d |\n", r.eredAlbum, r.eredFoto)
	p("| manuali | %d | %d |\n", r.manAlbum, r.manFoto)
	nf := func(v []voceRapporto) (n int) {
		for _, x := range v {
			n += x.foto
		}
		return
	}
	p("| ambigui | %d | %d |\n", len(r.ambigui), nf(r.ambigui))
	p("| da fare | %d | %d |\n\n", len(r.daFare), nf(r.daFare))
	if len(r.sconosciuti) > 0 {
		p("## Slug sconosciuti nelle righe manuali\n\n")
		for _, s := range r.sconosciuti {
			p("- %s\n", s)
		}
		p("\n")
	}
	p("## Ambigui\n\nPiù dighe con lo stesso nome nella stessa regione.\n\n")
	for _, v := range r.ambigui {
		p("- %d (%d foto) %s → %s\n", v.aid, v.foto, v.percorso, strings.Join(v.candidati, ", "))
	}
	p("\n## Da fare\n\nIn ordine di Foto: i primi sono gli album principali. Dopo la freccia, i candidati da confermare.\n\n")
	for _, v := range r.daFare {
		p("- %d (%d foto) %s", v.aid, v.foto, v.percorso)
		if len(v.candidati) > 0 {
			p(" → %s", strings.Join(v.candidati, ", "))
		}
		p("\n")
	}
	return b.Bytes()
}
