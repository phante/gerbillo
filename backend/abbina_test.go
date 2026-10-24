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
		"Delle Piazze (Pinè)":     "piazze",
		"Lago Delle Piazze":       "piazze",
		"La Stua":                 "stua",
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
		e("centrali/achille-gaggia-soverzene-bl", "Achille Gaggia - Soverzene (BL)", "Italia", "Veneto"),
		e("centrali/ligonchio-ligonchio-re", "Ligonchio - Ligonchio (RE)", "Italia", "Emilia-Romagna"),
		e("reportage/il-guardiano-della-diga", "Il guardiano della diga", "", ""),
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
		{"Achille Gaggia", nil, []string{"centrali/achille-gaggia-soverzene-bl"}, true},
		{"Centrale Ligonchio", nil, []string{"centrali/ligonchio-ligonchio-re"}, true},
		{"La Forra", friuli, []string{}, false},
	}
	for _, c := range casi {
		s, certo := trova(c.nome, c.percorso, k)
		if !slices.Equal(s, c.slug) || certo != c.certo {
			t.Errorf("trova(%q) = %v, %v; atteso %v, %v", c.nome, s, certo, c.slug, c.certo)
		}
	}
}

func TestMappature(t *testing.T) {
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
	manuali := fileMappature{Album: map[int]*mappatura{
		22: {Slug: []string{"dighe/ravedis"}, Tags: []string{"cellina"}},
	}}
	m, rap := mappature(cpg, indiceProva(), manuali)
	if a := m[14]; a == nil || !slices.Equal(a.Slug, []string{"dighe/vajont"}) || !a.auto {
		t.Errorf("album 14 eredita dalla categoria Vajont, due livelli sopra: %+v", a)
	}
	if a := m[20]; a == nil || !a.auto || !slices.Equal(a.Slug, []string{"dighe/ponte-antoi"}) {
		t.Errorf("album 20: %+v", a)
	}
	if a := m[21]; a == nil || a.auto || !slices.Equal(a.Tags, []string{daCollegare}) {
		t.Errorf("album 21: %+v", a)
	}
	if a := m[22]; a == nil || a.auto || a.Tags[0] != "cellina" {
		t.Errorf("album 22 manuale non applicato: %+v", a)
	}
	// Sotto Centrali si cercano solo le centrali: la diga Ponte Antoi non conta.
	if a := m[500]; a == nil || len(a.Slug) > 0 || !slices.Equal(a.Tags, []string{daCollegare}) {
		t.Errorf("album 500 sotto Centrali: %+v", a)
	}
	d := rap.radici[cidDighe]
	if d.album != 4 || d.autoAlbum != 2 || d.eredAlbum != 1 || d.manAlbum != 1 || d.faAlbum != 1 || len(rap.daFare) != 2 {
		t.Errorf("rapporto %+v %+v", d, rap)
	}
	if !slices.Equal(rap.sconosciuti, []string{"album 22: dighe/ravedis"}) {
		t.Errorf("sconosciuti %v", rap.sconosciuti)
	}
	if rap.interni[daCollegare] != 104 {
		t.Errorf("tag interni %v", rap.interni)
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

// Le categorie della #1: categories e tags si ereditano dalla categoria, anche nelle righe automatiche.
func TestMappatureCategorieNonDighe(t *testing.T) {
	cpg := albumCoppermine{
		Categorie: map[int]categoriaYAML{
			1:  {"User galleries", 0},
			3:  {"Centrali - Powerplants", 0},
			69: {"Italia", 3},
			55: {"Emilia Romagna", 69},
			11: {"Opere di Presa", 0},
			70: {"Italia", 11},
			34: {"Veneto", 70},
			16: {"Uploads Temporanei", 0},
			47: {"Off-Topics", 0},
		},
		Album: map[int]albumYAML{
			1: {55, 20, "Centrale Caprile - Ligonchio (RE)"}, // il comune è di un'altra centrale
			2: {55, 59, "Centrale Ligonchio - Ligonchio (RE)"},
			3: {34, 16, "Diga di Vajont"},            // automatica: eredita categories dalla categoria
			4: {34, 30, "Sbarramento sul Brenta"},    // eredita, nessuna riga
			5: {16, 153, "Uploads temporanei"},       // eredita il tag interno
			6: {47, 30, "Brenner Basis Tunnel"},      // eredita off-topic
			7: {10025, 4, "Il guardiano della diga"}, // album personale: reportage mai certo
			8: {10025, 3, "Vajont"},                  // album personale abbinato
			9: {47, 5, "Locomotive"},                 // riga manuale: vince per intero
		},
	}
	manuali := fileMappature{
		Categorie: map[int]*mappatura{
			34: {Categories: []string{"opere-idrauliche"}},
			16: {Tags: []string{"da-classificare"}},
			47: {Categories: []string{"off-topic"}},
		},
		Album: map[int]*mappatura{9: nil}, // "9: # Altro / Locomotive" nel YAML
	}
	m, rap := mappature(cpg, indiceProva(), manuali)
	if a := m[1]; a == nil || len(a.Slug) > 0 || !slices.Equal(a.Tags, []string{daCollegare}) {
		t.Errorf("album 1: %+v", a)
	}
	if a := m[2]; a == nil || !slices.Equal(a.Slug, []string{"centrali/ligonchio-ligonchio-re"}) {
		t.Errorf("album 2: %+v", a)
	}
	if a := m[3]; a == nil || !a.auto || !slices.Equal(a.Slug, []string{"dighe/vajont"}) ||
		!slices.Equal(a.Categories, []string{"opere-idrauliche"}) {
		t.Errorf("album 3: %+v", a)
	}
	if a := m[4]; a == nil || len(a.Slug) > 0 || !slices.Equal(a.Categories, []string{"opere-idrauliche"}) {
		t.Errorf("album 4 eredita dalla categoria: %+v", a)
	}
	if a := m[5]; a == nil || !slices.Equal(a.Tags, []string{"da-classificare"}) {
		t.Errorf("album 5 eredita il tag interno: %+v", a)
	}
	if a := m[6]; a == nil || !slices.Equal(a.Categories, []string{"off-topic"}) {
		t.Errorf("album 6 eredita off-topic: %+v", a)
	}
	if a := m[7]; a == nil || len(a.Slug) > 0 || !slices.Equal(a.Tags, []string{daCollegare}) {
		t.Errorf("album 7: %+v", a)
	}
	if v := rap.daFare[0]; v.aid != 1 || rap.daFare[1].aid != 7 ||
		!slices.Equal(rap.daFare[1].candidati, []string{"reportage/il-guardiano-della-diga"}) ||
		rap.daFare[1].percorso != "User galleries / Il guardiano della diga" {
		t.Errorf("da fare: %+v", rap.daFare)
	}
	if a := m[8]; a == nil || !a.auto {
		t.Errorf("album 8: %+v", a)
	}
	// Una riga senza slug vuol dire "nessuna Entità": vince per intero, anche sulla categoria.
	if a := m[9]; a == nil || !a.vuota() {
		t.Errorf("album 9: %+v", a)
	}
	if rap.interni["da-classificare"] != 153 || rap.interni[daCollegare] != 24 {
		t.Errorf("tag interni %v", rap.interni)
	}
}

func TestFuoriVocabolario(t *testing.T) {
	m := fileMappature{
		Categorie: map[int]*mappatura{47: {Categories: []string{"off-topic"}}},
		Album:     map[int]*mappatura{80: {Categories: []string{"tecnica/turbina/francis"}}},
	}
	fuori := fuoriVocabolario(m, vocabolario{Categories: []string{"off-topic", "tecnica/turbine/francis"}})
	if !slices.Equal(fuori, []string{"album 80: tecnica/turbina/francis"}) {
		t.Errorf("%v", fuori)
	}
}
