package main

// Modifica diretta di una Foto dal Backend (#43, ADR 0016): il Moderatore cambia i campi
// editoriali. Prima dello Switch si scrive in arricchimento/foto/<pid>.yaml solo ciò che
// differisce dall'Importazione, con un commit per salvataggio; dopo il congelamento (#40)
// la stessa funzione scriverà nel YAML della Foto, e il modulo non cambia. Dopo il commit
// parte un build completo in background; i salvataggi che arrivano mentre gira ne
// producono uno solo alla fine. La rigenerazione parziale è #15.

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"gopkg.in/yaml.v3"
)

// campoEditoriale è un campo del modulo. I valori si confrontano come testo: le liste
// hanno una voce per riga, il Punto geografico è "latitudine, longitudine".
type campoEditoriale struct {
	chiave    string // percorso nel YAML della Foto, con i punti
	etichetta string
	lista     bool
	righe     int // textarea; 0 = una riga sola
	valore    func(f *fotoPubblicata) string
}

var campiEditoriali = []campoEditoriale{
	{"title", "Titolo", false, 0, func(f *fotoPubblicata) string { return f.Title }},
	{"description", "Descrizione", false, 5, func(f *fotoPubblicata) string { return f.Description }},
	{"alt", "Testo alternativo", false, 2, func(f *fotoPubblicata) string { return f.Alt }},
	{"slug", "Pagine del sito (slug)", true, 3, func(f *fotoPubblicata) string { return strings.Join(f.Slug, "\n") }},
	{"categories", "Categorie", true, 3, func(f *fotoPubblicata) string { return strings.Join(f.Categories, "\n") }},
	{"tags", "Tag", true, 3, func(f *fotoPubblicata) string { return strings.Join(f.Tags, "\n") }},
	{"datazione", "Datazione", false, 0, func(f *fotoPubblicata) string { return f.Datazione }},
	{"authors", "Autori", true, 2, func(f *fotoPubblicata) string { return strings.Join(f.Authors, "\n") }},
	{"fonte", "Fonte", false, 0, func(f *fotoPubblicata) string { return f.Fonte }},
	{"geolocation.coords", "Punto geografico (latitudine, longitudine)", false, 0, func(f *fotoPubblicata) string {
		if c := f.Geolocation.Coords; c != nil {
			return strconv.FormatFloat(c.Latitudine, 'f', -1, 64) + ", " + strconv.FormatFloat(c.Longitudine, 'f', -1, 64)
		}
		return ""
	}},
}

// normalizza un valore del modulo: righe senza spazi ai bordi, liste senza righe vuote.
func (c campoEditoriale) normalizza(s string) string {
	s = strings.TrimSpace(strings.ReplaceAll(s, "\r\n", "\n"))
	if !c.lista {
		return s
	}
	var v []string
	for _, r := range strings.Split(s, "\n") {
		if r = strings.TrimSpace(r); r != "" {
			v = append(v, r)
		}
	}
	return strings.Join(v, "\n")
}

// yaml è il valore da scrivere nel file di arricchimento.
func (c campoEditoriale) yaml(s string) (any, error) {
	switch {
	case c.lista:
		return strings.Split(s, "\n"), nil
	case c.chiave == "geolocation.coords":
		lat, lon, ok := strings.Cut(s, ",")
		a, err1 := strconv.ParseFloat(strings.TrimSpace(lat), 64)
		b, err2 := strconv.ParseFloat(strings.TrimSpace(lon), 64)
		if !ok || err1 != nil || err2 != nil || a < -90 || a > 90 || b < -180 || b > 180 {
			return nil, fmt.Errorf("punto geografico %q: scrivi latitudine, longitudine (es. 46.2905, 8.2987)", s)
		}
		return map[string]any{"latitudine": a, "longitudine": b}, nil
	}
	return s, nil
}

