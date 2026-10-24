package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDecodifica(t *testing.T) {
	casi := []struct {
		in    []byte
		out   string
		esito esitoTesto
	}{
		{[]byte("Diga di Pont&#39;Ventoux &amp; centrale"), "Diga di Pont'Ventoux & centrale", inUTF8},
		{[]byte("L&#039;invaso &quot;pieno&quot;"), `L'invaso "pieno"`, inUTF8},
		{[]byte("Città"), "Città", inUTF8},
		{[]byte("Citt\xe0 \x93nuova\x94 \x80"), "Città “nuova” €", inCP1252},
		{[]byte("byte \x81 non definito"), "byte � non definito", nonDecodificato},
		{[]byte(" riga\r\naltra \n"), "riga\naltra", inUTF8},
	}
	for _, c := range casi {
		s, e := decodifica(c.in)
		if s != c.out || e != c.esito {
			t.Errorf("decodifica(%q) = %q, %d; atteso %q, %d", c.in, s, e, c.out, c.esito)
		}
		if mojibake(s) {
			t.Errorf("mojibake in %q", s)
		}
	}
	if !mojibake("CittÃ ") || !mojibake("â€œ") {
		t.Error("mojibake non riconosciuto")
	}
}

func TestDataExif(t *testing.T) {
	b := []byte(`a:3:{s:5:"Model";s:3:"X10";s:16:"DateTimeOriginal";s:19:"2005:06:03 11:31:56";s:12:"SerialNumber";s:4:"1234";}`)
	if d := dataExif(b); d != "2005:06:03 11:31:56" {
		t.Errorf("dataExif = %q", d)
	}
	if d := dataExif([]byte("a:0:{}")); d != "" {
		t.Errorf("dataExif vuoto = %q", d)
	}
}

func TestDatazioneExif(t *testing.T) {
	for in, out := range map[string]string{
		"2005:06:03 11:31:56": "2005-06-03",
		"2012-07-01 11:53:37": "2012-07-01",
		"2011:6:20 22:33:36":  "2011-06-20",
		"2010:03:28 10:11: 9": "2010-03-28",
		"0000:00:00 00:00:00": "", // anno 0000
		"2005:02:30 10:00:00": "", // data impossibile
		"2005:00:00 00:00:00": "",
		"6 Feb 2011 17:13:36": "", // malformato
		"20050603":            "",
		"":                    "",
	} {
		got, ok := datazioneExif(in)
		if got != out || ok != (out != "") {
			t.Errorf("datazioneExif(%q) = %q, %v; atteso %q", in, got, ok, out)
		}
	}
}

