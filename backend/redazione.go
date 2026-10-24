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
	"html/template"
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
	{"datazione", "Datazione (EDTF)", false, 0, func(f *fotoPubblicata) string { return f.Datazione }},
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

	mu              sync.Mutex
	inCorso, ancora bool
	esito           string // dell'ultimo build, per le pagine
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
			esito := fmt.Sprintf("riuscito alle %s, in %s", time.Now().In(roma).Format("15:04"), time.Since(inizio).Round(time.Second))
			if err != nil {
				esito = "fallito: " + err.Error()
				log.Printf("build: %v", err)
			}
			r.mu.Lock()
			r.esito = esito
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

func (r *redazione) statoBuild() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	switch {
	case r.inCorso && r.ancora:
		return "in corso, poi un altro"
	case r.inCorso:
		return "in corso"
	case r.esito != "":
		return "ultimo " + r.esito
	}
	return "nessuno da quando il Backend è partito"
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
		if v != c.valore(pub) {
			cambiati = append(cambiati, c.chiave)
		}
		percorso := strings.Split(c.chiave, ".")
		if v == c.valore(imp) {
			togli(arr, percorso) // il valore importato non si scrive
			continue
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
}

var paginaRedazione = template.Must(template.New("r").Parse(`<!doctype html>
<meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">
<meta name="robots" content="noindex">
<title>{{.Titolo}}</title>
<style>
body{font:16px/1.4 system-ui,sans-serif;max-width:40em;margin:0 auto;padding:0 16px 3em;color:#222;background:#fff}
label{display:block;margin-top:1em;font-weight:600}
input,textarea,button{width:100%;box-sizing:border-box;font:inherit;padding:.4em}
button{margin-top:1.5em;background:#fe0;border:1px solid #cb0}
.nota{color:#555;font-size:.9em;margin:.2em 0}
.errore{background:#fdd;padding:.5em}.salvato{background:#dfd;padding:.5em}
img{max-width:100%;height:auto}
</style>
<p style="font:12px monospace;background:#fe0;padding:4px">{{.Branch}} @ {{.Commit}}</p>
<p><a href="{{.Admin}}">admin</a>{{if .Pid}} · <a href="{{.Pubblica}}">pagina della Foto</a>{{end}}</p>
<h1>{{.Titolo}}</h1>
{{if .Errore}}<p class="errore">{{.Errore}}</p>{{end}}
{{if .Salvato}}<p class="salvato">{{.Salvato}}</p>{{end}}
<p class="nota">Build: {{.Build}}.</p>
{{if .Pid}}<form method="post">
{{range .Voci}}<label for="{{.Chiave}}">{{.Etichetta}}</label>
{{if .Righe}}<textarea id="{{.Chiave}}" name="{{.Chiave}}" rows="{{.Righe}}">{{.Valore}}</textarea>{{else}}<input id="{{.Chiave}}" name="{{.Chiave}}" value="{{.Valore}}">{{end}}
{{if .Diverso}}<p class="nota">Importato: {{if .Importato}}{{.Importato}}{{else}}(vuoto){{end}}</p>{{end}}
{{end}}<p class="nota">Una voce per riga nelle liste. Riportare un campo al valore importato lo toglie dal Livello di arricchimento.</p>
<button>Salva</button>
</form>{{else}}<form method="get" action="{{.Admin}}foto/"><label for="pid">Modifica una Foto: pid</label>
<input id="pid" name="pid" inputmode="numeric"><button>Apri</button></form>{{end}}`))

type datiRedazione struct {
	Branch, Commit, Titolo, Errore, Salvato, Build, Admin, Pubblica string
	Pid                                                             int
	Voci                                                            []voceModulo
}

