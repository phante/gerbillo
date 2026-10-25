package main

// Filtri e ordinamento dell'elenco delle Foto (#49, #57). Tutto sta nell'URL, così il tasto
// indietro funziona e un elenco si può salvare:
//   s   «Da sistemare» (più valori, tutti veri insieme)
//   e   Entità (slug)        a  album Coppermine (aid)   c  Categoria   t  Tag
//   u   Contributore (id)    au Autore                    f  Formato (immagine, documento)
//   q   testo libero su titolo e descrizione
//   o   ordinamento: album (predefinito, poi pid), pid, data (di caricamento), datazione
// Ogni filtro mostra quante Foto contiene insieme agli altri attivi.

import (
	"cmp"
	"net/url"
	"slices"
	"strconv"
	"strings"
)

// dimensione è un filtro a valore singolo: i valori che una Foto ha, con il loro testo.
type dimensione struct {
	param, nome string
	voci        func(x *fotoIndice, testo func(string) string) [][2]string
}

func uguali(v []string) [][2]string {
	out := make([][2]string, len(v))
	for i, x := range v {
		out[i] = [2]string{x, x}
	}
	return out
}

var dimensioni = []dimensione{
	{"e", "Entità", func(x *fotoIndice, testo func(string) string) [][2]string {
		out := make([][2]string, len(x.Slug))
		for i, s := range x.Slug {
			out[i] = [2]string{s, testo(s)}
		}
		return out
	}},
	{"a", "Album Coppermine", func(x *fotoIndice, _ func(string) string) [][2]string {
		return [][2]string{{strconv.Itoa(x.Aid), cmp.Or(x.Album, "album") + " (" + strconv.Itoa(x.Aid) + ")"}}
	}},
	{"c", "Categoria", func(x *fotoIndice, _ func(string) string) [][2]string { return uguali(x.Categorie) }},
	{"t", "Tag", func(x *fotoIndice, _ func(string) string) [][2]string { return uguali(x.Tag) }},
	{"u", "Contributore", func(x *fotoIndice, _ func(string) string) [][2]string {
		if x.Contributore == nil {
			return nil
		}
		return [][2]string{{strconv.Itoa(x.Contributore.ID), x.Contributore.Nome}}
	}},
	{"au", "Autore", func(x *fotoIndice, _ func(string) string) [][2]string { return uguali(x.Autori) }},
	{"f", "Formato", func(x *fotoIndice, _ func(string) string) [][2]string {
		if x.Formato == "immagine" {
			return [][2]string{{"immagine", "immagine"}}
		}
		return [][2]string{{"documento", "documento"}}
	}},
}

var ordinamenti = []struct{ chiave, nome string }{
	{"album", "album, poi pid"}, {"pid", "pid"}, {"data", "Data di caricamento"}, {"datazione", "Datazione"},
}

// filtro è una vista dell'elenco.
type filtro struct {
	sistemare []string          // chiavi di filtriDaSistemare, tutte vere insieme
	valori    map[string]string // param di una dimensione → valore
	testo     string
	ordine    string // "" = album
}

func leggiFiltro(q url.Values) filtro {
	f := filtro{valori: map[string]string{}}
	for _, s := range q["s"] {
		if slices.ContainsFunc(filtriDaSistemare(), func(d daSistemare) bool { return d.chiave == s }) && !slices.Contains(f.sistemare, s) {
			f.sistemare = append(f.sistemare, s)
		}
	}
	slices.Sort(f.sistemare)
	for _, d := range dimensioni {
		if v := strings.TrimSpace(q.Get(d.param)); v != "" {
			f.valori[d.param] = v
		}
	}
	f.testo = strings.TrimSpace(q.Get("q"))
	if o := q.Get("o"); slices.ContainsFunc(ordinamenti, func(x struct{ chiave, nome string }) bool { return x.chiave == o }) && o != "album" {
		f.ordine = o
	}
	return f
}

func (f filtro) clona() filtro {
	g := f
	g.sistemare = slices.Clone(f.sistemare)
	g.valori = map[string]string{}
	for k, v := range f.valori {
		g.valori[k] = v
	}
	return g
}

func (f filtro) valoriURL() url.Values {
	v := url.Values{}
	for _, s := range f.sistemare {
		v.Add("s", s)
	}
	for _, d := range dimensioni {
		if x := f.valori[d.param]; x != "" {
			v.Set(d.param, x)
		}
	}
	if f.testo != "" {
		v.Set("q", f.testo)
	}
	if f.ordine != "" {
		v.Set("o", f.ordine)
	}
	return v
}

// query è la parte dopo il ? (vuota senza filtri), da aggiungere ai link.
func (f filtro) query() string {
	if q := f.valoriURL().Encode(); q != "" {
		return "?" + q
	}
	return ""
}

// attivi è quanti filtri, oltre a «Da sistemare», sono attivi.
func (f filtro) attivi() int {
	n := len(f.valori)
	if f.testo != "" {
		n++
	}
	return n
}

func (f filtro) vale(x *fotoIndice) bool { return f.valeSenza(x, "") }

