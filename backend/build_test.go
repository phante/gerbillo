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
    10: {slug: [dighe/vajont]} # Dighe - Dams / Friuli / Vajont
album:
    20: {slug: [dighe/inesistente], tags: [da-separare, cellina]}
`)
	foto := map[string]string{
		"1": "pid: 1\ntitle: Diga\nformato: immagine\ndate: \"2006-12-27T12:23:55+01:00\"\ncontributore: {id: 25, nome: Mario}\ncoppermine: {aid: 14, filepath: userpics/1/, filename: a.jpg}\n",
		"2": "pid: 2\nauthors: [\"-\"]\nformato: pdf\ncoppermine: {aid: 14, filepath: userpics/1/, filename: b.pdf}\n",
		"3": "pid: 3\ntitle: Altrove\nauthors: [Il gaurdiano & paolog]\nformato: immagine\ncoppermine: {aid: 20, filepath: userpics/1/, filename: c.jpg}\n",
		"4": "pid: 4\nauthors: [Grezzo]\nformato: immagine\ncontributore: {id: 74, nome: Pomase}\ncoppermine: {aid: 30, filepath: userpics/2/, filename: d.jpg}\n",
	}
	for pid, y := range foto {
		scrivi("importazione/foto/"+pid+".yaml", y)
	}
	scrivi("arricchimento/autori.yaml", "\"Il gaurdiano & paolog\": [Il guardiano, paolog]\n\"-\": []\n")
	scrivi("arricchimento/foto/1.yaml", "datazione: 195X\nauthors: [Ente]\nlicenza: CC-BY-4.0\n")
	indice = filepath.Join(dir, "indice.json")
	scrivi("indice.json", `[{"slug":"dighe/vajont","title":"Vajont","geolocation":{"stato":"Italia","regione":"Friuli-Venezia Giulia"}}]`)
	return dir, indice
}

func TestLeggiContenuti(t *testing.T) {
	dir, indice := contenutiProva(t)
	c, err := leggiContenuti(dir, indice, nuovoLivello(dir, ""))
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
	if a := f[3].Authors; len(a) != 2 || a[0] != "Il guardiano" || a[1] != "paolog" {
		t.Errorf("alias composto: %v", a)
	}
	if len(f[2].Authors) != 0 || len(f[4].Authors) != 1 || f[4].Authors[0] != "Grezzo" {
		t.Errorf("alias scartato o assente: %v, %v", f[2].Authors, f[4].Authors)
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
	// Album personale senza riga né abbinamento: il tag interno si deriva al build (#38).
	if tg := f[4].Tags; len(tg) != 1 || tg[0] != daCollegare || len(f[4].tagPubblici()) != 0 {
		t.Errorf("pid 4 senza da-collegare: %v", tg)
	}
}

// La cartella di -arricchimento si sovrappone file per file a quella dei contenuti (#38, #39).
func TestLeggiContenutiArricchimentoSovrapposto(t *testing.T) {
	dir, indice := contenutiProva(t)
	sopra := t.TempDir()
	os.MkdirAll(filepath.Join(sopra, "foto"), 0o755)
	os.WriteFile(filepath.Join(sopra, "mappature.yaml"), []byte("album:\n    20: {slug: [dighe/vajont]}\n"), 0o644)
	os.WriteFile(filepath.Join(sopra, "foto", "3.yaml"), []byte("title: Sopra\n"), 0o644)
	c, err := leggiContenuti(dir, indice, nuovoLivello(dir, sopra))
	if err != nil {
		t.Fatal(err)
	}
	f := map[int]*fotoPubblicata{}
	for _, x := range c.foto {
		f[x.Pid] = x
	}
	if s := f[3].Slug; len(s) != 1 || s[0] != "dighe/vajont" || f[3].Title != "Sopra" {
		t.Errorf("pid 3 senza la cartella sovrapposta: %+v", f[3])
	}
	// mappature.yaml sovrapposto sostituisce quello dei contenuti: la categoria 10 non ha più riga,
	// e Vajont a due livelli da Dighe non si abbina in automatico.
	if len(f[1].Slug) != 0 || f[1].Datazione != "195X" {
		t.Errorf("pid 1: %+v", f[1])
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
	var cerca indiceRicerca
	if err := json.Unmarshal([]byte(leggi("statico/cerca.json")), &cerca); err != nil {
		t.Fatal(err)
	}
	if len(cerca.Pagine) != 3 || cerca.Pagine[0] != (paginaRicerca{Slug: "dighe/vajont", Titolo: "Vajont", Tipo: "Dighe",
		Luogo: "Friuli-Venezia Giulia, Italia", URL: "/g/dighe/vajont/", Mini: "/g/f/1/mini.webp", N: 2}) {
		t.Errorf("Pagine nell'Indice di ricerca: %+v", cerca.Pagine)
	}
	if c := cerca.Pagine[2]; c.Titolo != "Pomase" || c.Tipo != "Contributori" || c.Slug != "" {
		t.Errorf("Contributore nell'Indice di ricerca: %+v", c)
	}
	foto := map[int]fotoRicerca{}
	for _, f := range cerca.Foto {
		foto[f.Pid] = f
	}
	if f := foto[1]; f.Titolo != "Diga" || f.Meta != "Vajont Ente Mario" || len(f.Entita) != 1 || f.Entita[0] != 0 || f.Mini != "" {
		t.Errorf("Foto nell'Indice di ricerca: %+v", f)
	}
	if f := foto[2]; f.Titolo != "Vajont · foto 2" || f.Mini != "/g/statico/documento.svg" {
		t.Errorf("PDF senza titolo nell'Indice di ricerca: %+v", f)
	}
	if strings.Contains(foto[3].Tags, "da-separare") {
		t.Error("tag interno nell'Indice di ricerca")
	}
	if !strings.Contains(leggi("index.html"), `action="/g/cerca/"`) {
		t.Error("home senza il campo che porta a /cerca/")
	}
	if !strings.Contains(leggi("cerca/index.html"), `INDICE="/g/statico/cerca.json",FOTO="/g/f/"`) {
		t.Error("/cerca/ senza l'Indice di ricerca")
	}
	var idx map[string][]int
	if err := json.Unmarshal([]byte(leggi("indice-foto.json")), &idx); err != nil || len(idx["dighe/vajont"]) != 2 {
		t.Errorf("Indice delle Foto %v %v", idx, err)
	}
}
