package main

// Ricerca della Gallery (ADR 0014, #26): la pagina /cerca/ e l'Indice di ricerca,
// statico/cerca.json, che la pagina indicizza nel browser con FlexSearch. In cima le
// Pagine il cui nome corrisponde (Entità e Contributori), poi il filtro «Dove» con le
// Entità delle Foto trovate e la griglia delle Foto.

import (
	"encoding/json"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

// indiceRicerca ha chiavi corte: le Foto sono decine di migliaia.
type indiceRicerca struct {
	Pagine []paginaRicerca `json:"pagine"`
	Foto   []fotoRicerca   `json:"foto"`
}

type paginaRicerca struct {
	Slug   string `json:"s,omitempty"` // solo le Entità, per il filtro «Dove»
	Titolo string `json:"t"`
	Tipo   string `json:"k"` // Dighe, Centrali, Opere, Eventi, Altre pagine, Contributori
	Luogo  string `json:"l,omitempty"`
	URL    string `json:"u"`
	Mini   string `json:"m"`
	N      int    `json:"n"`
}

type fotoRicerca struct {
	Pid     int    `json:"p"`
	Titolo  string `json:"t"`
	Meta    string `json:"a,omitempty"` // Entità, Autori, Contributore
	Tags    string `json:"g,omitempty"` // Tag e Categorie
	Content string `json:"c,omitempty"` // descrizione, luogo, Datazione
	Entita  []int  `json:"e,omitempty"` // posizioni in Pagine
	Mini    string `json:"m,omitempty"` // solo se non è f/<pid>/mini.webp (anteprima, PDF)
}

// tipoEntita è il nome della sezione dello Slug, per la ricerca e il Cosa della Bussola.
// Le sezioni che arrivano solo dal sito stanno in «Altre pagine».
func tipoEntita(slug string) string {
	switch strings.SplitN(slug, "/", 2)[0] {
	case "dighe":
		return "Dighe"
	case "centrali":
		return "Centrali"
	case "opere":
		return "Opere"
	case "eventi":
		return "Eventi"
	}
	return "Altre pagine"
}

type corpoCerca struct {
	Indice, FlexSearch, Foto string // URL dell'Indice di ricerca, della libreria, prefisso delle pagine Foto
	Dighe                    string // ripiego senza JavaScript
}

// cerca scrive l'Indice di ricerca e la pagina /cerca/.
func (b *builder) cerca(perEntita map[string][]*fotoPubblicata, slugs []string, perContributore map[int][]*fotoPubblicata, ids []int, nomi map[int]string) error {
	var x indiceRicerca
	pos := map[string]int{}
	for _, s := range slugs {
		e, fs := b.c.entita[s], perEntita[s]
		pos[s] = len(x.Pagine)
		x.Pagine = append(x.Pagine, paginaRicerca{Slug: s, Titolo: e.Title, Tipo: tipoEntita(s), Luogo: luogoEntita(e),
			URL: b.url(s), Mini: b.mini(fs[0]), N: len(fs)})
	}
	for _, id := range ids {
		fs := perContributore[id]
		x.Pagine = append(x.Pagine, paginaRicerca{Titolo: nomi[id], Tipo: "Contributori",
			URL: b.url("contributori", strconv.Itoa(id)), Mini: b.mini(fs[0]), N: len(fs)})
	}
	for _, f := range b.c.foto {
		x.Foto = append(x.Foto, b.fotoRicerca(f, pos))
	}
	d, err := json.Marshal(x)
	if err != nil {
		return err
	}
	if err := b.scriviFile(filepath.Join("statico", "cerca.json"), d); err != nil {
		return err
	}
	corpo := corpoCerca{Indice: b.statico("cerca.json"), FlexSearch: b.statico("flexsearch.js"), Foto: b.url("f"), Dighe: b.url("dighe")}
	return b.scrivi("cerca", paginaSito{Titolo: "Cerca", Corpo: corpo, Briciole: []voce{{URL: b.url(), Testo: "Gallery"}}}, "cerca")
}

func (b *builder) fotoRicerca(f *fotoPubblicata, pos map[string]int) fotoRicerca {
	var entita []string
	var e []int
	for _, s := range f.Slug {
		entita = append(entita, b.c.entita[s].Title)
		e = append(e, pos[s])
	}
	meta := slices.Concat(entita, f.Authors)
	if f.Contributore != nil {
		meta = append(meta, f.Contributore.Nome)
	}
	g := f.Geolocation
	mini := b.mini(f)
	if mini == b.url("f", strconv.Itoa(f.Pid))+"mini.webp" {
		mini = ""
	}
	return fotoRicerca{
		Pid:     f.Pid,
		Titolo:  b.titolo(f),
		Meta:    strings.Join(meta, " "),
		Tags:    strings.Join(slices.Concat(f.tagPubblici(), f.Categories), " "),
		Content: strings.Join(strings.Fields(strings.Join([]string{f.Description, g.Provincia, g.Regione, g.Stato, f.datazioneTesto()}, " ")), " "),
		Entita:  e,
		Mini:    mini,
	}
}