type redazione struct {
	contenuti string       // Repository dei contenuti
	indice    string       // Indice delle Entità
	build     func() error // build completo della Gallery pubblica

	salvataggio sync.Mutex // un salvataggio alla volta

	// Indice della redazione (#49): fileIndice lo riscrivono i build del Backend, semeIndice
	// è quello del seme, finché non ne finisce uno.
	fileIndice, semeIndice string
	elenco                 atomic.Pointer[indiceRedazione]

	voc   *vocabolario // nil: le Categorie non si controllano (nessun vocabolario nel seme)
	cache cacheVoci    // voci note per i chip (#53)

	mu              sync.Mutex
	inCorso, ancora bool
	ultimo          *esitoBuild // nil finché non finisce il primo build
	dopoBuild       func()      // a fine build riuscito, per esempio per ricaricare l'indice
}

type esitoBuild struct {
	Riuscito bool
	Fine     time.Time
	Durata   time.Duration
	Errore   string
}

func (r *redazione) usaVocabolario(voc vocabolario) {
	r.voc = &voc
	for _, t := range voc.TagsInterni {
		descrizioniTagInterni[t.Nome] = t.Descrizione
	}
}

// richiediBuild lancia un build in background. Se ne gira già uno, ne segna un altro da
// fare alla fine: più richieste durante un build producono un solo build in più.
func (r *redazione) richiediBuild() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.inCorso {
		r.ancora = true
		return
	}
	r.inCorso = true
	go func() {
		for {
			inizio := time.Now()
			err := r.build()
			e := &esitoBuild{Riuscito: err == nil, Fine: time.Now(), Durata: time.Since(inizio).Round(time.Second)}
			if err != nil {
				e.Errore = err.Error()
				log.Printf("build: %v", err)
			} else if r.dopoBuild != nil {
				r.dopoBuild()
			}
			r.mu.Lock()
			r.ultimo = e
			if !r.ancora {
				r.inCorso = false
				r.mu.Unlock()
				return
			}
			r.ancora = false
			r.mu.Unlock()
		}
	}()
}

// statoDelBuild dice se un build gira, se ne segue un altro e com'è andato l'ultimo.
func (r *redazione) statoDelBuild() (inCorso, inCoda bool, ultimo *esitoBuild) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.inCorso, r.inCorso && r.ancora, r.ultimo
}

func (r *redazione) statoBuild() string {
	inCorso, inCoda, e := r.statoDelBuild()
	switch {
	case inCoda:
		return "in corso, poi un altro"
	case inCorso:
		return "in corso"
	case e == nil:
		return "nessuno da quando il Backend è partito"
	case !e.Riuscito:
		return "ultimo fallito: " + e.Errore
	}
	return fmt.Sprintf("ultimo riuscito alle %s, in %s", e.Fine.In(roma).Format("15:04"), e.Durata)
}

// fotoDaModificare è la Foto pubblicata e quella importata, con cui si confronta.
func (r *redazione) fotoDaModificare(pid int) (pub, imp *fotoPubblicata, idx []entita, err error) {
	var album albumCoppermine
	st, idx, err := leggiStrati(r.contenuti, r.indice, nuovoLivello(r.contenuti, ""), &album)
	if err != nil {
		return nil, nil, nil, err
	}
	p := filepath.Join(r.contenuti, "importazione", "foto", strconv.Itoa(pid)+".yaml")
	if imp, err = st.importata(p); err != nil {
		return nil, nil, nil, err
	}
	if pub, err = st.importata(p); err != nil {
		return nil, nil, nil, err
	}
	return pub, imp, idx, st.arricchisci(pub)
}

