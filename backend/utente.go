package main

// Utente corrente del Backend (ADR 0008, #42). Le pagine dipendono solo da utenteCorrente,
// mai da come l'utente è stato riconosciuto: nell'anteprima è chi si sceglie dopo la password
// usa e getta, con il bridge phpBB (bridge.go, #66) è l'utente della sessione del forum.

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
	"strings"
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
	gruppoNuovi      = "NEWLY_REGISTERED"
	gruppoUnlimited  = "Gallery unlimited"
)

// Moderatore dice se l'utente può approvare, rifiutare e organizzare le Foto.
func (u *utente) Moderatore() bool {
	return slices.Contains(u.Gruppi, gruppoAdmin) || slices.Contains(u.Gruppi, gruppoModeratori)
}

// Admin dice se l'utente è un amministratore del forum: per le azioni sull'intera Gallery.
func (u *utente) Admin() bool { return u != nil && slices.Contains(u.Gruppi, gruppoAdmin) }

// utenteCorrente riconosce chi fa una richiesta.
type utenteCorrente interface {
	// Utente è chi fa la richiesta; nil per un ospite.
	Utente(r *http.Request) *utente
	// Accedi risponde a un ospite che deve farsi riconoscere.
	Accedi(w http.ResponseWriter, r *http.Request)
	// Esci dimentica chi ha fatto la richiesta.
	Esci(w http.ResponseWriter, r *http.Request)
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
			nonModeratore(w, r, u)
		default:
			segnalaModeratore(w)
			h(w, r.WithContext(context.WithValue(r.Context(), chiaveUtente{}, u)))
		}
	}
}

// nonModeratore dice con chi si è entrati e offre di uscire: nell'anteprima serve a cambiare utente.
func nonModeratore(w http.ResponseWriter, r *http.Request, u *utente) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusForbidden)
	fmt.Fprintf(w, paginaBreve("Serve un Moderatore", `Sei entrato come %s (%s): serve un Moderatore. <a href="%sesci">Esci</a>`),
		template.HTMLEscapeString(u.Nome), template.HTMLEscapeString(strings.Join(u.Gruppi, ", ")), radice(r))
}

// paginaBreve è una pagina di una riga, per chi non vede il Backend.
func paginaBreve(titolo, corpo string) string {
	return `<!doctype html><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>` + titolo + `</title>` +
		`<div style="font:16px system-ui;margin:2em">` + corpo + `</div>`
}

// cookieModeratore segnala alla Gallery pubblica che chi guarda è un Moderatore, per mostrare
// «Modifica» nella pagina della Foto (#58). Non è segreto e non dà nessun permesso: il Backend
// controlla sempre l'utente. Path=/ perché la Gallery sta fuori da /admin/. Con l'HTTP
// Basic dell'anteprima non c'è un login vero: il cookie si mette a ogni pagina del Backend
// vista da un Moderatore, e /admin/esci lo toglie.
const cookieModeratore = "gerbillo_moderatore"

func segnalaModeratore(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: cookieModeratore, Value: "1", Path: "/", MaxAge: 30 * 24 * 3600, SameSite: http.SameSiteLaxMode})
}

// esci toglie il cookie segnale e l'utente. L'HTTP Basic dell'anteprima resta nel browser fino
// alla sua chiusura: rientrando si sceglie di nuovo con chi entrare.
func esci(uc utenteCorrente) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		uc.Esci(w, r)
		http.SetCookie(w, &http.Cookie{Name: cookieModeratore, Value: "", Path: "/", MaxAge: -1, SameSite: http.SameSiteLaxMode})
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, paginaBreve("Uscito", `Sei uscito: la Gallery non mostra più «Modifica». <a href="../">Torna alla Gallery</a> · <a href="./">Rientra nel Backend</a>`))
	}
}

// passwordAnteprima è l'utente corrente dell'anteprima dev (ADR 0011, #66): HTTP Basic con una
// password usa e getta inviata via Telegram, rigenerata alla scadenza. Chi la conosce sceglie
// con chi entrare, tra gli utentiAnteprima. Vive solo in memoria: un riavvio ne genera un'altra.
const (
	durataPassword   = 24 * time.Hour
	utenteAnteprima  = "gerbillo" // lo username dell'HTTP Basic
	idAdminAnteprima = 2          // in phpBB l'admin creato all'installazione ha user_id 2
	// cookieAnteprima dice chi si è scelto. Non è segreto: la password dà già tutti gli utenti.
	cookieAnteprima = "gerbillo_anteprima"
	campoAnteprima  = "anteprima_utente"
)

