package main

// Mappatura degli Album di Coppermine verso Slug, categories e tags (G2, #10, #1). Le proposte
// automatiche vanno in arricchimento/mappature.yaml nel Repository dei contenuti, accanto alle
// righe scritte a mano, che non vengono mai toccate.

import (
	"bytes"
	"cmp"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/BurntSushi/toml"
	"gopkg.in/yaml.v3"
)

const (
	cidUtenti    = 1 // User galleries
	cidDighe     = 2
	cidCentrali  = 3
	cidPersonali = 10000 // gli album delle User galleries hanno categoria 10000 + owner
)

const (
	automatica = "automatica" // proposta dal confronto dei nomi
	manuale    = "manuale"    // scritta o confermata a mano: mai riscritta
	daFare     = "da-fare"    // nessun abbinamento certo
)

// daCollegare è il tag interno delle righe da-fare: manca lo Slug, lo si sceglie dal Backend.
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

// Le chiavi sono in ordine alfabetico, come nei YAML delle Foto.
type mappatura struct {
	Candidati  []string `yaml:"candidati,omitempty"`
	Categories []string `yaml:"categories,omitempty"`
	Coppermine string   `yaml:"coppermine"` // percorso dell'album, solo per orientarsi
	Origine    string   `yaml:"origine"`
	Slug       []string `yaml:"slug,omitempty"`
	Tags       []string `yaml:"tags,omitempty"`
}

type fileMappature struct {
	Album     map[int]*mappatura `yaml:"album"`
	Categorie map[int]*mappatura `yaml:"categorie"`
}

const intestazioneMappature = `# Mappature degli Album di Coppermine (G2, #10; ADR 0009, 0013).
# Le righe "origine: automatica" e "da-fare" le riscrive gerbillo abbina.
# Per correggerne o completarne una, scrivi "origine: manuale": da lì non viene più toccata.
# Un album senza riga eredita la Mappatura della sua categoria. I commenti non si conservano.

`

