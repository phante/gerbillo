package main

// Mappatura degli Album di Coppermine verso Slug, categories e tags (G2, #10, #1, #38). È un
// dato derivato: si calcola a ogni build e a ogni Importazione, con la catena riga manuale →
// abbinamento automatico sul nome → Mappatura della categoria. Le righe manuali stanno in
// arricchimento/mappature.yaml, scritto solo a mano come autori.yaml (#19).

import (
	"bytes"
	"cmp"
	"fmt"
	"regexp"
	"slices"
	"strings"
)

const (
	cidUtenti    = 1 // User galleries
	cidDighe     = 2
	cidCentrali  = 3
	cidPersonali = 10000 // gli album delle User galleries hanno categoria 10000 + owner
)

// daCollegare è il tag interno degli album senza riga manuale né abbinamento certo:
// manca lo Slug, lo si sceglie dal Backend.
const daCollegare = "da-collegare"

// vocabolario è dati/vocabolario.yaml: i valori ammessi di categories e i tag interni,
// che il build non mostra e che vanno azzerati prima dello Switch (#1).
type vocabolario struct {
	Categories  []string `yaml:"categories"`
	TagsInterni []string `yaml:"tags_interni"`
}

// fuoriVocabolario elenca le categories delle righe che il vocabolario non conosce.
func fuoriVocabolario(m fileMappature, voc vocabolario) []string {
	var fuori []string
	for _, righe := range []struct {
		tipo string
		m    map[int]*mappatura
	}{{"categoria", m.Categorie}, {"album", m.Album}} {
		for id, r := range righe.m {
			if r == nil {
				continue
			}
			for _, c := range r.Categories {
				if !slices.Contains(voc.Categories, c) {
					fuori = append(fuori, fmt.Sprintf("%s %d: %s", righe.tipo, id, c))
				}
			}
		}
	}
	slices.Sort(fuori)
	return fuori
}

// mappatura è una riga di arricchimento/mappature.yaml. Una riga sostituisce in blocco
// l'abbinamento automatico; una riga senza slug vuol dire "nessuna Entità".
type mappatura struct {
	Categories []string `yaml:"categories,omitempty"`
	Slug       []string `yaml:"slug,omitempty"`
	Tags       []string `yaml:"tags,omitempty"`
	auto       bool     // abbinamento automatico, non una riga manuale
}

func (m *mappatura) vuota() bool { return len(m.Slug)+len(m.Categories)+len(m.Tags) == 0 }

type fileMappature struct {
	Album     map[int]*mappatura `yaml:"album"`
	Categorie map[int]*mappatura `yaml:"categorie"`
}

func perc(a, b int) float64 {
	if b == 0 {
		return 0
	}
	return 100 * float64(a) / float64(b)
}

type voceRapporto struct {
	aid, foto int
	percorso  string
	candidati []string
}

// contoRadice è una riga della tabella del rapporto: una radice di Coppermine.
type contoRadice struct {
	nome                                     string
	album, foto                              int
	autoAlbum, autoFoto, eredAlbum, eredFoto int
	manAlbum, manFoto                        int
	ambAlbum, ambFoto, faAlbum, faFoto       int
}

type rapportoMappature struct {
	radici          map[int]*contoRadice
	daFare, ambigui []voceRapporto
	sconosciuti     []string       // "album 12: dighe/xxx"
	interni         map[string]int // tag interno → Foto che lo hanno, da azzerare prima dello Switch
}

func (r *rapportoMappature) totale() (t contoRadice) {
	for _, c := range r.radici {
		t.album += c.album
		t.foto += c.foto
		t.autoAlbum += c.autoAlbum
		t.autoFoto += c.autoFoto
		t.eredAlbum += c.eredAlbum
		t.eredFoto += c.eredFoto
		t.manAlbum += c.manAlbum
		t.manFoto += c.manFoto
		t.ambAlbum += c.ambAlbum
		t.ambFoto += c.ambFoto
		t.faAlbum += c.faAlbum
		t.faFoto += c.faFoto
	}
	t.nome = "totale"
	return
}

