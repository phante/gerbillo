package main

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestRadice(t *testing.T) {
	for in, out := range map[string]string{"/admin/": "./", "/admin/stato": "./", "/admin/foto/": "../", "/admin/foto/12": "../"} {
		if r := radice(httptest.NewRequest("GET", in, nil)); r != out {
			t.Errorf("%s: %q, atteso %q", in, r, out)
		}
	}
}

// Ogni file statico del Backend è nel binario, e nessuna pagina carica risorse da altri domini.
func TestStaticiBackend(t *testing.T) {
	for nome := range staticiBackend {
		w := httptest.NewRecorder()
		statico(w, httptest.NewRequest("GET", "/admin/statico/"+nome, nil))
		if w.Code != 200 || w.Body.Len() == 0 {
			t.Errorf("%s: HTTP %d, %d byte", nome, w.Code, w.Body.Len())
		}
	}
	w := httptest.NewRecorder()
	statico(w, httptest.NewRequest("GET", "/admin/statico/altro.js", nil))
	if w.Code != 404 {
		t.Errorf("file non elencato: HTTP %d", w.Code)
	}
	esterno := regexp.MustCompile(`(src|href)="(https?:)?//`)
	for _, m := range modelliBackend.Templates() {
		if m.Tree != nil && esterno.MatchString(m.Tree.Root.String()) {
			t.Errorf("il modello %s carica risorse da un altro dominio", m.Name())
		}
	}
}

func TestStatoWidget(t *testing.T) {
	dir := t.TempDir()
	dirCgroup = dir + "/"
	t.Cleanup(func() { dirCgroup = "/sys/fs/cgroup/" })
	os.WriteFile(dir+"/memory.current", []byte("314572800\n"), 0o644) // 300 MiB
	os.WriteFile(dir+"/memory.max", []byte("402653184\n"), 0o644)     // 384 MiB
	os.WriteFile(dir+"/memory.stat", []byte("anon 52428800\nfile 262144000\nshmem 10485760\n"), 0o644)
	host := dir + "/stato.json"
	os.WriteFile(host, []byte("container: running=true oom_killed=false restart=2\n"), 0o644)
	t.Setenv("GERBILLO_STATO_HOST", host)
	r := &redazione{build: func() error { return nil }}
	r.ultimo = &esitoBuild{Riuscito: true, Fine: time.Date(2026, 10, 25, 9, 30, 0, 0, roma), Durata: 13 * time.Second}
	w := httptest.NewRecorder()
	statoWidget(r)(w, httptest.NewRequest("GET", "/admin/stato.json", nil))
	var s statoJSON
	if err := json.Unmarshal(w.Body.Bytes(), &s); err != nil {
		t.Fatal(err)
	}
	// 300 MiB meno 240 di cache: la tmpfs (shmem) resta memoria vera.
	if s.Memoria != (memoria{Vera: 60, Cache: 240, Max: 384}) || s.Riavvii != 2 {
		t.Errorf("stato: %+v", s)
	}
	if b := s.Build; b == nil || b.Esito != "riuscito" || b.Fine != "09:30" || b.Durata != "13s" || b.InCorso {
		t.Errorf("build: %+v", b)
	}
	w = httptest.NewRecorder()
	statoWidget(nil)(w, httptest.NewRequest("GET", "/admin/stato.json", nil))
	if strings.Contains(w.Body.String(), "build") {
		t.Errorf("senza redazione c'è il build: %s", w.Body)
	}
}
