// Command gerbillo è il Backend di Gerbillo (ADR 0007).
// Sottocomandi: serve, l'anteprima dev (ADR 0011) con un admin protetto da una
// password usa e getta inviata via Telegram e la modifica diretta delle Foto (#43);
// importa, l'Importazione da Coppermine (#9), con il rapporto sulla Mappatura degli
// album (#10, #38); indice-entita, l'Indice delle Entità provvisorio.
// varianti, le Varianti WebP (#11); build, la Gallery pubblica (#12).
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
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
	// Con GERBILLO_BRIDGE l'utente corrente è quello del forum (#66). Nell'anteprima, che non ha
	// il forum, è chi si sceglie dopo la password usa e getta.
	var uc utenteCorrente
	if u := os.Getenv("GERBILLO_BRIDGE"); u != "" {
		uc = nuovoBridge(u, os.Getenv("GERBILLO_FORUM_LOGIN"))
	} else {
		uc = nuovaPasswordAnteprima(os.Getenv("GERBILLO_PASSWORD"))
	}
	// Nell'anteprima la Gallery pubblica generata in CI sta in /dist (ADR 0011, 0012); quella
	// rigenerata dopo una modifica (#43) sta in /tmp/dist, con i soli file cambiati, e vince.
	dist := envOr("GERBILLO_DIST", "/dist")
	modifiche := envOr("GERBILLO_DIST_MODIFICHE", "/tmp/dist")
	red, err := redazioneAnteprima(envOr("GERBILLO_SEME", "/seme"), envOr("GERBILLO_CONTENUTI", "/tmp/contenuti"), dist, modifiche,
		envOr("GERBILLO_REDAZIONE", "/tmp/redazione.json"))
	if err != nil {
		log.Printf("modifica delle Foto non disponibile: %v", err)
	}
	mux := http.NewServeMux()
	if esiste(filepath.Join(dist, "index.html")) {
		cartelle := sovrapposte{http.Dir(modifiche), http.Dir(dist)}
		mux.Handle("/", http.FileServer(cartelle))
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

// apiTelegram è una variabile per i test.
var apiTelegram = "https://api.telegram.org"

// telegram manda il messaggio e ne restituisce l'id.
func telegram(testo string) (int, error) {
	var risposta struct {
		Result struct {
			MessageID int `json:"message_id"`
		} `json:"result"`
	}
	err := chiamaTelegram("sendMessage", url.Values{"text": {testo}, "parse_mode": {"HTML"}}, &risposta)
	return risposta.Result.MessageID, err
}

func telegramCancella(id int) error {
	return chiamaTelegram("deleteMessage", url.Values{"message_id": {strconv.Itoa(id)}}, nil)
}

// chiamaTelegram chiama un metodo della Bot API sulla chat configurata. Con risposta non nil
// vi decodifica il JSON restituito.
func chiamaTelegram(metodo string, v url.Values, risposta any) error {
	tok, chat := os.Getenv("TELEGRAM_TOKEN"), os.Getenv("TELEGRAM_CHAT_ID")
	if tok == "" || chat == "" {
		return fmt.Errorf("TELEGRAM_TOKEN o TELEGRAM_CHAT_ID mancanti")
	}
	v.Set("chat_id", chat)
	r, err := http.PostForm(apiTelegram+"/bot"+tok+"/"+metodo, v)
	if err != nil {
		return err
	}
	defer r.Body.Close()
	if r.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", r.StatusCode)
	}
	if risposta == nil {
		return nil
	}
	return json.NewDecoder(r.Body).Decode(risposta)
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
	mostraPagina(w, r, "home", "Gerbillo", corpoHome{})
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
