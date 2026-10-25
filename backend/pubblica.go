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
	"strings"

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
	Fonte        string // da dove proviene la Foto, se non è opera di chi la carica
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
// Gli stessi di tags_interni in dati/vocabolario.yaml (TestVocabolarioDelRepo).
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
	st, idx, err := leggiStrati(dir, indice, l, &c.album)
	if err != nil {
		return nil, err
	}
	for i := range idx {
		c.entita[idx[i].Slug] = &idx[i]
	}
	file, err := filepath.Glob(filepath.Join(dir, "importazione", "foto", "*.yaml"))
	if err != nil {
		return nil, err
	}
	for _, p := range file {
		f, err := st.importata(p)
		if err != nil {
			return nil, err
		}
		switch st.decisioni[f.Coppermine.Aid] {
		case cancellaAlbum:
			continue
		case pubblicaAlbum:
			f.Riservata = false
		}
		if err := st.arricchisci(f); err != nil {
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

// strati sono i livelli che compongono la Foto pubblicata sopra l'Importazione.
type strati struct {
	l         livelloArricchimento
	mappe     map[int]*mappatura // aid → Mappatura
	decisioni map[int]string     // riservati.yaml
	alias     aliasAutori
}

func leggiStrati(dir, indice string, l livelloArricchimento, album *albumCoppermine) (*strati, []entita, error) {
	idx, err := leggiIndice(indice)
	if err != nil {
		return nil, nil, err
	}
	b, err := os.ReadFile(filepath.Join(dir, "importazione", "album.yaml"))
	if err != nil {
		return nil, nil, err
	}
	if err := yaml.Unmarshal(b, album); err != nil {
		return nil, nil, err
	}
	st := &strati{l: l}
	var manuali fileMappature
	if err := l.leggi("mappature.yaml", &manuali); err != nil {
		return nil, nil, err
	}
	st.mappe, _ = mappature(*album, idx, manuali)
	if st.decisioni, err = leggiRiservati(l, *album); err != nil {
		return nil, nil, err
	}
	if st.alias, err = leggiAlias(l); err != nil {
		return nil, nil, err
	}
	return st, idx, nil
}

// importata è la Foto dell'Importazione con gli alias degli Autori e la Mappatura dell'album:
// tutto tranne arricchimento/foto/<pid>.yaml. Il Backend confronta con questa (#43).
func (st *strati) importata(p string) (*fotoPubblicata, error) {
	b, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	f := &fotoPubblicata{}
	if err := yaml.Unmarshal(b, f); err != nil {
		return nil, fmt.Errorf("%s: %w", p, err)
	}
	f.Authors = st.alias.applica(f.Authors)
	if mp := st.mappe[f.Coppermine.Aid]; mp != nil {
		f.Slug, f.Tags, f.Categories = slices.Clone(mp.Slug), slices.Clone(mp.Tags), slices.Clone(mp.Categories)
	}
	return f, nil
}

// arricchisci applica arricchimento/foto/<pid>.yaml, che sovrascrive solo i campi presenti.
func (st *strati) arricchisci(f *fotoPubblicata) error {
	return st.l.leggi(fileFoto(f.Pid), f)
}

func fileFoto(pid int) string { return filepath.Join("foto", strconv.Itoa(pid)+".yaml") }

// Decisioni su un album non pubblico di Coppermine, in arricchimento/riservati.yaml (#35).
// Un album senza riga resta riservato: la protezione di #9 regge anche senza arricchimento.
// Le eccezioni sulla singola Foto vanno in foto/<pid>.yaml con riservata: false.
const (
	pubblicaAlbum = "pubblica" // toglie riservata alle Foto dell'album
	cancellaAlbum = "cancella" // le Foto restano fuori dal build; Coppermine non si tocca
)

func leggiRiservati(l livelloArricchimento, album albumCoppermine) (map[int]string, error) {
	var r struct {
		Album map[int]string `yaml:"album"`
	}
	if err := l.leggi("riservati.yaml", &r); err != nil {
		return nil, err
	}
	var errori []string
	for _, aid := range chiaviOrdinate(r.Album) {
		v := r.Album[aid]
		if _, ok := album.Album[aid]; !ok {
			errori = append(errori, fmt.Sprintf("album %d inesistente", aid))
		} else if v != pubblicaAlbum && v != cancellaAlbum {
			errori = append(errori, fmt.Sprintf("album %d: %q, ammessi %s o %s", aid, v, pubblicaAlbum, cancellaAlbum))
		}
	}
	if len(errori) > 0 {
		return nil, fmt.Errorf("riservati.yaml:\n%s", strings.Join(errori, "\n"))
	}
	return r.Album, nil
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
