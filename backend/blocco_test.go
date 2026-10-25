package main

import (
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestModificaInBlocco(t *testing.T) {
	r := redazioneConIndice(t)
	admin := &utente{ID: 2, Nome: "admin"}
	v := url.Values{"togli.tags": {"da-collegare"}, "aggiungi.slug": {"dighe/vajont"}, "modo.fonte": {"imposta"}, "valore.fonte": {"Archivio"}}
	m := leggiModificaBlocco(v)
	// 4 e 5 sono da collegare, 1 ha già lo slug ma non la Fonte.
	e, err := r.calcolaBlocco([]int{4, 5, 1}, m)
	if err != nil || len(e.file) != 3 || len(e.aPosto) != 0 || e.perCampo["slug"] != 2 || e.perCampo["fonte"] != 3 || e.perCampo["tags"] != 2 {
		t.Fatalf("anteprima: %+v, %v", e, err)
	}
	prima := gitLog(t, r.contenuti, "%H")
	e, id, err := r.applicaBlocco([]int{4, 5, 1}, m, admin)
	if err != nil || id == "" {
		t.Fatal(err)
	}
	if n, _ := r.git("rev-list", "--count", prima+"..HEAD"); strings.TrimSpace(n) != "1" {
		t.Errorf("%s commit, atteso uno", n)
	}
	if b := gitLog(t, r.contenuti, "%s|%an|%(trailers:key=Operazione,valueonly)"); b != "blocco: 3 Foto: slug, tags, fonte|admin|"+id {
		t.Errorf("commit: %q", b)
	}
	// Ogni file ha solo ciò che differisce dall'Importazione.
	b, _ := os.ReadFile(filepath.Join(r.contenuti, "arricchimento", "foto", "4.yaml"))
	if string(b) != "fonte: Archivio\nslug:\n    - dighe/vajont\ntags: []\n" {
		t.Errorf("4.yaml:\n%s", b)
	}
	// Ripetuta, l'operazione trova tutto a posto e non fa commit.
	if e, id, err := r.applicaBlocco([]int{4, 5, 1}, m, admin); err != nil || id != "" || len(e.aPosto) != 3 {
		t.Errorf("ripetuta: %+v, %q, %v", e, id, err)
	}
	// Uno slug non valido si segnala prima, e non si applica niente.
	m = leggiModificaBlocco(url.Values{"aggiungi.slug": {"dighe/inventata"}})
	if e, _, err := r.applicaBlocco([]int{4, 5}, m, admin); err == nil || len(e.errori) != 2 {
		t.Errorf("slug non valido: %+v, %v", e, err)
	}
	// Le Operazioni recenti la riconoscono come in blocco, e si annulla.
	ops, _ := r.operazioni()
	if ops[0].Tipo != "In blocco" || len(ops[0].Pid) != 3 {
		t.Fatalf("operazione: %+v", ops[0])
	}
	if _, err := r.annulla(ops[0].Hash, admin); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(r.contenuti, "arricchimento", "foto", "4.yaml")); !os.IsNotExist(err) {
		t.Errorf("4.yaml dopo l'annullamento: %v", err)
	}
}

// Oltre 500 Foto la conferma chiede di scrivere il numero.
func TestBloccoConfermaEsplicita(t *testing.T) {
	r := redazioneConIndice(t)
	h := soloModeratore(&utenteFinto{u: &utente{ID: 2, Nome: "admin", Gruppi: []string{gruppoAdmin}}}, r.blocco)
	molti := make([]int, 501)
	for i := range molti {
		molti[i] = 100000 + i // pid inesistenti: bastano per la soglia
	}
	v := url.Values{"pid": {scriviPids(molti)}, "modo.fonte": {"svuota"}}
	req := httptest.NewRequest("POST", "/admin/blocco/applica", strings.NewReader(v.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	h(w, req)
	if !strings.Contains(w.Body.String(), "Per confermare su 501 Foto scrivi 501") {
		t.Errorf("senza la conferma:\n%s", w.Body)
	}
}