// Le radici di Coppermine che hanno una sezione del sito: i loro album si abbinano solo lì.
// Gli altri album (Opere di presa, Tecnica, album personali, …) si abbinano a tutte le sezioni.
var sezioniRadice = map[int][]string{cidDighe: {"dighe"}, cidCentrali: {"centrali"}}

// Una proposta verso reportage e tecnica non è mai certa: i titoli sono generici (#1).
func sezioneIncerta(slug string) bool {
	return strings.HasPrefix(slug, "reportage/") || strings.HasPrefix(slug, "tecnica/")
}

// mappature risolve la Mappatura di ogni album: la riga manuale, se c'è, altrimenti
// l'abbinamento automatico sul nome, altrimenti la Mappatura della categoria antenata più
// vicina. Gli album senza nessuna delle tre hanno il tag interno da-collegare.
func mappature(cpg albumCoppermine, idx []entita, manuali fileMappature) (map[int]*mappatura, *rapportoMappature) {
	perSezione := func(sez []string) map[string][]*entita {
		var v []entita
		for _, e := range idx {
			if slices.Contains(sez, e.Slug[:strings.Index(e.Slug, "/")]) {
				v = append(v, e)
			}
		}
		return chiaviEntita(v)
	}
	chiavi := map[int]map[string][]*entita{0: perSezione(sezioniEntita)}
	for r, sez := range sezioniRadice {
		chiavi[r] = perSezione(sez)
	}
	noti := map[string]bool{}
	for _, e := range idx {
		noti[e.Slug] = true
	}
	// Gli album personali hanno categoria 10000 + owner, che non esiste: stanno sotto User galleries.
	padre := func(cid int) (categoriaYAML, bool) {
		if c, ok := cpg.Categorie[cid]; ok {
			return c, true
		}
		if cid > 10000 {
			return categoriaYAML{"", cidUtenti}, true
		}
		return categoriaYAML{}, false
	}
	percorso := func(cid int) []string {
		var p []string
		for cid != 0 && len(p) < 10 {
			c, ok := padre(cid)
			if !ok {
				break
			}
			if c.Nome != "" {
				p = append([]string{c.Nome}, p...)
			}
			cid = c.Parent
		}
		return p
	}
	radice := func(cid int) int {
		for i := 0; i < 10; i++ {
			c, ok := padre(cid)
			if !ok || c.Parent == 0 {
				return cid
			}
			cid = c.Parent
		}
		return cid
	}
	categorie := map[int]*mappatura{}
	out := map[int]*mappatura{}
	rap := &rapportoMappature{radici: map[int]*contoRadice{}, interni: map[string]int{}}
	controlla := func(dove string, m *mappatura) {
		for _, s := range m.Slug {
			if !noti[s] {
				rap.sconosciuti = append(rap.sconosciuti, dove+": "+s)
			}
		}
	}

	// Una riga vuota ("33: # Svizzera / Verzasca") nel YAML è nil: è comunque una riga manuale.
	manuale := func(m map[int]*mappatura, id int) *mappatura {
		v, ok := m[id]
		if ok && v == nil {
			v = &mappatura{}
		}
		return v
	}
	for cid, c := range cpg.Categorie {
		if v := manuale(manuali.Categorie, cid); v != nil {
			categorie[cid] = v
			controlla(fmt.Sprintf("categoria %d", cid), v)
			continue
		}
		// In automatico solo dal terzo livello sotto Dighe e Centrali (Dighe > Italia > regione > diga),
		// perché stati e regioni non sono dighe.
		p := percorso(cid)
		r := radice(cid)
		if sezioniRadice[r] == nil || len(p) < 4 {
			continue
		}
		if s, certo := trova(c.Nome, p, chiavi[r]); certo && len(s) == 1 {
			categorie[cid] = &mappatura{Slug: s, auto: true}
		}
	}

	// ereditata è la Mappatura della categoria antenata più vicina che ha uno Slug, delle categories o dei tags.
	ereditata := func(cid int) *mappatura {
		for i := 0; cid != 0 && i < 10; i++ {
			if m := categorie[cid]; m != nil && !m.vuota() {
				return m
			}
			c, ok := padre(cid)
			if !ok {
				return nil
			}
			cid = c.Parent
		}
		return nil
	}
	for aid, a := range cpg.Album {
		r := radice(a.Categoria)
		cont := rap.radici[r]
		if cont == nil {
			c, _ := padre(r)
			cont = &contoRadice{nome: cmp.Or(c.Nome, fmt.Sprintf("categoria %d", r))}
			rap.radici[r] = cont
		}
		p := percorso(a.Categoria)
		cont.album++
		cont.foto += a.Foto
		effettiva := func(m *mappatura) {
			out[aid] = m
			for _, t := range m.Tags {
				if tagInterni[t] {
					rap.interni[t] += a.Foto
				}
			}
		}
		if v := manuale(manuali.Album, aid); v != nil {
			cont.manAlbum++
			cont.manFoto += a.Foto
			controlla(fmt.Sprintf("album %d", aid), v)
			effettiva(v)
			continue
		}
		ch := chiavi[r]
		if ch == nil {
			ch = chiavi[0]
		}
		nome := a.Title
		if r == cidCentrali {
			// "Centrale Caprile - Ligonchio (RE)": dopo il trattino c'è il comune, che è di un'altra centrale.
			nome = reSeparati.Split(reParentesi.ReplaceAllString(nome, " "), 2)[0]
		}
		s, certo := trova(nome, p, ch)
		if slices.ContainsFunc(s, sezioneIncerta) {
			certo = false
		}
		ered := ereditata(a.Categoria)
		switch {
		case certo && len(s) == 1:
			// Lo Slug viene dal nome, categories e tags dalla categoria (#1).
			m := &mappatura{Slug: s, auto: true}
			if ered != nil {
				m.Categories, m.Tags = slices.Clone(ered.Categories), slices.Clone(ered.Tags)
			}
			cont.autoAlbum++
			cont.autoFoto += a.Foto
			effettiva(m)
		case ered != nil:
			// Sottoalbum di una diga ("La Forra" sotto "Vajont"): eredita dalla categoria.
			cont.eredAlbum++
			cont.eredFoto += a.Foto
			if ered.auto {
				cont.autoAlbum++
				cont.autoFoto += a.Foto
			}
			effettiva(ered)
		default:
			effettiva(&mappatura{Tags: []string{daCollegare}})
			v := voceRapporto{aid, a.Foto, strings.Join(append(p, a.Title), " / "), s}
			if certo && len(s) > 1 {
				rap.ambigui = append(rap.ambigui, v)
				cont.ambAlbum++
				cont.ambFoto += a.Foto
			} else {
				rap.daFare = append(rap.daFare, v)
				cont.faAlbum++
				cont.faFoto += a.Foto
			}
		}
	}
	ordina := func(v []voceRapporto) {
		slices.SortFunc(v, func(a, b voceRapporto) int { return cmp.Or(b.foto-a.foto, a.aid-b.aid) })
	}
	ordina(rap.daFare)
	ordina(rap.ambigui)
	slices.Sort(rap.sconosciuti)
	return out, rap
}

