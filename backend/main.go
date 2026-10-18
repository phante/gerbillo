// Command gerbillo è il Backend di Gerbillo (ADR 0007).
// Per ora contiene solo l'anteprima dev (ADR 0011): una pagina con branch e commit
// e un admin protetto da una password usa e getta inviata via Telegram.
package main

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
	_ "time/tzdata" // l'immagine distroless non ha i fusi orari
)

// Impostati in build con -ldflags "-X main.branch=… -X main.commit=…".
var branch, commit = "locale", "sconosciuto"

const durataPassword = 24 * time.Hour

var roma, _ = time.LoadLocation("Europe/Rome")

var (
	mu       sync.Mutex
	password string
	scadenza time.Time
	avvio    = time.Now()
)

func main() {
	if len(os.Args) < 2 || os.Args[1] != "serve" {
		fmt.Fprintln(os.Stderr, "uso: gerbillo serve")
		os.Exit(2)
	}
	nuovaPassword()
	http.HandleFunc("/", home)
	http.HandleFunc("/admin/", soloModeratore(admin))
	http.HandleFunc("/admin/stato", soloModeratore(stato))
	addr := envOr("GERBILLO_ADDR", ":8080")
	log.Printf("gerbillo %s@%s in ascolto su %s", branch, commit, addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

// nuovaPassword genera la password usa e getta e la invia via Telegram.
// Vive solo in memoria: un riavvio ne genera un'altra.
func nuovaPassword() {
	b := make([]byte, 12)
	rand.Read(b)
	mu.Lock()
	password = base64.RawURLEncoding.EncodeToString(b)
	scadenza = time.Now().Add(durataPassword)
	p := password
	mu.Unlock()
	msg := fmt.Sprintf("Gerbillo dev %s@%s\nutente: gerbillo\npassword: %s\nscade: %s",
		branch, commit, p, scadenza.In(roma).Format("02/01 15:04"))
	if err := telegram(msg); err != nil {
		// ponytail: senza Telegram (in locale) la password va nel log del container.
		log.Printf("telegram: %v; password: %s", err, p)
	}
}

func telegram(testo string) error {
	tok, chat := os.Getenv("TELEGRAM_TOKEN"), os.Getenv("TELEGRAM_CHAT_ID")
	if tok == "" || chat == "" {
		return fmt.Errorf("TELEGRAM_TOKEN o TELEGRAM_CHAT_ID mancanti")
	}
	r, err := http.PostForm("https://api.telegram.org/bot"+tok+"/sendMessage",
		url.Values{"chat_id": {chat}, "text": {testo}})
	if err != nil {
		return err
	}
	r.Body.Close()
	if r.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", r.StatusCode)
	}
	return nil
}

// soloModeratore è l'implementazione dev di "utente corrente" (ADR 0008):
// HTTP Basic con la password usa e getta, rigenerata alla scadenza.
func soloModeratore(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		scaduta := time.Now().After(scadenza)
		mu.Unlock()
		if scaduta {
			nuovaPassword()
		}
		mu.Lock()
		p := password
		mu.Unlock()
		u, pw, ok := r.BasicAuth()
		if !ok || u != "gerbillo" || subtle.ConstantTimeCompare([]byte(pw), []byte(p)) != 1 {
			w.Header().Set("WWW-Authenticate", `Basic realm="Gerbillo dev"`)
			http.Error(w, "password su Telegram", http.StatusUnauthorized)
			return
		}
		h(w, r)
	}
}

var pagina = template.Must(template.New("p").Parse(`<!doctype html>
<meta charset="utf-8"><meta name="viewport" content="width=device-width">
<meta name="robots" content="noindex">
<title>Gerbillo dev</title>
<style>pre{white-space:pre-wrap}</style>
<p style="font:12px monospace;background:#fe0;padding:4px">{{.Branch}} @ {{.Commit}}</p>
<h1>{{.Titolo}}</h1>
<pre>{{.Corpo}}</pre>`))

func mostra(w http.ResponseWriter, titolo, corpo string) {
	pagina.Execute(w, map[string]string{"Branch": branch, "Commit": commit, "Titolo": titolo, "Corpo": corpo})
}

func home(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	mostra(w, "Gerbillo: anteprima dev", "Gallery pubblica in arrivo.\nAdmin: admin/")
}

func admin(w http.ResponseWriter, r *http.Request) {
	mostra(w, "Gerbillo: admin", "Stato: admin/stato")
}

// stato mostra la memoria del processo, quella del container (cgroup v2)
// e il file scritto dall'host dal timer (montato in sola lettura).
func stato(w http.ResponseWriter, r *http.Request) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	var b strings.Builder
	fmt.Fprintf(&b, "avvio: %s (%s fa)\n", avvio.In(roma).Format(time.RFC3339), time.Since(avvio).Round(time.Second))
	fmt.Fprintf(&b, "go: heap %d MiB, sys %d MiB, goroutine %d\n", m.HeapAlloc>>20, m.Sys>>20, runtime.NumGoroutine())
	for _, f := range []string{"memory.current", "memory.peak", "memory.max"} {
		v, _ := os.ReadFile("/sys/fs/cgroup/" + f)
		fmt.Fprintf(&b, "cgroup %s: %s\n", f, strings.TrimSpace(string(v)))
	}
	host, err := os.ReadFile(envOr("GERBILLO_STATO_HOST", "/stato/stato.json"))
	if err != nil {
		host = []byte(err.Error())
	}
	fmt.Fprintf(&b, "\nhost:\n%s", host)
	mostra(w, "Gerbillo: stato", b.String())
}

func envOr(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
