package main

import (
	"net/http"
	"net/http/httptest"
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
	r.SetBasicAuth(utenteAnteprima, "segreta")
	if u := p.Utente(r); u == nil || !u.Moderatore() || u.ID != idAdminAnteprima {
		t.Errorf("con la password l'utente è l'admin: %+v", u)
	}
	w := httptest.NewRecorder()
	p.Accedi(w, r)
	if w.Code != http.StatusUnauthorized || w.Header().Get("WWW-Authenticate") == "" {
		t.Errorf("Accedi: HTTP %d, %v", w.Code, w.Header())
	}
}
