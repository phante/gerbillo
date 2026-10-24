package main

// Foto pubblicata (ADR 0009, 0013): l'Importazione con gli alias degli Autori, poi la
// Mappatura dell'album (abbina.go), poi arricchimento/foto/<pid>.yaml.
// Per ogni campo vince il livello più specifico.

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

type fotoPubblicata struct {
	Pid          int
	Title        string
	Description  string
	Alt          string
	Date         string
	Authors      []string
	Datazione    string
	Licenza      string // identificativo SPDX; vuota = quella del sito (ADR 0010)
	Riservata    bool   // album non pubblico di Coppermine (#9): fuori dalla Gallery pubblica
	Slug         []string
	Tags         []string
	Categories   []string
	Geolocation  geolocFoto
	Contributore *struct {
		ID   int
		Nome string
	}
	Formato    string
	Larghezza  int
	Altezza    int
	Coppermine struct {
		Aid      int
		Filepath string
		Filename string
		Keywords string
		Luogo    string
		Data     string
		DataExif string `yaml:"data_exif"`
	}
}

type geolocFoto struct {
	Stato     string
	Regione   string
	Provincia string
	Coords    *struct{ Latitudine, Longitudine float64 }
}

// tagInterni sono promemoria di lavoro, mai mostrati nella Gallery pubblica.
// Gli stessi di tags_interni in dati/vocabolario.yaml.
var tagInterni = map[string]bool{"da-classificare": true, "da-collegare": true, "da-separare": true}

type contenuti struct {
	foto   []*fotoPubblicata
	album  albumCoppermine
	entita map[string]*entita
	ignoti []string // "pid: slug" non presenti nell'Indice delle Entità
}

// livelloArricchimento sono le cartelle del Livello di arricchimento, dalla più specifica:
// prima quella di -arricchimento (nell'anteprima dati/arricchimento/, #38, #39), poi
// arricchimento/ del Repository dei contenuti. Si sovrappongono file per file.
type livelloArricchimento []string

func nuovoLivello(contenuti, sopra string) livelloArricchimento {
	l := livelloArricchimento{filepath.Join(contenuti, "arricchimento")}
	if sopra != "" {
		l = append(livelloArricchimento{sopra}, l...)
	}
	return l
}

// leggi decodifica in v il file rel della cartella più specifica che lo ha.
// Se nessuna lo ha, v resta com'è.
func (l livelloArricchimento) leggi(rel string, v any) error {
	for _, d := range l {
		p := filepath.Join(d, rel)
		b, err := os.ReadFile(p)
		if errors.Is(err, os.ErrNotExist) {
			continue
		} else if err != nil {
			return err
		}
		if err := yaml.Unmarshal(b, v); err != nil {
			return fmt.Errorf("%s: %w", p, err)
		}
		return nil
	}
	return nil
}

func leggiContenuti(dir, indice string, l livelloArricchimento) (*contenuti, error) {
	c := &contenuti{entita: map[string]*entita{}}
	idx, err := leggiIndice(indice)
	if err != nil {
		return nil, err
	}
	for i := range idx {
		c.entita[idx[i].Slug] = &idx[i]
	}
	b, err := os.ReadFile(filepath.Join(dir, "importazione", "album.yaml"))
	if err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal(b, &c.album); err != nil {
		return nil, err
	}
	var manuali fileMappature
	if err := l.leggi("mappature.yaml", &manuali); err != nil {
		return nil, err
	}
	mappe, _ := mappature(c.album, idx, manuali)
	alias, err := leggiAlias(l)
	if err != nil {
		return nil, err
	}
	file, err := filepath.Glob(filepath.Join(dir, "importazione", "foto", "*.yaml"))
	if err != nil {
		return nil, err
	}
	for _, p := range file {
		b, err := os.ReadFile(p)
		if err != nil {
			return nil, err
		}
		f := &fotoPubblicata{}
		if err := yaml.Unmarshal(b, f); err != nil {
			return nil, fmt.Errorf("%s: %w", p, err)
		}
		f.Authors = alias.applica(f.Authors)
		if mp := mappe[f.Coppermine.Aid]; mp != nil {
			f.Slug, f.Tags, f.Categories = mp.Slug, mp.Tags, mp.Categories
		}
		// Sovrascrive solo i campi presenti.
		if err := l.leggi(filepath.Join("foto", strconv.Itoa(f.Pid)+".yaml"), f); err != nil {
			return nil, err
		}
		if f.Riservata {
			continue
		}
		c.completa(f)
		c.foto = append(c.foto, f)
	}
	slices.SortFunc(c.foto, func(a, b *fotoPubblicata) int { return a.Pid - b.Pid })
	return c, nil
}