// utentiAnteprima sono gli utenti tra cui scegliere, nell'ordine della pagina.
var utentiAnteprima = []struct {
	chiave, etichetta string
	u                 utente
}{
	{"admin", "Amministratore", utente{ID: idAdminAnteprima, Nome: "admin", Gruppi: []string{gruppoAdmin, gruppoRegistrati}}},
	{"moderatore", "Moderatore", utente{ID: 3, Nome: "moderatore", Gruppi: []string{gruppoModeratori, gruppoRegistrati}}},
	{"unlimited", "Gallery unlimited", utente{ID: 4, Nome: "unlimited", Gruppi: []string{gruppoRegistrati, gruppoUnlimited}}},
	{"nuovo", "Utente nuovo", utente{ID: 5, Nome: "nuovo", Gruppi: []string{gruppoRegistrati, gruppoNuovi}}},
}

// utenteScelto è l'utente di utentiAnteprima con quella chiave, nil se non c'è.
func utenteScelto(chiave string) *utente {
	for _, s := range utentiAnteprima {
		if s.chiave == chiave {
			u := s.u
			return &u
		}
	}
	return nil
}

type passwordAnteprima struct {
	mu       sync.Mutex
	password string
	scadenza time.Time
}

// nuovaPasswordAnteprima genera la password usa e getta. Con fissa non vuota (GERBILLO_PASSWORD,
// per i test end-to-end in locale e in CI) la password è quella, non scade e non va su Telegram.
func nuovaPasswordAnteprima(fissa string) *passwordAnteprima {
	if fissa != "" {
		return &passwordAnteprima{password: fissa, scadenza: time.Now().AddDate(100, 0, 0)}
	}
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
	id, err := telegram(msg)
	if err != nil {
		// ponytail: senza Telegram (in locale) la password va nel log del container.
		log.Printf("telegram: %v; password: %s", err, pw)
		return
	}
	// Questa riga la legge gerbillo-dev-aggiorna.sh: al deploy cancella il messaggio del container
	// vecchio, il cui timer qui sotto muore con lui.
	log.Printf("telegram message_id=%d", id)
	// ponytail: dopo un riavvio per crash il timer è perso, ci pensa il prossimo deploy (se entro
	// le 48 ore oltre le quali Telegram non lascia cancellare).
	time.AfterFunc(durataPassword, func() {
		if err := telegramCancella(id); err != nil {
			log.Printf("telegram: cancellazione di %d: %v", id, err)
		}
	})
}

// passwordGiusta dice se la richiesta porta la password usa e getta, che rinnova se è scaduta.
func (p *passwordAnteprima) passwordGiusta(r *http.Request) bool {
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
	return ok && u == utenteAnteprima && subtle.ConstantTimeCompare([]byte(inviata), []byte(pw)) == 1
}

func (p *passwordAnteprima) Utente(r *http.Request) *utente {
	if !p.passwordGiusta(r) {
		return nil
	}
	c, err := r.Cookie(cookieAnteprima)
	if err != nil {
		return nil
	}
	return utenteScelto(c.Value)
}

// Accedi chiede la password, poi con chi entrare. La scelta torna alla stessa pagina.
func (p *passwordAnteprima) Accedi(w http.ResponseWriter, r *http.Request) {
	if !p.passwordGiusta(r) {
		w.Header().Set("WWW-Authenticate", `Basic realm="Gerbillo dev"`)
		http.Error(w, "password su Telegram", http.StatusUnauthorized)
		return
	}
	if scelta := r.PostFormValue(campoAnteprima); r.Method == http.MethodPost && utenteScelto(scelta) != nil {
		http.SetCookie(w, &http.Cookie{Name: cookieAnteprima, Value: scelta, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode})
		rimanda(w, "?"+r.URL.RawQuery) // relativo: dietro nginx il Backend non vede /gerbillo/
		return
	}
	var bottoni strings.Builder
	for _, s := range utentiAnteprima {
		fmt.Fprintf(&bottoni, `<p><button name="%s" value="%s">%s</button></p>`, campoAnteprima, s.chiave, s.etichetta)
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, paginaBreve("Con chi entri?", `<h1>Con chi entri?</h1><form method="post">`+bottoni.String()+`</form>`))
}

func (p *passwordAnteprima) Esci(w http.ResponseWriter, _ *http.Request) {
	http.SetCookie(w, &http.Cookie{Name: cookieAnteprima, Value: "", Path: "/", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteLaxMode})
}
