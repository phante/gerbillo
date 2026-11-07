package main

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
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

// #57: gli altri filtri, l'ordinamento e il campo della barra.
func TestAltriFiltri(t *testing.T) {
	r := redazioneConIndice(t)
	idx := r.indiceCorrente()
	casi := map[string][]int{
		"e=dighe/vajont":            {1, 2},
		"a=30":                      {4, 5},
		"f=documento":               {2},
		"f=immagine&a=14":           {1},
		"q=diga":                    {1},
		"u=74":                      {4},
		"au=Ente":                   {1},
		"au=Ente&s=senza-datazione": nil,
		"t=da-collegare&a=30":       {4, 5},
		"o=pid&a=30":                {4, 5},
	}
	for q, atteso := range casi {
		v, _ := url.ParseQuery(q)
		if p := pids(leggiFiltro(v).applica(idx)); !slices.Equal(p, atteso) {
			t.Errorf("%s: %v, atteso %v", q, p, atteso)
		}
	}
	// I conteggi di una faccetta rispettano gli altri filtri, non il suo.
	v, _ := url.ParseQuery("a=30&f=immagine")
	for _, fc := range leggiFiltro(v).faccette(idx, r.vociNote().testoSlug) {
		if fc.Param == "a" {
			n := map[string]int{}
			for _, o := range fc.Opzioni {
				n[o.Valore] = o.N
			}
			if fc.Scelto != "30" || n["30"] != 2 || n["14"] != 1 {
				t.Errorf("faccetta album: %+v", fc)
			}
		}
		if fc.Param == "e" && len(fc.Opzioni) != 0 {
			t.Errorf("faccetta Entità con a=30: %+v", fc.Opzioni)
		}
	}
	// L'ordinamento per Datazione mette in fondo le Foto senza.
	v, _ = url.ParseQuery("o=datazione")
	if p := pids(leggiFiltro(v).applica(idx)); p[0] != 1 || p[1] != 3 {
		t.Errorf("per Datazione: %v", p)
	}
	for in, out := range map[string]string{"3": "3", "999": "./?q=999", "https://progettodighe.it/gallery/f/4/": "4", "/gerbillo/f/5": "5", "la diga": "./?q=la+diga"} {
		if d := r.dove(in); d != out {
			t.Errorf("barra %q: %q, atteso %q", in, d, out)
		}
	}
}

// #91: gli avvisi del build arrivano ai Moderatori nella home del Backend.
func TestAvvisiNellaHome(t *testing.T) {
	r := redazioneProva(t)
	conEntitaGallery(t, r.contenuti)
	// Foto 1 e 2 sullo stesso Slug sconosciuto: una riga sola.
	for _, pid := range []string{"1", "2"} {
		os.WriteFile(filepath.Join(r.contenuti, "arricchimento", "foto", pid+".yaml"), []byte("slug: [dighe/tolta]\n"), 0o644)
	}
	r.fileIndice = filepath.Join(t.TempDir(), "redazione.json")
	dist := filepath.Join(t.TempDir(), "dist")
	if err := build([]string{"-contenuti", r.contenuti, "-indice", r.indice, "-dist", dist, "-redazione", r.fileIndice}); err != nil {
		t.Fatal(err)
	}
	if err := r.caricaIndice(); err != nil {
		t.Fatal(err)
	}
	home := soloModeratore(&utenteFinto{u: &utente{ID: 3, Nome: "x", Gruppi: []string{gruppoModeratori}}}, r.admin)
	w := httptest.NewRecorder()
	home(w, httptest.NewRequest("GET", "/admin/", nil))
	for _, atteso := range []string{"Avvisi del build", "slug dighe/tolta non è fra le Entità: 2 Foto", "entita/dighe/vajont.yaml: il sito ha la stessa pagina"} {
		if !strings.Contains(w.Body.String(), atteso) {
			t.Errorf("home senza %q:\n%s", atteso, w.Body)
		}
	}
	// Sono informazioni interne: fuori dalla Gallery pubblica (ADR 0019).
	filepath.WalkDir(dist, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if b, _ := os.ReadFile(p); strings.Contains(string(b), "non è fra le Entità") {
			t.Errorf("avviso pubblicato in %s", p)
		}
		return nil
	})
}
