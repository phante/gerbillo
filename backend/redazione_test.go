package main

import (
	"maps"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// redazioneProva è una redazione sul Repository dei contenuti di contenutiProva, già in git.
func redazioneProva(t *testing.T) *redazione {
	dir, indice := contenutiProva(t)
	if err := registra(dir, ".", "contenuti di prova", ""); err != nil {
		t.Fatal(err)
	}
	return &redazione{contenuti: dir, indice: indice, build: func() error { return nil }}
}

func gitLog(t *testing.T, dir, formato string) string {
	out, err := exec.Command("git", "-C", dir, "log", "-1", "--format="+formato).Output()
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(string(out))
}

// moduloDi sono i valori che il modulo mostra per la Foto: quelli pubblicati.
func moduloDi(t *testing.T, r *redazione, pid int) map[string]string {
	pub, _, _, err := r.fotoDaModificare(pid)
	if err != nil {
		t.Fatal(err)
	}
	m := map[string]string{}
	for _, c := range campiEditoriali {
		m[c.chiave] = c.valore(pub)
	}
	return m
}

func TestSalvaFoto(t *testing.T) {
	r := redazioneProva(t)
	admin := &utente{ID: 2, Nome: "admin", Gruppi: []string{gruppoAdmin}}
	file := filepath.Join(r.contenuti, "arricchimento", "foto", "1.yaml")
	m := moduloDi(t, r, 1)
	if cambiati, err := r.salva(1, m, admin); err != nil || len(cambiati) != 0 {
		t.Fatalf("salvataggio senza modifiche: %v, %v", cambiati, err)
	}
	if gitLog(t, r.contenuti, "%s") != "contenuti di prova" {
		t.Error("commit senza modifiche")
	}
	m["title"] = "  Diga del Vajont  "
	m["tags"] = "vajont\n\n longarone \n"
	m["geolocation.coords"] = "46.2672, 12.3292"
	cambiati, err := r.salva(1, m, admin)
	if err != nil || strings.Join(cambiati, ",") != "title,tags,geolocation.coords" {
		t.Fatalf("salvataggio: %v, %v", cambiati, err)
	}
	b, _ := os.ReadFile(file)
	// Solo i campi diversi dall'Importazione; licenza, che il modulo non tocca, resta.
	atteso := `authors:
    - Ente
datazione: 195X
geolocation:
    coords:
        latitudine: 46.2672
        longitudine: 12.3292
licenza: CC-BY-4.0
tags:
    - vajont
    - longarone
title: Diga del Vajont
`
	if string(b) != atteso {
		t.Errorf("arricchimento:\n%s\natteso:\n%s", b, atteso)
	}
	if a := gitLog(t, r.contenuti, "%an <%ae>|%cn|%s"); a != "admin <utente-2@gerbillo.invalid>|gerbillo|foto 1: title, tags, geolocation.coords" {
		t.Errorf("commit: %s", a)
	}
	c, err := leggiContenuti(r.contenuti, r.indice, nuovoLivello(r.contenuti, ""))
	if err != nil {
		t.Fatal(err)
	}
	if f := c.foto[0]; f.Title != "Diga del Vajont" || f.Geolocation.Coords == nil || f.Geolocation.Coords.Longitudine != 12.3292 || f.Datazione != "195X" {
		t.Errorf("Foto pubblicata: %+v", f)
	}
	// Riportare i campi al valore importato li toglie dal file.
	m["title"], m["tags"], m["geolocation.coords"], m["authors"], m["datazione"] = "Diga", "", "", "", ""
	if _, err := r.salva(1, m, admin); err != nil {
		t.Fatal(err)
	}
	b, _ = os.ReadFile(file)
	if string(b) != "licenza: CC-BY-4.0\n" {
		t.Errorf("arricchimento dopo il ritorno all'Importazione:\n%s", b)
	}
	// Un campo che l'Importazione ha e il modulo svuota si scrive vuoto.
	m = moduloDi(t, r, 3)
	m["title"] = ""
	if _, err := r.salva(3, m, admin); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(filepath.Join(r.contenuti, "arricchimento", "foto", "3.yaml")); string(b) != "title: \"\"\n" {
		t.Errorf("titolo svuotato:\n%s", b)
	}
	m["title"] = "Altrove" // di nuovo quello importato: il file sparisce
	if _, err := r.salva(3, m, admin); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(r.contenuti, "arricchimento", "foto", "3.yaml")); !os.IsNotExist(err) {
		t.Error("file di arricchimento non tolto")
	}
	for campo, valore := range map[string]string{"slug": "dighe/inventata", "geolocation.coords": "46,2672"} {
		m := moduloDi(t, r, 4)
		m[campo] = valore
		if _, err := r.salva(4, m, admin); err == nil {
			t.Errorf("%s %q accettato", campo, valore)
		}
	}
}

