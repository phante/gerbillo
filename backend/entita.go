package main

// Le Entità vengono da due fonti unite (ADR 0019): i file entita/<sezione>/<nome>.yaml della
// Gallery e l'Indice delle Entità (ADR 0002). L'Indice lo produrrà il Sito principale; finché
// non c'è, `gerbillo indice-entita` lo ricava dalla directory content di Hugo.

import (
	"bytes"
	"cmp"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type entita struct {
	Slug  string   `json:"slug"` // sezione/nome (ADR 0013)
	Title string   `json:"title"`
	Draft bool     `json:"draft,omitempty"`         // nessuna pagina pubblica sul sito a cui rimandare: in bozza, o senza pagina (ADR 0018)
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

// Sezioni del sito lette da `gerbillo indice-entita`.
var sezioniEntita = []string{"dighe", "centrali", "reportage", "tecnica"}

var (
	reSlug = regexp.MustCompile(`^[a-z0-9_-]+/[a-z0-9_-]+$`)
	// Prime parti degli URL della Gallery pubblica che non sono Entità.
	percorsiRiservati = []string{"altre", "cerca", "contributori", "f", "statico"}
)

func slugValido(s string) bool {
	return reSlug.MatchString(s) && !slices.Contains(percorsiRiservati, s[:strings.Index(s, "/")])
}

// entitaUnite sono le Entità delle due fonti. rimandi va da un Alias allo Slug attuale.
type entitaUnite struct {
	idx     []entita
	rimandi map[string]string
	avvisi  []string
}

// leggiEntita unisce l'Indice delle Entità (facoltativo) e i file entita/<sezione>/<nome>.yaml
// del Repository dei contenuti e delle cartelle del Livello di arricchimento (nell'anteprima
// dati/arricchimento/entita/): fra i file vince il più specifico, fra file e Indice il sito.
func leggiEntita(dir, indice string, l livelloArricchimento) (*entitaUnite, error) {
	u := &entitaUnite{rimandi: map[string]string{}}
	if indice != "" {
		var err error
		if u.idx, err = leggiIndice(indice); err != nil {
			return nil, err
		}
		// Lo Slug diventa un percorso in dist/: la voce non valida non ci arriva (#83).
		u.idx = slices.DeleteFunc(u.idx, func(e entita) bool {
			if slugValido(e.Slug) {
				return false
			}
			u.avvisi = append(u.avvisi, fmt.Sprintf("Indice delle Entità: %q non si può usare come Slug, voce scartata", e.Slug))
			return true
		})
	}
	noti := map[string]bool{}
	for _, e := range u.idx {
		noti[e.Slug] = true
	}
	type fileEntita struct {
		entita  `yaml:",inline"`
		Aliases []string
	}
	file := map[string]*fileEntita{}
	cartelle := []string{dir}
	for _, d := range slices.Backward(l) {
		cartelle = append(cartelle, d)
	}
	for _, d := range cartelle {
		trovati, err := filepath.Glob(filepath.Join(d, "entita", "*", "*.yaml"))
		if err != nil {
			return nil, err
		}
		for _, p := range trovati {
			slug := filepath.Base(filepath.Dir(p)) + "/" + strings.TrimSuffix(filepath.Base(p), ".yaml")
			if !slugValido(slug) {
				return nil, fmt.Errorf("%s: %q non si può usare come Slug", p, slug)
			}
			b, err := os.ReadFile(p)
			if err != nil {
				return nil, err
			}
			f := &fileEntita{}
			if err := yaml.Unmarshal(b, f); err != nil {
				return nil, fmt.Errorf("%s: %w", p, err)
			}
			if f.Title == "" {
				return nil, fmt.Errorf("%s: manca title", p)
			}
			file[slug] = f
		}
	}
	slugFile := chiaviOrdinate(file)
	for _, s := range slugFile {
		f := file[s]
		if noti[s] {
			if len(f.Aliases) == 0 {
				u.avvisi = append(u.avvisi, "entita/"+s+".yaml: il sito ha la stessa pagina, il file si può togliere")
			}
			continue
		}
		f.Slug, f.Draft = s, true
		u.idx = append(u.idx, f.entita)
		noti[s] = true
	}
	slices.SortFunc(u.idx, func(a, b entita) int { return strings.Compare(a.Slug, b.Slug) })
	for _, s := range slugFile {
		for _, a := range file[s].Aliases {
			_, doppio := u.rimandi[a]
			switch {
			case !slugValido(a):
				u.avvisi = append(u.avvisi, fmt.Sprintf("alias %q di %s: non si può usare come Slug", a, s))
			case noti[a]:
				u.avvisi = append(u.avvisi, fmt.Sprintf("alias %s di %s: esiste una pagina con quello Slug, vince la pagina", a, s))
			case doppio:
				u.avvisi = append(u.avvisi, fmt.Sprintf("alias %s di %s: è già di %s", a, s, u.rimandi[a]))
			default:
				u.rimandi[a] = s
			}
		}
	}
	return u, nil
}

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
