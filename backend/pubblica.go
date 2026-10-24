package main

// Foto pubblicata (ADR 0009, 0013): l'Importazione con gli alias degli Autori, poi la
// Mappatura della categoria antenata, poi quella dell'album, poi arricchimento/foto/<pid>.yaml.
// Per ogni campo vince il livello più specifico.

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"

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

func leggiContenuti(dir, indice string) (*contenuti, error) {
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
	var m fileMappature
	if b, err := os.ReadFile(filepath.Join(dir, "arricchimento", "mappature.yaml")); err == nil {
		if err := yaml.Unmarshal(b, &m); err != nil {
			return nil, err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	alias, err := leggiAlias(dir)
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
		if mp := c.mappaturaAlbum(f.Coppermine.Aid, m); mp != nil {
			f.Slug, f.Tags, f.Categories = mp.Slug, mp.Tags, mp.Categories
		}
		arr := filepath.Join(dir, "arricchimento", "foto", strconv.Itoa(f.Pid)+".yaml")
		if b, err := os.ReadFile(arr); err == nil {
			if err := yaml.Unmarshal(b, f); err != nil { // sovrascrive solo i campi presenti
				return nil, fmt.Errorf("%s: %w", arr, err)
			}
		} else if !errors.Is(err, os.ErrNotExist) {
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

func leggiAlias(dir string) (aliasAutori, error) {
	a := aliasAutori{}
	b, err := os.ReadFile(filepath.Join(dir, "arricchimento", "autori.yaml"))
	if errors.Is(err, os.ErrNotExist) {
		return a, nil
	} else if err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal(b, &a); err != nil {
		return nil, fmt.Errorf("autori.yaml: %w", err)
	}
	return a, nil
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

// mappaturaAlbum è la riga dell'album se ha qualcosa, altrimenti quella della categoria
// antenata più vicina che ha uno slug.
func (c *contenuti) mappaturaAlbum(aid int, m fileMappature) *mappatura {
	if r := m.Album[aid]; r != nil && len(r.Slug)+len(r.Tags)+len(r.Categories) > 0 {
		return r
	}
	cid := c.album.Album[aid].Categoria
	for i := 0; cid != 0 && i < 10; i++ {
		if r := m.Categorie[cid]; r != nil && len(r.Slug) > 0 {
			return r
		}
		cid = c.album.Categorie[cid].Parent
	}
	return nil
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
