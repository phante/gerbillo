package main

import (
	"net/http/httptest"
	"regexp"
	"testing"
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