// salva scrive i valori del modulo e ne fa il commit con autore u. Restituisce i campi
// cambiati rispetto alla Foto pubblicata; nessuno vuol dire nessun commit.
func (r *redazione) salva(pid int, modulo map[string]string, u *utente) ([]string, error) {
	r.salvataggio.Lock()
	defer r.salvataggio.Unlock()
	pub, imp, idx, err := r.fotoDaModificare(pid)
	if err != nil {
		return nil, err
	}
	rel := filepath.Join("arricchimento", fileFoto(pid))
	p := filepath.Join(r.contenuti, rel)
	vecchio, err := os.ReadFile(p)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	arr := map[string]any{}
	if err := yaml.Unmarshal(vecchio, &arr); err != nil {
		return nil, fmt.Errorf("%s: %w", rel, err)
	}
	if arr == nil {
		arr = map[string]any{}
	}
	var cambiati []string
	for _, c := range campiEditoriali {
		v := c.normalizza(modulo[c.chiave])
		if c.chiave == "datazione" {
			if v, err = normalizzaDatazione(v); err != nil {
				return nil, err
			}
		}
		if v != c.valore(pub) {
			cambiati = append(cambiati, c.chiave)
		}
		percorso := strings.Split(c.chiave, ".")
		if v == c.valore(imp) {
			togli(arr, percorso) // il valore importato non si scrive
			continue
		}
		if c.chiave == "categories" && r.voc != nil {
			for _, k := range strings.Split(v, "\n") {
				if k != "" && !slices.Contains(r.voc.Categories, k) {
					return nil, fmt.Errorf("Categoria %q non è nel vocabolario (dati/vocabolario.yaml)", k)
				}
			}
		}
		if c.chiave == "slug" {
			for _, s := range strings.Split(v, "\n") {
				if s != "" && !slices.ContainsFunc(idx, func(e entita) bool { return e.Slug == s }) {
					return nil, fmt.Errorf("slug %q non è nell'Indice delle Entità", s)
				}
			}
		}
		// Vuoto ma diverso dall'Importazione: si scrive vuoto, per toglierlo.
		var y any = ""
		switch {
		case v == "" && c.lista:
			y = []string{}
		case v == "" && len(percorso) > 1:
			y = nil // coords: null
		case v != "":
			if y, err = c.yaml(v); err != nil {
				return nil, err
			}
		}
		metti(arr, percorso, y)
	}
	if len(cambiati) == 0 {
		return nil, nil // nemmeno si riscrive il file
	}
	var nuovo []byte
	if len(arr) > 0 {
		if nuovo, err = yaml.Marshal(arr); err != nil {
			return nil, err
		}
	}
	if bytes.Equal(nuovo, vecchio) {
		return nil, nil
	}
	if len(nuovo) == 0 {
		err = os.Remove(p)
	} else if err = os.MkdirAll(filepath.Dir(p), 0o755); err == nil {
		err = os.WriteFile(p, nuovo, 0o644)
	}
	if err != nil {
		return nil, err
	}
	msg := fmt.Sprintf("foto %d: %s\n\nModifica diretta dal Backend (#43).", pid, strings.Join(cambiati, ", "))
	if err := registra(r.contenuti, rel, msg, autoreCommit(u)); err != nil {
		return nil, err
	}
	return cambiati, nil
}

// autoreCommit è l'autore git di una modifica: il nome dell'utente del forum, con un
// indirizzo che non esiste (.invalid) ma che porta il suo user_id.
func autoreCommit(u *utente) string {
	return fmt.Sprintf("%s <utente-%d@gerbillo.invalid>", u.Nome, u.ID)
}

func metti(m map[string]any, percorso []string, v any) {
	for _, k := range percorso[:len(percorso)-1] {
		sotto, ok := m[k].(map[string]any)
		if !ok {
			sotto = map[string]any{}
			m[k] = sotto
		}
		m = sotto
	}
	m[percorso[len(percorso)-1]] = v
}

// togli toglie la chiave e le mappe rimaste vuote.
func togli(m map[string]any, percorso []string) {
	if len(percorso) == 1 {
		delete(m, percorso[0])
		return
	}
	if sotto, ok := m[percorso[0]].(map[string]any); ok {
		togli(sotto, percorso[1:])
		if len(sotto) == 0 {
			delete(m, percorso[0])
		}
	}
}

