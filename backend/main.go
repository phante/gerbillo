// Command gerbillo è il Backend di Gerbillo (ADR 0007).
// Sottocomandi: serve, l'anteprima dev (ADR 0011) con un admin protetto da una
// password usa e getta inviata via Telegram e la modifica diretta delle Foto (#43);
// importa, l'Importazione da Coppermine (#9), con il rapporto sulla Mappatura degli
// album (#10, #38); indice-entita, l'Indice delle Entità provvisorio.
// varianti, le Varianti WebP (#11); build, la Gallery pubblica (#12).
package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
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
	var uc utenteCorrente = nuovaPasswordAnteprima(os.Getenv("GERBILLO_PASSWORD"))
	// Nell'anteprima la Gallery pubblica generata in CI sta in /dist (ADR 0011, 0012); quella
	// rigenerata dopo una modifica (#43) sta in /tmp/dist, con i soli file cambiati, e vince.
	dist := envOr("GERBILLO_DIST", "/dist")
	modifiche := envOr("GERBILLO_DIST_MODIFICHE", "/tmp/dist")
	red, err := redazioneAnteprima(envOr("GERBILLO_SEME", "/seme"), envOr("GERBILLO_CONTENUTI", "/tmp/contenuti"), dist, modifiche)
	if err != nil {
		log.Printf("modifica delle Foto non disponibile: %v", err)
	}
	mux := http.NewServeMux()
	if esiste(filepath.Join(dist, "index.html")) {
		cartelle := sovrapposte{http.Dir(modifiche), http.Dir(dist)}
		mux.Handle("/", staticiSenzaEstensione(cartelle, http.FileServer(cartelle)))
	} else {
		mux.HandleFunc("/", home)
	}
	rotteBackend(mux, uc, red)
	var h http.Handler = mux
	// Come nginx nell'anteprima, che toglie /gerbillo/: serve ai test end-to-end (e2e/).
	if p := os.Getenv("GERBILLO_PREFISSO"); p != "" {
		h = http.StripPrefix(strings.TrimSuffix(p, "/"), mux)
	}
	addr := envOr("GERBILLO_ADDR", ":8080")
	log.Printf("gerbillo %s@%s in ascolto su %s", branch, commit, addr)
	log.Fatal(http.ListenAndServe(addr, h))
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

func home(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	fmt.Fprintf(w, "Gerbillo %s: la Gallery pubblica non è stata generata. Backend: admin/\n", striscia())
}

// adminSenzaRedazione è la home del Backend quando manca il seme della modifica delle Foto.
func adminSenzaRedazione(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/admin/" {
		http.NotFound(w, r)
		return
	}
	mostraPagina(w, r, "home", "Backend", corpoHome{})
}

type corpoStato struct {
	Avvio, Da            string
	Heap, Sys, Goroutine uint64
	Cgroup               [][2]string
	Host                 string
}

// stato mostra la memoria del processo, quella del container (cgroup v2)
// e il file scritto dall'host dal timer (montato in sola lettura).
func stato(w http.ResponseWriter, r *http.Request) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	c := corpoStato{Avvio: avvio.In(roma).Format("02/01/2006 15:04:05"), Da: time.Since(avvio).Round(time.Second).String(),
		Heap: m.HeapAlloc >> 20, Sys: m.Sys >> 20, Goroutine: uint64(runtime.NumGoroutine())}
	for _, f := range []string{"memory.current", "memory.peak", "memory.max"} {
		v, _ := os.ReadFile("/sys/fs/cgroup/" + f)
		c.Cgroup = append(c.Cgroup, [2]string{f, strings.TrimSpace(string(v))})
	}
	host, err := os.ReadFile(envOr("GERBILLO_STATO_HOST", "/stato/stato.json"))
	if err != nil {
		host = []byte(err.Error())
	}
	c.Host = string(host)
	mostraPagina(w, r, "stato", "Stato del server", c)
}

// staticiSenzaEstensione serve /statico/stile come /statico/stile.css (build -anteprima).
func staticiSenzaEstensione(fs sovrapposte, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/statico/") && path.Ext(r.URL.Path) == "" {
			for _, ext := range []string{".css", ".svg", ".js", ".json"} {
				if f, err := fs.Open(r.URL.Path + ext); err == nil {
					st, _ := f.Stat()
					http.ServeContent(w, r, path.Base(r.URL.Path)+ext, st.ModTime(), f)
					f.Close()
					return
				}
			}
		}
		h.ServeHTTP(w, r)
	})
}

// conBarra rimanda alla stessa pagina con la barra finale, con un Location relativo.
func conBarra(w http.ResponseWriter, r *http.Request) {
	rimanda(w, path.Base(r.URL.Path)+"/")
}

// sovrapposte serve ogni file dalla prima cartella che lo ha.
type sovrapposte []http.Dir

func (s sovrapposte) Open(nome string) (http.File, error) {
	var primo error
	for _, d := range s {
		f, err := d.Open(nome)
		if err == nil {
			return f, nil
		}
		if primo == nil {
			primo = err
		}
	}
	return nil, primo
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
