package main

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"
)

// utenteFinto riconosce sempre lo stesso utente, o nessuno.
type utenteFinto struct {
	u       *utente
	accessi int
}

func (f *utenteFinto) Utente(*http.Request) *utente { return f.u }
func (f *utenteFinto) Accedi(w http.ResponseWriter, _ *http.Request) {
	f.accessi++
	w.WriteHeader(http.StatusUnauthorized)
}
func (f *utenteFinto) Esci(http.ResponseWriter, *http.Request) { f.u = nil }

// La password inviata riporta l'id del messaggio, che poi si cancella dalla stessa chat.
func TestTelegramMandaECancella(t *testing.T) {
	var cancellato, chat string
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/botTOK/sendMessage":
			w.Write([]byte(`{"ok":true,"result":{"message_id":42}}`))
		case "/botTOK/deleteMessage":
			cancellato, chat = r.FormValue("message_id"), r.FormValue("chat_id")
		default:
			t.Errorf("percorso inatteso: %s", r.URL.Path)
		}
	}))
	defer s.Close()
	vecchia := apiTelegram
	apiTelegram = s.URL
	defer func() { apiTelegram = vecchia }()
	t.Setenv("TELEGRAM_TOKEN", "TOK")
	t.Setenv("TELEGRAM_CHAT_ID", "7")

	id, err := telegram("ciao")
	if err != nil || id != 42 {
		t.Fatalf("telegram = %d, %v", id, err)
	}
	if err := telegramCancella(id); err != nil {
		t.Fatal(err)
	}
	if cancellato != "42" || chat != "7" {
		t.Errorf("cancellato %q dalla chat %q", cancellato, chat)
	}
}

func TestSoloModeratore(t *testing.T) {
	var visto *utente
	pagina := func(w http.ResponseWriter, r *http.Request) { visto = utenteDi(r) }
	casi := []struct {
		nome   string
		u      *utente
		codice int
	}{
		{"ospite", nil, http.StatusUnauthorized},
		{"utente registrato", &utente{ID: 7, Nome: "nuovo", Gruppi: []string{gruppoRegistrati}}, http.StatusForbidden},
		{"moderatore globale", &utente{ID: 8, Nome: "mod", Gruppi: []string{gruppoRegistrati, gruppoModeratori}}, http.StatusOK},
		{"admin", &utente{ID: 2, Nome: "admin", Gruppi: []string{gruppoAdmin}}, http.StatusOK},
	}
	for _, c := range casi {
		visto = nil
		f := &utenteFinto{u: c.u}
		w := httptest.NewRecorder()
		soloModeratore(f, pagina)(w, httptest.NewRequest("GET", "/admin/", nil))
		if w.Code != c.codice {
			t.Errorf("%s: HTTP %d, atteso %d", c.nome, w.Code, c.codice)
		}
		if (c.codice == http.StatusOK) != (visto == c.u && visto != nil) {
			t.Errorf("%s: la pagina ha visto %+v", c.nome, visto)
		}
		if (c.u == nil) != (f.accessi == 1) {
			t.Errorf("%s: Accedi chiamato %d volte", c.nome, f.accessi)
		}
	}
}

func TestPasswordAnteprima(t *testing.T) {
	p := &passwordAnteprima{password: "segreta", scadenza: time.Now().Add(time.Hour)}
	r := httptest.NewRequest("GET", "/admin/", nil)
	if p.Utente(r) != nil {
		t.Error("senza password non è un ospite")
	}
	r.SetBasicAuth(utenteAnteprima, "sbagliata")
	if p.Utente(r) != nil {
		t.Error("password sbagliata riconosciuta")
	}
	w := httptest.NewRecorder()
	p.Accedi(w, r)
	if w.Code != http.StatusUnauthorized || w.Header().Get("WWW-Authenticate") == "" {
		t.Errorf("Accedi senza password: HTTP %d, %v", w.Code, w.Header())
	}
	// La scelta da sola non basta.
	r.AddCookie(&http.Cookie{Name: cookieAnteprima, Value: "admin"})
	if p.Utente(r) != nil {
		t.Error("cookie senza password riconosciuto")
	}
}