func abbina(args []string) error {
	fs := flag.NewFlagSet("abbina", flag.ExitOnError)
	conf := fs.String("config", "gerbillo.toml", "file di configurazione TOML")
	indice := fs.String("indice", "", "Indice delle Entità (JSON)")
	manuali := fs.String("manuali", "", "righe manuali da aggiungere (es. dati/mappature-manuali.yaml): vincono su quelle esistenti")
	pVoc := fs.String("vocabolario", "", "vocabolario controllato (dati/vocabolario.yaml)")
	fs.Parse(args)
	var c configImporta
	if _, err := toml.DecodeFile(*conf, &c); err != nil {
		return err
	}
	if c.Contenuti == "" || *indice == "" || *pVoc == "" {
		return errors.New("servono contenuti nella configurazione, -indice e -vocabolario")
	}
	var voc vocabolario
	if b, err := os.ReadFile(*pVoc); err != nil {
		return err
	} else if err := yaml.Unmarshal(b, &voc); err != nil {
		return fmt.Errorf("%s: %w", *pVoc, err)
	}
	idx, err := leggiIndice(*indice)
	if err != nil {
		return err
	}
	b, err := os.ReadFile(filepath.Join(c.Contenuti, "importazione", "album.yaml"))
	if err != nil {
		return fmt.Errorf("manca l'Importazione: %w", err)
	}
	var cpg albumCoppermine
	if err := yaml.Unmarshal(b, &cpg); err != nil {
		return err
	}
	p := filepath.Join(c.Contenuti, "arricchimento", "mappature.yaml")
	var vecchie fileMappature
	if b, err := os.ReadFile(p); err == nil {
		if err := yaml.Unmarshal(b, &vecchie); err != nil {
			return fmt.Errorf("%s: %w", p, err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if *manuali != "" {
		b, err := os.ReadFile(*manuali)
		if err != nil {
			return err
		}
		var m fileMappature
		if err := yaml.Unmarshal(b, &m); err != nil {
			return fmt.Errorf("%s: %w", *manuali, err)
		}
		vecchie.Album, vecchie.Categorie = unisciManuali(vecchie.Album, m.Album), unisciManuali(vecchie.Categorie, m.Categorie)
	}
	if fuori := fuoriVocabolario(vecchie, voc); len(fuori) > 0 {
		return fmt.Errorf("categories fuori dal vocabolario, niente scritto:\n%s", strings.Join(fuori, "\n"))
	}
	nuove, rap := mappature(cpg, idx, vecchie, voc)
	out, err := yaml.Marshal(nuove)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(p, append([]byte(intestazioneMappature), out...), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(c.Contenuti, "arricchimento", "rapporto-mappature.md"), rap.markdown(), 0o644); err != nil {
		return err
	}
	t := rap.totale()
	fmt.Printf("album: %d, abbinati in automatico %.1f%% (foto %.1f%%), da collegare %d (foto %d)\n",
		t.album, perc(t.autoAlbum, t.album), perc(t.autoFoto, t.foto), t.faAlbum+t.ambAlbum, rap.interni[daCollegare])
	return registra(c.Contenuti, "arricchimento", fmt.Sprintf("mappature: %d album, %d da fare", t.album, t.faAlbum+t.ambAlbum))
}

func unisciManuali(dst, src map[int]*mappatura) map[int]*mappatura {
	if dst == nil {
		dst = map[int]*mappatura{}
	}
	for k, v := range src {
		if v.Origine == manuale {
			dst[k] = v
		}
	}
	return dst
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

// mappature produce le Mappature: le righe manuali restano, le altre si rigenerano.
func mappature(cpg albumCoppermine, idx []entita, vecchie fileMappature, voc vocabolario) (fileMappature, *rapportoMappature) {
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
	out := fileMappature{Album: map[int]*mappatura{}, Categorie: map[int]*mappatura{}}
	rap := &rapportoMappature{radici: map[int]*contoRadice{}, interni: map[string]int{}}
	controlla := func(dove string, m *mappatura) {
		for _, s := range m.Slug {
			if !noti[s] {
				rap.sconosciuti = append(rap.sconosciuti, dove+": "+s)
			}
		}
	}

	for cid, c := range cpg.Categorie {
		p := percorso(cid)
		desc := strings.Join(p, " / ")
		if v := vecchie.Categorie[cid]; v != nil && v.Origine == manuale {
			v.Coppermine = desc
			out.Categorie[cid] = v
			controlla(fmt.Sprintf("categoria %d", cid), v)
			continue
		}
		// In automatico solo dal terzo livello sotto Dighe e Centrali (Dighe > Italia > regione > diga),
		// perché stati e regioni non sono dighe.
		r := radice(cid)
		if sezioniRadice[r] == nil || len(p) < 4 {
			continue
		}
		if s, certo := trova(c.Nome, p, chiavi[r]); certo && len(s) == 1 {
			out.Categorie[cid] = &mappatura{Coppermine: desc, Origine: automatica, Slug: s}
		}
	}

	// ereditata è la Mappatura della categoria antenata più vicina che ha uno Slug, delle categories o dei tags.
	ereditata := func(cid int) *mappatura {
		for i := 0; cid != 0 && i < 10; i++ {
			if m := out.Categorie[cid]; m != nil && (len(m.Slug) > 0 || len(m.Categories) > 0 || len(m.Tags) > 0) {
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
		desc := strings.Join(append(p, a.Title), " / ")
		cont.album++
		cont.foto += a.Foto
		effettiva := func(m *mappatura) {
			for _, t := range m.Tags {
				if slices.Contains(voc.TagsInterni, t) {
					rap.interni[t] += a.Foto
				}
			}
		}
		if v := vecchie.Album[aid]; v != nil && v.Origine == manuale {
			v.Coppermine = desc
			out.Album[aid] = v
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
			m := &mappatura{Coppermine: desc, Origine: automatica, Slug: s}
			if ered != nil {
				m.Categories, m.Tags = slices.Clone(ered.Categories), slices.Clone(ered.Tags)
			}
			out.Album[aid] = m
			cont.autoAlbum++
			cont.autoFoto += a.Foto
			effettiva(m)
		case ered != nil:
			// Sottoalbum di una diga ("La Forra" sotto "Vajont"): eredita, nessuna riga.
			cont.eredAlbum++
			cont.eredFoto += a.Foto
			if ered.Origine == automatica {
				cont.autoAlbum++
				cont.autoFoto += a.Foto
			}
			effettiva(ered)
		default:
			m := &mappatura{Coppermine: desc, Origine: daFare, Candidati: s, Tags: []string{daCollegare}}
			out.Album[aid] = m
			effettiva(m)
			v := voceRapporto{aid, a.Foto, desc, s}
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
	p("# Mappature\n\nUna riga per radice di Coppermine. Gli abbinati in automatico comprendono gli ereditati da una categoria abbinata in automatico.\n\n")
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
	p("\n## Tag interni\n\nFoto con un tag interno: allo Switch devono essere zero.\n\n")
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
		p("## Slug sconosciuti nelle righe manuali\n\n")
		for _, s := range r.sconosciuti {
			p("- %s\n", s)
		}
		p("\n")
	}
	p("## Ambigui\n\nPiù pagine con lo stesso nome nella stessa regione.\n\n")
	for _, v := range r.ambigui {
		p("- %d (%d foto) %s → %s\n", v.aid, v.foto, v.percorso, strings.Join(v.candidati, ", "))
	}
	p("\n## Da fare\n\nIn ordine di Foto: i primi sono gli album principali. Dopo la freccia, i candidati da confermare.\n\n")
	for _, v := range r.daFare {
		p("- %d (%d foto) %s", v.aid, v.foto, v.percorso)
		if len(v.candidati) > 0 {
			p(" → %s", strings.Join(v.candidati, ", "))
		}
		p("\n")
	}
	return b.Bytes()
}