type voceModulo struct {
	Chiave, Etichetta, Valore, Importato string
	Righe                                int
	Diverso                              bool
	Anteprima                            anteprimaDatazione // solo per la Datazione
	User3, User3EDTF                     string             // idem: il suggerimento da Coppermine
	Gruppi                               []gruppoCategorie  // solo per le Categorie, dal vocabolario
	TagInterni                           []tagInterno       // solo per i Tag: l'aiuto accanto
	Chip                                 string             // slug, tags, authors: campo a chip (#53)
	Opzioni                              []opzioneChip      // i valori già scelti del campo a chip
}

// gruppoCategorie sono le Categorie del vocabolario con lo stesso primo pezzo di percorso.
type gruppoCategorie struct {
	Nome    string
	Opzioni []opzione
}

type opzione struct {
	Valore, Testo string
	Scelta        bool
}

// gruppiCategorie raggruppa il vocabolario per percorso; le Categorie scelte che non ci
// sono restano visibili, così si possono togliere.
func gruppiCategorie(voc *vocabolario, scelte []string) []gruppoCategorie {
	var g []gruppoCategorie
	aggiungi := func(gruppo, valore, testo string) {
		if len(g) == 0 || g[len(g)-1].Nome != gruppo {
			g = append(g, gruppoCategorie{Nome: gruppo})
		}
		o := &g[len(g)-1].Opzioni
		*o = append(*o, opzione{valore, testo, slices.Contains(scelte, valore)})
	}
	for _, c := range voc.Categories {
		primo, resto, _ := strings.Cut(c, "/")
		if resto == "" {
			resto = primo
		}
		aggiungi(primo, c, strings.ReplaceAll(resto, "/", " › "))
	}
	for _, c := range scelte {
		if c != "" && !slices.Contains(voc.Categories, c) {
			aggiungi("fuori dal vocabolario", c, c)
		}
	}
	return g
}

// corpoFotoBackend è la scheda di una Foto: il modulo e l'esito dell'ultimo salvataggio.
type corpoFotoBackend struct {
	Pid             int
	Pubblica, Build string
	Errore, Salvato string
	Voci            []voceModulo
	// Navigazione nell'elenco filtrato da cui si è aperta la scheda (#49).
	Elenco, Prima, Dopo string
	Pos, Tot            int
}