// chiaviEntita indicizza le Entità per nome normalizzato: titolo e nome dello slug,
// senza il suffisso numerico degli omonimi (montagna-spaccata-1).
func chiaviEntita(idx []entita) map[string][]*entita {
	k := map[string][]*entita{}
	agg := func(chiave string, e *entita) {
		if chiave != "" && !slices.Contains(k[chiave], e) {
			k[chiave] = append(k[chiave], e)
		}
	}
	for i := range idx {
		e := &idx[i]
		nome := e.Slug[strings.LastIndex(e.Slug, "/")+1:]
		agg(normNome(e.Title), e)
		agg(normNome(e.Nome), e)
		// Nomi alternativi: "Lumiei o di La Maina di Sauris", "Vicaglia, Silano".
		for _, alt := range reAlternativa.Split(e.Title, -1) {
			agg(normNome(alt), e)
			for _, parte := range strings.Split(alt, ",") {
				agg(normNome(parte), e)
			}
		}
		// Le centrali si chiamano "Achille Gaggia - Soverzene (BL)": conta il nome, non il comune,
		// che darebbe falsi abbinamenti ("Torrite - Castelnuovo Garfagnana").
		if strings.HasPrefix(e.Slug, "centrali/") {
			agg(normNome(reSeparati.Split(reParentesi.ReplaceAllString(e.Title, " "), 2)[0]), e)
		}
		agg(normNome(nome), e)
		agg(reOmonimo.ReplaceAllString(nome, ""), e)
	}
	return k
}

