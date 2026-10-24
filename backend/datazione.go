package main

// Datazione (CONTEXT). L'Importazione la prende dall'EXIF (#18) e, solo per le Foto
// candidate, da user3 (#39): le Foto senza EXIF con user3 più di un anno prima della Data
// di caricamento, e le scansioni, con l'EXIF più di un anno dopo user3. Nelle scansioni la
// data di scansione si butta. L'interpretazione di user3 è codice di Migrazione: sparisce
// con il congelamento dello Switch (#40).

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

// La Datazione dell'EXIF è in ISO 8601 senza l'ora (#18). Si tollerano i separatori "-" e
// il mese o il giorno a una cifra; "6 Feb 2011", gli anni 0000 e le date impossibili si scartano.
var reDatazioneExif = regexp.MustCompile(`^(\d{4})[:-](\d{1,2})[:-](\d{1,2})(\s|$)`)

func datazioneExif(v string) (string, bool) {
	m := reDatazioneExif.FindStringSubmatch(v)
	if m == nil || m[1] == "0000" {
		return "", false
	}
	a, _ := strconv.Atoi(m[1])
	me, _ := strconv.Atoi(m[2])
	g, _ := strconv.Atoi(m[3])
	d, ok := dataGiorno(a, me, g)
	return d.edtf, ok
}

// Precisione di una Datazione letta da user3, per il rapporto.
const (
	precGiorno     = "giorno"
	precMese       = "mese"
	precAnno       = "anno"
	precDecennio   = "decennio"
	precIntervallo = "intervallo"
	precIncerto    = "incerto"
)

// dataUser3 è un valore di user3 interpretato: EDTF fino al livello 1.
type dataUser3 struct {
	edtf string
	fine time.Time // ultimo giorno del periodo: per un anno o un mese si confronta la fine
	prec string
}

func dataGiorno(a, m, g int) (dataUser3, bool) {
	t := time.Date(a, time.Month(m), g, 0, 0, 0, 0, time.UTC)
	if t.Year() != a || int(t.Month()) != m || t.Day() != g {
		return dataUser3{}, false // 29-02-1993, 32-08-2011
	}
	return dataUser3{t.Format("2006-01-02"), t, precGiorno}, true
}

func dataMese(a, m int) (dataUser3, bool) {
	if m < 1 || m > 12 {
		return dataUser3{}, false
	}
	t := time.Date(a, time.Month(m), 1, 0, 0, 0, 0, time.UTC)
	return dataUser3{t.Format("2006-01"), t.AddDate(0, 1, -1), precMese}, true
}

func dataAnno(a int) dataUser3 {
	return dataUser3{fmt.Sprintf("%04d", a), time.Date(a, 12, 31, 0, 0, 0, 0, time.UTC), precAnno}
}

var mesi = map[string]int{
	"gennaio": 1, "febbraio": 2, "marzo": 3, "aprile": 4, "maggio": 5, "giugno": 6, "luglio": 7,
	"agosto": 8, "settembre": 9, "ottobre": 10, "novembre": 11, "dicembre": 12,
	"gen": 1, "feb": 2, "mar": 3, "apr": 4, "mag": 5, "giu": 6, "lug": 7, "ago": 8, "set": 9, "ott": 10, "nov": 11, "dic": 12,
	// Il mese in numeri romani: IV/2005, IX.2005.
	"i": 1, "ii": 2, "iii": 3, "iv": 4, "v": 5, "vi": 6, "vii": 7, "viii": 8, "ix": 9, "x": 10, "xi": 11, "xii": 12,
}

var nomiMesi = [...]string{"gennaio", "febbraio", "marzo", "aprile", "maggio", "giugno", "luglio",
	"agosto", "settembre", "ottobre", "novembre", "dicembre"}

// reMese è l'alternativa dei nomi dei mesi, i più lunghi prima.
var reMese = func() string {
	n := make([]string, 0, len(mesi))
	for k := range mesi {
		n = append(n, k)
	}
	slices.SortFunc(n, func(a, b string) int { return len(b) - len(a) })
	return strings.Join(n, "|")
}()

