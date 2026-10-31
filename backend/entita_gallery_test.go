package main

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// conEntitaGallery aggiunge ai contenuti di prova l'Entità senza pagina dighe/pontesei con le
// Foto 3 e 4, e un file per dighe/vajont, che è anche sul sito. La Foto 5 usa uno Slug vecchio.
func conEntitaGallery(t *testing.T, dir string) {
	t.Helper()
	for p, s := range map[string]string{
		"entita/dighe/pontesei.yaml": "title: Pontesei\naliases: [dighe/pontesei-vecchia, dighe/vajont, altre/1, ../fuori]\ngeolocation: {stato: Italia, regione: Veneto, provincia: Belluno}\n",
		"entita/dighe/vajont.yaml":   "title: Doppione\n",
		"arricchimento/foto/3.yaml":  "slug: [dighe/pontesei]\n",
		"arricchimento/foto/4.yaml":  "slug: [dighe/pontesei]\n",
		"arricchimento/foto/5.yaml":  "slug: [dighe/pontesei-vecchia]\n",
	} {
		p = filepath.Join(dir, p)
		os.MkdirAll(filepath.Dir(p), 0o755)
		if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestLeggiEntitaUnite(t *testing.T) {
	dir, indice := contenutiProva(t)
	conEntitaGallery(t, dir)
	sopra := t.TempDir()
	os.MkdirAll(filepath.Join(sopra, "entita", "opere"), 0o755)
	os.WriteFile(filepath.Join(sopra, "entita", "opere", "caprizi.yaml"), []byte("title: Caprizi\n"), 0o644)
	u, err := leggiEntita(dir, indice, nuovoLivello(dir, sopra))
	if err != nil {
		t.Fatal(err)
	}
	var slug []string
	for _, e := range u.idx {
		slug = append(slug, e.Slug)
	}
	if !slices.Equal(slug, []string{"dighe/pontesei", "dighe/vajont", "opere/caprizi"}) {
		t.Fatalf("Entità: %v", slug)
	}
	// Se lo Slug è in entrambe le fonti vincono i dati del sito, che ha il link.
	if v := u.idx[1]; v.Title != "Vajont" || v.Draft {
		t.Errorf("il sito non vince: %+v", v)
	}
	if !u.idx[0].Draft || u.idx[0].Geolocation.Provincia != "Belluno" {
		t.Errorf("Entità senza pagina: %+v", u.idx[0])
	}
	if len(u.rimandi) != 1 || u.rimandi["dighe/pontesei-vecchia"] != "dighe/pontesei" {
		t.Errorf("rimandi: %v", u.rimandi)
	}
	avvisi := strings.Join(u.avvisi, "\n")
	for _, atteso := range []string{"entita/dighe/vajont.yaml: il sito ha la stessa pagina", "alias dighe/vajont di dighe/pontesei: esiste una pagina", `alias "altre/1"`, `alias "../fuori"`} {
		if !strings.Contains(avvisi, atteso) {
			t.Errorf("avvisi senza %q:\n%s", atteso, avvisi)
		}
	}
}

func TestEntitaNonValide(t *testing.T) {
	for p, errore := range map[string]string{"entita/dighe/x.yaml": "manca title", "entita/altre/x.yaml": "non si può usare come Slug"} {
		dir := t.TempDir()
		p = filepath.Join(dir, p)
		os.MkdirAll(filepath.Dir(p), 0o755)
		os.WriteFile(p, []byte("aliases: []\n"), 0o644)
		if _, err := leggiEntita(dir, "", nuovoLivello(dir, "")); err == nil || !strings.Contains(err.Error(), errore) {
			t.Errorf("%s: errore %v, atteso %q", p, err, errore)
		}
	}
}

func TestBuildEntitaSenzaPagina(t *testing.T) {
	dir, indice := contenutiProva(t)
	conEntitaGallery(t, dir)
	// Una Foto con una sola Categoria non è più «Da classificare» (ADR 0019).
	os.WriteFile(filepath.Join(dir, "arricchimento", "foto", "2.yaml"), []byte("slug: []\ncategories: [tecnica/turbine]\n"), 0o644)
	dist := filepath.Join(dir, "dist")
	if err := build([]string{"-contenuti", dir, "-indice", indice, "-dist", dist, "-base", "/g/"}); err != nil {
		t.Fatal(err)
	}
	leggi := func(p string) string {
		b, err := os.ReadFile(filepath.Join(dist, p))
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	// Una pagina come le altre, senza il link al sito; le Foto 3, 4 e, con lo Slug vecchio, 5.
	pagina := leggi("dighe/pontesei/index.html")
	for _, atteso := range []string{"Pontesei", "3 foto · Belluno, Veneto, Italia", `href="/g/f/3/"`, `href="/g/f/4/"`, `href="/g/f/5/"`, "OpenStreetMap"} {
		if !strings.Contains(pagina, atteso) {
			t.Errorf("pagina senza %q", atteso)
		}
	}
	if strings.Contains(pagina, "sul sito") {
		t.Error("link al sito su un'Entità senza pagina")
	}
	if !strings.Contains(leggi("dighe/vajont/index.html"), "Pagina di Vajont sul sito") {
		t.Error("il sito non vince per dighe/vajont")
	}
	if !strings.Contains(leggi("dighe/index.html"), `href="/g/dighe/pontesei/"`) {
		t.Error("Entità senza pagina fuori dall'albero")
	}
	// L'Alias rimanda; i percorsi non validi non si scrivono.
	rimando := leggi("dighe/pontesei-vecchia/index.html")
	if !strings.Contains(rimando, `<meta http-equiv="refresh" content="0; url=/g/dighe/pontesei/">`) || !strings.Contains(rimando, `rel="canonical" href="/g/dighe/pontesei/"`) {
		t.Errorf("rimando: %s", rimando)
	}
	if _, err := os.Stat(filepath.Join(dir, "fuori")); err == nil {
		t.Error("un Alias è uscito da dist/")
	}
	if strings.Contains(leggi("f/2/index.html"), `/g/altre/`) {
		t.Error("la Foto con una Categoria è ancora «Da classificare»")
	}
	// Le informazioni interne non finiscono in dist/ (ADR 0019).
	filepath.WalkDir(dist, func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && (strings.HasSuffix(p, ".html") || strings.HasSuffix(p, ".json")) {
			b, _ := os.ReadFile(p)
			for _, interno := range []string{"da-collegare", "da-classificare", "da classificare", "da-separare", "senza pagina", "non ancora sul sito", "non ancora collegate", "paginasito"} {
				if strings.Contains(strings.ToLower(string(b)), interno) {
					t.Errorf("%s contiene %q", p, interno)
				}
			}
		}
		return nil
	})
}

// La Gallery non presenta come dighe le Entità delle altre sezioni (ADR 0019, #74).
func TestBuildNonSoloDighe(t *testing.T) {
	dir, indice := contenutiProva(t)
	for p, s := range map[string]string{
		"entita/centrali/goglio.yaml":   "title: Goglio\n",
		"entita/eventi/fiera-2008.yaml": "title: Fiera 2008\n",
		"entita/opere/presa-agaro.yaml": "title: Presa Agaro\n",
		"arricchimento/foto/3.yaml":     "slug: [centrali/goglio]\n",
		"arricchimento/foto/4.yaml":     "slug: [eventi/fiera-2008]\n",
		"arricchimento/foto/5.yaml":     "slug: [opere/presa-agaro]\n",
	} {
		p = filepath.Join(dir, p)
		os.MkdirAll(filepath.Dir(p), 0o755)
		if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	dist := filepath.Join(dir, "dist")
	if err := build([]string{"-contenuti", dir, "-indice", indice, "-dist", dist, "-base", "/g/"}); err != nil {
		t.Fatal(err)
	}
	leggi := func(p string) string {
		b, err := os.ReadFile(filepath.Join(dist, p))
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	attesi := map[string]string{"dighe/vajont": "Dighe", "centrali/goglio": "Centrali", "eventi/fiera-2008": "Eventi", "opere/presa-agaro": "Opere"}

	_, dati, _ := strings.Cut(leggi("index.html"), `id="dati-bussola">`)
	dati, _, _ = strings.Cut(dati, "</script>")
	var voci []voceBussola
	if err := json.Unmarshal([]byte(dati), &voci); err != nil {
		t.Fatal(err)
	}
	cosa := map[string]string{}
	for _, v := range voci {
		cosa[v.URL] = v.Cosa
	}
	var cerca indiceRicerca
	if err := json.Unmarshal([]byte(leggi("statico/cerca.json")), &cerca); err != nil {
		t.Fatal(err)
	}
	tipo := map[string]string{}
	for _, p := range cerca.Pagine {
		tipo[p.Slug] = p.Tipo
	}
	for s, atteso := range attesi {
		if cosa["/g/"+s+"/"] != atteso {
			t.Errorf("Cosa di %s = %q, atteso %q", s, cosa["/g/"+s+"/"], atteso)
		}
		if tipo[s] != atteso {
			t.Errorf("tipo di %s nella ricerca = %q, atteso %q", s, tipo[s], atteso)
		}
	}
	// Foto 9: senza Entità e senza Categoria.
	home, altre := leggi("index.html"), leggi("altre/index.html")
	if !strings.Contains(home, "Altre foto") || !strings.Contains(altre, "<title>Altre foto") {
		t.Error("manca «Altre foto»")
	}
	if strings.Contains(home, "Tutte le dighe") || strings.Contains(leggi("dighe/index.html"), "Tutte le dighe") {
		t.Error("«Tutte le dighe» per Entità che non sono dighe")
	}
	if !strings.Contains(leggi("dighe/index.html"), "collegate a 4 soggetti") {
		t.Error("il sottotitolo di /dighe/ non conta le Entità")
	}
	filepath.WalkDir(dist, func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.HasSuffix(p, ".html") {
			b, _ := os.ReadFile(p)
			if l := strings.ToLower(string(b)); strings.Contains(l, "da classificare") || strings.Contains(l, "pagina del sito") {
				t.Errorf("%s nomina un'informazione interna", p)
			}
		}
		return nil
	})
}

// La mappa dà a ogni Entità il tipo della sua sezione: un evento non è una diga.
func TestTipoMappa(t *testing.T) {
	c := &contenuti{entita: map[string]*entita{}}
	per := map[string][]*fotoPubblicata{}
	var slugs []string
	for _, s := range []string{"centrali/goglio", "dighe/agaro", "eventi/fiera-2008"} {
		e := &entita{Slug: s, Title: s}
		e.Geolocation.Coords = &struct {
			Latitudine  float64 `json:"latitudine"`
			Longitudine float64 `json:"longitudine"`
			Zoom        int     `json:"zoom,omitempty"`
		}{46.29, 8.30, 0}
		c.entita[s], per[s] = e, []*fotoPubblicata{{Pid: 1, Formato: "immagine"}}
		slugs = append(slugs, s)
	}
	b := &builder{opzioniBuild: opzioniBuild{base: "/g/"}, c: c}
	var geo struct {
		Features []struct {
			Properties map[string]any `json:"properties"`
		} `json:"features"`
	}
	_, g, _ := b.geojsonDiga("eventi/fiera-2008", per, slugs)
	if err := json.Unmarshal([]byte(g), &geo); err != nil {
		t.Fatal(err)
	}
	tipi := map[string]any{}
	for _, f := range geo.Features {
		tipi[strings.Fields(f.Properties["name"].(string))[0]] = f.Properties["type"]
	}
	if tipi["centrali/goglio"] != "centrale" || tipi["dighe/agaro"] != "diga" || tipi["eventi/fiera-2008"] != "" {
		t.Errorf("tipi sulla mappa: %v", tipi)
	}
}

// Una Foto con uno Slug perde da-collegare: si sa cosa ritrae.
func TestDaCollegareConSlug(t *testing.T) {
	dir, indice := contenutiProva(t)
	conEntitaGallery(t, dir)
	c, err := leggiContenuti(dir, indice, nuovoLivello(dir, ""))
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range c.foto {
		if f.Pid == 4 && (len(f.Tags) != 0 || !slices.Equal(f.Slug, []string{"dighe/pontesei"}) || f.Geolocation.Regione != "Veneto") {
			t.Errorf("pid 4: %+v", f)
		}
	}
}

// #75: indice-entita legge ogni sezione del Sito, tranne quelle fuori dall'Indice.
func TestLeggiSitoSezioni(t *testing.T) {
	dir := t.TempDir()
	for _, p := range []string{"dighe/vajont", "biblioteca/il-guardiadiga", "nuova/pagina", "blog/2022", "noi/ode"} {
		os.MkdirAll(filepath.Join(dir, p), 0o755)
		if err := os.WriteFile(filepath.Join(dir, p, "index.md"), []byte("---\ntitle: "+p+"\n---\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	idx, err := leggiSito(dir)
	if err != nil {
		t.Fatal(err)
	}
	var slug []string
	for _, e := range idx {
		slug = append(slug, e.Slug)
	}
	if !slices.Equal(slug, []string{"biblioteca/il-guardiadiga", "dighe/vajont", "nuova/pagina"}) {
		t.Errorf("Indice: %v", slug)
	}
}

// #75: senza Indice il build va chiesto, perché un -indice dimenticato toglierebbe le Entità del Sito.
func TestBuildSenzaIndice(t *testing.T) {
	dir, indice := contenutiProva(t)
	conEntitaGallery(t, dir)
	conf := filepath.Join(dir, "gerbillo.toml")
	os.WriteFile(conf, []byte("contenuti = \""+dir+"\"\n"), 0o644)
	dist := filepath.Join(dir, "dist")
	for _, flag := range [][]string{nil, {"-indice", indice, "-senza-indice"}} {
		if err := build(append([]string{"-config", conf, "-dist", dist, "-base", "/g/"}, flag...)); err == nil {
			t.Errorf("build con %v riuscito", flag)
		}
	}
	if err := build([]string{"-config", conf, "-dist", dist, "-base", "/g/", "-senza-indice"}); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dist, "dighe", "pontesei", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "sul sito") {
		t.Error("link al sito senza Indice")
	}
}