func TestConverti(t *testing.T) {
	d := &datiCoppermine{
		foto: []rigaFoto{
			{pid: 1, aid: 3, owner: 25, ctime: 1167214711, larghezza: 800, altezza: 600,
				filepath: []byte("userpics/10025/"), filename: []byte("a.JPG"),
				titolo: []byte("Citt\xe0"), user1: []byte("Mario Rossi")},
			{pid: 2, owner: 0, filename: []byte("b.zip")},
			{pid: 3, owner: 1, filename: []byte("c.pdf")},
			{pid: 4, owner: 999, filename: []byte("d.png")},
			{pid: 5, aid: 390, owner: 0, filename: []byte("e.jpg"), user3: []byte("18/07/2004")},
			// Candidate (#39): storica e scansione, caricate il 27/12/2006.
			{pid: 6, ctime: 1167214711, filename: []byte("f.jpg"), user3: []byte("anni '60")},
			{pid: 7, ctime: 1167214711, filename: []byte("g.jpg"), user3: []byte("1985")},
			{pid: 8, ctime: 1167214711, filename: []byte("h.jpg"), user3: []byte("Inverno 1970")},
		},
		exif: map[int]string{1: "2005:06:03 11:31:56", 2: "0000:00:00 00:00:00", 3: "6 Feb 2011 17:13:36",
			7: "2005:01:01 10:00:00"},
		utenti:    map[int]string{25: "utente", 1: "Anonymous"},
		riservati: map[int]bool{390: true},
	}
	foto, rap := converti(d)
	atteso := `altezza: 600
authors:
    - Mario Rossi
contributore:
    id: 25
    nome: utente
coppermine:
    aid: 3
    data_exif: 2005:06:03 11:31:56
    filename: a.JPG
    filepath: userpics/10025/
datazione: "2005-06-03"
date: "2006-12-27T11:18:31+01:00"
formato: immagine
larghezza: 800
pid: 1
title: Città
`
	if got := string(foto[0].file); got != atteso {
		t.Errorf("YAML:\n%s\natteso:\n%s", got, atteso)
	}
	for _, f := range foto[1:] {
		if strings.Contains(string(f.file), "contributore") || strings.Contains(string(f.file), "authors") {
			t.Errorf("pid %d: contributore o autore non attesi:\n%s", f.pid, f.file)
		}
	}
	if !strings.Contains(string(foto[4].file), "riservata: true") || strings.Contains(string(foto[0].file), "riservata") || rap.riservate != 1 {
		t.Errorf("riservata: %d\n%s", rap.riservate, foto[4].file)
	}
	if rap.ospiti != 6 || rap.cancellati[999] != 1 || rap.senzaAutore != 7 {
		t.Errorf("rapporto: ospiti %d, cancellati %v, senza autore %d", rap.ospiti, rap.cancellati, rap.senzaAutore)
	}
	for _, f := range foto[1:5] {
		if strings.Contains(string(f.file), "datazione") {
			t.Errorf("pid %d: Datazione senza EXIF valido:\n%s", f.pid, f.file)
		}
	}
	if !strings.Contains(string(foto[4].file), "data: 18/07/2004") {
		t.Errorf("user3 non conservato grezzo:\n%s", foto[4].file)
	}
	if !strings.Contains(string(foto[5].file), "datazione: 196X") || !strings.Contains(string(foto[6].file), "datazione: \"1985\"") ||
		strings.Contains(string(foto[7].file), "datazione") {
		t.Errorf("Datazione da user3:\n%s\n%s\n%s", foto[5].file, foto[6].file, foto[7].file)
	}
	if !strings.Contains(string(foto[6].file), "data_exif: 2005:01:01 10:00:00") {
		t.Errorf("EXIF della scansione non conservato grezzo:\n%s", foto[6].file)
	}
	if rap.candidate[candStorica][precDecennio] != 1 || rap.candidate[candScansione][precAnno] != 1 ||
		rap.candidate[candStorica]["residui"] != 1 || len(rap.residui) != 1 || len(rap.scansioni) != 1 {
		t.Errorf("candidate %v, residui %v, scansioni %v", rap.candidate, rap.residui, rap.scansioni)
	}
	if rap.datazioneExif != 1 || rap.soloUser3 != 1 || rap.senzaData != 3 || len(rap.exifScartati) != 2 {
		t.Errorf("Datazione nel rapporto: %d dall'EXIF, %d solo user3, %d senza, scartati %v",
			rap.datazioneExif, rap.soloUser3, rap.senzaData, rap.exifScartati)
	}
	if rap.formati["altro"] != 1 || rap.formati["pdf"] != 1 || rap.formati["immagine"] != 6 {
		t.Errorf("formati %v", rap.formati)
	}
	if len(rap.cp1252) != 1 || rap.cp1252[0] != 1 {
		t.Errorf("cp1252 %v", rap.cp1252)
	}
}

func TestScrivi(t *testing.T) {
	dir := t.TempDir()
	cart := filepath.Join(dir, "importazione", "foto")
	os.MkdirAll(cart, 0o755)
	os.WriteFile(filepath.Join(cart, "1.yaml"), []byte("pid: 1\n"), 0o644)
	os.WriteFile(filepath.Join(cart, "7.yaml"), []byte("pid: 7\n"), 0o644)
	rap := &rapporto{formati: map[string]int{}, cancellati: map[int]int{},
		autori: map[string]int{"Mario Rossi": 3, "Rossi M.": 1, "-": 1, "A | B": 1},
		alias:  aliasAutori{"-": {}, "Rossi M.": {"Mario Rossi"}}}
	rap.mappature = &rapportoMappature{radici: map[int]*contoRadice{}, interni: map[string]int{daCollegare: 5},
		daFare: []voceRapporto{{aid: 21, foto: 5, percorso: "Dighe - Dams / Sconosciuta"}}}
	foto := []fotoYAML{{1, []byte("pid: 1\n")}, {2, []byte("pid: 2\n")}}
	if err := scrivi(dir, foto, rap); err != nil {
		t.Fatal(err)
	}
	if rap.scritte != 1 || len(rap.spariti) != 1 || rap.spariti[0] != 7 {
		t.Errorf("scritte %d, spariti %v", rap.scritte, rap.spariti)
	}
	if _, err := os.Stat(filepath.Join(cart, "7.yaml")); !os.IsNotExist(err) {
		t.Error("7.yaml non rimosso")
	}
	md, err := os.ReadFile(filepath.Join(dir, "importazione", "anomalie.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range []string{
		"Valori distinti di `user1`: 4, di cui 2 con alias",
		"| Mario Rossi | 3 |  |\n| - | 1 | scartato |\n| A \\| B | 1 |  |\n| Rossi M. | 1 | → Mario Rossi |",
		"## Mappature", "- da-collegare: 5", "- 21 (5 foto) Dighe - Dams / Sconosciuta",
	} {
		if !strings.Contains(string(md), r) {
			t.Errorf("anomalie.md senza %q:\n%s", r, md)
		}
	}
}
