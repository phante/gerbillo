package main

// Entità da censire (ADR 0018, #61): un'Entità senza pagina sul Sito principale, in
// entita/<id>.yaml del Repository dei contenuti. Le Foto la indicano con da_censire. Quando
// nel sito nasce la pagina resta come «diventata» quello Slug, e la sua pagina rimanda lì.

import (
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type entitaDaCensire struct {
	ID          int
	Nome        string
	Tipo        string
	Geolocation geolocFoto
	Diventata   string // lo Slug, dopo il collegamento
}

const nonSulSito = "non ancora sul sito"

// tipiDaCensire: niente reportage, che non è un luogo da censire (ADR 0018).
var tipiDaCensire = map[string]struct{ nome, ricerca string }{
	"diga":     {"Diga", "Dighe"},
	"centrale": {"Centrale", "Centrali"},
	"altra":    {"Altra opera", "Altre pagine"},
}

// leggiDaCensire legge entita/ del Repository dei contenuti e poi quella delle cartelle del
// Livello di arricchimento (nell'anteprima dati/arricchimento/entita/): vince la più specifica.
func leggiDaCensire(dir string, l livelloArricchimento) (map[int]*entitaDaCensire, error) {
	cartelle := []string{dir}
	for _, d := range slices.Backward(l) {
		cartelle = append(cartelle, d)
	}
	out := map[int]*entitaDaCensire{}
	var errori []string
	for _, d := range cartelle {
		file, err := filepath.Glob(filepath.Join(d, "entita", "*.yaml"))
		if err != nil {
			return nil, err
		}
		for _, p := range file {
			b, err := os.ReadFile(p)
			if err != nil {
				return nil, err
			}
			e := &entitaDaCensire{}
			if err := yaml.Unmarshal(b, e); err != nil {
				return nil, fmt.Errorf("%s: %w", p, err)
			}
			_, tipoNoto := tipiDaCensire[e.Tipo]
			switch {
			case strconv.Itoa(e.ID)+".yaml" != filepath.Base(p):
				errori = append(errori, fmt.Sprintf("%s: id %d diverso dal nome del file", p, e.ID))
			case e.Nome == "":
				errori = append(errori, p+": manca il nome")
			case !tipoNoto:
				errori = append(errori, fmt.Sprintf("%s: tipo %q, ammessi diga, centrale, altra", p, e.Tipo))
			}
			out[e.ID] = e
		}
	}
	if len(errori) > 0 {
		return nil, fmt.Errorf("Entità da censire:\n%s", strings.Join(errori, "\n"))
	}
	return out, nil
}

// comeEntita serve ai calcoli sul luogo, gli stessi delle Entità del sito.
func (e *entitaDaCensire) comeEntita() *entita {
	x := &entita{Title: e.Nome}
	x.Geolocation.Stato, x.Geolocation.Regione, x.Geolocation.Provincia = e.Geolocation.Stato, e.Geolocation.Regione, e.Geolocation.Provincia
	return x
}

func parteDaCensire(id int) []string { return []string{"da-censire", strconv.Itoa(id)} }

// chiaveDaCensire è il suo posto nell'Indice di ricerca, per il filtro «Dove».
func chiaveDaCensire(id int) string { return strings.Join(parteDaCensire(id), "/") }

// conFoto sono gli id delle Entità da censire con almeno una Foto, nell'ordine degli id:
// solo queste stanno nell'albero, nella ricerca e nella Bussola, come le Entità del sito.
func (b *builder) conFoto() []int {
	return chiaviOrdinate(b.perCensire)
}

// pagineDaCensire scrive /da-censire/<id>/: la griglia, oppure il rimando dopo il collegamento.
func (b *builder) pagineDaCensire(casa voce) error {
	for _, id := range chiaviOrdinate(b.c.daCensire) {
		e := b.c.daCensire[id]
		if e.Diventata != "" {
			u := b.url(e.Diventata)
			corpo := corpoRimando{URL: u, Testo: b.c.entita[e.Diventata].Title,
				Refresh: template.HTMLAttr(`content="0; url=` + template.HTMLEscapeString(u) + `"`)}
			if err := b.scrivi("rimando", paginaSito{Titolo: e.Nome, Corpo: corpo}, parteDaCensire(id)...); err != nil {
				return err
			}
			continue
		}
		fs := b.perCensire[id]
		b.registraContesto(fs, voce{URL: b.url(parteDaCensire(id)...), Testo: e.Nome})
		sotto := []string{tipiDaCensire[e.Tipo].nome}
		if l := luogoEntita(e.comeEntita()); l != "" {
			sotto = append(sotto, l)
		}
		sotto = append(sotto, nonSulSito)
		if err := b.griglie(fs, e.Nome, strings.Join(sotto, " · "), nil, []voce{casa}, nil, nil, parteDaCensire(id)...); err != nil {
			return err
		}
	}
	return nil
}

type corpoRimando struct {
	Refresh    template.HTMLAttr
	URL, Testo string
}

// voceIndiceDaCensire è una riga di da-censire.json, per il Sito principale (ADR 0018).
type voceIndiceDaCensire struct {
	ID          int        `json:"id"`
	Nome        string     `json:"nome"`
	Tipo        string     `json:"tipo"`
	Geolocation geolocFoto `json:"geolocation"`
	N           int        `json:"n"`
}

// scriviIndiceDaCensire elenca le Entità ancora da censire: quelle diventate uno Slug al sito
// non servono più.
func (b *builder) scriviIndiceDaCensire() error {
	v := []voceIndiceDaCensire{}
	for _, id := range chiaviOrdinate(b.c.daCensire) {
		if e := b.c.daCensire[id]; e.Diventata == "" {
			v = append(v, voceIndiceDaCensire{id, e.Nome, e.Tipo, e.Geolocation, len(b.perCensire[id])})
		}
	}
	d, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return b.scriviFile("da-censire.json", d)
}
