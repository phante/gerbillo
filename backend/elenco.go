package main

// Elenco delle Foto del Backend (#49): l'indice redazione.json, scritto dal build con i
// valori risolti delle Foto pubblicate, compresi i tag interni che la Gallery pubblica non
// mostra. Il Backend lo tiene in memoria e lo ricarica a fine build: l'elenco e i filtri
// non leggono mai i YAML. Non si calcolano risultati parziali durante un build.

import (
	"cmp"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime/debug"
	"slices"
	"strconv"
	"strings"
	"time"
)

// fotoIndice è una Foto nell'indice della redazione.
type fotoIndice struct {
	Pid          int      `json:"pid"`
	Titolo       string   `json:"titolo,omitempty"`
	Descrizione  string   `json:"descrizione,omitempty"`
	Slug         []string `json:"slug,omitempty"`
	Categorie    []string `json:"categorie,omitempty"`
	Tag          []string `json:"tag,omitempty"` // compresi gli interni
	Autori       []string `json:"autori,omitempty"`
	Datazione    string   `json:"datazione,omitempty"`
	User3        string   `json:"user3,omitempty"`
	Aid          int      `json:"aid"`
	Album        string   `json:"album,omitempty"`
	Contributore *struct {
		ID   int    `json:"id"`
		Nome string `json:"nome"`
	} `json:"contributore,omitempty"`
	Formato string `json:"formato"`
	Data    string `json:"data,omitempty"` // Data di caricamento, RFC 3339
	Mini    string `json:"mini"`           // URL della miniatura, come nella Gallery pubblica
}

type indiceRedazione struct {
	Generato time.Time    `json:"generato"`
	Foto     []fotoIndice `json:"foto"` // per album, poi per pid
	perPid   map[int]int
}

// scriviIndiceRedazione è la parte del build che scrive redazione.json (build -redazione).
func (b *builder) scriviIndiceRedazione(p string) error {
	idx := indiceRedazione{Generato: time.Now()}
	for _, f := range b.c.foto {
		v := fotoIndice{Pid: f.Pid, Titolo: f.Title, Descrizione: f.Description, Slug: f.Slug, Categorie: f.Categories,
			Tag: f.Tags, Autori: f.Authors, Datazione: f.Datazione, User3: f.Coppermine.Data, Aid: f.Coppermine.Aid,
			Album: b.c.album.Album[f.Coppermine.Aid].Title, Formato: f.Formato, Data: f.Date, Mini: b.mini(f)}
		if c := f.Contributore; c != nil {
			v.Contributore = &struct {
				ID   int    `json:"id"`
				Nome string `json:"nome"`
			}{c.ID, c.Nome}
		}
		idx.Foto = append(idx.Foto, v)
	}
	ordinaIndice(idx.Foto)
	d, err := json.Marshal(idx)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(p+".nuovo", d, 0o644); err != nil {
		return err
	}
	return os.Rename(p+".nuovo", p)
}

func ordinaIndice(fs []fotoIndice) {
	slices.SortFunc(fs, func(a, b fotoIndice) int { return cmp.Or(a.Aid-b.Aid, a.Pid-b.Pid) })
}

func leggiIndiceRedazione(p string) (*indiceRedazione, error) {
	d, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	idx := &indiceRedazione{}
	if err := json.Unmarshal(d, idx); err != nil {
		return nil, fmt.Errorf("%s: %w", p, err)
	}
	idx.perPid = make(map[int]int, len(idx.Foto))
	for i, f := range idx.Foto {
		idx.perPid[f.Pid] = i
	}
	return idx, nil
}

// caricaIndice legge l'indice più recente: quello dei build del Backend, altrimenti quello
// del seme. Senza nessuno dei due l'elenco è vuoto finché non finisce un build.
func (r *redazione) caricaIndice() error {
	var ultimo error
	for _, p := range []string{r.fileIndice, r.semeIndice} {
		if p == "" {
			continue
		}
		idx, err := leggiIndiceRedazione(p)
		if err == nil {
			r.elenco.Store(idx)
			// Il vecchio indice e il JSON letto sono qualche decina di MiB: si rendono subito
			// al sistema, perché il container ha 384 MiB (ADR 0011).
			debug.FreeOSMemory()
			return nil
		}
		ultimo = err
	}
	return ultimo
}