// foto risponde a /admin/foto/<pid>: la scheda in GET, il salvataggio in POST. Il modulo
// salva con htmx: la risposta è il solo modulo, con i valori salvati e l'esito.
func (r *redazione) foto(w http.ResponseWriter, req *http.Request) {
	resto := strings.Trim(strings.TrimPrefix(req.URL.Path, "/admin/foto"), "/")
	switch resto {
	case "":
		if q := strings.TrimSpace(req.URL.Query().Get("pid")); q != "" {
			rimanda(w, url.PathEscape(q))
			return
		}
		r.elencoFoto(w, req)
		return
	case "freschezza":
		r.freschezza(w, req)
		return
	}
	pid, err := strconv.Atoi(resto)
	if err != nil {
		http.NotFound(w, req)
		return
	}
	c := corpoFotoBackend{Pid: pid, Pubblica: fmt.Sprintf("../f/%d/", pid)}
	{
		f := leggiFiltro(req.URL.Query())
		prima, dopo, pos, tot := r.vicine(pid, f)
		c.Elenco, c.Pos, c.Tot = "foto/"+f.query(), pos, tot
		if prima > 0 {
			c.Prima = fmt.Sprintf("foto/%d%s", prima, f.query())
		}
		if dopo > 0 {
			c.Dopo = fmt.Sprintf("foto/%d%s", dopo, f.query())
		}
	}
	modulo := map[string]string{}
	if req.Method == http.MethodPost {
		req.ParseForm()
		for _, k := range campiEditoriali {
			// Le liste arrivano come più valori (select multiple, chip) o come righe di un testo.
			modulo[k.chiave] = strings.Join(req.PostForm[k.chiave], "\n")
		}
		cambiati, err := r.salva(pid, modulo, utenteDi(req))
		switch {
		case err != nil:
			c.Errore = "Non salvato: " + err.Error()
		case len(cambiati) == 0:
			c.Salvato = "Nessun campo cambiato."
			modulo = map[string]string{}
		default:
			r.richiediBuild()
			c.Salvato = "Salvato: " + strings.Join(cambiati, ", ") + ". La pagina della Foto cambia dopo il build."
			modulo = map[string]string{}
		}
	}
	pub, imp, _, err := r.fotoDaModificare(pid)
	if errors.Is(err, os.ErrNotExist) {
		http.NotFound(w, req)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for _, k := range campiEditoriali {
		v := k.valore(pub)
		if inviato, ok := modulo[k.chiave]; ok {
			v = inviato // dopo un errore si ripropone quello che si era scritto
		}
		vm := voceModulo{Chiave: k.chiave, Etichetta: k.etichetta, Valore: v, Importato: k.valore(imp),
			Righe: k.righe, Diverso: k.valore(pub) != k.valore(imp)}
		if k.lista && k.chiave != "categories" {
			vm.Chip, vm.Opzioni = k.chiave, r.opzioniChip(k.chiave, strings.Split(v, "\n"))
		}
		if k.chiave == "categories" && r.voc != nil {
			vm.Gruppi = gruppiCategorie(r.voc, strings.Split(v, "\n"))
		}
		if k.chiave == "tags" && r.voc != nil {
			vm.TagInterni = r.voc.TagsInterni
		}
		if k.chiave == "datazione" {
			vm.Anteprima = anteprimaDi(v)
			if pub.Datazione == "" && strings.TrimSpace(pub.Coppermine.Data) != "" {
				vm.User3 = pub.Coppermine.Data
				if d, ok := interpretaUser3(pub.Coppermine.Data, annoCaricamento(pub)); ok {
					vm.User3EDTF = d.edtf
				}
			}
		}
		c.Voci = append(c.Voci, vm)
	}
	c.Build = r.statoBuild()
	if req.Method == http.MethodPost && daHtmx(req) {
		mostraFrammento(w, "modulo", c)
		return
	}
	mostraPagina(w, req, "foto", fmt.Sprintf("Foto %d", pid), c)
}

// anteprimaDatazione è la resa in italiano della Datazione mentre si scrive (#51).
type anteprimaDatazione struct {
	Testo, EDTF, Errore string
}

func anteprimaDi(v string) anteprimaDatazione {
	if strings.TrimSpace(v) == "" {
		return anteprimaDatazione{}
	}
	e, err := normalizzaDatazione(v)
	if err != nil {
		return anteprimaDatazione{Errore: "Non valida: non si salva."}
	}
	a := anteprimaDatazione{Testo: datazioneItaliana(e)}
	if e != strings.TrimSpace(v) {
		a.EDTF = e // forma italiana: si salva in EDTF
	}
	return a
}

// datazione risponde a /admin/datazione?datazione=…, che il campo chiede mentre si scrive.
func datazione(w http.ResponseWriter, req *http.Request) {
	mostraFrammento(w, "anteprima-datazione", anteprimaDi(req.URL.Query().Get("datazione")))
}

func annoCaricamento(f *fotoPubblicata) int {
	if t, err := time.Parse(time.RFC3339, f.Date); err == nil {
		return t.Year()
	}
	return time.Now().Year()
}

// rimanda con un Location relativo: http.Redirect lo renderebbe assoluto, e dietro nginx
// si perderebbe il prefisso /gerbillo/ dell'anteprima.
func rimanda(w http.ResponseWriter, dove string) {
	w.Header().Set("Location", dove)
	w.WriteHeader(http.StatusSeeOther)
}

// admin è la home del Backend.
func (r *redazione) admin(w http.ResponseWriter, req *http.Request) {
	if req.URL.Path != "/admin/" {
		http.NotFound(w, req)
		return
	}
	idx := r.indiceCorrente()
	mostraPagina(w, req, "home", "Backend", corpoHome{Build: r.statoBuild(), Aggiornato: aggiornatoAlle(idx),
		Filtri: filtro{}.vociDaSistemare(idx), Totale: len(idx.Foto), Admin: utenteDi(req).Admin()})
}

// ricalcola rilancia il build completo, e con lui l'indice. Solo per l'admin.
func (r *redazione) ricalcola(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "serve un POST", http.StatusMethodNotAllowed)
		return
	}
	if !utenteDi(req).Admin() {
		http.Error(w, "serve l'admin", http.StatusForbidden)
		return
	}
	r.richiediBuild()
	fmt.Fprint(w, `<span class="nota" role="status">Build richiesto: segui il widget in alto.</span>`)
}

