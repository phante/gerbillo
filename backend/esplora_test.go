package main

import (
	"math"
	"testing"
)

func TestFamigliaDecennio(t *testing.T) {
	for in, out := range map[string]string{
		"Gravità ordinaria in calcestruzzo":          "Gravità",
		"Arco gravità":                               "Arco e cupola",
		"Cupola":                                     "Arco e cupola",
		"Gravità a speroni a vani interni":           "Contrafforti e speroni",
		"Terra e/o pietrame con nucleo verticale":    "Terra e pietrame",
		"Traversa in muratura di pietrame con malta": "Traverse",
		"Muratura in pietrame a secco":               "Gravità",
		"":                                           "",
	} {
		if got := famiglia(in); got != out {
			t.Errorf("famiglia(%q) = %q, atteso %q", in, got, out)
		}
	}
	for in, out := range map[string]string{"1957": "Anni 1950", "1900-06": "Prima del 1910", "19": "", "abcd": ""} {
		if got := decennio(in); got != out {
			t.Errorf("decennio(%q) = %q, atteso %q", in, got, out)
		}
	}
	if d := km(46.2922, 8.3016, 46.2922, 8.3016); d != 0 {
		t.Errorf("km stesso punto %f", d)
	}
	if d := km(45.0, 7.0, 46.0, 7.0); math.Abs(d-111.2) > 0.5 {
		t.Errorf("un grado di latitudine = %f km", d)
	}
}

func TestIntorno(t *testing.T) {
	e := func(slug, tip, anno string, lat, lon float64, mappe ...string) entita {
		var x entita
		x.Slug, x.Title = slug, slug
		x.Geolocation.MapIDs = mappe
		x.Geolocation.Coords = &struct {
			Latitudine  float64 `json:"latitudine"`
			Longitudine float64 `json:"longitudine"`
		}{lat, lon}
		x.Diga = &struct {
			Tipologia      string   `json:"tipologia,omitempty"`
			Anno           string   `json:"anno,omitempty"`
			Uso            []string `json:"uso,omitempty"`
			Concessionario []string `json:"concessionario,omitempty"`
			Corsoacqua     string   `json:"corsoacqua,omitempty"`
		}{Tipologia: tip, Anno: anno}
		return x
	}
	idx := []entita{
		e("dighe/agaro", "Gravità ordinaria", "1940", 46.29, 8.30, "devero"),
		e("dighe/codelago", "Gravità ordinaria", "1912", 46.29, 8.20, "devero"),
		e("dighe/vicina", "Arco semplice", "1960", 46.20, 8.30),
		e("dighe/lontana", "Gravità ordinaria", "1945", 40.0, 15.0),
	}
	c := &contenuti{entita: map[string]*entita{}}
	per := map[string][]*fotoPubblicata{}
	var slugs []string
	for i := range idx {
		c.entita[idx[i].Slug] = &idx[i]
		per[idx[i].Slug] = []*fotoPubblicata{{Pid: i + 1, Formato: "immagine"}}
		slugs = append(slugs, idx[i].Slug)
	}
	b := &builder{opzioniBuild: opzioniBuild{base: "/g/"}, c: c}
	g := b.intorno("dighe/agaro", per, slugs)
	nomi := map[string]string{}
	for _, x := range g {
		for _, v := range x.Voci {
			nomi[v.Testo] = x.Nome
		}
	}
	if nomi["dighe/codelago"] != "Stesso impianto" || nomi["dighe/vicina"] != "Vicine" {
		t.Errorf("intorno: %v", nomi)
	}
	if nomi["dighe/lontana"] != "Simili: gravità, anni 1940" {
		t.Errorf("simili: %v", nomi)
	}
	if _, ok := nomi["dighe/agaro"]; ok {
		t.Error("la diga stessa fra quelle intorno")
	}
}
