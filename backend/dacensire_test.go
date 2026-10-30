package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// conDaCensire aggiunge ai contenuti di prova: l'Entità 1 con le Foto 1 (che ha anche uno
// Slug), 3 e 4, l'Entità 2 diventata dighe/vajont con la Foto 5, e un id che non esiste.
func conDaCensire(t *testing.T, dir string) {
	t.Helper()
	for p, s := range map[string]string{
		"entita/1.yaml":             "id: 1\nnome: Pontesei\ntipo: diga\ngeolocation: {stato: Italia, regione: Veneto, provincia: Belluno}\n",
		"entita/2.yaml":             "id: 2\nnome: Vecchia Vajont\ntipo: centrale\ndiventata: dighe/vajont\n",
		"arricchimento/foto/1.yaml": "datazione: 195X\nauthors: [Ente]\nlicenza: CC-BY-4.0\nda_censire: [1]\n",
		"arricchimento/foto/3.yaml": "da_censire: [1, 99]\n",
		"arricchimento/foto/4.yaml": "da_censire: [1]\n",
		"arricchimento/foto/5.yaml": "da_censire: [2]\n",
	} {
		p = filepath.Join(dir, p)
		os.MkdirAll(filepath.Dir(p), 0o755)
		if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestDaCensireNeiContenuti(t *testing.T) {
	dir, indice := contenutiProva(t)
	conDaCensire(t, dir)
	c, err := leggiContenuti(dir, indice, nuovoLivello(dir, ""))
	if err != nil {
		t.Fatal(err)
	}
	f := map[int]*fotoPubblicata{}
	for _, x := range c.foto {
		f[x.Pid] = x
	}
	// La Foto 4 aveva da-collegare: ora si sa cosa ritrae.
	if len(f[4].Tags) != 0 || !slices.Equal(f[4].DaCensire, []int{1}) || f[4].Geolocation.Regione != "Veneto" {
		t.Errorf("pid 4: %+v", f[4])
	}
	// Una Foto può avere entrambi i campi.
	if !slices.Equal(f[1].Slug, []string{"dighe/vajont"}) || !slices.Equal(f[1].DaCensire, []int{1}) {
		t.Errorf("pid 1: %v %v", f[1].Slug, f[1].DaCensire)
	}
	// Diventata uno Slug: vale come collegata.
	if !slices.Equal(f[5].Slug, []string{"dighe/vajont"}) || len(f[5].DaCensire) != 0 || len(f[5].Tags) != 0 {
		t.Errorf("pid 5: %v %v %v", f[5].Slug, f[5].DaCensire, f[5].Tags)
	}
	// L'id che non esiste si segnala come gli slug sconosciuti.
	if !slices.Contains(c.ignoti, "3: da censire 99") || !slices.Equal(f[3].DaCensire, []int{1}) {
		t.Errorf("id sconosciuto: %v, %v", c.ignoti, f[3].DaCensire)
	}
}

// Un file nella cartella sovrapposta vince su quello dei contenuti, come nel resto del Livello.
func TestDaCensireSovrapposta(t *testing.T) {
	dir, _ := contenutiProva(t)
	conDaCensire(t, dir)
	sopra := t.TempDir()
	os.MkdirAll(filepath.Join(sopra, "entita"), 0o755)
	os.WriteFile(filepath.Join(sopra, "entita", "1.yaml"), []byte("id: 1\nnome: Sopra\ntipo: altra\n"), 0o644)
	m, err := leggiDaCensire(dir, nuovoLivello(dir, sopra))
	if err != nil {
		t.Fatal(err)
	}
	if m[1].Nome != "Sopra" || m[2] == nil {
		t.Errorf("sovrapposizione: %+v, %+v", m[1], m[2])
	}
}

func TestDaCensireNonValide(t *testing.T) {
	for y, errore := range map[string]string{
		"id: 2\nnome: X\ntipo: diga\n":      "diverso dal nome del file",
		"id: 1\ntipo: diga\n":               "manca il nome",
		"id: 1\nnome: X\ntipo: reportage\n": `tipo "reportage"`,
	} {
		dir := t.TempDir()
		os.MkdirAll(filepath.Join(dir, "entita"), 0o755)
		os.WriteFile(filepath.Join(dir, "entita", "1.yaml"), []byte(y), 0o644)
		if _, err := leggiDaCensire(dir, nuovoLivello(dir, "")); err == nil || !strings.Contains(err.Error(), errore) {
			t.Errorf("%q: errore %v, atteso %q", y, err, errore)
		}
	}
}

func TestBuildDaCensire(t *testing.T) {
	dir, indice := contenutiProva(t)
	conDaCensire(t, dir)
	dist, redazione := filepath.Join(dir, "dist"), filepath.Join(dir, "redazione.json")
	if err := build([]string{"-contenuti", dir, "-indice", indice, "-dist", dist, "-base", "/g/", "-redazione", redazione}); err != nil {
		t.Fatal(err)
	}
	leggi := func(p string) string {
		b, err := os.ReadFile(filepath.Join(dist, p))
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	pagina := leggi("da-censire/1/index.html")
	for _, atteso := range []string{"Pontesei", "3 foto · Diga · Belluno, Veneto, Italia · non ancora sul sito",
		`href="/g/f/1/"`, `href="/g/f/3/"`, `href="/g/f/4/"`, "OpenStreetMap", "Nominatim"} {
		if !strings.Contains(pagina, atteso) {
			t.Errorf("pagina senza %q", atteso)
		}
	}
	// Nell'albero, sotto la sua regione.
	albero := leggi("dighe/index.html")
	if i, j := strings.Index(albero, ">Veneto<"), strings.Index(albero, `href="/g/da-censire/1/"`); i < 0 || j < i {
		t.Errorf("albero: Veneto a %d, Entità da censire a %d", i, j)
	}
	// Nella Bussola solo sotto «Dove».
	var bussola []voceBussola
	home := leggi("index.html")
	_, resto, _ := strings.Cut(home, `id="dati-bussola">`)
	dati, _, _ := strings.Cut(resto, "</script>")
	if err := json.Unmarshal([]byte(dati), &bussola); err != nil {
		t.Fatal(err)
	}
	i := slices.IndexFunc(bussola, func(v voceBussola) bool { return v.URL == "/g/da-censire/1/" })
	if i < 0 || bussola[i].Dove != "Veneto" || bussola[i].Cosa != "" || bussola[i].Come != "" || bussola[i].Quando != "" || bussola[i].Uso != "" {
		t.Errorf("Bussola: %+v", bussola)
	}
	// Nell'Indice di ricerca, con le sue Foto.
	var cerca indiceRicerca
	if err := json.Unmarshal([]byte(leggi("statico/cerca.json")), &cerca); err != nil {
		t.Fatal(err)
	}
	p := slices.IndexFunc(cerca.Pagine, func(v paginaRicerca) bool { return v.Titolo == "Pontesei" })
	if p < 0 || cerca.Pagine[p].Tipo != "Dighe" || cerca.Pagine[p].N != 3 {
		t.Fatalf("Pagine: %+v", cerca.Pagine)
	}
	for _, f := range cerca.Foto {
		if f.Pid == 4 && (!slices.Contains(f.Entita, p) || !strings.Contains(f.Meta, "Pontesei")) {
			t.Errorf("Foto 4 nell'Indice di ricerca: %+v", f)
		}
	}
	// «Ritrae» elenca entrambe; le Foto con un'Entità da censire non sono «Da classificare».
	if f1 := leggi("f/1/index.html"); !strings.Contains(f1, "Ritrae:") || !strings.Contains(f1, `href="/g/da-censire/1/"`) || !strings.Contains(f1, `href="/g/dighe/vajont/"`) {
		t.Error("f/1 senza «Ritrae» con le due Entità")
	}
	if f4 := leggi("f/4/index.html"); !strings.Contains(f4, `<a class="dib pa2 mr1 mb1 f5 b ttu tracked link white bg-dark-gray" href="/g/da-censire/1/">Pontesei</a>`) {
		t.Error("f/4 senza l'Entità da censire nelle briciole")
	}
	if _, err := os.Stat(filepath.Join(dist, "altre", "1", "index.html")); err == nil {
		t.Error("le Foto dell'album personale sono ancora «Da classificare»")
	}
	// Diventata: la pagina rimanda, e le Foto stanno nella pagina dell'Entità.
	rimando := leggi("da-censire/2/index.html")
	if !strings.Contains(rimando, `<meta http-equiv="refresh" content="0; url=/g/dighe/vajont/">`) || !strings.Contains(rimando, `rel="canonical" href="/g/dighe/vajont/"`) {
		t.Errorf("rimando: %s", rimando)
	}
	if !strings.Contains(leggi("dighe/vajont/index.html"), `href="/g/f/5/"`) {
		t.Error("la Foto 5 non è nella pagina di dighe/vajont")
	}
	// da-censire.json: solo quelle ancora da censire, con il numero di Foto.
	var el []voceIndiceDaCensire
	if err := json.Unmarshal([]byte(leggi("da-censire.json")), &el); err != nil {
		t.Fatal(err)
	}
	if len(el) != 1 || el[0].ID != 1 || el[0].Nome != "Pontesei" || el[0].Tipo != "diga" || el[0].N != 3 || el[0].Geolocation.Provincia != "Belluno" {
		t.Errorf("da-censire.json: %+v", el)
	}
	if !strings.Contains(leggi("da-censire.json"), `"provincia":"Belluno"`) {
		t.Errorf("da-censire.json: %s", leggi("da-censire.json"))
	}
	// redazione.json porta da_censire.
	idx, err := leggiIndiceRedazione(redazione)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(idx.Foto[idx.perPid[4]].DaCensire, []int{1}) {
		t.Errorf("redazione.json: %+v", idx.Foto[idx.perPid[4]])
	}
}
