package main

// Utente corrente sul bridge phpBB (ADR 0008, #66): chiede al controller dell'estensione
// (ext/gerbillo/bridge) chi è il proprietario della sessione del forum.

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	versioneBridge = 1
	// durataSessione è per quanto il Backend si fida della risposta del bridge: anche il tempo
	// massimo in cui un Moderatore rimosso dal gruppo conserva i permessi.
	durataSessione = 15 * time.Minute
)

// bridgeForum tiene una sessione propria per ogni sessione del forum. Sta in memoria e la
// chiave è il cookie _sid del forum: niente cookie in più, e chi esce dal forum, che cambia
// _sid, non è più riconosciuto subito. Dopo un riavvio del Backend si richiede al bridge.
type bridgeForum struct {
	url    string // il controller, su 127.0.0.1
	login  string // la pagina di accesso del forum, per chi arriva da ospite
	client *http.Client

	mu       sync.Mutex
	sessioni map[string]sessioneBackend
}

type sessioneBackend struct {
	u        *utente
	scadenza time.Time
}

func nuovoBridge(url, login string) *bridgeForum {
	return &bridgeForum{url: url, login: login, sessioni: map[string]sessioneBackend{}, client: &http.Client{
		Timeout: 2 * time.Second,
		// Il controller non rimanda altrove: se succede, i cookie del forum non lo seguono.
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}}
}

// sidForum è il valore del cookie _sid del forum. Il nome ha il prefisso scelto in phpBB.
func sidForum(r *http.Request) string {
	for _, c := range r.Cookies() {
		if strings.HasSuffix(c.Name, "_sid") && c.Value != "" {
			return c.Value
		}
	}
	return ""
}

func (b *bridgeForum) Utente(r *http.Request) *utente {
	sid := sidForum(r)
	if sid == "" { // senza sessione del forum il bridge risponderebbe comunque "ospite"
		return nil
	}
	b.mu.Lock()
	s, ok := b.sessioni[sid]
	b.mu.Unlock()
	if ok && time.Now().Before(s.scadenza) {
		return s.u
	}
	u := b.chiedi(r)
	b.mu.Lock()
	defer b.mu.Unlock()
	now := time.Now()
	for k, v := range b.sessioni {
		if now.After(v.scadenza) {
			delete(b.sessioni, k)
		}
	}
	if u != nil {
		b.sessioni[sid] = sessioneBackend{u, now.Add(durataSessione)}
	} else {
		delete(b.sessioni, sid)
	}
	return u
}

// chiedi interroga il bridge. Se non risponde, o risponde altro, l'utente è un ospite.
func (b *bridgeForum) chiedi(r *http.Request) *utente {
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, b.url, nil)
	if err != nil {
		log.Printf("bridge: %v", err)
		return nil
	}
	// Tutti i cookie tranne _k, quello di «ricordami» (ADR 0008): app.php di phpBB chiama
	// session_begin() prima del controller, e con _k rigenererebbe la chiave nella risposta
	// al Backend. Il browser non la riceverebbe mai e l'utente perderebbe il «ricordami».
	for _, c := range r.Cookies() {
		if !strings.HasSuffix(c.Name, "_k") {
			req.AddCookie(c)
		}
	}
	res, err := b.client.Do(req)
	if err != nil {
		log.Printf("bridge: %v", err)
		return nil
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		log.Printf("bridge: HTTP %d", res.StatusCode)
		return nil
	}
	var v struct {
		V             int      `json:"v"`
		Authenticated bool     `json:"authenticated"`
		UserID        int      `json:"user_id"`
		Username      string   `json:"username"`
		Groups        []string `json:"groups"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(nil, res.Body, 1<<16)).Decode(&v); err != nil {
		log.Printf("bridge: %v", err)
		return nil
	}
	if v.V != versioneBridge {
		log.Printf("bridge: versione %d, attesa %d", v.V, versioneBridge)
		return nil
	}
	if !v.Authenticated {
		return nil
	}
	return &utente{ID: v.UserID, Nome: v.Username, Gruppi: v.Groups}
}

func (b *bridgeForum) Accedi(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusUnauthorized)
	fmt.Fprint(w, paginaBreve("Accedi", `Per entrare nel Backend <a href="`+template.HTMLEscapeString(b.login)+`">accedi al forum</a>.`))
}

// Esci dimentica la sessione del Backend, non quella del forum: alla prossima pagina del
// Backend il bridge riconosce di nuovo l'utente.
func (b *bridgeForum) Esci(_ http.ResponseWriter, r *http.Request) {
	b.mu.Lock()
	delete(b.sessioni, sidForum(r))
	b.mu.Unlock()
}
