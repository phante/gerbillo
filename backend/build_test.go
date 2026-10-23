package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// contenutiProva scrive un Repository dei contenuti minimo: due categorie, tre album, quattro Foto.
func contenutiProva(t *testing.T) (dir, indice string) {
	dir = t.TempDir()
	scrivi := func(p, s string) {
		p = filepath.Join(dir, p)
		os.MkdirAll(filepath.Dir(p), 0o755)
		if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	scrivi("importazione/album.yaml", `album:
    14: {categoria: 10, foto: 2, title: La Forra}
    20: {categoria: 6, foto: 1, title: Sconosciuta}
    30: {categoria: 10074, foto: 1, title: Varie}
categorie:
    2: {nome: Dighe - Dams, parent: 0}
    6: {nome: Friuli, parent: 2}
    10: {nome: Vajont, parent: 6}
`)
	scrivi("arricchimento/mappature.yaml", `categorie:
    10: {coppermine: x, origine: automatica, slug: [dighe/vajont]}
album:
    20: {coppermine: x, origine: manuale, slug: [dighe/inesistente], tags: [da-separare, cellina]}
`)
	foto := map[string]string{
		"1": "pid: 1\ntitle: Diga\nformato: immagine\ndate: \"2006-12-27T12:23:55+01:00\"\ncontributore: {id: 25, nome: Mario}\ncoppermine: {aid: 14, filepath: userpics/1/, filename: a.jpg}\n",
		"2": "pid: 2\nformato: pdf\ncoppermine: {aid: 14, filepath: userpics/1/, filename: b.pdf}\n",
		"3": "pid: 3\ntitle: Altrove\nformato: immagine\ncoppermine: {aid: 20, filepath: userpics/1/, filename: c.jpg}\n",
		"4": "pid: 4\nformato: immagine\ncontributore: {id: 74, nome: Pomase}\ncoppermine: {aid: 30, filepath: userpics/2/, filename: d.jpg}\n",
	}
	for pid, y := range foto {
		scrivi("importazione/foto/"+pid+".yaml", y)
	}
	scrivi("arricchimento/foto/1.yaml", "datazione: 195X\nauthors: [Ente]\nlicenza: CC-BY-4.0\n")
	indice = filepath.Join(dir, "indice.json")
	scrivi("indice.json", `[{"slug":"dighe/vajont","title":"Vajont","geolocation":{"stato":"Italia","regione":"Friuli-Venezia Giulia"}}]`)
	return dir, indice
}

func TestLeggiContenuti(t *testing.T) {
	dir, indice := contenutiProva(t)
	c, err := leggiContenuti(dir, indice)
	if err != nil {
		t.Fatal(err)
	}
	f := map[int]*fotoPubblicata{}
	for _, x := range c.foto {
		f[x.Pid] = x
	}
	if s := f[1].Slug; len(s) != 1 || s[0] != "dighe/vajont" {
		t.Errorf("pid 1 eredita dalla categoria Vajont: %v", s)
	}
	if f[1].Datazione != "195X" || f[1].Authors[0] != "Ente" || f[1].Title != "Diga" {
		t.Errorf("arricchimento della Foto non applicato: %+v", f[1])
	}
	if f[1].Geolocation.Regione != "Friuli-Venezia Giulia" {
		t.Errorf("geolocation non ereditata: %+v", f[1].Geolocation)
	}
	if len(f[3].Slug) != 0 || len(c.ignoti) != 1 {
		t.Errorf("slug sconosciuto: %v, ignoti %v", f[3].Slug, c.ignoti)
	}
	if tg := f[3].tagPubblici(); len(tg) != 1 || tg[0] != "cellina" {
		t.Errorf("tag pubblici %v", tg)
	}
}

func TestBuild(t *testing.T) {
	dir, indice := contenutiProva(t)
	conf := filepath.Join(dir, "gerbillo.toml")
	os.WriteFile(conf, []byte("contenuti = \""+dir+"\"\n"), 0o644)
	dist := filepath.Join(dir, "dist")
	if err := build([]string{"-config", conf, "-indice", indice, "-dist", dist, "-base", "/g/"}); err != nil {
		t.Fatal(err)
	}
	leggi := func(p string) string {
		b, err := os.ReadFile(filepath.Join(dist, p))
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	foto1 := leggi("f/1/index.html")
	for _, atteso := range []string{
		`src="/g/f/1/schermo.webp"`,
		"[url=https://progettodighe.it/gallery/f/1/][img]https://progettodighe.it/gallery/f/1/mini.webp[/img][/url]",
		`href="/g/f/2/"`, // successiva nello stesso elenco (Vajont)
		"195X", "27/12/2006", `href="/g/contributori/25/"`,
		`href="https://creativecommons.org/licenses/by/4.0/">CC BY 4.0`, // scelta per la Foto
		`data-copy-url="https://progettodighe.it/gallery/f/1/"`,
	} {
		if !strings.Contains(foto1, atteso) {
			t.Errorf("f/1 senza %q", atteso)
		}
	}
	if !strings.Contains(leggi("f/3/index.html"), `by-sa/4.0/">CC BY-SA 4.0</a>`) { // nei metadati, non nel footer
		t.Error("licenza predefinita assente")
	}
	if strings.Contains(leggi("f/3/index.html"), "da-separare") {
		t.Error("tag interno pubblicato")
	}
	if !strings.Contains(leggi("f/2/index.html"), "statico/documento.svg") {
		t.Error("PDF senza icona")
	}
	for _, p := range []string{"index.html", "dighe/vajont/index.html", "altre/index.html", "altre/2/index.html", "altre/1/index.html", "contributori/74/index.html", "statico/stile.css"} {
		leggi(p)
	}
	var idx map[string][]int
	if err := json.Unmarshal([]byte(leggi("indice-foto.json")), &idx); err != nil || len(idx["dighe/vajont"]) != 2 {
		t.Errorf("Indice delle Foto %v %v", idx, err)
	}
}
