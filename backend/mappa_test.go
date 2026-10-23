package main

import (
	"slices"
	"testing"
)

func TestNormNome(t *testing.T) {
	for in, out := range map[string]string{
		"Ravedis (PN)":            "ravedis",
		"Diga di Pont'Ventoux":    "pontventoux",
		"Cà Zul":                  "ca-zul",
		"Lago Brugneto (GE)":      "brugneto",
		"S. Caterina di Auronzo":  "s-caterina-di-auronzo",
		"Punt dal Gall (Livigno)": "punt-dal-gall",
		"  Crosis  (UD) ":         "crosis",
	} {
		if got := normNome(in); got != out {
			t.Errorf("normNome(%q) = %q, atteso %q", in, got, out)
		}
	}
}

func indiceProva() []entita {
	e := func(slug, title, stato, regione string) entita {
		var x entita
		x.Slug, x.Title = slug, title
		x.Geolocation.Stato, x.Geolocation.Regione = stato, regione
		return x
	}
	return []entita{
		e("dighe/vajont", "Vajont", "Italia", "Friuli-Venezia Giulia"),
		e("dighe/ponte-antoi", "Ponte Antoi", "Italia", "Friuli-Venezia Giulia"),
		e("dighe/lumiei-o-di-la-maina-di-sauris", "Lumiei o di La Maina di Sauris", "Italia", "Friuli-Venezia Giulia"),
		e("dighe/santa-caterina", "Santa Caterina", "Italia", "Veneto"),
		e("dighe/castel", "Castel", "Italia", "Piemonte"),
		e("dighe/castel-1", "Castel", "Italia", "Lombardia"),
	}
}

func TestTrova(t *testing.T) {
	k := chiaviEntita(indiceProva())
	friuli := []string{"Dighe - Dams", "Italia", "Friuli Venezia-Giulia"}
	casi := []struct {
		nome     string
		percorso []string
		slug     []string
		certo    bool
	}{
		{"Vajont", friuli, []string{"dighe/vajont"}, true},
		{"Barcis - Ponte Antoi (PN)", friuli, []string{"dighe/ponte-antoi"}, true},
		{"La Maina di Sauris (UD)", friuli, []string{"dighe/lumiei-o-di-la-maina-di-sauris"}, true},
		{"S. Caterina di Auronzo (BL)", []string{"Dighe - Dams", "Italia", "Veneto"}, []string{"dighe/santa-caterina"}, false},
		{"Castel", []string{"Dighe - Dams", "Italia", "Piemonte"}, []string{"dighe/castel"}, true},
		{"Castel", []string{"Dighe - Dams", "Francia"}, []string{"dighe/castel", "dighe/castel-1"}, true},
		{"La Forra", friuli, []string{}, false},
	}
	for _, c := range casi {
		s, certo := trova(c.nome, c.percorso, k)
		if !slices.Equal(s, c.slug) || certo != c.certo {
			t.Errorf("trova(%q) = %v, %v; atteso %v, %v", c.nome, s, certo, c.slug, c.certo)
		}
	}
}

func TestAbbina(t *testing.T) {
	cpg := albumCoppermine{
		Categorie: map[int]categoriaYAML{
			2:  {"Dighe - Dams", 0},
			27: {"Italia", 2},
			6:  {"Friuli Venezia-Giulia", 27},
			10: {"Vajont", 6},
			57: {"Percorsi della Memoria", 10},
			3:  {"Centrali - Powerplants", 0},
		},
		Album: map[int]albumYAML{
			14:  {57, 288, "I percorsi della memoria 2008"}, // eredita da Vajont, due livelli sopra
			20:  {6, 10, "Ponte Antoi"},
			21:  {6, 5, "Sconosciuta"},
			22:  {6, 7, "Ravedis"},
			500: {3, 99, "Ponte Antoi"}, // fuori da Dighe
		},
	}
	vecchie := fileMappature{Album: map[int]*mappatura{
		22: {Origine: manuale, Slug: []string{"dighe/ravedis"}, Tags: []string{"cellina"}},
		21: {Origine: daFare},
	}}
	m, rap := abbina(cpg, indiceProva(), vecchie)
	if c := m.Categorie[10]; c == nil || !slices.Equal(c.Slug, []string{"dighe/vajont"}) {
		t.Errorf("categoria Vajont: %+v", c)
	}
	if m.Album[14] != nil {
		t.Errorf("album 14 eredita, non deve avere una riga: %+v", m.Album[14])
	}
	if a := m.Album[20]; a == nil || a.Origine != automatica {
		t.Errorf("album 20: %+v", a)
	}
	if a := m.Album[21]; a == nil || a.Origine != daFare {
		t.Errorf("album 21: %+v", a)
	}
	if a := m.Album[22]; a == nil || a.Origine != manuale || a.Tags[0] != "cellina" || a.Coppermine == "" {
		t.Errorf("album 22 manuale non conservato: %+v", a)
	}
	if m.Album[500] != nil {
		t.Error("album fuori da Dighe mappato")
	}
	if rap.album != 4 || rap.autoAlbum != 2 || rap.eredAlbum != 1 || rap.manAlbum != 1 || len(rap.daFare) != 1 {
		t.Errorf("rapporto %+v", rap)
	}
	if !slices.Equal(rap.sconosciuti, []string{"album 22: dighe/ravedis"}) {
		t.Errorf("sconosciuti %v", rap.sconosciuti)
	}
}

func TestFrontMatterChiaviRipetute(t *testing.T) {
	var e entita
	b := []byte("---\ntitle: Alpe Gera\ncategories: ''\ncategories: ''\ntags:\n  - valmalenco\ndraft: true\n---\ntesto\n")
	if err := frontMatter(b, &e); err != nil {
		t.Fatal(err)
	}
	if e.Title != "Alpe Gera" || !e.Draft || e.Tags[0] != "valmalenco" {
		t.Errorf("%+v", e)
	}
}