// #66: dopo la password si sceglie con chi entrare, e i permessi seguono la scelta.
func TestSceltaAnteprima(t *testing.T) {
	p := &passwordAnteprima{password: "segreta", scadenza: time.Now().Add(time.Hour)}
	r := httptest.NewRequest("GET", "/admin/", nil)
	r.SetBasicAuth(utenteAnteprima, "segreta")
	if p.Utente(r) != nil {
		t.Error("utente riconosciuto prima della scelta")
	}
	w := httptest.NewRecorder()
	p.Accedi(w, r)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "Gallery unlimited") {
		t.Errorf("pagina di scelta: HTTP %d: %s", w.Code, w.Body)
	}

	for _, c := range []struct {
		chiave     string
		moderatore bool
		gruppo     string
	}{{"admin", true, gruppoAdmin}, {"moderatore", true, gruppoModeratori}, {"unlimited", false, gruppoUnlimited}, {"nuovo", false, gruppoNuovi}} {
		post := httptest.NewRequest("POST", "/admin/foto/?x=1", strings.NewReader(campoAnteprima+"="+c.chiave))
		post.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		post.SetBasicAuth(utenteAnteprima, "segreta")
		w := httptest.NewRecorder()
		p.Accedi(w, post)
		if w.Code != http.StatusSeeOther || w.Header().Get("Location") != "?x=1" {
			t.Fatalf("%s: HTTP %d, Location %q", c.chiave, w.Code, w.Header().Get("Location"))
		}
		g := httptest.NewRequest("GET", "/admin/", nil)
		g.SetBasicAuth(utenteAnteprima, "segreta")
		g.AddCookie(w.Result().Cookies()[0])
		u := p.Utente(g)
		if u == nil || u.Moderatore() != c.moderatore || !slices.Contains(u.Gruppi, c.gruppo) {
			t.Errorf("%s: utente %+v", c.chiave, u)
		}
	}

	// Una scelta che non esiste non entra.
	r.AddCookie(&http.Cookie{Name: cookieAnteprima, Value: "root"})
	if u := p.Utente(r); u != nil {
		t.Errorf("scelta inventata: %+v", u)
	}
}

// #66: chi non è Moderatore vede con chi è entrato, con il nome sottoposto a escape.
func TestNonModeratore(t *testing.T) {
	w := httptest.NewRecorder()
	u := &utente{ID: 7, Nome: "<b>x", Gruppi: []string{gruppoUnlimited}}
	soloModeratore(&utenteFinto{u: u}, func(http.ResponseWriter, *http.Request) {})(w, httptest.NewRequest("GET", "/admin/foto/8", nil))
	if w.Code != http.StatusForbidden || strings.Contains(w.Body.String(), "<b>x") ||
		!strings.Contains(w.Body.String(), gruppoUnlimited) || !strings.Contains(w.Body.String(), `href="../esci"`) {
		t.Errorf("HTTP %d: %s", w.Code, w.Body)
	}
}

// Dietro nginx il Backend non vede il prefisso /gerbillo/: il redirect deve essere relativo.
func TestConBarra(t *testing.T) {
	for in, out := range map[string]string{"/admin": "admin/", "/admin/foto": "foto/"} {
		w := httptest.NewRecorder()
		conBarra(w, httptest.NewRequest("GET", in, nil))
		if w.Code != http.StatusSeeOther || w.Header().Get("Location") != out {
			t.Errorf("%s: HTTP %d, Location %q", in, w.Code, w.Header().Get("Location"))
		}
	}
}

// #58: il cookie segnale va ai Moderatori, non agli altri, ed esci lo toglie.
func TestCookieModeratore(t *testing.T) {
	pagina := func(http.ResponseWriter, *http.Request) {}
	for _, c := range []struct {
		u         *utente
		segnalato bool
	}{{nil, false}, {&utente{ID: 7, Gruppi: []string{gruppoRegistrati}}, false}, {&utente{ID: 8, Gruppi: []string{gruppoModeratori}}, true}} {
		w := httptest.NewRecorder()
		soloModeratore(&utenteFinto{u: c.u}, pagina)(w, httptest.NewRequest("GET", "/admin/", nil))
		if got := strings.Contains(w.Header().Get("Set-Cookie"), cookieModeratore+"=1; Path=/"); got != c.segnalato {
			t.Errorf("%+v: Set-Cookie %q", c.u, w.Header().Get("Set-Cookie"))
		}
	}
	w := httptest.NewRecorder()
	f := &utenteFinto{u: &utente{ID: 8}}
	esci(f)(w, httptest.NewRequest("GET", "/admin/esci", nil))
	if !strings.Contains(w.Header().Get("Set-Cookie"), "Max-Age=0") || f.u != nil {
		t.Errorf("esci: %q, utente %+v", w.Header().Get("Set-Cookie"), f.u)
	}
}
