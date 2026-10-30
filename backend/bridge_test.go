package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const rispostaModeratore = `{"v":1,"authenticated":true,"user_id":8,"username":"mod","groups":["REGISTERED","GLOBAL_MODERATORS"]}`

// bridgeFinto è un controller che dà sempre la stessa risposta e conta le chiamate.
func bridgeFinto(t *testing.T, codice int, risposta string) (*bridgeForum, *int, *string) {
	t.Helper()
	chiamate, cookie := 0, ""
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		chiamate++
		cookie = r.Header.Get("Cookie")
		w.WriteHeader(codice)
		w.Write([]byte(risposta))
	}))
	t.Cleanup(s.Close)
	return nuovoBridge(s.URL, "https://forum/ucp.php?mode=login"), &chiamate, &cookie
}

func richiestaForum() *http.Request {
	r := httptest.NewRequest("GET", "/admin/", nil)
	r.Header.Set("Cookie", "phpbb3_x_u=8; phpbb3_x_k=segreta; phpbb3_x_sid=abc")
	return r
}

func TestBridgeRiconosceUtenteEGruppi(t *testing.T) {
	b, chiamate, cookie := bridgeFinto(t, http.StatusOK, rispostaModeratore)
	u := b.Utente(richiestaForum())
	if u == nil || u.ID != 8 || u.Nome != "mod" || !u.Moderatore() {
		t.Fatalf("utente %+v", u)
	}
	if !strings.Contains(*cookie, "phpbb3_x_sid=abc") || strings.Contains(*cookie, "_k=") {
		t.Errorf("cookie inviati al bridge: %q", *cookie)
	}
	// La sessione del Backend evita altre chiamate, finché non scade.
	b.Utente(richiestaForum())
	if *chiamate != 1 {
		t.Errorf("%d chiamate con la sessione valida", *chiamate)
	}
	b.sessioni["abc"] = sessioneBackend{u, time.Now().Add(-time.Second)}
	b.Utente(richiestaForum())
	if *chiamate != 2 {
		t.Errorf("%d chiamate con la sessione scaduta", *chiamate)
	}
	// Uscendo la sessione del Backend sparisce.
	b.Esci(nil, richiestaForum())
	if _, ok := b.sessioni["abc"]; ok {
		t.Error("Esci non ha tolto la sessione")
	}
}

func TestBridgeOspite(t *testing.T) {
	for _, c := range []struct {
		nome     string
		codice   int
		risposta string
	}{
		{"ospite", http.StatusOK, `{"v":1,"authenticated":false,"user_id":1,"username":"","groups":[]}`},
		{"versione sconosciuta", http.StatusOK, strings.Replace(rispostaModeratore, `"v":1`, `"v":2`, 1)},
		{"errore HTTP", http.StatusForbidden, rispostaModeratore},
		{"redirect", http.StatusFound, rispostaModeratore},
		{"non JSON", http.StatusOK, "<html>"},
	} {
		b, _, _ := bridgeFinto(t, c.codice, c.risposta)
		if u := b.Utente(richiestaForum()); u != nil {
			t.Errorf("%s: utente %+v", c.nome, u)
		}
		if len(b.sessioni) != 0 {
			t.Errorf("%s: sessione creata", c.nome)
		}
	}
}

func TestBridgeNonRisponde(t *testing.T) {
	b, _, _ := bridgeFinto(t, http.StatusOK, rispostaModeratore)
	b.url = "http://127.0.0.1:1/"
	if u := b.Utente(richiestaForum()); u != nil {
		t.Errorf("utente %+v", u)
	}
}

func TestBridgeSenzaSessioneDelForum(t *testing.T) {
	b, chiamate, _ := bridgeFinto(t, http.StatusOK, rispostaModeratore)
	if u := b.Utente(httptest.NewRequest("GET", "/admin/", nil)); u != nil || *chiamate != 0 {
		t.Errorf("utente %+v, %d chiamate", u, *chiamate)
	}
}

func TestBridgeAccedi(t *testing.T) {
	b, _, _ := bridgeFinto(t, http.StatusOK, "")
	w := httptest.NewRecorder()
	b.Accedi(w, httptest.NewRequest("GET", "/admin/", nil))
	if w.Code != http.StatusUnauthorized || !strings.Contains(w.Body.String(), "ucp.php?mode=login") {
		t.Errorf("HTTP %d: %s", w.Code, w.Body)
	}
}
