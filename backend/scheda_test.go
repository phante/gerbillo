package main

import (
	"bytes"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestFileDellaFoto(t *testing.T) {
	var buf bytes.Buffer
	png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 320, 200)))
	chiesti := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		chiesti++
		http.ServeContent(w, r, "a.png", time.Now(), bytes.NewReader(buf.Bytes()))
	}))
	defer srv.Close()
	r := redazioneProva(t)
	r.opz = opzioniBuild{base: "/gallery/", media: srv.URL + "/albums/", pubblico: "https://progettodighe.it/gallery/"}
	w := httptest.NewRecorder()
	r.fileDellaFoto(w, httptest.NewRequest("GET", "/admin/foto/1/file", nil), 1)
	b := w.Body.String()
	for _, atteso := range []string{`href="` + srv.URL + `/albums/userpics/1/a.jpg">Originale`, "thumb_a.jpg", "320 × 200", "kB</td>"} {
		if !strings.Contains(b, atteso) {
			t.Errorf("file senza %q:\n%s", atteso, b)
		}
	}
	// Le informazioni si ricordano: nessuna richiesta in più.
	n := chiesti
	r.fileDellaFoto(httptest.NewRecorder(), httptest.NewRequest("GET", "/admin/foto/1/file", nil), 1)
	if chiesti != n {
		t.Errorf("%d richieste in più", chiesti-n)
	}
	// Senza -media: le Varianti, con l'URL pubblico; l'Originale non è pubblico.
	r.opz.media = ""
	if u := r.opz.assoluto(r.builderScheda().mini(&fotoPubblicata{Pid: 7, Formato: "immagine"})); u != "https://progettodighe.it/gallery/f/7/mini.webp" {
		t.Errorf("mini: %s", u)
	}
	if peso(1536) != "2 kB" || peso(3<<20) != "3,0 MB" {
		t.Errorf("peso: %s, %s", peso(1536), peso(3<<20))
	}
}
