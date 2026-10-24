package main

// Indice di ricerca (ADR 0014, #26): statico/cerca.json nel formato di /index.json del Sito
// principale, letto da search.js copiato dal sito. Contiene le pagine delle Entità,
// i Contributori e ogni Foto; i campi pesano come nel sito (title 5, meta 4, tags 3).

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

type voceRicerca struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Meta    string `json:"meta"`
	Tags    string `json:"tags"`
	Content string `json:"content"`
	Section string `json:"section"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
}

// Le pillole della ricerca, nell'ordine in cui compaiono.
var sezioniRicerca = []string{"Dighe", "Centrali", "Altre pagine", "Contributori", "Foto"}

// sezioneEntita mette ogni sezione del sito diversa da dighe e centrali in «Altre pagine».
func sezioneEntita(slug string) string {
	switch strings.SplitN(slug, "/", 2)[0] {
	case "dighe":
		return "Dighe"
	case "centrali":
		return "Centrali"
	}
	return "Altre pagine"
}

type pillola struct {
	Sezione, Testo string // data-section (in minuscolo, come lo confronta search.js) ed etichetta
	N              int
}

// cerca scrive cerca.json e restituisce le pillole, «Tutti» per prima, con il numero di voci.
func (b *builder) cerca(perEntita map[string][]*fotoPubblicata, slugs []string, perContributore map[int][]*fotoPubblicata, ids []int, nomi map[int]string) ([]pillola, error) {
	var v []voceRicerca
	for _, s := range slugs {
		e, luogo := b.c.entita[s], luogoEntita(b.c.entita[s])
		v = append(v, voceRicerca{ID: "e:" + s, Title: e.Title, Meta: luogo, Tags: strings.Join(e.Tags, " "),
			Section: sezioneEntita(s), URL: b.url(s), Snippet: conteggio(len(perEntita[s]), luogo)})
	}
	for _, id := range ids {
		v = append(v, voceRicerca{ID: "c:" + strconv.Itoa(id), Title: nomi[id], Section: "Contributori",
			URL: b.url("contributori", strconv.Itoa(id)), Snippet: conteggio(len(perContributore[id]), "")})
	}
	for _, f := range b.c.foto {
		v = append(v, b.vociFoto(f))
	}
	n := map[string]int{}
	for _, x := range v {
		n[x.Section]++
	}
	p := []pillola{{"all", "Tutti", len(v)}}
	for _, s := range sezioniRicerca {
		if n[s] > 0 {
			p = append(p, pillola{strings.ToLower(s), s, n[s]})
		}
	}
	d, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(b.dir, "statico")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return p, os.WriteFile(filepath.Join(dir, "cerca.json"), d, 0o644)
}

func (b *builder) vociFoto(f *fotoPubblicata) voceRicerca {
	var entita []string
	for _, s := range f.Slug {
		entita = append(entita, b.c.entita[s].Title)
	}
	meta := slices.Concat(entita, f.Authors)
	if f.Contributore != nil {
		meta = append(meta, f.Contributore.Nome)
	}
	g := f.Geolocation
	// Snippet: Entità · Datazione · Autore — descrizione, per riconoscere la Foto senza miniatura.
	var testa []string
	if len(entita) > 0 {
		testa = append(testa, entita[0])
	}
	if f.Datazione != "" {
		testa = append(testa, f.Datazione)
	}
	if len(f.Authors) > 0 {
		testa = append(testa, f.Authors[0])
	}
	snippet := strings.Join(testa, " · ")
	if d := tronca(f.Description, 120); d != "" {
		if snippet != "" {
			snippet += " — "
		}
		snippet += d
	}
	return voceRicerca{
		ID:      "f:" + strconv.Itoa(f.Pid),
		Title:   b.titolo(f),
		Meta:    strings.Join(meta, " "),
		Tags:    strings.Join(slices.Concat(f.tagPubblici(), f.Categories), " "),
		Content: strings.Join([]string{f.Description, g.Provincia, g.Regione, g.Stato, f.Datazione}, " "),
		Section: "Foto",
		URL:     b.url("f", strconv.Itoa(f.Pid)),
		Snippet: snippet,
	}
}

// tronca taglia s a n caratteri, sull'ultimo spazio, con i puntini.
func tronca(s string, n int) string {
	r := []rune(strings.Join(strings.Fields(s), " "))
	if len(r) <= n {
		return string(r)
	}
	t := string(r[:n])
	if i := strings.LastIndex(t, " "); i > n/2 {
		t = t[:i]
	}
	return t + "…"
}