// aliasAutori viene da arricchimento/autori.yaml, scritto solo a mano (#19): valore grezzo
// di Coppermine → Autori. Più Autori per i composti, nessuno per scartare il valore.
type aliasAutori map[string][]string

func leggiAlias(l livelloArricchimento) (aliasAutori, error) {
	a := aliasAutori{}
	return a, l.leggi("autori.yaml", &a)
}

// applica sostituisce i valori che hanno un alias; gli altri passano grezzi.
func (a aliasAutori) applica(grezzi []string) []string {
	var out []string
	for _, g := range grezzi {
		if v, ok := a[g]; ok {
			out = append(out, v...)
		} else {
			out = append(out, g)
		}
	}
	return out
}

// licenzaSito è la Licenza predefinita delle Foto, la stessa del Sito principale.
const licenzaSito = "CC-BY-SA-4.0"

// licenze sono quelle che chi carica potrà scegliere (ADR 0010 rivisto): nome e testo.
var licenze = map[string]struct{ nome, url string }{
	"CC-BY-SA-4.0":    {"CC BY-SA 4.0", "https://creativecommons.org/licenses/by-sa/4.0/"},
	"CC-BY-4.0":       {"CC BY 4.0", "https://creativecommons.org/licenses/by/4.0/"},
	"CC-BY-NC-SA-4.0": {"CC BY-NC-SA 4.0", "https://creativecommons.org/licenses/by-nc-sa/4.0/"},
	"CC0-1.0":         {"CC0 1.0", "https://creativecommons.org/publicdomain/zero/1.0/"},
}

// completa valida gli slug ed eredita dalla prima Entità i campi geografici mancanti.
// Le coords no: il Punto geografico mancante lo sostituisce la Vista per mappa.
func (c *contenuti) completa(f *fotoPubblicata) {
	if f.Licenza == "" {
		f.Licenza = licenzaSito
	}
	var validi []string
	for _, s := range f.Slug {
		if c.entita[s] == nil {
			c.ignoti = append(c.ignoti, fmt.Sprintf("%d: %s", f.Pid, s))
			continue
		}
		validi = append(validi, s)
	}
	f.Slug = validi
	if len(f.Slug) == 0 {
		return
	}
	g := c.entita[f.Slug[0]].Geolocation
	if f.Geolocation.Stato == "" {
		f.Geolocation.Stato, f.Geolocation.Regione, f.Geolocation.Provincia = g.Stato, g.Regione, g.Provincia
	}
}

// datazioneTesto è la Datazione come si mostra nella pagina della Foto e nell'Indice di
// ricerca: in italiano se c'è, altrimenti user3 di Coppermine così com'è (#18).
func (f *fotoPubblicata) datazioneTesto() string {
	if f.Datazione == "" {
		return f.Coppermine.Data
	}
	if t, err := time.Parse("2006-01-02", f.Datazione); err == nil {
		return t.Format("02/01/2006")
	}
	return f.Datazione
}

// tagPubblici sono i tag senza i promemoria interni.
func (f *fotoPubblicata) tagPubblici() []string {
	var t []string
	for _, s := range f.Tags {
		if !tagInterni[s] {
			t = append(t, s)
		}
	}
	return t
}