func TestRichiediBuild(t *testing.T) {
	var mu sync.Mutex
	build := 0
	sblocca := make(chan bool)
	r := &redazione{build: func() error {
		mu.Lock()
		build++
		mu.Unlock()
		<-sblocca
		return nil
	}}
	r.richiediBuild()
	for range 3 { // durante il primo build
		r.richiediBuild()
	}
	if s := r.statoBuild(); s != "in corso, poi un altro" {
		t.Errorf("stato %q", s)
	}
	sblocca <- true
	sblocca <- true
	for i := 0; i < 100 && strings.Contains(r.statoBuild(), "in corso"); i++ {
		time.Sleep(10 * time.Millisecond)
	}
	if build != 2 || !strings.HasPrefix(r.statoBuild(), "ultimo riuscito") {
		t.Errorf("%d build, stato %q", build, r.statoBuild())
	}
}

func TestPaginaRedazione(t *testing.T) {
	r := redazioneProva(t)
	builds := 0
	r.build = func() error { builds++; return nil }
	h := soloModeratore(&utenteFinto{u: &utente{ID: 2, Nome: "admin", Gruppi: []string{gruppoAdmin}}}, r.foto)
	w := httptest.NewRecorder()
	h(w, httptest.NewRequest("GET", "/admin/foto/1", nil))
	pagina := w.Body.String()
	for _, atteso := range []string{`name="viewport"`, `value="Diga"`, `<option value="Ente" data-opzione="{&#34;value&#34;:&#34;Ente&#34;,&#34;text&#34;:&#34;Ente&#34;}" selected>Ente</option>`, "Importato: (vuoto)", `<base href="../">`, `href="../f/1/"`, `hx-post="foto/1"`, `<input type="hidden" name="modulo_completo" value="1">`} {
		if !strings.Contains(pagina, atteso) {
			t.Errorf("modulo senza %q:\n%s", atteso, pagina)
		}
	}
	v := url.Values{}
	for k, x := range moduloDi(t, r, 1) {
		v.Set(k, x)
	}
	v.Set("description", "Il coronamento")
	v.Set("modulo_completo", "1")
	req := httptest.NewRequest("POST", "/admin/foto/1", strings.NewReader(v.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	w = httptest.NewRecorder()
	h(w, req)
	// htmx riceve solo il modulo, con il valore salvato e l'esito.
	if b := w.Body.String(); w.Code != http.StatusOK || strings.Contains(b, "<html") ||
		!strings.Contains(b, "Salvato: description") || !strings.Contains(b, ">Il coronamento</textarea>") {
		t.Errorf("POST: HTTP %d\n%s", w.Code, b)
	}
	for i := 0; i < 100 && strings.Contains(r.statoBuild(), "in corso"); i++ {
		time.Sleep(10 * time.Millisecond)
	}
	if builds != 1 {
		t.Errorf("%d build dopo il salvataggio", builds)
	}
	w = httptest.NewRecorder()
	h(w, httptest.NewRequest("GET", "/admin/foto/?pid=3", nil))
	if w.Header().Get("Location") != "3" {
		t.Errorf("apri per pid: %v", w.Header())
	}
	w = httptest.NewRecorder()
	h(w, httptest.NewRequest("GET", "/admin/foto/99", nil))
	if w.Code != http.StatusNotFound {
		t.Errorf("Foto inesistente: HTTP %d", w.Code)
	}
}

// #78: il Punto della prima Entità si mostra come riferimento e non finisce nella Foto.
func TestPuntoEreditatoNonSiCopia(t *testing.T) {
	r := redazioneProva(t)
	os.WriteFile(r.indice, []byte(`[{"slug":"dighe/vajont","title":"Vajont","geolocation":{"coords":{"latitudine":46.2672,"longitudine":12.3292}}}]`), 0o644)
	h := soloModeratore(&utenteFinto{u: &utente{ID: 2, Nome: "admin", Gruppi: []string{gruppoAdmin}}}, r.foto)
	w := httptest.NewRecorder()
	h(w, httptest.NewRequest("GET", "/admin/foto/1", nil))
	b := w.Body.String()
	if !strings.Contains(b, "è il Punto di Vajont: vale se la Foto non ne ha uno.") || strings.Contains(b, "usa-diga") {
		t.Errorf("scheda senza il riferimento, o con il pulsante:\n%s", b)
	}
	m := moduloDi(t, r, 1)
	m["description"] = "Il coronamento"
	if _, err := r.salva(1, m, &utente{ID: 2, Nome: "admin"}); err != nil {
		t.Fatal(err)
	}
	y, err := os.ReadFile(filepath.Join(r.contenuti, "arricchimento", "foto", "1.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(y), "coords") {
		t.Errorf("Punto copiato nella Foto:\n%s", y)
	}
}

// #85: un POST incompleto non svuota la Foto e non risponde «Salvato».
func TestPostIncompleto(t *testing.T) {
	r := redazioneProva(t)
	h := soloModeratore(&utenteFinto{u: &utente{ID: 2, Nome: "admin", Gruppi: []string{gruppoAdmin}}}, r.foto)
	prima := moduloDi(t, r, 1)
	testa := gitLog(t, r.contenuti, "%H")
	for _, c := range []struct{ nome, tipo, corpo string }{
		{"vuoto", "application/x-www-form-urlencoded", ""},
		{"senza sentinella", "application/x-www-form-urlencoded", "title=Solo+il+titolo"},
		{"troncato", "application/x-www-form-urlencoded", "title=Ponte&slug=%ZZ"},
		{"multipart", "multipart/form-data; boundary=x", "--x\r\nContent-Disposition: form-data; name=\"modulo_completo\"\r\n\r\n1\r\n--x--\r\n"},
	} {
		req := httptest.NewRequest("POST", "/admin/foto/1", strings.NewReader(c.corpo))
		req.Header.Set("Content-Type", c.tipo)
		w := httptest.NewRecorder()
		h(w, req)
		if w.Code != http.StatusBadRequest || strings.Contains(w.Body.String(), "Salvato") {
			t.Errorf("%s: HTTP %d\n%s", c.nome, w.Code, w.Body)
		}
	}
	if !maps.Equal(moduloDi(t, r, 1), prima) || gitLog(t, r.contenuti, "%H") != testa {
		t.Error("un POST incompleto ha cambiato la Foto")
	}
}

// #51: la Datazione non valida non si salva; quella in italiano si salva in EDTF.
func TestSalvaDatazione(t *testing.T) {
	r := redazioneProva(t)
	admin := &utente{ID: 2, Nome: "admin"}
	m := moduloDi(t, r, 4)
	m["datazione"] = "Pippo"
	if _, err := r.salva(4, m, admin); err == nil || !strings.Contains(err.Error(), "non valida") {
		t.Errorf("Pippo: %v", err)
	}
	m["datazione"] = "maggio 2006"
	if _, err := r.salva(4, m, admin); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(filepath.Join(r.contenuti, "arricchimento", "foto", "4.yaml")); string(b) != "datazione: 2006-05\n" {
		t.Errorf("arricchimento:\n%s", b)
	}
	// Il suggerimento da user3 compare solo se la Datazione manca.
	h := soloModeratore(&utenteFinto{u: &utente{ID: 2, Nome: "admin", Gruppi: []string{gruppoAdmin}}}, r.foto)
	w := httptest.NewRecorder()
	h(w, httptest.NewRequest("GET", "/admin/foto/5", nil))
	if b := w.Body.String(); !strings.Contains(b, "«Ottobre 2006»") || !strings.Contains(b, "Usa 2006-10") {
		t.Errorf("scheda 5 senza il suggerimento da user3")
	}
	w = httptest.NewRecorder()
	datazione(w, httptest.NewRequest("GET", "/admin/datazione?datazione=anni+%2760", nil))
	if b := w.Body.String(); !strings.Contains(b, "= anni &#39;60") || !strings.Contains(b, "si salva come 196X") {
		t.Errorf("anteprima: %s", b)
	}
}

// #52: le Categorie si scelgono dal vocabolario; fuori dal vocabolario non si salvano.
func TestSalvaCategorie(t *testing.T) {
	r := redazioneProva(t)
	r.usaVocabolario(vocabolario{Categories: []string{"documenti", "tecnica/turbine/pelton"},
		TagsInterni: []tagInterno{{"da-collegare", "manca lo Slug"}}})
	admin := &utente{ID: 2, Nome: "admin"}
	m := moduloDi(t, r, 4)
	m["categories"] = "tecnica/turbine/inventata"
	if _, err := r.salva(4, m, admin); err == nil || !strings.Contains(err.Error(), "vocabolario") {
		t.Errorf("Categoria fuori dal vocabolario: %v", err)
	}
	// Dal modulo le Categorie arrivano come più valori dello stesso campo.
	h := soloModeratore(&utenteFinto{u: &utente{ID: 2, Nome: "admin", Gruppi: []string{gruppoAdmin}}}, r.foto)
	v := url.Values{}
	for k, x := range moduloDi(t, r, 4) {
		v.Set(k, x)
	}
	v["categories"] = []string{"documenti", "tecnica/turbine/pelton"}
	v.Set("modulo_completo", "1")
	req := httptest.NewRequest("POST", "/admin/foto/4", strings.NewReader(v.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	h(w, req)
	b := w.Body.String()
	for _, atteso := range []string{"Salvato: categories", `<optgroup label="tecnica">`, `<option value="tecnica/turbine/pelton" selected>turbine › pelton</option>`,
		`<span class="tag-interno">da-collegare</span> manca lo Slug`} {
		if !strings.Contains(b, atteso) {
			t.Errorf("scheda senza %q", atteso)
		}
	}
}