var (
	reU3Giorno     = regexp.MustCompile(`^(\d{1,2})\s*[/.-]\s*(\d{1,2})\s*[/.-]\s*(\d{4}|\d{2})$`)
	reU3ISO        = regexp.MustCompile(`^(\d{4})-(\d{1,2})-(\d{1,2})$`)
	reU3GiornoNome = regexp.MustCompile(`^(\d{1,2})\s+(` + reMese + `)\s+(\d{4}|\d{2})$`)
	reU3MeseNome   = regexp.MustCompile(`^(` + reMese + `)[\s/.-]+(\d{4}|\d{2})$`)
	reU3MeseNum    = regexp.MustCompile(`^(\d{1,2})\s*[/.-]\s*(\d{4})$`)
	reU3Mesi       = regexp.MustCompile(`^(` + reMese + `)\s*[/-]\s*(` + reMese + `)\s+(\d{4})$`)
	reU3Anno       = regexp.MustCompile(`^(\d{4})$`)
	reU3Incerto    = regexp.MustCompile(`^(\d{4})\s*\?+$|^(?:circa|ca\.?|c\.)\s*(\d{4})$|^(\d{4})\s*(?:circa|ca\.?)$`)
	reU3Intervallo = regexp.MustCompile(`^(\d{4})\s*[-/]\s*(\d{4})$`)
	reU3Decennio   = regexp.MustCompile(`^anni\s*'?\s*(\d0)$|^anni\s+(\d{3}0)$`)
	reU3UnAnno     = regexp.MustCompile(`(?:^|\D)(1[89]\d\d|20\d\d)(?:\D|$)`)
)

// interpretaUser3 legge user3 con regole fisse (#39). Il giorno/mese si legge sempre così;
// un anno a due cifre è 20xx se non supera l'anno del caricamento, altrimenti 19xx.
func interpretaUser3(s string, caricamento int) (dataUser3, bool) {
	d, ok := regoleUser3(s, caricamento)
	if ok && d.fine.Year() < 1800 { // "26 agosto 1011"
		return dataUser3{}, false
	}
	return d, ok
}

func regoleUser3(s string, caricamento int) (dataUser3, bool) {
	s = strings.Join(strings.Fields(strings.ReplaceAll(strings.ToLower(s), "’", "'")), " ")
	anno := func(y string) int {
		a, _ := strconv.Atoi(y)
		if len(y) == 2 {
			if a += 2000; a > caricamento {
				a -= 100
			}
		}
		return a
	}
	n := func(x string) int { v, _ := strconv.Atoi(x); return v }
	if m := reU3Giorno.FindStringSubmatch(s); m != nil {
		return dataGiorno(anno(m[3]), n(m[2]), n(m[1]))
	}
	if m := reU3ISO.FindStringSubmatch(s); m != nil {
		return dataGiorno(n(m[1]), n(m[2]), n(m[3]))
	}
	if m := reU3GiornoNome.FindStringSubmatch(s); m != nil {
		return dataGiorno(anno(m[3]), mesi[m[2]], n(m[1]))
	}
	if m := reU3MeseNome.FindStringSubmatch(s); m != nil {
		return dataMese(anno(m[2]), mesi[m[1]])
	}
	if m := reU3MeseNum.FindStringSubmatch(s); m != nil {
		return dataMese(n(m[2]), n(m[1]))
	}
	if m := reU3Mesi.FindStringSubmatch(s); m != nil {
		a, b := mesi[m[1]], mesi[m[2]]
		if a >= b {
			return dataUser3{}, false
		}
		inizio, _ := dataMese(n(m[3]), a)
		fine, _ := dataMese(n(m[3]), b)
		return dataUser3{inizio.edtf + "/" + fine.edtf, fine.fine, precIntervallo}, true
	}
	if m := reU3Anno.FindStringSubmatch(s); m != nil {
		return dataAnno(n(m[1])), true
	}
	if m := reU3Incerto.FindStringSubmatch(s); m != nil {
		d := dataAnno(n(m[1] + m[2] + m[3]))
		d.edtf += "?"
		d.prec = precIncerto
		return d, true
	}
	if m := reU3Intervallo.FindStringSubmatch(s); m != nil {
		a, b := n(m[1]), n(m[2])
		if a >= b {
			return dataUser3{}, false
		}
		return dataUser3{fmt.Sprintf("%04d/%04d", a, b), dataAnno(b).fine, precIntervallo}, true
	}
	if m := reU3Decennio.FindStringSubmatch(s); m != nil {
		a := anno(m[1] + m[2])
		return dataUser3{fmt.Sprintf("%03dX", a/10), dataAnno(a + 9).fine, precDecennio}, true
	}
	return dataUser3{}, false
}

