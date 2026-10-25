package main

// Campi a chip per slug, Categorie, Tag e Autori (#53), con Tom Select (ADR 0017). I
// suggerimenti mentre si scrive vengono da /admin/suggerimenti/<campo>?q=…:
//   - slug: l'Indice delle Entità, per titolo, comprese le bozze. Lo slug si sceglie,
//     non si scrive, e il salvataggio lo valida ancora contro l'Indice;
//   - Tag: quelli del Sito principale e quelli già usati nella Gallery. Un Tag che non è
//     in nessuno dei due si accetta, ma il chip lo segna «nuovo» perché i refusi si notino;
//   - Autori: quelli già usati, più i nomi degli alias di autori.yaml (#19).

import (
	"cmp"
	"encoding/json"
	"log"
	"net/http"
	"path"
	"slices"
	"strings"
	"sync"
)

// suggerimento è un'opzione di Tom Select.
type suggerimento struct {
	Value   string `json:"value"`
	Text    string `json:"text"`
	Nuovo   bool   `json:"nuovo,omitempty"`
	Interno bool   `json:"interno,omitempty"`
}

// voci calcolate una volta per indice della redazione: si ricalcolano dopo ogni build.
type vociNote struct {
	idx           *indiceRedazione
	tag, autori   map[string]int // valore → Foto che lo usano
	entita        []entita
	perSlug       map[string]*entita
	tagSito       map[string]bool
	aliasCaricati bool
	nomiAlias     []string
}

type cacheVoci struct {
	mu   sync.Mutex
	voci *vociNote
}

func (r *redazione) vociNote() *vociNote {
	r.cache.mu.Lock()
	defer r.cache.mu.Unlock()
	idx := r.indiceCorrente()
	if v := r.cache.voci; v != nil && v.idx == idx {
		return v
	}
	v := &vociNote{idx: idx, tag: map[string]int{}, autori: map[string]int{}, perSlug: map[string]*entita{}, tagSito: map[string]bool{}}
	for _, f := range idx.Foto {
		for _, t := range f.Tag {
			v.tag[t]++
		}
		for _, a := range f.Autori {
			v.autori[a]++
		}
	}
	if old := r.cache.voci; old != nil {
		v.entita, v.perSlug, v.tagSito = old.entita, old.perSlug, old.tagSito // l'Indice delle Entità non cambia
	} else if e, err := leggiIndice(r.indice); err != nil {
		log.Printf("Indice delle Entità: %v", err)
	} else {
		v.entita = e
		for i := range v.entita {
			v.perSlug[v.entita[i].Slug] = &v.entita[i]
			for _, t := range v.entita[i].Tags {
				v.tagSito[t] = true
			}
		}
	}
	if alias, err := leggiAlias(nuovoLivello(r.contenuti, "")); err == nil {
		for _, nomi := range alias {
			for _, n := range nomi {
				if !slices.Contains(v.nomiAlias, n) {
					v.nomiAlias = append(v.nomiAlias, n)
				}
			}
		}
	}
	r.cache.voci = v
	return v
}

func (v *vociNote) tagNoto(t string) bool { return v.tagSito[t] || v.tag[t] > 0 || tagInterni[t] }

// testoSlug è come lo slug compare nel chip e nei suggerimenti: «Barcis · dighe/barcis».
func (v *vociNote) testoSlug(s string) string {
	if e := v.perSlug[s]; e != nil {
		return e.Title + " · " + s
	}
	return s
}

const maxSuggerimenti = 30

// suggerimenti risponde a /admin/suggerimenti/<campo>?q=…, in JSON per Tom Select.
func (r *redazione) suggerimenti(w http.ResponseWriter, req *http.Request) {
	q := strings.ToLower(strings.TrimSpace(req.URL.Query().Get("q")))
	v := r.vociNote()
	var out []suggerimento
	contiene := func(s ...string) bool {
		for _, x := range s {
			if strings.Contains(strings.ToLower(x), q) {
				return true
			}
		}
		return false
	}
	// per frequenza, poi in ordine alfabetico
	perUso := func(conta map[string]int, extra []string) []string {
		var k []string
		for x := range conta {
			k = append(k, x)
		}
		for _, x := range extra {
			if conta[x] == 0 {
				k = append(k, x)
			}
		}
		slices.SortFunc(k, func(a, b string) int { return cmp.Or(conta[b]-conta[a], strings.Compare(a, b)) })
		return k
	}
	switch path.Base(req.URL.Path) {
	case "slug":
		for _, e := range v.entita {
			if q != "" && contiene(e.Title, e.Slug, e.Nome) {
				out = append(out, suggerimento{Value: e.Slug, Text: v.testoSlug(e.Slug)})
			}
		}
		// prima chi comincia con la ricerca, poi per titolo
		slices.SortStableFunc(out, func(a, b suggerimento) int {
			pa, pb := strings.HasPrefix(strings.ToLower(a.Text), q), strings.HasPrefix(strings.ToLower(b.Text), q)
			if pa != pb {
				if pa {
					return -1
				}
				return 1
			}
			return strings.Compare(a.Text, b.Text)
		})
	case "tags":
		var sito []string
		for t := range v.tagSito {
			sito = append(sito, t)
		}
		slices.Sort(sito)
		for _, t := range perUso(v.tag, sito) {
			if contiene(t) {
				out = append(out, suggerimento{Value: t, Text: t, Interno: tagInterni[t]})
			}
		}
	case "authors":
		for _, a := range perUso(v.autori, v.nomiAlias) {
			if contiene(a) {
				out = append(out, suggerimento{Value: a, Text: a})
			}
		}
	default:
		http.NotFound(w, req)
		return
	}
	if len(out) > maxSuggerimenti {
		out = out[:maxSuggerimenti]
	}
	w.Header().Set("Content-Type", "application/json")
	if out == nil {
		out = []suggerimento{}
	}
	json.NewEncoder(w).Encode(out)
}

// opzioniChip sono i valori di un campo lista, già scelti, come opzioni del select.
func (r *redazione) opzioniChip(chiave string, valori []string) []opzioneChip {
	v := r.vociNote()
	var o []opzioneChip
	for _, x := range valori {
		if x == "" {
			continue
		}
		s := suggerimento{Value: x, Text: x}
		switch chiave {
		case "slug":
			s.Text = v.testoSlug(x)
		case "tags":
			s.Interno, s.Nuovo = tagInterni[x], !v.tagNoto(x)
		}
		d, _ := json.Marshal(s)
		o = append(o, opzioneChip{Valore: x, Testo: s.Text, Dati: string(d)})
	}
	return o
}

type opzioneChip struct {
	Valore, Testo, Dati string // Dati è il JSON che Tom Select legge da data-opzione
}