func (r *redazione) indiceCorrente() *indiceRedazione {
	if idx := r.elenco.Load(); idx != nil {
		return idx
	}
	return &indiceRedazione{}
}

// Filtri «Da sistemare»: ciascun tag interno, più due sulla Datazione.
const (
	senzaDatazione       = "senza-datazione"
	datazioneDaSistemare = "datazione-da-sistemare" // user3 c'è, ma nessuna Datazione: i residui di #39
	fotoPerPaginaBackend = 60
)

type daSistemare struct {
	chiave, nome, aiuto string
	vale                func(f *fotoIndice) bool
}

func filtriDaSistemare() []daSistemare {
	var d []daSistemare
	for _, t := range chiaviOrdinate(tagInterni) {
		d = append(d, daSistemare{t, t, descrizioneTagInterno(t), func(f *fotoIndice) bool { return slices.Contains(f.Tag, t) }})
	}
	return append(d,
		daSistemare{senzaDatazione, "senza Datazione", "", func(f *fotoIndice) bool { return f.Datazione == "" }},
		daSistemare{datazioneDaSistemare, "Datazione da sistemare", "user3 di Coppermine c'è, ma non è diventato una Datazione",
			func(f *fotoIndice) bool { return f.Datazione == "" && strings.TrimSpace(f.User3) != "" }})
}

// filtro è una vista dell'elenco, tutta nell'URL: indietro e ricarica la mantengono.
type filtro struct {
	sistemare []string // chiavi di filtriDaSistemare, tutte vere insieme
}

func leggiFiltro(q url.Values) filtro {
	var f filtro
	for _, s := range q["s"] {
		if slices.ContainsFunc(filtriDaSistemare(), func(d daSistemare) bool { return d.chiave == s }) && !slices.Contains(f.sistemare, s) {
			f.sistemare = append(f.sistemare, s)
		}
	}
	slices.Sort(f.sistemare)
	return f
}

func (f filtro) valori() url.Values {
	v := url.Values{}
	for _, s := range f.sistemare {
		v.Add("s", s)
	}
	return v
}

// query è la parte dopo il ? (vuota senza filtri), da aggiungere ai link.
func (f filtro) query() string {
	if q := f.valori().Encode(); q != "" {
		return "?" + q
	}
	return ""
}

func (f filtro) vale(x *fotoIndice) bool {
	for _, d := range filtriDaSistemare() {
		if slices.Contains(f.sistemare, d.chiave) && !d.vale(x) {
			return false
		}
	}
	return true
}

func (f filtro) applica(idx *indiceRedazione) []*fotoIndice {
	var out []*fotoIndice
	for i := range idx.Foto {
		if f.vale(&idx.Foto[i]) {
			out = append(out, &idx.Foto[i])
		}
	}
	return out
}

// conAltro è il filtro con s acceso o spento.
func (f filtro) conAltro(s string) filtro {
	g := filtro{sistemare: slices.Clone(f.sistemare)}
	if i := slices.Index(g.sistemare, s); i >= 0 {
		g.sistemare = slices.Delete(g.sistemare, i, i+1)
	} else {
		g.sistemare = append(g.sistemare, s)
		slices.Sort(g.sistemare)
	}
	return g
}

type voceFiltro struct {
	Chiave, Nome, Aiuto, Link string
	N                         int
	Attivo                    bool
}

// vociDaSistemare sono i filtri con quante Foto ha ciascuno insieme agli altri attivi.
func (f filtro) vociDaSistemare(idx *indiceRedazione) []voceFiltro {
	var v []voceFiltro
	for _, d := range filtriDaSistemare() {
		attivo := slices.Contains(f.sistemare, d.chiave)
		con := f
		if !attivo {
			con = f.conAltro(d.chiave)
		}
		n := 0
		for i := range idx.Foto {
			if con.vale(&idx.Foto[i]) {
				n++
			}
		}
		v = append(v, voceFiltro{Chiave: d.chiave, Nome: d.nome, Aiuto: d.aiuto, N: n, Attivo: attivo, Link: "foto/" + f.conAltro(d.chiave).query()})
	}
	return v
}

type miniaturaBackend struct {
	Pid                 int
	Titolo, Album, Mini string
	Link                string
}

type corpoElencoBackend struct {
	Filtri               []voceFiltro
	Foto                 []miniaturaBackend
	Totale, Pagina       int
	Pagine               int
	Indietro, Avanti     string
	Aggiornato, Versione string
	Avviso               string // build in corso o in coda
	Query                string
}

