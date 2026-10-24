package main

// Indice delle Entità (ADR 0002). Lo produrrà il Sito principale; finché non c'è,
// `gerbillo indice-entita` lo ricava dalla directory content di Hugo.

import (
	"bytes"
	"cmp"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type entita struct {
	Slug  string   `json:"slug"` // sezione/nome (ADR 0013)
	Title string   `json:"title"`
	Draft bool     `json:"draft,omitempty"`         // pagina in bozza: è comunque un'Entità ("ha o avrà una pagina")
	Nome  string   `json:"nome,omitempty" yaml:"-"` // diga.nome o centrale.nome, un nome in più per l'abbinamento
	Tags  []string `json:"tags,omitempty"`
	// Diga è la scheda dati del front matter del sito, intera: serve a esplorare la
	// Gallery (#28) e allo specchietto della mappa (#27), come nella mappa generale.
	Diga        map[string]any `json:"diga,omitempty"`
	Geolocation struct {
		MapIDs    []string `json:"map_ids,omitempty" yaml:"map_ids"`
		Stato     string   `json:"stato,omitempty"`
		Regione   string   `json:"regione,omitempty"`
		Provincia string   `json:"provincia,omitempty"`
		Coords    *struct {
			Latitudine  float64 `json:"latitudine"`
			Longitudine float64 `json:"longitudine"`
			Zoom        int     `json:"zoom,omitempty"`
		} `json:"coords,omitempty"`
	} `json:"geolocation"`
}

// Sezioni del sito a cui una Foto si può collegare (ADR 0013).
var sezioniEntita = []string{"dighe", "centrali", "reportage", "tecnica"}

func indiceEntita(args []string) error {
	fs := flag.NewFlagSet("indice-entita", flag.ExitOnError)
	sito := fs.String("sito", "", "directory content del Sito principale")
	fs.Parse(args)
	if *sito == "" {
		return fmt.Errorf("uso: gerbillo indice-entita -sito <content> > indice.json")
	}
	idx, err := leggiSito(*sito)
	if err != nil {
		return err
	}
	e := json.NewEncoder(os.Stdout)
	e.SetIndent("", "  ")
	return e.Encode(idx)
}

func leggiSito(dir string) ([]entita, error) {
	var idx []entita
	for _, sez := range sezioniEntita {
		pagine, err := filepath.Glob(filepath.Join(dir, sez, "*", "index.md"))
		if err != nil {
			return nil, err
		}
		for _, p := range pagine {
			b, err := os.ReadFile(p)
			if err != nil {
				return nil, err
			}
			var e entita
			if err := frontMatter(b, &e); err != nil {
				return nil, fmt.Errorf("%s: %w", p, err)
			}
			var blocchi struct {
				Diga, Centrale struct{ Nome string }
			}
			if err := frontMatter(b, &blocchi); err != nil {
				return nil, fmt.Errorf("%s: %w", p, err)
			}
			e.Nome = cmp.Or(blocchi.Diga.Nome, blocchi.Centrale.Nome)
			e.Slug = sez + "/" + filepath.Base(filepath.Dir(p))
			idx = append(idx, e)
		}
	}
	slices.SortFunc(idx, func(a, b entita) int { return strings.Compare(a.Slug, b.Slug) })
	return idx, nil
}

// frontMatter decodifica il YAML fra i due "---". Alcune pagine hanno chiavi ripetute
// (es. due "categories: ”"): passando da yaml.Node si tiene la prima.
func frontMatter(b []byte, v any) error {
	b = bytes.TrimPrefix(b, []byte("---\n"))
	fine := bytes.Index(b, []byte("\n---"))
	if fine < 0 {
		return fmt.Errorf("front matter non chiuso")
	}
	var n yaml.Node
	if err := yaml.Unmarshal(b[:fine], &n); err != nil {
		return err
	}
	if len(n.Content) == 1 && n.Content[0].Kind == yaml.MappingNode {
		m := n.Content[0]
		visti := map[string]bool{}
		var tieni []*yaml.Node
		for i := 0; i+1 < len(m.Content); i += 2 {
			if k := m.Content[i].Value; !visti[k] {
				visti[k] = true
				tieni = append(tieni, m.Content[i], m.Content[i+1])
			}
		}
		m.Content = tieni
	}
	return n.Decode(v)
}

func leggiIndice(p string) ([]entita, error) {
	b, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	var idx []entita
	return idx, json.Unmarshal(b, &idx)
}

// campo legge un valore testuale della scheda diga: «anno» può essere un numero o un
// intervallo («1900-06»), gli elenchi come «uso» danno il primo elemento.
func (e *entita) campo(k string) string {
	switch v := e.Diga[k].(type) {
	case string:
		return v
	case int:
		return strconv.Itoa(v)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case []any:
		if len(v) > 0 {
			if s, ok := v[0].(string); ok {
				return s
			}
		}
	}
	return ""
}