// foto risponde a /admin/foto/<pid>: il modulo in GET, il salvataggio in POST.
func (r *redazione) foto(w http.ResponseWriter, req *http.Request) {
	d := datiRedazione{Branch: branch, Commit: commit, Titolo: "Modifica di una Foto", Build: r.statoBuild(), Admin: "../"}
	resto := strings.Trim(strings.TrimPrefix(req.URL.Path, "/admin/foto"), "/")
	if resto == "" {
		if q := req.URL.Query().Get("pid"); q != "" {
			rimanda(w, url.PathEscape(strings.TrimSpace(q)))
			return
		}
		paginaRedazione.Execute(w, d)
		return
	}
	pid, err := strconv.Atoi(resto)
	if err != nil {
		http.NotFound(w, req)
		return
	}
	d.Admin, d.Pid, d.Pubblica = "../../", pid, fmt.Sprintf("../../f/%d/", pid)
	d.Titolo = fmt.Sprintf("Modifica della Foto %d", pid)
	modulo := map[string]string{}
	if req.Method == http.MethodPost {
		req.ParseForm()
		for _, c := range campiEditoriali {
			modulo[c.chiave] = req.PostForm.Get(c.chiave)
		}
		cambiati, err := r.salva(pid, modulo, utenteDi(req))
		switch {
		case err != nil:
			d.Errore = "Non salvato: " + err.Error()
		case len(cambiati) == 0:
			rimanda(w, resto+"?salvato=0")
			return
		default:
			r.richiediBuild()
			rimanda(w, resto+"?salvato="+strings.Join(cambiati, ","))
			return
		}
	}
	switch s := req.URL.Query().Get("salvato"); s {
	case "":
	case "0":
		d.Salvato = "Nessun campo cambiato."
	default:
		d.Salvato = "Salvato: " + strings.ReplaceAll(s, ",", ", ") + ". La pagina della Foto cambia dopo il build."
	}
	pub, imp, _, err := r.fotoDaModificare(pid)
	if errors.Is(err, os.ErrNotExist) {
		http.NotFound(w, req)
		return
	} else if err != nil {
		d.Errore, d.Pid = err.Error(), 0
		paginaRedazione.Execute(w, d)
		return
	}
	for _, c := range campiEditoriali {
		v := c.valore(pub)
		if inviato, ok := modulo[c.chiave]; ok {
			v = inviato // dopo un errore si ripropone quello che si era scritto
		}
		d.Voci = append(d.Voci, voceModulo{Chiave: c.chiave, Etichetta: c.etichetta, Valore: v, Importato: c.valore(imp),
			Righe: c.righe, Diverso: c.valore(pub) != c.valore(imp)})
	}
	paginaRedazione.Execute(w, d)
}

// rimanda con un Location relativo: http.Redirect lo renderebbe assoluto, e dietro nginx
// si perderebbe il prefisso /gerbillo/ dell'anteprima.
func rimanda(w http.ResponseWriter, dove string) {
	w.Header().Set("Location", dove)
	w.WriteHeader(http.StatusSeeOther)
}

// admin è la home del Backend: chi è l'utente, il modulo per aprire una Foto, lo stato.
func (r *redazione) admin(w http.ResponseWriter, req *http.Request) {
	if req.URL.Path != "/admin/" {
		http.NotFound(w, req)
		return
	}
	u := utenteDi(req)
	d := datiRedazione{Branch: branch, Commit: commit, Titolo: "Gerbillo: admin", Build: r.statoBuild(), Admin: "./",
		Salvato: fmt.Sprintf("Utente: %s (%d), gruppi %s. Stato del server: admin/stato.", u.Nome, u.ID, strings.Join(u.Gruppi, ", "))}
	paginaRedazione.Execute(w, d)
}

// redazioneAnteprima prepara la modifica diretta nell'anteprima (ADR 0011). L'immagine ha in
// seme/ l'Importazione dei dati di test, l'Indice delle Entità, dati/arricchimento e gli
// argomenti del build della CI. All'avvio il Backend crea in /tmp un Repository dei
// contenuti con il solo Livello di arricchimento (l'Importazione resta nell'immagine), e i
// build scrivono in /tmp solo i file diversi dalla Gallery dell'immagine. Senza seme/
// la modifica non c'è.
func redazioneAnteprima(seme, contenuti, dist, modifiche string) (*redazione, error) {
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
	args := []string{"build", "-contenuti", contenuti, "-indice", filepath.Join(seme, "indice.json"), "-dist", modifiche, "-confronta", dist}
	for sc := bufio.NewScanner(f); sc.Scan(); {
		args = append(args, sc.Text())
	}
	eseguibile, err := os.Executable()
	if err != nil {
		return nil, err
	}
	r := &redazione{contenuti: contenuti, indice: filepath.Join(seme, "indice.json")}
	r.build = func() error {
		cmd := exec.Command(eseguibile, args...)
		cmd.Stdout, cmd.Stderr = os.Stderr, os.Stderr
		return cmd.Run()
	}
	return r, nil
}
