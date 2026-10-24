package main

import (
	"testing"
	"time"
)

func TestInterpretaUser3(t *testing.T) {
	casi := []struct {
		in          string
		caricamento int
		edtf, prec  string
	}{
		{"25/08/2011", 2015, "2011-08-25", precGiorno},
		{"30-09-2013", 2015, "2013-09-30", precGiorno},
		{"28.03.2010", 2015, "2010-03-28", precGiorno},
		{"2008-3-24", 2015, "2008-03-24", precGiorno},
		{"10 Febbraio 2006", 2015, "2006-02-10", precGiorno},
		{"03/04/2005", 2015, "2005-04-03", precGiorno}, // sempre giorno/mese
		{"maggio 2006", 2015, "2006-05", precMese},
		{"Ottobre  2006", 2015, "2006-10", precMese},
		{"06/2004", 2015, "2004-06", precMese},
		{"IV/2005", 2015, "2005-04", precMese},
		{"IX.2005", 2015, "2005-09", precMese},
		{"1963", 2015, "1963", precAnno},
		// Anno a due cifre: 20xx se non supera l'anno del caricamento, altrimenti 19xx.
		{"apr-06", 2007, "2006-04", precMese},
		{"18/07/04", 2007, "2004-07-18", precGiorno},
		{"18/07/07", 2007, "2007-07-18", precGiorno},
		{"18/07/08", 2007, "1908-07-18", precGiorno},
		{"1950-1960", 2015, "1950/1960", precIntervallo},
		{"1980/1981", 2015, "1980/1981", precIntervallo},
		{"maggio / giugno 2016", 2018, "2016-05/2016-06", precIntervallo},
		{"anni '60", 2015, "196X", precDecennio},
		{"Anni 1950", 2015, "195X", precDecennio},
		{"1950?", 2015, "1950?", precIncerto},
		{"circa 1950", 2015, "1950?", precIncerto},
	}
	for _, c := range casi {
		d, ok := interpretaUser3(c.in, c.caricamento)
		if !ok || d.edtf != c.edtf || d.prec != c.prec {
			t.Errorf("interpretaUser3(%q, %d) = %+v, %v; atteso %s (%s)", c.in, c.caricamento, d, ok, c.edtf, c.prec)
		}
	}
	// Date impossibili e valori senza data: nessuna Datazione.
	for _, in := range []string{"29-02-1993", "32-08-2011", "13/2005", "-", "tempo fa....", "Barcis", "Inverno 2012-13", "1960-1950", "26 agosto 1011"} {
		if d, ok := interpretaUser3(in, 2015); ok {
			t.Errorf("interpretaUser3(%q) = %+v, atteso nessuna regola", in, d)
		}
	}
	if d, _ := interpretaUser3("maggio 2006", 2015); !d.fine.Equal(time.Date(2006, 5, 31, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("fine del mese %v", d.fine)
	}
}

func TestCandidata(t *testing.T) {
	giorno := func(s string) time.Time { d, _ := time.Parse("2006-01-02", s); return d }
	casi := []struct {
		nome, user3, exif, caricata string
		tipo, edtf                  string
		interpretata                bool
	}{
		{"storica", "Aprile 1962", "", "2020-10-29", candStorica, "1962-04", true},
		{"recente: non candidata", "18/07/2004", "", "2005-03-01", "", "2004-07-18", true},
		// Per un anno si confronta la fine: 2019 finisce meno di un anno prima del caricamento.
		{"anno a cavallo", "2019", "", "2020-06-01", "", "2019", true},
		{"anno di due anni prima", "2018", "", "2020-06-01", candStorica, "2018", true},
		{"scansione", "1985", "2010-05-01", "2011-01-01", candScansione, "1985", true},
		{"EXIF concorde: non candidata", "18/07/2004", "2004-07-18", "2010-01-01", "", "2004-07-18", true},
		{"data impossibile: residuo", "29-02-1993", "", "2010-01-01", candStorica, "", false},
		{"senza regola: residuo", "Foto eseguita nel 1994", "", "2008-09-26", candStorica, "", false},
		{"senza anno", "tempo fa....", "", "2008-09-26", "", "", false},
	}
	for _, c := range casi {
		tipo, d, ok := candidata(c.user3, c.exif, giorno(c.caricata))
		if tipo != c.tipo || d.edtf != c.edtf || ok != c.interpretata {
			t.Errorf("%s: candidata(%q, %q, %s) = %q, %q, %v", c.nome, c.user3, c.exif, c.caricata, tipo, d.edtf, ok)
		}
	}
}

func TestDatazioneItaliana(t *testing.T) {
	for in, out := range map[string]string{
		"2004-07-18":      "18/07/2004",
		"2006-05":         "maggio 2006",
		"1963":            "1963",
		"196X":            "anni '60",
		"201X":            "anni 2010",
		"1950?":           "1950 circa",
		"1950~":           "1950 circa",
		"1950/1960":       "1950–1960",
		"2016-05/2016-06": "maggio 2016–giugno 2016",
		"Inverno 2012-13": "Inverno 2012-13",
		"2006-13":         "2006-13",
	} {
		if got := datazioneItaliana(in); got != out {
			t.Errorf("datazioneItaliana(%q) = %q, atteso %q", in, got, out)
		}
	}
}
