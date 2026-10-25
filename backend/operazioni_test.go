package main

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestOperazioniEAnnulla(t *testing.T) {
	r := redazioneProva(t)
	admin := &utente{ID: 2, Nome: "admin"}
	mod := &utente{ID: 8, Nome: "mod"}
	cambia := func(pid int, campo, v string) {
		m := moduloDi(t, r, pid)
		m[campo] = v
		if _, err := r.salva(pid, m, admin); err != nil {
			t.Fatal(err)
		}
	}
	file4 := filepath.Join(r.contenuti, "arricchimento", "foto", "4.yaml")
	cambia(4, "title", "Primo")
	cambia(3, "fonte", "Archivio")
	ops, err := r.operazioni()
	if err != nil || len(ops) != 2 || ops[0].Campi != "fonte" || !slices.Equal(ops[0].Pid, []int{3}) || ops[1].Tipo != "Modifica" || ops[1].Autore != "admin" {
		t.Fatalf("operazioni: %+v, %v", ops, err)
	}
	// Annullare la prima: il file di 4 torna com'era (non c'era).
	if _, err := r.annulla(ops[1].Hash, mod); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(file4); !os.IsNotExist(err) {
		t.Errorf("4.yaml dopo l'annullamento: %v", err)
	}
	ops, _ = r.operazioni()
	if ops[0].Tipo != "Annullamento" || ops[0].Autore != "mod" || ops[2].AnnullataDa != ops[0].Hash || ops[2].Annullabile() {
		t.Errorf("dopo l'annullamento: %+v", ops)
	}
	if gitLog(t, r.contenuti, "%an <%ae>") != "mod <utente-8@gerbillo.invalid>" {
		t.Errorf("autore del revert: %s", gitLog(t, r.contenuti, "%an <%ae>"))
	}
	// Se una Foto è cambiata dopo, non si annulla e si dice quale.
	cambia(3, "fonte", "Altro archivio")
	ops, _ = r.operazioni()
	prima := ops[len(ops)-1] // fonte di 3: «Archivio»
	if dopo, err := r.annulla(ops[2].Hash, mod); !errors.Is(err, errCambiate) || !slices.Equal(dopo, []int{3}) {
		t.Errorf("annullamento con 3 cambiata dopo: %v, %v (%s)", dopo, err, prima.Campi)
	}
	if b, _ := os.ReadFile(filepath.Join(r.contenuti, "arricchimento", "foto", "3.yaml")); string(b) != "fonte: Altro archivio\n" {
		t.Errorf("3.yaml toccato:\n%s", b)
	}
}
