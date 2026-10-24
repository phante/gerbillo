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

func TestConverti(t *testing.T) {
	d := &datiCoppermine{
		foto: []rigaFoto{
			{pid: 1, aid: 3, owner: 25, ctime: 1167214711, larghezza: 800, altezza: 600,
				filepath: []byte("userpics/10025/"), filename: []byte("a.JPG"),
				titolo: []byte("Citt\xe0"), user1: []byte("Mario Rossi")},
			{pid: 2, owner: 0, filename: []byte("b.zip")},
			{pid: 3, owner: 1, filename: []byte("c.pdf")},
			{pid: 4, owner: 999, filename: []byte("d.png")},
		},
		exif:   map[int]string{1: "2005:06:03 11:31:56"},
		utenti: map[int]string{25: "utente", 1: "Anonymous"},
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
	if rap.ospiti != 2 || rap.cancellati[999] != 1 || rap.senzaAutore != 3 {
		t.Errorf("rapporto: ospiti %d, cancellati %v, senza autore %d", rap.ospiti, rap.cancellati, rap.senzaAutore)
	}
	if rap.formati["altro"] != 1 || rap.formati["pdf"] != 1 || rap.formati["immagine"] != 2 {
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
	os.MkdirAll(filepath.Join(dir, "arricchimento"), 0o755)
	os.WriteFile(filepath.Join(dir, "arricchimento", "autori.yaml"), []byte("\"-\": []\nRossi M.: [Mario Rossi]\n"), 0o644)
	rap := &rapporto{formati: map[string]int{}, cancellati: map[int]int{},
		autori: map[string]int{"Mario Rossi": 3, "Rossi M.": 1, "-": 1, "A | B": 1}}
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
	} {
		if !strings.Contains(string(md), r) {
			t.Errorf("anomalie.md senza %q:\n%s", r, md)
		}
	}
}
