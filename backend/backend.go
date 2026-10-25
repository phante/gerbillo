package main

// Interfaccia del Backend (ADR 0017): HTML generato dal server con un solo modello di
// pagina, htmx per gli aggiornamenti parziali e Alpine.js per lo stato locale dei
// componenti. Le librerie sono incorporate nel binario (scripts/librerie-backend.sh):
// nessuna pagina carica risorse da altri domini.

import (
	"bytes"
	"compress/gzip"
	"html/template"
	"log"
	"mime"
	"net/http"
	"path"
	"strings"
	"sync"
)

var modelliBackend = template.Must(template.New("").Funcs(template.FuncMap{
	"icona": icona,
	// dict passa più valori a un modello: {{template "x" (dict "a" 1 "b" 2)}}.
	"dict": func(kv ...any) map[string]any {
		m := map[string]any{}
		for i := 0; i+1 < len(kv); i += 2 {
			m[kv[i].(string)] = kv[i+1]
		}
		return m
	},
}).ParseFS(modelli, "modelli/backend/*.html"))

// paginaBackend è il dato di ogni pagina del Backend: la barra in alto più il Corpo.
type paginaBackend struct {
	Titolo   string
	R        string // percorso relativo di /admin/: la <base> della pagina
	Striscia string // branch e commit dell'anteprima
	Utente   *utente
	Corpo    any
}

// radice è il percorso relativo che dalla pagina richiesta porta a /admin/. I link sono
// tutti relativi: dietro nginx l'anteprima sta sotto /gerbillo/, che il Backend non vede.
func radice(r *http.Request) string {
	resto := strings.TrimPrefix(r.URL.Path, "/admin/")
	if n := strings.Count(resto, "/"); n > 0 {
		return strings.Repeat("../", n)
	}
	return "./"
}

func striscia() string {
	c := commit
	if len(c) > 7 {
		c = c[:7]
	}
	return branch + " @ " + c
}

// mostraPagina scrive la pagina intera del modello nome.
func mostraPagina(w http.ResponseWriter, r *http.Request, nome, titolo string, corpo any) {
	esegui(w, nome, paginaBackend{Titolo: titolo, R: radice(r), Striscia: striscia(), Utente: utenteDi(r), Corpo: corpo})
}

// mostraFrammento scrive solo un pezzo di pagina, per le risposte a htmx.
func mostraFrammento(w http.ResponseWriter, nome string, dati any) {
	esegui(w, nome, dati)
}

func esegui(w http.ResponseWriter, nome string, dati any) {
	var b bytes.Buffer
	if err := modelliBackend.ExecuteTemplate(&b, nome, dati); err != nil {
		log.Printf("modello %s: %v", nome, err)
		http.Error(w, "errore interno", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(b.Bytes())
}

// daHtmx dice se la richiesta viene da htmx, che vuole solo il frammento.
func daHtmx(r *http.Request) bool { return r.Header.Get("HX-Request") == "true" }

// staticiBackend sono i file serviti a /admin/statico/: nome pubblicato → file in modelli/.
// I nomi non hanno l'estensione: nell'anteprima il vhost di nginx intercetta .css, .js, .svg
// e .json prima di /gerbillo/ (finché manca ^~ nello snippet), come per la Gallery con
// build -anteprima. Il tipo viene dall'estensione del file in modelli/.
var staticiBackend = map[string]string{
	"tachyons":         "sito/tachyons.css",
	"logo":             "sito/PDLogo.svg",
	"backend-stile":    "backend/backend.css",
	"backend":          "backend/backend.js",
	"htmx":             "backend/librerie/htmx.min.js",
	"alpine":           "backend/librerie/alpine.min.js",
	"tom-select":       "backend/librerie/tom-select.min.js",
	"tom-select-stile": "backend/librerie/tom-select.min.css",
	"mappa":            "sito/mappa.js", // Leaflet del Sito principale (#27), per la mappa del Punto geografico
	"leaflet-stile":    "sito/leaflet.css",
}

// statico serve i file di staticiBackend. Non richiede l'utente: sono librerie e stili.
func statico(w http.ResponseWriter, r *http.Request) {
	f, ok := staticiBackend[path.Base(r.URL.Path)]
	if !ok {
		http.NotFound(w, r)
		return
	}
	d, err := modelli.ReadFile("modelli/" + f)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	// Il binario cambia a ogni deploy: il browser ricontrolla, e riceve 304 finché non cambia.
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Vary", "Accept-Encoding")
	// nginx dell'anteprima comprime il CSS ma non il JavaScript: lo fa il Backend, una volta
	// per file. Le librerie passano da 470 a circa 150 kB e la pagina è pronta prima.
	if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") && path.Ext(f) != ".svg" {
		w.Header().Set("Content-Type", mime.TypeByExtension(path.Ext(f)))
		w.Header().Set("Content-Encoding", "gzip")
		d = compresso(f, d)
	}
	http.ServeContent(w, r, path.Base(f), avvio, bytes.NewReader(d))
}

var compressi sync.Map // file in modelli/ → gzip

func compresso(f string, d []byte) []byte {
	if c, ok := compressi.Load(f); ok {
		return c.([]byte)
	}
	var b bytes.Buffer
	z, _ := gzip.NewWriterLevel(&b, gzip.BestCompression)
	z.Write(d)
	z.Close()
	compressi.Store(f, b.Bytes())
	return b.Bytes()
}

// rotteBackend registra le pagine del Backend. Senza redazione (manca il seme) ci sono
// solo la home e lo stato.
func rotteBackend(mux *http.ServeMux, uc utenteCorrente, red *redazione) {
	// Il browser rimanda l'HTTP Basic anche dalle richieste di altri siti: CrossOriginProtection
	// rifiuta quelle che cambiano qualcosa.
	mod := func(h http.HandlerFunc) http.Handler {
		return http.NewCrossOriginProtection().Handler(soloModeratore(uc, h))
	}
	// /admin senza barra: il ServeMux di Go rimanderebbe a /admin/ con un percorso assoluto,
	// e dietro nginx si perderebbe il prefisso /gerbillo/ dell'anteprima.
	mux.HandleFunc("/admin", conBarra)
	mux.HandleFunc("/admin/statico/", statico)
	mux.HandleFunc("/admin/esci", esci) // senza utente: chi esce non deve rifare l'accesso
	mux.Handle("/admin/stato", mod(stato))
	mux.Handle("/admin/widget", mod(statoWidget(red))) // senza .json: vedi staticiBackend
	if red == nil {
		mux.Handle("/admin/", mod(adminSenzaRedazione))
		return
	}
	mux.HandleFunc("/admin/foto", conBarra)
	mux.Handle("/admin/", mod(red.admin))
	mux.Handle("/admin/foto/", mod(red.foto))
	mux.Handle("/admin/ricalcola", mod(red.ricalcola))
	mux.Handle("/admin/datazione", mod(datazione))
	mux.Handle("/admin/suggerimenti/", mod(red.suggerimenti))
	mux.Handle("/admin/punto", mod(punto))
	mux.Handle("/admin/blocco", mod(red.blocco))
	mux.Handle("/admin/blocco/", mod(red.blocco))
	mux.Handle("/admin/operazioni", mod(red.paginaOperazioni))
	mux.Handle("/admin/operazioni/", mod(red.paginaOperazioni))
}
