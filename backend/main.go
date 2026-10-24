// Command gerbillo è il Backend di Gerbillo (ADR 0007).
// Sottocomandi: serve, l'anteprima dev (ADR 0011) con un admin protetto da una
// password usa e getta inviata via Telegram; importa, l'Importazione da Coppermine (#9),
// con il rapporto sulla Mappatura degli album (#10, #38); indice-entita, l'Indice delle Entità provvisorio.
// varianti, le Varianti WebP (#11); build, la Gallery pubblica (#12).
package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
	_ "time/tzdata" // l'immagine distroless non ha i fusi orari
)

// Impostati in build con -ldflags "-X main.branch=… -X main.commit=…".
var branch, commit = "locale", "sconosciuto"

var roma, _ = time.LoadLocation("Europe/Rome")

var avvio = time.Now()

func main() {
	comandi := map[string]func([]string) error{"importa": importa, "indice-entita": indiceEntita, "varianti": generaVarianti, "build": build}
	if len(os.Args) >= 2 && comandi[os.Args[1]] != nil {
		if err := comandi[os.Args[1]](os.Args[2:]); err != nil {
			log.Fatal(err)
		}
		return
	}
	if len(os.Args) < 2 || os.Args[1] != "serve" {
		fmt.Fprintln(os.Stderr, "uso: gerbillo serve | importa | indice-entita | varianti | build")
		os.Exit(2)
	}
	// Nell'anteprima l'utente corrente è solo l'admin, con la password usa e getta (#42).
	var uc utenteCorrente = nuovaPasswordAnteprima()
	// Nell'anteprima la Gallery pubblica generata in CI sta in /dist (ADR 0011, 0012).
	if dist := envOr("GERBILLO_DIST", "/dist"); esiste(filepath.Join(dist, "index.html")) {
		http.Handle("/", staticiSenzaEstensione(dist, http.FileServer(http.Dir(dist))))
	} else {
		http.HandleFunc("/", home)
	}
	http.HandleFunc("/admin/", soloModeratore(uc, admin))
	http.HandleFunc("/admin/stato", soloModeratore(uc, stato))
	addr := envOr("GERBILLO_ADDR", ":8080")
	log.Printf("gerbillo %s@%s in ascolto su %s", branch, commit, addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

func telegram(testo string) error {
	tok, chat := os.Getenv("TELEGRAM_TOKEN"), os.Getenv("TELEGRAM_CHAT_ID")
	if tok == "" || chat == "" {
		return fmt.Errorf("TELEGRAM_TOKEN o TELEGRAM_CHAT_ID mancanti")
	}
	r, err := http.PostForm("https://api.telegram.org/bot"+tok+"/sendMessage",
		url.Values{"chat_id": {chat}, "text": {testo}, "parse_mode": {"HTML"}})
	if err != nil {
		return err
	}
	r.Body.Close()
	if r.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", r.StatusCode)
	}
	return nil
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
	mostra(w, "Gerbillo: anteprima dev", "Gallery pubblica in arrivo (prova remota).\nAdmin: admin/")
}

func admin(w http.ResponseWriter, r *http.Request) {
	u := utenteDi(r)
	mostra(w, "Gerbillo: admin", fmt.Sprintf("Utente: %s (%d), gruppi %s\nStato: admin/stato", u.Nome, u.ID, strings.Join(u.Gruppi, ", ")))
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

// staticiSenzaEstensione serve /statico/stile come /statico/stile.css (build -anteprima).
func staticiSenzaEstensione(dist string, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/statico/") && filepath.Ext(r.URL.Path) == "" {
			for _, ext := range []string{".css", ".svg", ".js", ".json"} {
				if p := filepath.Join(dist, filepath.FromSlash(r.URL.Path)+ext); esiste(p) {
					http.ServeFile(w, r, p)
					return
				}
			}
		}
		h.ServeHTTP(w, r)
	})
}

func esiste(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func envOr(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
