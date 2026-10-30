package main

// Esplorazione della Gallery (prototipo, #25): la Bussola della home, che restringe
// le dighe per Dove, Quando, Come e Per cosa senza elenchi lunghi, e «Esplora
// intorno» nella pagina della diga, che porta agli album collegati: stessa
// mappa (map_ids), vicine (coords), simili (tipologia ed epoca).

import (
	"cmp"
	"encoding/json"
	"math"
	"slices"
	"strconv"
	"strings"
)

// famiglia riduce le 23 tipologie della scheda diga a poche famiglie leggibili.
func famiglia(tipologia string) string {
	t := strings.ToLower(tipologia)
	switch {
	case t == "":
		return ""
	case strings.Contains(t, "traversa"):
		return "Traverse"
	case strings.Contains(t, "speroni") || strings.Contains(t, "contrafforti"):
		return "Contrafforti e speroni"
	case strings.Contains(t, "arco") || strings.Contains(t, "cupola"):
		return "Arco e cupola"
	case strings.Contains(t, "gravità") || strings.Contains(t, "blocchi") || strings.Contains(t, "muratura"):
		return "Gravità"
	case strings.Contains(t, "terra") || strings.Contains(t, "pietrame"):
		return "Terra e pietrame"
	}
	return "Altro"
}

// decennio legge le prime quattro cifre di anno («1957», «1900-06»).
func decennio(anno string) string {
	if len(anno) < 4 {
		return ""
	}
	a, err := strconv.Atoi(anno[:4])
	if err != nil {
		return ""
	}
	if a < 1910 {
		return "Prima del 1910"
	}
	return "Anni " + strconv.Itoa(a/10*10)
}

func dove(e *entita) string {
	if e.Geolocation.Stato != "" && e.Geolocation.Stato != "Italia" {
		return e.Geolocation.Stato
	}
	return e.Geolocation.Regione
}

func usoPrincipale(e *entita) string {
	return e.campo("uso")
}

// voceBussola è una diga con foto nel JSON della home.
type voceBussola struct {
	URL    string `json:"u"`
	Titolo string `json:"t"`
	Mini   string `json:"m"`
	N      int    `json:"n"`
	Dove   string `json:"d,omitempty"`
	Quando string `json:"q,omitempty"`
	Come   string `json:"c,omitempty"`
	Uso    string `json:"p,omitempty"`
	Cosa   string `json:"k"`
	// Gruppo: un insieme di Foto ancora da classificare, non una diga. Compare solo
	// quando si sceglie il suo Cosa.
	Gruppo bool   `json:"g,omitempty"`
	Nota   string `json:"o,omitempty"`
}

// cosaCoppermine dà il nome della Categoria Coppermine radice nella faccetta Cosa:
// «Centrali - Powerplants» → «Centrali».
func cosaCoppermine(cid int, nome string) string {
	switch cid {
	case cidDighe:
		return "Dighe"
	case 1:
		return "Album personali"
	}
	nome, _, _ = strings.Cut(nome, " - ")
	nome, _, _ = strings.Cut(nome, ",")
	return strings.TrimSpace(nome)
}

// vociGruppi aggiunge alla Bussola le Foto senza slug, una scheda per Categoria Coppermine.
func (b *builder) vociGruppi(altre map[int][]*fotoPubblicata) []voceBussola {
	var v []voceBussola
	for _, cid := range chiaviOrdinate(altre) {
		fs := altre[cid]
		cosa := cosaCoppermine(cid, b.c.album.Categorie[cid].Nome)
		titolo := cosa
		if cid == cidDighe {
			titolo = "Altre foto di dighe"
		}
		v = append(v, voceBussola{URL: b.url("altre", strconv.Itoa(cid)), Titolo: titolo, Mini: b.mini(fs[0]),
			N: len(fs), Cosa: cosa, Gruppo: true, Nota: "da classificare"})
	}
	return v
}