// Tipi di Foto candidate (#39).
const (
	candStorica   = "storica"   // senza EXIF, user3 più di un anno prima del caricamento
	candScansione = "scansione" // EXIF più di un anno dopo user3
)

// candidata dice se la Foto prende la Datazione da user3. exif è la Datazione dall'EXIF,
// vuota se non c'è. Un user3 senza regola che contiene un anno è comunque candidato, ma
// resta da sistemare a mano: d è vuota e interpretata è falso.
func candidata(user3, exif string, caricata time.Time) (tipo string, d dataUser3, interpretata bool) {
	d, interpretata = interpretaUser3(user3, caricata.Year())
	fine := d.fine
	if !interpretata {
		var ultimo int
		for _, m := range reU3UnAnno.FindAllStringSubmatch(user3, -1) {
			if a, _ := strconv.Atoi(m[1]); a > ultimo {
				ultimo = a
			}
		}
		if ultimo == 0 {
			return "", d, false
		}
		fine = dataAnno(ultimo).fine
	}
	if exif == "" {
		if fine.Before(caricata.AddDate(-1, 0, 0)) {
			tipo = candStorica
		}
	} else if e, err := time.Parse("2006-01-02", exif); err == nil && e.After(fine.AddDate(1, 0, 0)) {
		tipo = candScansione
	}
	return tipo, d, interpretata
}

var (
	reEDTFGiorno   = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	reEDTFMese     = regexp.MustCompile(`^(\d{4})-(\d{2})$`)
	reEDTFAnno     = regexp.MustCompile(`^\d{4}$`)
	reEDTFDecennio = regexp.MustCompile(`^(\d{3})X$`)
	reEDTFIncerto  = regexp.MustCompile(`^(\d{4})[?~]$`)
)

// datazioneItaliana mostra una Datazione EDTF in italiano: 18/07/2004, maggio 2006, 1963,
// anni '60, 1950 circa, 1950–1960. Un valore che non riconosce passa così com'è.
func datazioneItaliana(edtf string) string {
	if a, b, ok := strings.Cut(edtf, "/"); ok {
		ia, ib := datazioneItaliana(a), datazioneItaliana(b)
		if ia == a && !reEDTFAnno.MatchString(a) || ib == b && !reEDTFAnno.MatchString(b) {
			return edtf
		}
		return ia + "–" + ib
	}
	switch {
	case reEDTFGiorno.MatchString(edtf):
		if t, err := time.Parse("2006-01-02", edtf); err == nil {
			return t.Format("02/01/2006")
		}
	case reEDTFMese.MatchString(edtf):
		m := reEDTFMese.FindStringSubmatch(edtf)
		if me, _ := strconv.Atoi(m[2]); me >= 1 && me <= 12 {
			return nomiMesi[me-1] + " " + m[1]
		}
	case reEDTFDecennio.MatchString(edtf):
		d := reEDTFDecennio.FindStringSubmatch(edtf)[1]
		if d[:2] == "19" {
			return "anni '" + d[2:] + "0"
		}
		return "anni " + d + "0"
	case reEDTFIncerto.MatchString(edtf):
		return edtf[:4] + " circa"
	}
	return edtf
}

// datazioneTesto è la Datazione come si mostra nella pagina della Foto e nell'Indice di
// ricerca: in italiano se c'è, altrimenti user3 di Coppermine così com'è (#18).
func (f *fotoPubblicata) datazioneTesto() string {
	if f.Datazione == "" {
		return f.Coppermine.Data
	}
	return datazioneItaliana(f.Datazione)
}