// redazioneAnteprima prepara la modifica diretta nell'anteprima (ADR 0011). L'immagine ha in
// seme/ l'Importazione dei dati di test, l'Indice delle Entità, dati/arricchimento e gli
// argomenti del build della CI. All'avvio il Backend crea in /tmp un Repository dei
// contenuti con il solo Livello di arricchimento (l'Importazione resta nell'immagine), e i
// build scrivono in /tmp solo i file diversi dalla Gallery dell'immagine. Senza seme/
// la modifica non c'è.
func redazioneAnteprima(seme, contenuti, dist, modifiche, indiceRed string) (*redazione, error) {
	if !esiste(filepath.Join(seme, "indice.json")) {
		return nil, nil
	}
	if !esiste(filepath.Join(contenuti, ".git")) {
		if err := os.MkdirAll(contenuti, 0o755); err != nil {
			return nil, err
		}
		if err := os.Symlink(filepath.Join(seme, "importazione"), filepath.Join(contenuti, "importazione")); err != nil {
			return nil, err
		}
		if err := os.CopyFS(filepath.Join(contenuti, "arricchimento"), os.DirFS(filepath.Join(seme, "arricchimento"))); err != nil {
			return nil, err
		}
		if err := registra(contenuti, "arricchimento", "anteprima: Livello di arricchimento da dati/arricchimento", ""); err != nil {
			return nil, err
		}
	}
	f, err := os.Open(filepath.Join(seme, "build.args"))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	args := []string{"build", "-contenuti", contenuti, "-indice", filepath.Join(seme, "indice.json"), "-dist", modifiche, "-confronta", dist,
		"-redazione", indiceRed}
	for sc := bufio.NewScanner(f); sc.Scan(); {
		args = append(args, sc.Text())
	}
	eseguibile, err := os.Executable()
	if err != nil {
		return nil, err
	}
	r := &redazione{contenuti: contenuti, indice: filepath.Join(seme, "indice.json"),
		fileIndice: indiceRed, semeIndice: filepath.Join(seme, "redazione.json")}
	if err := r.caricaIndice(); err != nil {
		log.Printf("indice della redazione: %v", err)
	}
	if voc, err := leggiVocabolario(filepath.Join(seme, "vocabolario.yaml")); err != nil {
		log.Printf("vocabolario: %v; le Categorie non si controllano", err)
	} else {
		r.usaVocabolario(voc)
	}
	r.dopoBuild = func() {
		if err := r.caricaIndice(); err != nil {
			log.Printf("indice della redazione: %v", err)
		}
	}
	r.build = func() error {
		cmd := exec.Command(eseguibile, args...)
		cmd.Stdout, cmd.Stderr = os.Stderr, os.Stderr
		return cmd.Run()
	}
	return r, nil
}
