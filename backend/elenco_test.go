package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// redazioneConIndice è redazioneProva con l'indice scritto da un build vero, che i build
// successivi (dopo un salvataggio) riscrivono.
func redazioneConIndice(t *testing.T) *redazione {
	r := redazioneProva(t)
	r.fileIndice = filepath.Join(t.TempDir(), "redazione.json")
	dist := filepath.Join(t.TempDir(), "dist")
	r.build = func() error {
		return build([]string{"-contenuti", r.contenuti, "-indice", r.indice, "-dist", dist, "-redazione", r.fileIndice})
	}
	r.dopoBuild = func() { r.caricaIndice() }
	if err := r.build(); err != nil {
		t.Fatal(err)
	}
	if err := r.caricaIndice(); err != nil {
		t.Fatal(err)
	}
	return r
}

func pids(fs []*fotoIndice) []int {
	var p []int
	for _, f := range fs {
		p = append(p, f.Pid)
	}
	return p
}

func TestIndiceRedazione(t *testing.T) {
	r := redazioneConIndice(t)
	idx := r.indiceCorrente()
	// Per album, poi per pid. Delle riservate ci sono solo quelle pubblicate: l'album di 6, e 9 a mano.
	if p := pids(filtro{}.applica(idx)); !slices.Equal(p, []int{1, 2, 3, 4, 5, 6, 9}) {
		t.Errorf("ordine: %v", p)
	}
	f := idx.Foto[idx.perPid[4]]
	if f.Contributore == nil || f.Contributore.Nome != "Pomase" || f.Album != "Varie" || f.Mini == "" || !strings.Contains(strings.Join(f.Tag, ","), daCollegare) {
		t.Errorf("pid 4: %+v", f)
	}
	casi := map[string][]int{
		"s=da-collegare":                          {4, 5, 6, 9}, // senza slug: il tag si deriva al build
		"s=da-separare":                           {3},
		"s=senza-datazione":                       {2, 4, 5, 6, 9},
		"s=datazione-da-sistemare":                {5},
		"s=da-collegare&s=datazione-da-sistemare": {5},
		"s=inventato":                             {1, 2, 3, 4, 5, 6, 9},
		"s=senza-datazione&s=da-separare":         nil,
	}
	for q, atteso := range casi {
		v, _ := url.ParseQuery(q)
		if p := pids(leggiFiltro(v).applica(idx)); !slices.Equal(p, atteso) {
			t.Errorf("%s: %v, atteso %v", q, p, atteso)
		}
	}
	// I conteggi rispettano gli altri filtri attivi.
	v, _ := url.ParseQuery("s=da-collegare")
	n := map[string]int{}
	for _, x := range leggiFiltro(v).vociDaSistemare(idx) {
		n[x.Chiave] = x.N
	}
	if n["da-collegare"] != 4 || n["datazione-da-sistemare"] != 1 || n["da-separare"] != 0 || n["senza-datazione"] != 4 {
		t.Errorf("conteggi: %v", n)
	}
}

func TestElencoEScheda(t *testing.T) {
	r := redazioneConIndice(t)
	h := soloModeratore(&utenteFinto{u: &utente{ID: 2, Nome: "admin", Gruppi: []string{gruppoAdmin}}}, r.foto)
	get := func(u string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		h(w, httptest.NewRequest("GET", u, nil))
		return w
	}
	b := get("/admin/foto/?s=da-collegare").Body.String()
	for _, atteso := range []string{"<b>4</b> Foto", `href="foto/4?s=da-collegare"`, `aria-current="true"`, "Aggiornato alle"} {
		if !strings.Contains(b, atteso) {
			t.Errorf("elenco senza %q", atteso)
		}
	}
	// La scheda aperta dall'elenco torna all'elenco e passa alla successiva dello stesso filtro.
	b = get("/admin/foto/5?s=da-collegare").Body.String()
	for _, atteso := range []string{`href="foto/?s=da-collegare"`, `href="foto/4?s=da-collegare" rel="prev"`, `href="foto/6?s=da-collegare" rel="next"`, "2 di 4"} {
		if !strings.Contains(b, atteso) {
			t.Errorf("scheda senza %q", atteso)
		}
	}
	// Freschezza: stessa versione → l'avviso; versione diversa → ricarica.
	v := versioneIndice(r.indiceCorrente())
	if w := get("/admin/foto/freschezza?v=" + v); w.Header().Get("HX-Refresh") != "" {
		t.Error("ricarica senza un indice nuovo")
	}
	if w := get("/admin/foto/freschezza?v=0"); w.Header().Get("HX-Refresh") != "true" {
		t.Error("indice nuovo senza ricarica")
	}
	// Un salvataggio che toglie da-collegare fa uscire la Foto dall'elenco a fine build.
	m := moduloDi(t, r, 4)
	m["slug"], m["tags"] = "dighe/vajont", ""
	if _, err := r.salva(4, m, &utente{ID: 2, Nome: "admin"}); err != nil {
		t.Fatal(err)
	}
	if err := r.build(); err != nil {
		t.Fatal(err)
	}
	r.dopoBuild()
	if p := pids(leggiFiltro(url.Values{"s": {"da-collegare"}}).applica(r.indiceCorrente())); !slices.Equal(p, []int{5, 6, 9}) {
		t.Errorf("da-collegare dopo il salvataggio: %v", p)
	}
}

func TestRicalcolaSoloAdmin(t *testing.T) {
	r := redazioneProva(t)
	builds := make(chan bool, 2)
	r.build = func() error { builds <- true; return nil }
	for _, c := range []struct {
		gruppi []string
		codice int
	}{{[]string{gruppoModeratori}, http.StatusForbidden}, {[]string{gruppoAdmin}, http.StatusOK}} {
		h := soloModeratore(&utenteFinto{u: &utente{ID: 3, Nome: "x", Gruppi: c.gruppi}}, r.ricalcola)
		w := httptest.NewRecorder()
		h(w, httptest.NewRequest("POST", "/admin/ricalcola", nil))
		if w.Code != c.codice {
			t.Errorf("%v: HTTP %d", c.gruppi, w.Code)
		}
	}
	<-builds
	home := soloModeratore(&utenteFinto{u: &utente{ID: 3, Nome: "x", Gruppi: []string{gruppoModeratori}}}, r.admin)
	w := httptest.NewRecorder()
	home(w, httptest.NewRequest("GET", "/admin/", nil))
	if strings.Contains(w.Body.String(), "Ricalcola") {
		t.Error("Ricalcola visibile a un Moderatore")
	}
}