func (r *redazione) avvisoBuild() string {
	inCorso, inCoda, _ := r.statoDelBuild()
	switch {
	case inCoda:
		return "Build in corso, poi un altro: l'elenco si aggiorna alla fine."
	case inCorso:
		return "Build in corso: l'elenco si aggiorna alla fine."
	}
	return ""
}

// elencoFoto risponde a /admin/foto/: la griglia filtrata e paginata.
func (r *redazione) elencoFoto(w http.ResponseWriter, req *http.Request) {
	idx := r.indiceCorrente()
	f := leggiFiltro(req.URL.Query())
	trovate := f.applica(idx)
	c := corpoElencoBackend{Filtri: f.vociDaSistemare(idx), Totale: len(trovate), Query: f.query()}
	c.Pagine = max(1, (len(trovate)+fotoPerPaginaBackend-1)/fotoPerPaginaBackend)
	c.Pagina = min(max(1, atoi(req.URL.Query().Get("p"))), c.Pagine)
	pagina := func(k int) string {
		v := f.valori()
		if k > 1 {
			v.Set("p", strconv.Itoa(k))
		}
		if q := v.Encode(); q != "" {
			return "foto/?" + q
		}
		return "foto/"
	}
	if c.Pagina > 1 {
		c.Indietro = pagina(c.Pagina - 1)
	}
	if c.Pagina < c.Pagine {
		c.Avanti = pagina(c.Pagina + 1)
	}
	for _, x := range trovate[min((c.Pagina-1)*fotoPerPaginaBackend, len(trovate)):min(c.Pagina*fotoPerPaginaBackend, len(trovate))] {
		c.Foto = append(c.Foto, miniaturaBackend{Pid: x.Pid, Titolo: x.Titolo, Album: x.Album, Mini: x.Mini, Link: fmt.Sprintf("foto/%d%s", x.Pid, c.Query)})
	}
	c.Versione, c.Aggiornato, c.Avviso = versioneIndice(idx), aggiornatoAlle(idx), r.avvisoBuild()
	mostraPagina(w, req, "elenco", "Foto", c)
}

func versioneIndice(idx *indiceRedazione) string {
	return strconv.FormatInt(idx.Generato.UnixMilli(), 10)
}

func aggiornatoAlle(idx *indiceRedazione) string {
	if idx.Generato.IsZero() {
		return ""
	}
	return idx.Generato.In(roma).Format("02/01 15:04")
}

// freschezza risponde a /admin/foto/freschezza, che l'elenco chiede ogni 5 secondi con htmx:
// se l'indice è cambiato fa ricaricare la pagina, altrimenti rimanda l'avviso del build.
func (r *redazione) freschezza(w http.ResponseWriter, req *http.Request) {
	idx := r.indiceCorrente()
	if req.URL.Query().Get("v") != versioneIndice(idx) {
		w.Header().Set("HX-Refresh", "true")
		return
	}
	mostraFrammento(w, "freschezza", corpoElencoBackend{Aggiornato: aggiornatoAlle(idx), Avviso: r.avvisoBuild(), Versione: versioneIndice(idx)})
}

// vicine sono la Foto precedente e la successiva di pid nell'elenco filtrato, e la sua posizione.
func (r *redazione) vicine(pid int, f filtro) (prima, dopo, pos, tot int) {
	trovate := f.applica(r.indiceCorrente())
	i := slices.IndexFunc(trovate, func(x *fotoIndice) bool { return x.Pid == pid })
	if i < 0 {
		return 0, 0, 0, len(trovate)
	}
	if i > 0 {
		prima = trovate[i-1].Pid
	}
	if i+1 < len(trovate) {
		dopo = trovate[i+1].Pid
	}
	return prima, dopo, i + 1, len(trovate)
}

type corpoHome struct {
	Build      string
	Aggiornato string
	Filtri     []voceFiltro
	Totale     int
	Admin      bool
}

func atoi(s string) int { n, _ := strconv.Atoi(s); return n }

// descrizioneTagInterno è l'aiuto accanto a un tag interno.
func descrizioneTagInterno(t string) string {
	return map[string]string{
		"da-classificare": "manca la classificazione",
		"da-collegare":    "manca lo Slug",
		"da-separare":     "l'album copre più pagine: le Foto vanno divise",
	}[t]
}