func (b *builder) vociBussola(perEntita map[string][]*fotoPubblicata, slugs []string) []voceBussola {
	var v []voceBussola
	for _, s := range slugs {
		e, fs := b.c.entita[s], perEntita[s]
		x := voceBussola{URL: b.url(s), Titolo: e.Title, Mini: b.mini(fs[0]), N: len(fs), Dove: dove(e), Uso: usoPrincipale(e), Cosa: "Dighe"}
		x.Quando, x.Come = decennio(e.campo("anno")), famiglia(e.campo("tipologia"))
		v = append(v, x)
	}
	// Le Entità da censire stanno solo sotto «Dove»: senza Cosa, senza scheda (ADR 0018).
	for _, id := range b.conFoto() {
		e, fs := b.c.daCensire[id], b.perCensire[id]
		v = append(v, voceBussola{URL: b.url(parteDaCensire(id)...), Titolo: e.Nome, Mini: b.mini(fs[0]), N: len(fs), Dove: dove(e.comeEntita()), Nota: nonSulSito})
	}
	// Prima le dighe con più foto: sono l'anteprima della Bussola.
	slices.SortStableFunc(v, func(a, b voceBussola) int { return b.N - a.N })
	return v
}

type gruppoEsplora struct {
	Nome string
	Voci []voce
}

const (
	zoomPredefinito = 14 // se la pagina del sito non ne indica uno
	maxIntorno      = 6
	raggioKm        = 30.0
	// maxDigheMappa: una mappa del sito con più Entità non lega davvero le dighe (#36).
	maxDigheMappa = 20
)

// mappaGenerale è la mappa di tutta Italia: è in quasi tutte le dighe (#36).
const mappaGenerale = "italia"

// stessaMappa dice se due Entità condividono una mappa del sito che non sia generale.
// Una mappa non è sempre un Impianto, che nella Gallery non è modellato (CONTEXT).
func (b *builder) stessaMappa(e, o *entita) bool {
	if b.mappe == nil {
		b.mappe = map[string]int{}
		for _, x := range b.c.entita {
			for _, m := range x.Geolocation.MapIDs {
				b.mappe[m]++
			}
		}
	}
	for _, m := range o.Geolocation.MapIDs {
		if m != mappaGenerale && b.mappe[m] <= maxDigheMappa && slices.Contains(e.Geolocation.MapIDs, m) {
			return true
		}
	}
	return false
}

// intorno calcola gli album collegati a una diga, fra quelle che hanno foto.
func (b *builder) intorno(s string, perEntita map[string][]*fotoPubblicata, slugs []string) []gruppoEsplora {
	e := b.c.entita[s]
	visti := map[string]bool{s: true}
	scheda := func(x string) voce {
		f := perEntita[x]
		return voce{URL: b.url(x), Testo: b.c.entita[x].Title, Mini: b.mini(f[0]), N: len(f)}
	}
	var gruppi []gruppoEsplora
	aggiungi := func(nome string, candidati []string) {
		var v []voce
		for _, x := range candidati {
			if !visti[x] && len(v) < maxIntorno {
				visti[x] = true
				v = append(v, scheda(x))
			}
		}
		if len(v) > 0 {
			gruppi = append(gruppi, gruppoEsplora{nome, v})
		}
	}
	// Stessa mappa: condividono una mappa del sito.
	var mappa []string
	for _, x := range slugs {
		if b.stessaMappa(e, b.c.entita[x]) {
			mappa = append(mappa, x)
		}
	}
	aggiungi("Stessa mappa", mappa)
	// Vicine: entro raggioKm, le più vicine prima.
	perDistanza := func(lista []string) []string {
		if e.Geolocation.Coords == nil {
			return lista
		}
		d := map[string]float64{}
		var out []string
		for _, x := range lista {
			if c := b.c.entita[x].Geolocation.Coords; c != nil {
				d[x] = km(e.Geolocation.Coords.Latitudine, e.Geolocation.Coords.Longitudine, c.Latitudine, c.Longitudine)
				out = append(out, x)
			}
		}
		slices.SortStableFunc(out, func(a, b string) int { return cmp.Compare(d[a], d[b]) })
		return out
	}
	var vicine []string
	for _, x := range perDistanza(slugs) {
		c := b.c.entita[x].Geolocation.Coords
		if e.Geolocation.Coords != nil && km(e.Geolocation.Coords.Latitudine, e.Geolocation.Coords.Longitudine, c.Latitudine, c.Longitudine) <= raggioKm {
			vicine = append(vicine, x)
		}
	}
	aggiungi("Vicine", vicine)
	// Simili: stessa famiglia di tipologia e stesso decennio, le più vicine prima.
	if fam := famiglia(e.campo("tipologia")); fam != "" {
		var simili []string
		for _, x := range slugs {
			if o := b.c.entita[x]; famiglia(o.campo("tipologia")) == fam && decennio(o.campo("anno")) == decennio(e.campo("anno")) {
				simili = append(simili, x)
			}
		}
		nome := "Simili: " + strings.ToLower(fam)
		if q := decennio(e.campo("anno")); q != "" {
			nome += ", " + strings.ToLower(q)
		}
		aggiungi(nome, perDistanza(simili))
	}
	return gruppi
}