// valeSenza dice se x passa il filtro ignorando la dimensione param: serve ai conteggi.
func (f filtro) valeSenza(x *fotoIndice, param string) bool {
	for _, d := range filtriDaSistemare() {
		if slices.Contains(f.sistemare, d.chiave) && !d.vale(x) {
			return false
		}
	}
	for _, d := range dimensioni {
		v, ok := f.valori[d.param]
		if !ok || d.param == param {
			continue
		}
		if !slices.ContainsFunc(d.voci(x, func(s string) string { return s }), func(c [2]string) bool { return c[0] == v }) {
			return false
		}
	}
	if f.testo != "" {
		q := strings.ToLower(f.testo)
		if !strings.Contains(strings.ToLower(x.Titolo), q) && !strings.Contains(strings.ToLower(x.Descrizione), q) {
			return false
		}
	}
	return true
}

func (f filtro) applica(idx *indiceRedazione) []*fotoIndice {
	var out []*fotoIndice
	for i := range idx.Foto {
		if f.vale(&idx.Foto[i]) {
			out = append(out, &idx.Foto[i])
		}
	}
	switch f.ordine { // l'indice è già per album, poi per pid
	case "pid":
		slices.SortFunc(out, func(a, b *fotoIndice) int { return a.Pid - b.Pid })
	case "data":
		slices.SortStableFunc(out, func(a, b *fotoIndice) int { return cmp.Or(strings.Compare(a.Data, b.Data), a.Pid-b.Pid) })
	case "datazione": // le Foto senza Datazione in fondo
		slices.SortStableFunc(out, func(a, b *fotoIndice) int {
			if (a.Datazione == "") != (b.Datazione == "") {
				if a.Datazione == "" {
					return 1
				}
				return -1
			}
			return cmp.Or(strings.Compare(a.Datazione, b.Datazione), a.Pid-b.Pid)
		})
	}
	return out
}

// conAltro è il filtro con il «Da sistemare» s acceso o spento.
func (f filtro) conAltro(s string) filtro {
	g := f.clona()
	if i := slices.Index(g.sistemare, s); i >= 0 {
		g.sistemare = slices.Delete(g.sistemare, i, i+1)
	} else {
		g.sistemare = append(g.sistemare, s)
		slices.Sort(g.sistemare)
	}
	return g
}

type voceFiltro struct {
	Chiave, Nome, Aiuto, Link string
	N                         int
	Attivo                    bool
}

// vociDaSistemare sono i filtri «Da sistemare» con quante Foto ha ciascuno insieme agli altri attivi.
func (f filtro) vociDaSistemare(idx *indiceRedazione) []voceFiltro {
	var v []voceFiltro
	for _, d := range filtriDaSistemare() {
		attivo := slices.Contains(f.sistemare, d.chiave)
		con := f
		if !attivo {
			con = f.conAltro(d.chiave)
		}
		n := 0
		for i := range idx.Foto {
			if con.vale(&idx.Foto[i]) {
				n++
			}
		}
		v = append(v, voceFiltro{Chiave: d.chiave, Nome: d.nome, Aiuto: d.aiuto, N: n, Attivo: attivo, Link: "foto/" + f.conAltro(d.chiave).query()})
	}
	return v
}

// faccetta è un filtro a valore singolo con le sue opzioni: i valori delle Foto che passano
// tutti gli altri filtri, con quante sono.
type faccetta struct {
	Param, Nome, Scelto string
	Opzioni             []opzioneFaccetta
}

type opzioneFaccetta struct {
	Valore, Testo string
	N             int
}

const maxOpzioniFaccetta = 600

func (f filtro) faccette(idx *indiceRedazione, testo func(string) string) []faccetta {
	var out []faccetta
	for _, d := range dimensioni {
		conta, testi := map[string]int{}, map[string]string{}
		for i := range idx.Foto {
			x := &idx.Foto[i]
			if !f.valeSenza(x, d.param) {
				continue
			}
			for _, c := range d.voci(x, testo) {
				conta[c[0]]++
				testi[c[0]] = c[1]
			}
		}
		fc := faccetta{Param: d.param, Nome: d.nome, Scelto: f.valori[d.param]}
		for v, n := range conta {
			fc.Opzioni = append(fc.Opzioni, opzioneFaccetta{v, testi[v], n})
		}
		slices.SortFunc(fc.Opzioni, func(a, b opzioneFaccetta) int { return cmp.Or(b.N-a.N, strings.Compare(a.Testo, b.Testo)) })
		if len(fc.Opzioni) > maxOpzioniFaccetta {
			fc.Opzioni = fc.Opzioni[:maxOpzioniFaccetta]
		}
		if fc.Scelto != "" && !slices.ContainsFunc(fc.Opzioni, func(o opzioneFaccetta) bool { return o.Valore == fc.Scelto }) {
			fc.Opzioni = append([]opzioneFaccetta{{fc.Scelto, cmp.Or(testi[fc.Scelto], fc.Scelto), conta[fc.Scelto]}}, fc.Opzioni...)
		}
		out = append(out, fc)
	}
	return out
}