var (
	reParentesi   = regexp.MustCompile(`\([^)]*\)`)
	reSeparati    = regexp.MustCompile(`\s+[-–/]\s+|,\s*`)
	reNonAlnum    = regexp.MustCompile(`[^a-z0-9]+`)
	reOmonimo     = regexp.MustCompile(`-\d+$`)
	reAlternativa = regexp.MustCompile(`\s+o\s+(di\s+)?`)
	reMultipla    = regexp.MustCompile(`[-.]|\s+e\s+`) // "Arno-Campellio-Isola", "Pian Sapeio e Giacopiane"
	reLocalita    = regexp.MustCompile(`-(di|del|della|in|sul|sulla|presso)-.*$`)
	reIniziale    = regexp.MustCompile(`^(il|lo|la|l|i|gli|le|di|del|della|dello|dei|delle|degli)-`) // "Delle Piazze", "La Stua"
	reArticolo    = regexp.MustCompile(`^(diga|lago|invaso|serbatoio|traversa|sbarramento|centrale)(-(di|del|della|dello|dei|delle|degli|d|dell))?-`)
	accenti       = strings.NewReplacer("à", "a", "á", "a", "â", "a", "ä", "a", "è", "e", "é", "e", "ê", "e", "ë", "e",
		"ì", "i", "í", "i", "î", "i", "ï", "i", "ò", "o", "ó", "o", "ô", "o", "ö", "o",
		"ù", "u", "ú", "u", "û", "u", "ü", "u", "ç", "c", "ñ", "n", "ß", "ss", "'", "", "’", "")
)

// normNome riduce un nome a una chiave confrontabile: "Diga di Pont'Ventoux (TO)" → "pontventoux".
func normNome(s string) string {
	s = reParentesi.ReplaceAllString(s, " ")
	s = accenti.Replace(strings.ToLower(s))
	s = strings.Trim(reNonAlnum.ReplaceAllString(s, "-"), "-")
	s = reArticolo.ReplaceAllString(s, "")
	return reIniziale.ReplaceAllString(s, "")
}

// trova cerca le dighe che corrispondono a un nome Coppermine: prima il nome intero,
// poi le sue parti ("Barcis - Ponte Antoi"). Più candidati si separano con stato e regione.
// Se nessuno corrisponde, prova il nome accorciato ("S. Caterina di Auronzo" → santa-caterina):
// quei candidati non sono certi e vanno confermati a mano.
func trova(nome string, percorso []string, chiavi map[string][]*entita) ([]string, bool) {
	var trovate []*entita
	aggiungi := func(k string) {
		for _, e := range chiavi[k] {
			if !slices.Contains(trovate, e) {
				trovate = append(trovate, e)
			}
		}
	}
	parti := reSeparati.Split(reParentesi.ReplaceAllString(nome, " "), -1)
	certo := true
	if aggiungi(normNome(nome)); len(trovate) == 0 {
		for _, parte := range parti {
			aggiungi(normNome(parte))
		}
	}
	if len(trovate) == 0 {
		certo = false
		for _, parte := range append(reMultipla.Split(nome, -1), append(parti, nome)...) {
			k := reLocalita.ReplaceAllString(normNome(parte), "")
			aggiungi(k)
			if r, ok := strings.CutPrefix(k, "s-"); ok {
				aggiungi("santa-" + r)
				aggiungi("san-" + r)
				aggiungi("santo-" + r)
			}
		}
	}
	if len(trovate) > 1 {
		stato, regione := luogo(percorso)
		var stessa []*entita
		for _, e := range trovate {
			if normLuogo(e.Geolocation.Stato) == stato && (regione == "" || normLuogo(e.Geolocation.Regione) == regione) {
				stessa = append(stessa, e)
			}
		}
		if len(stessa) > 0 {
			trovate = stessa
		}
	}
	s := make([]string, len(trovate))
	for i, e := range trovate {
		s[i] = e.Slug
	}
	slices.Sort(s)
	return s, certo
}

