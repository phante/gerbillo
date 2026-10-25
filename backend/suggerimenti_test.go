package main

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSuggerimenti(t *testing.T) {
	r := redazioneConIndice(t)
	h := soloModeratore(&utenteFinto{u: &utente{ID: 2, Nome: "admin", Gruppi: []string{gruppoAdmin}}}, r.suggerimenti)
	chiedi := func(u string) []suggerimento {
		w := httptest.NewRecorder()
		h(w, httptest.NewRequest("GET", u, nil))
		var s []suggerimento
		if err := json.Unmarshal(w.Body.Bytes(), &s); err != nil {
			t.Fatalf("%s: %v %s", u, err, w.Body)
		}
		return s
	}
	if s := chiedi("/admin/suggerimenti/slug?q=vaj"); len(s) != 1 || s[0] != (suggerimento{Value: "dighe/vajont", Text: "Vajont · dighe/vajont"}) {
		t.Errorf("slug: %+v", s)
	}
	if s := chiedi("/admin/suggerimenti/tags?q=da-"); len(s) < 2 || !s[0].Interno {
		t.Errorf("tag interni: %+v", s)
	}
	// Gli Autori già usati e i nomi degli alias di autori.yaml.
	a := chiedi("/admin/suggerimenti/authors?q=")
	var nomi []string
	for _, x := range a {
		nomi = append(nomi, x.Value)
	}
	if j := strings.Join(nomi, ","); !strings.Contains(j, "Ente") || !strings.Contains(j, "paolog") {
		t.Errorf("autori: %s", j)
	}
	// Nel modulo un Tag che nessuno usa è «nuovo», un tag interno no.
	o := r.opzioniChip("tags", []string{"inventato", "da-collegare"})
	if !strings.Contains(o[0].Dati, `"nuovo":true`) || !strings.Contains(o[1].Dati, `"interno":true`) || strings.Contains(o[1].Dati, "nuovo") {
		t.Errorf("chip: %+v", o)
	}
}