// km è la distanza sul globo fra due punti, con la formula dell'emisenoverso.
func km(lat1, lon1, lat2, lon2 float64) float64 {
	const r = 6371.0
	rad := math.Pi / 180
	dlat, dlon := (lat2-lat1)*rad, (lon2-lon1)*rad
	a := math.Sin(dlat/2)*math.Sin(dlat/2) + math.Cos(lat1*rad)*math.Cos(lat2*rad)*math.Sin(dlon/2)*math.Sin(dlon/2)
	return 2 * r * math.Asin(math.Sqrt(a))
}

// geojsonDiga è la mappa della pagina della diga (#27), per il componente mappa del
// sito (modelli/sito/mappa.js): la diga stessa, che porta alla sua pagina sul sito,
// e le dighe con foto della stessa mappa o entro raggioKm, che portano ai loro
// album. Vuota se la diga non ha coordinate.
func (b *builder) geojsonDiga(s string, perEntita map[string][]*fotoPubblicata, slugs []string) (centro, geo string, zoom int) {
	e := b.c.entita[s]
	c := e.Geolocation.Coords
	if c == nil {
		return "", "", 0
	}
	type feature struct {
		Type       string         `json:"type"`
		ID         string         `json:"id,omitempty"`
		Properties map[string]any `json:"properties"`
		Geometry   struct {
			Type        string     `json:"type"`
			Coordinates [2]float64 `json:"coordinates"`
		} `json:"geometry"`
	}
	punto := func(x *entita, id, nome, url string) feature {
		// Come le feature della mappa generale del sito: la scheda diga intera, per lo
		// specchietto del popup, più tipo, nome e collegamento.
		p := map[string]any{}
		for k, v := range x.Diga {
			p[k] = v
		}
		p["type"], p["name"] = "diga", nome
		if url != "" {
			p["url"] = url
		}
		f := feature{Type: "Feature", ID: id, Properties: p}
		f.Geometry.Type = "Point"
		f.Geometry.Coordinates = [2]float64{x.Geolocation.Coords.Longitudine, x.Geolocation.Coords.Latitudine}
		return f
	}
	sito := ""
	if !e.Draft {
		sito = b.sito + s + "/"
	}
	fs := []feature{punto(e, "main", e.Title, sito)}
	for _, x := range slugs {
		o := b.c.entita[x]
		if x == s || o.Geolocation.Coords == nil {
			continue
		}
		if b.stessaMappa(e, o) || km(c.Latitudine, c.Longitudine, o.Geolocation.Coords.Latitudine, o.Geolocation.Coords.Longitudine) <= raggioKm {
			fs = append(fs, punto(o, "", o.Title+" · "+strconv.Itoa(len(perEntita[x]))+" foto", b.url(x)))
		}
	}
	// Lo zoom della pagina della diga sul sito, come le mappe degli articoli: le dighe
	// intorno sono sulla mappa, ma si raggiungono spostandola.
	zoom = c.Zoom
	if zoom == 0 {
		zoom = zoomPredefinito
	}
	g, _ := json.Marshal(map[string]any{"type": "FeatureCollection", "features": fs})
	ctr, _ := json.Marshal([2]float64{c.Latitudine, c.Longitudine})
	return string(ctr), string(g), zoom
}