// luogo ricava stato e regione dal percorso Coppermine: Dighe > Italia > Piemonte > …
func luogo(p []string) (stato, regione string) {
	if len(p) > 1 {
		stato = normLuogo(strings.SplitN(p[1], " - ", 2)[0])
	}
	if stato == "italia" && len(p) > 2 {
		regione = normLuogo(p[2])
	}
	return
}

// normLuogo rende confrontabili "Friuli Venezia-Giulia" e "Friuli-Venezia Giulia".
func normLuogo(s string) string {
	return reNonAlnum.ReplaceAllString(accenti.Replace(strings.ToLower(s)), "")
}

func (r *rapportoMappature) markdown() []byte {
	var b bytes.Buffer
	p := func(f string, a ...any) { fmt.Fprintf(&b, f, a...) }
	p("## Mappature\n\nUna riga per radice di Coppermine. Gli abbinati in automatico comprendono gli ereditati da una categoria abbinata in automatico.\n\n")
	p("| Radice | Album | Foto | Automatici | Ereditati | Manuali | Ambigui | Da fare |\n|---|---:|---:|---:|---:|---:|---:|---:|\n")
	riga := func(c contoRadice) {
		p("| %s | %d | %d | %d (%.1f%%), %d foto (%.1f%%) | %d | %d | %d | %d, %d foto |\n", c.nome, c.album, c.foto,
			c.autoAlbum, perc(c.autoAlbum, c.album), c.autoFoto, perc(c.autoFoto, c.foto),
			c.eredAlbum, c.manAlbum, c.ambAlbum, c.faAlbum, c.faFoto)
	}
	ids := make([]int, 0, len(r.radici))
	for id := range r.radici {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	for _, id := range ids {
		riga(*r.radici[id])
	}
	riga(r.totale())
	p("\n### Tag interni\n\nFoto con un tag interno: allo Switch devono essere zero.\n\n")
	tags := make([]string, 0, len(r.interni))
	for t := range r.interni {
		tags = append(tags, t)
	}
	slices.Sort(tags)
	for _, t := range tags {
		p("- %s: %d\n", t, r.interni[t])
	}
	p("\n")
	if len(r.sconosciuti) > 0 {
		p("### Slug sconosciuti nelle righe manuali\n\n")
		for _, s := range r.sconosciuti {
			p("- %s\n", s)
		}
		p("\n")
	}
	p("### Ambigui\n\nPiù pagine con lo stesso nome nella stessa regione. Per sceglierne una, scrivi la riga in `arricchimento/mappature.yaml`.\n\n")
	for _, v := range r.ambigui {
		p("- %d (%d foto) %s → %s\n", v.aid, v.foto, v.percorso, strings.Join(v.candidati, ", "))
	}
	p("\n### Da fare\n\nIn ordine di Foto: i primi sono gli album principali. Dopo la freccia, i candidati da confermare.\n\n")
	for _, v := range r.daFare {
		p("- %d (%d foto) %s", v.aid, v.foto, v.percorso)
		if len(v.candidati) > 0 {
			p(" → %s", strings.Join(v.candidati, ", "))
		}
		p("\n")
	}
	return b.Bytes()
}
