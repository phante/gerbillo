package main

import (
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
	for _, atteso := range []string{`name="viewport"`, `value="Diga"`, `>Ente</textarea>`, "Importato: (vuoto)", `href="../../f/1/"`} {
		if !strings.Contains(pagina, atteso) {
			t.Errorf("modulo senza %q:\n%s", atteso, pagina)
		}
	}
	v := url.Values{}
	for k, x := range moduloDi(t, r, 1) {
		v.Set(k, x)
	}
	v.Set("description", "Il coronamento")
	req := httptest.NewRequest("POST", "/admin/foto/1", strings.NewReader(v.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w = httptest.NewRecorder()
	h(w, req)
	if w.Code != http.StatusSeeOther || w.Header().Get("Location") != "1?salvato=description" {
		t.Errorf("POST: HTTP %d, %v", w.Code, w.Header())
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
