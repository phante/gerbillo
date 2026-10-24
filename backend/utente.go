package main

// Utente corrente del Backend (ADR 0008, #42). Le pagine dipendono solo da utenteCorrente,
// mai da come l'utente è stato riconosciuto: nell'anteprima è l'admin con la password usa e
// getta, con il bridge phpBB (#44) sarà l'utente della sessione del forum.

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"slices"
	"sync"
	"time"
)

// utente è chi fa la richiesta. I gruppi hanno i nomi di phpBB.
type utente struct {
	ID     int
	Nome   string
	Gruppi []string
}

// Gruppi phpBB. I Moderatori sono gli amministratori e i moderatori globali del forum.
const (
	gruppoAdmin      = "ADMINISTRATORS"
	gruppoModeratori = "GLOBAL_MODERATORS"
	gruppoRegistrati = "REGISTERED"
)

// Moderatore dice se l'utente può approvare, rifiutare e organizzare le Foto.
func (u *utente) Moderatore() bool {
	return slices.Contains(u.Gruppi, gruppoAdmin) || slices.Contains(u.Gruppi, gruppoModeratori)
}

// utenteCorrente riconosce chi fa una richiesta.
type utenteCorrente interface {
	// Utente è chi fa la richiesta; nil per un ospite.
	Utente(r *http.Request) *utente
	// Accedi risponde a un ospite che deve farsi riconoscere.
	Accedi(w http.ResponseWriter, r *http.Request)
}

type chiaveUtente struct{}

// utenteDi è l'utente che soloModeratore ha messo nella richiesta.
func utenteDi(r *http.Request) *utente {
	u, _ := r.Context().Value(chiaveUtente{}).(*utente)
	return u
}

// soloModeratore protegge le pagine che richiedono un Moderatore e passa loro l'utente.
func soloModeratore(uc utenteCorrente, h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u := uc.Utente(r)
		switch {
		case u == nil:
			uc.Accedi(w, r)
		case !u.Moderatore():
			http.Error(w, "serve un Moderatore", http.StatusForbidden)
		default:
			h(w, r.WithContext(context.WithValue(r.Context(), chiaveUtente{}, u)))
		}
	}
}

// passwordAnteprima è l'utente corrente dell'anteprima dev (ADR 0011): HTTP Basic con una
// password usa e getta inviata via Telegram, rigenerata alla scadenza. Chi la conosce è
// l'admin, con tutti i permessi del Moderatore. Vive solo in memoria: un riavvio ne genera un'altra.
const (
	durataPassword     = 24 * time.Hour
	utenteAnteprima    = "gerbillo" // lo username dell'HTTP Basic
	idAdminAnteprima   = 2          // in phpBB l'admin creato all'installazione ha user_id 2
	nomeAdminAnteprima = "admin"
)

type passwordAnteprima struct {
	mu       sync.Mutex
	password string
	scadenza time.Time
}

func nuovaPasswordAnteprima() *passwordAnteprima {
	p := &passwordAnteprima{}
	p.rinnova()
	return p
}

func (p *passwordAnteprima) rinnova() {
	b := make([]byte, 12)
	rand.Read(b)
	p.mu.Lock()
	p.password = base64.RawURLEncoding.EncodeToString(b)
	p.scadenza = time.Now().Add(durataPassword)
	pw, scade := p.password, p.scadenza
	p.mu.Unlock()
	// <code>: su Telegram un tocco copia il testo. base64url non ha caratteri HTML.
	msg := fmt.Sprintf("Gerbillo dev %s@%s\nutente: <code>%s</code>\npassword: <code>%s</code>\nscade: %s",
		template.HTMLEscapeString(branch), commit, utenteAnteprima, pw, scade.In(roma).Format("02/01 15:04"))
	if err := telegram(msg); err != nil {
		// ponytail: senza Telegram (in locale) la password va nel log del container.
		log.Printf("telegram: %v; password: %s", err, pw)
	}
}

func (p *passwordAnteprima) Utente(r *http.Request) *utente {
	p.mu.Lock()
	scaduta := time.Now().After(p.scadenza)
	p.mu.Unlock()
	if scaduta {
		p.rinnova()
	}
	p.mu.Lock()
	pw := p.password
	p.mu.Unlock()
	u, inviata, ok := r.BasicAuth()
	if !ok || u != utenteAnteprima || subtle.ConstantTimeCompare([]byte(inviata), []byte(pw)) != 1 {
		return nil
	}
	return &utente{ID: idAdminAnteprima, Nome: nomeAdminAnteprima, Gruppi: []string{gruppoAdmin, gruppoRegistrati}}
}

func (p *passwordAnteprima) Accedi(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("WWW-Authenticate", `Basic realm="Gerbillo dev"`)
	http.Error(w, "password su Telegram", http.StatusUnauthorized)
}
