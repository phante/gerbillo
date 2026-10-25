package main

// Importazione Coppermine → YAML (G1, #9). Ogni esecuzione rigenera da zero
// importazione/foto/<pid>.yaml e importazione/anomalie.md nel Repository dei
// contenuti e ne fa un commit locale (ADR 0004, 0009). Con -indice il rapporto
// comprende la Mappatura degli album, con gli album da fare e quelli ambigui (#38).

import (
	"bytes"
	"cmp"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"html"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/go-sql-driver/mysql"
	"gopkg.in/yaml.v3"
)

type configImporta struct {
	MySQL struct {
		Utente    string
		Password  string
		Socket    string // socket locale sul server; con [ssh] è quello remoto
		Indirizzo string // host:porta TCP al posto del socket (MySQL di GitHub Actions per l'anteprima)
		DBGallery string `toml:"db_gallery"`
		DBForum   string `toml:"db_forum"`
	}
	// SSH è opzionale: dal Mac il socket del server arriva da un tunnel.
	SSH struct {
		Host string // es. ubuntu@prdi.it
	}
	Contenuti string // cartella del Repository dei contenuti
}

func importa(args []string) error {
	fs := flag.NewFlagSet("importa", flag.ExitOnError)
	conf := fs.String("config", "gerbillo.toml", "file di configurazione TOML")
	indice := fs.String("indice", "", "Indice delle Entità (JSON), per il rapporto sulla Mappatura")
	arr := fs.String("arricchimento", "", "cartella sovrapposta al Livello di arricchimento (anteprima: dati/arricchimento)")
	pVoc := fs.String("vocabolario", "", "vocabolario controllato (dati/vocabolario.yaml): rifiuta le categories che non conosce")
	fs.Parse(args)
	var c configImporta
	c.MySQL.Socket = "/var/run/mysqld/mysqld.sock"
	c.MySQL.DBGallery, c.MySQL.DBForum = "progettodighe_it_gallery", "progettodighe_it_forum"
	if _, err := toml.DecodeFile(*conf, &c); err != nil {
		return err
	}
	if c.Contenuti == "" {
		return errors.New("contenuti mancante nella configurazione")
	}
	liv := nuovoLivello(c.Contenuti, *arr)
	var manuali fileMappature
	if err := liv.leggi("mappature.yaml", &manuali); err != nil {
		return err
	}
	if *pVoc != "" {
		voc, err := leggiVocabolario(*pVoc)
		if err != nil {
			return err
		}
		if fuori := fuoriVocabolario(manuali, voc); len(fuori) > 0 {
			return fmt.Errorf("categories fuori dal vocabolario in mappature.yaml:\n%s", strings.Join(fuori, "\n"))
		}
	}
	var idx []entita
	if *indice != "" {
		var err error
		if idx, err = leggiIndice(*indice); err != nil {
			return err
		}
	}
	sock := c.MySQL.Socket
	if c.SSH.Host != "" {
		locale, chiudi, err := tunnel(c.SSH.Host, sock)
		if err != nil {
			return err
		}
		defer chiudi()
		sock = locale
	}
	cfg := mysql.NewConfig()
	cfg.User, cfg.Passwd, cfg.Net, cfg.Addr = c.MySQL.Utente, c.MySQL.Password, "unix", sock
	if c.MySQL.Indirizzo != "" {
		cfg.Net, cfg.Addr = "tcp", c.MySQL.Indirizzo
	}
	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return err
	}
	defer db.Close()
	righe, err := leggiCoppermine(db, c.MySQL.DBGallery, c.MySQL.DBForum)
	if err != nil {
		return err
	}
	foto, rap := converti(righe)
	if rap.alias, err = leggiAlias(liv); err != nil {
		return err
	}
	if idx != nil {
		_, rap.mappature = mappature(righe.album, idx, manuali)
	}
	decisioni, err := leggiRiservati(liv, righe.album)
	if err != nil {
		return err
	}
	rap.senzaDecisione = senzaDecisione(righe, decisioni)
	if err := scrivi(c.Contenuti, foto, rap); err != nil {
		return err
	}
	b, err := yaml.Marshal(righe.album)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(c.Contenuti, "importazione", "album.yaml"), b, 0o644); err != nil {
		return err
	}
	if len(foto) != len(righe.foto) {
		return fmt.Errorf("gate: %d YAML per %d righe", len(foto), len(righe.foto))
	}
	fmt.Printf("importate %d foto: %d scritte, %d sparite, %d non decodificate, %d mojibake\n",
		len(foto), rap.scritte, len(rap.spariti), len(rap.nonDecodificati), len(rap.mojibake))
	if m := rap.mappature; m != nil {
		t := m.totale()
		fmt.Printf("album: %d, abbinati in automatico %.1f%% (foto %.1f%%), da collegare %d (foto %d)\n",
			t.album, perc(t.autoAlbum, t.album), perc(t.autoFoto, t.foto), t.faAlbum+t.ambAlbum, m.interni[daCollegare])
	}
	return registra(c.Contenuti, "importazione", fmt.Sprintf("importazione: %d foto, %d scritte, %d sparite", rap.totale, rap.scritte, len(rap.spariti)), "")
}

// tunnel inoltra il socket MySQL del server su un socket locale con ssh -L.
func tunnel(host, remoto string) (string, func(), error) {
	dir, err := os.MkdirTemp("", "gerbillo-ssh")
	if err != nil {
		return "", nil, err
	}
	locale := filepath.Join(dir, "mysql.sock")
	cmd := exec.Command("ssh", "-N", "-o", "ExitOnForwardFailure=yes", "-L", locale+":"+remoto, host)
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		os.RemoveAll(dir)
		return "", nil, err
	}
	chiudi := func() { cmd.Process.Kill(); cmd.Wait(); os.RemoveAll(dir) }
	for range 150 {
		if _, err := os.Stat(locale); err == nil {
			return locale, chiudi, nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	chiudi()
	return "", nil, fmt.Errorf("tunnel ssh verso %s non pronto", host)
}

// rigaFoto contiene i byte grezzi delle colonne ammesse di cpg1410_pictures.
// Mai IP né guest_token: le colonne lette sono solo queste.
type rigaFoto struct {
	pid, aid, larghezza, altezza, owner int
	ctime                               int64
	filepath, filename                  []byte
	titolo, descrizione, keywords       []byte
	user1, user2, user3                 []byte
}

type datiCoppermine struct {
	foto      []rigaFoto
	exif      map[int]string // pid → DateTimeOriginal
	utenti    map[int]string // user_id → username
	album     albumCoppermine
	riservati map[int]int // aid degli album non pubblici → visibility (diversa da 0)
}

// albumCoppermine è il contenuto di importazione/album.yaml: l'input della Mappatura (G2).
type albumCoppermine struct {
	Album     map[int]albumYAML     `yaml:"album"`
	Categorie map[int]categoriaYAML `yaml:"categorie"`
}

type albumYAML struct {
	Categoria int    `yaml:"categoria"` // da 10000 in su: User galleries (10000 + owner)
	Foto      int    `yaml:"foto"`
	Title     string `yaml:"title"`
}

type categoriaYAML struct {
	Nome   string `yaml:"nome"`
	Parent int    `yaml:"parent"`
}

func leggiCoppermine(db *sql.DB, gallery, forum string) (*datiCoppermine, error) {
	d := &datiCoppermine{exif: map[int]string{}, utenti: map[int]string{}}
	// BINARY: i byte così come sono, senza conversioni del driver (latin1 con UTF-8 dentro).
	r, err := db.Query(`SELECT pid, aid, pwidth, pheight, owner_id, ctime,
		BINARY filepath, BINARY filename, BINARY title, BINARY caption, BINARY keywords,
		BINARY user1, BINARY user2, BINARY user3
		FROM ` + gallery + `.cpg1410_pictures ORDER BY pid`)
	if err != nil {
		return nil, err
	}
	for r.Next() {
		var f rigaFoto
		if err := r.Scan(&f.pid, &f.aid, &f.larghezza, &f.altezza, &f.owner, &f.ctime,
			&f.filepath, &f.filename, &f.titolo, &f.descrizione, &f.keywords,
			&f.user1, &f.user2, &f.user3); err != nil {
			return nil, err
		}
		d.foto = append(d.foto, f)
	}
	if err := r.Err(); err != nil {
		return nil, err
	}
	r, err = db.Query(`SELECT pid, BINARY exifData FROM ` + gallery + `.cpg1410_exif`)
	if err != nil {
		return nil, err
	}
	for r.Next() {
		var pid int
		var b []byte
		if err := r.Scan(&pid, &b); err != nil {
			return nil, err
		}
		if v := dataExif(b); v != "" {
			d.exif[pid] = v
		}
	}
	if err := r.Err(); err != nil {
		return nil, err
	}
	r, err = db.Query(`SELECT user_id, username FROM ` + forum + `.phpbb_users`)
	if err != nil {
		return nil, err
	}
	for r.Next() {
		var id int
		var nome string
		if err := r.Scan(&id, &nome); err != nil {
			return nil, err
		}
		d.utenti[id] = html.UnescapeString(nome) // phpBB salva lo username escapato
	}
	if err := r.Err(); err != nil {
		return nil, err
	}
	return d, leggiAlbum(db, gallery, d)
}

func leggiAlbum(db *sql.DB, gallery string, d *datiCoppermine) error {
	d.album = albumCoppermine{Album: map[int]albumYAML{}, Categorie: map[int]categoriaYAML{}}
	d.riservati = map[int]int{}
	foto := map[int]int{}
	for _, f := range d.foto {
		foto[f.aid]++
	}
	r, err := db.Query(`SELECT aid, category, visibility, BINARY title FROM ` + gallery + `.cpg1410_albums`)
	if err != nil {
		return err
	}
	for r.Next() {
		var aid, cat, vis int
		var t []byte
		if err := r.Scan(&aid, &cat, &vis, &t); err != nil {
			return err
		}
		titolo, _ := decodifica(t)
		d.album.Album[aid] = albumYAML{cat, foto[aid], titolo}
		if vis != 0 {
			d.riservati[aid] = vis
		}
	}
	if err := r.Err(); err != nil {
		return err
	}
	r, err = db.Query(`SELECT cid, parent, BINARY name FROM ` + gallery + `.cpg1410_categories`)
	if err != nil {
		return err
	}
	for r.Next() {
		var cid, parent int
		var n []byte
		if err := r.Scan(&cid, &parent, &n); err != nil {
			return err
		}
		nome, _ := decodifica(n)
		d.album.Categorie[cid] = categoriaYAML{nome, parent}
	}
	return r.Err()
}

// exifData è un array PHP serializzato e piatto. Se ne legge solo DateTimeOriginal:
// numero di serie e altri campi identificativi non escono mai di qui. Coppermine
// non salva il GPS, quindi non c'è nulla da portare per la Vista per mappa.
var reDataExif = regexp.MustCompile(`s:16:"DateTimeOriginal";s:\d+:"([^"]*)"`)

func dataExif(b []byte) string {
	m := reDataExif.FindSubmatch(b)
	if m == nil {
		return ""
	}
	return strings.TrimSpace(string(m[1]))
}

type fotoYAML struct {
	pid  int
	file []byte
}

type rapporto struct {
	totale          int
	formati         map[string]int
	altro           []string // "pid filename"
	cp1252          []int
	nonDecodificati []string // "pid campo"
	mojibake        []string // "pid campo"
	ospiti          int
	cancellati      map[int]int // owner_id → foto
	senzaAutore     int
	autori          map[string]int // valore grezzo → Foto
	alias           aliasAutori
	riservate       int
	senzaDecisione  []string                  // righe di tabella: album riservati senza riga in riservati.yaml (#35)
	datazioneExif   int                       // Datazione dall'EXIF (#18)
	soloUser3       int                       // senza Datazione, con user3 mostrato come testo
	senzaData       int                       // né l'una né l'altro
	exifScartati    []string                  // "pid valore": EXIF senza una data valida
	candidate       map[string]map[string]int // Datazione da user3 (#39): tipo → precisione → Foto
	residui         []string                  // righe di tabella: candidate senza regola
	scansioni       []string                  // righe di tabella, da controllare
	spariti         []int
	scritte         int
	mappature       *rapportoMappature // solo con importa -indice
}

func formato(nome string) string {
	switch strings.ToLower(filepath.Ext(nome)) {
	case ".jpg", ".jpeg", ".png", ".gif":
		return "immagine"
	case ".pdf":
		return "pdf"
	}
	return "altro"
}

func converti(d *datiCoppermine) ([]fotoYAML, *rapporto) {
	rap := &rapporto{totale: len(d.foto), formati: map[string]int{}, cancellati: map[int]int{}, autori: map[string]int{},
		candidate: map[string]map[string]int{}}
	var out []fotoYAML
	for _, f := range d.foto {
		cp := false
		testo := func(campo string, b []byte) string {
			s, e := decodifica(b)
			switch e {
			case inCP1252:
				cp = true
			case nonDecodificato:
				cp = true
				rap.nonDecodificati = append(rap.nonDecodificati, fmt.Sprintf("%d %s", f.pid, campo))
			}
			if mojibake(s) {
				rap.mojibake = append(rap.mojibake, fmt.Sprintf("%d %s", f.pid, campo))
			}
			return s
		}
		// Le mappe di yaml.v3 escono con le chiavi ordinate: output deterministico.
		y := map[string]any{
			"pid":       f.pid,
			"date":      time.Unix(f.ctime, 0).In(roma).Format(time.RFC3339),
			"larghezza": f.larghezza,
			"altezza":   f.altezza,
		}
		// Niente hits: cambia a ogni visita e sporcherebbe il diff di ogni Importazione.
		cpg := map[string]any{"aid": f.aid}
		metti := func(m map[string]any, k, v string) {
			if v != "" {
				m[k] = v
			}
		}
		metti(y, "title", testo("title", f.titolo))
		metti(y, "description", testo("caption", f.descrizione))
		autore := testo("user1", f.user1)
		// authors è una lista come nel sito (ADR 0013); gli alias (#19) si applicano al build.
		if autore != "" {
			y["authors"] = []string{autore}
			rap.autori[autore]++
		}
		nome := testo("filename", f.filename)
		metti(cpg, "filepath", testo("filepath", f.filepath))
		metti(cpg, "filename", nome)
		metti(cpg, "keywords", testo("keywords", f.keywords))
		metti(cpg, "luogo", testo("user2", f.user2))
		user3 := testo("user3", f.user3)
		metti(cpg, "data", user3)
		metti(cpg, "data_exif", d.exif[f.pid])
		exif, exifValido := datazioneExif(d.exif[f.pid])
		if !exifValido && d.exif[f.pid] != "" {
			rap.exifScartati = append(rap.exifScartati, fmt.Sprintf("%d %s", f.pid, d.exif[f.pid]))
		}
		caricata := time.Unix(f.ctime, 0).In(roma)
		tipo, du, interpretata := candidata(user3, exif, caricata)
		switch {
		case tipo != "":
			// Le regole valgono solo per le candidate (#39). Nelle scansioni l'EXIF è la data di
			// scansione: si butta, e resta grezzo in coppermine.data_exif.
			prec := du.prec
			riga := fmt.Sprintf("| %d | %s | %s | %s |", f.pid, cella(user3), d.exif[f.pid], caricata.Format("02/01/2006"))
			if interpretata {
				y["datazione"] = du.edtf
			} else {
				prec = "residui"
				rap.residui = append(rap.residui, riga)
			}
			if tipo == candScansione {
				rap.scansioni = append(rap.scansioni, riga+" "+du.edtf+" |")
			}
			if rap.candidate[tipo] == nil {
				rap.candidate[tipo] = map[string]int{}
			}
			rap.candidate[tipo][prec]++
		case exifValido:
			y["datazione"] = exif
			rap.datazioneExif++
		case user3 != "":
			rap.soloUser3++
		default:
			rap.senzaData++
		}
		y["coppermine"] = cpg
		// Un album non pubblico in Coppermine resta fuori dalla Gallery pubblica (#1). È un dato
		// dell'Importazione, non del Livello di arricchimento: la protezione non dipende da quello.
		if d.riservati[f.aid] != 0 {
			y["riservata"] = true
			rap.riservate++
		}

		fm := formato(nome)
		y["formato"] = fm
		rap.formati[fm]++
		if fm == "altro" {
			rap.altro = append(rap.altro, fmt.Sprintf("%d %s", f.pid, nome))
		}
		// Ospiti: owner_id 0 e 1 (Anonymous). Un id non più in phpbb_users non è un'informazione.
		if n, ok := d.utenti[f.owner]; ok && f.owner > 1 {
			y["contributore"] = map[string]any{"id": f.owner, "nome": n}
		} else if f.owner <= 1 {
			rap.ospiti++
		} else {
			rap.cancellati[f.owner]++
		}
		if autore == "" {
			rap.senzaAutore++
		}
		if cp {
			rap.cp1252 = append(rap.cp1252, f.pid)
		}
		b, err := yaml.Marshal(y)
		if err != nil {
			panic(err) // solo stringhe e interi: non può fallire
		}
		out = append(out, fotoYAML{f.pid, b})
	}
	return out, rap
}

// scrivi riscrive solo i YAML cambiati e toglie quelli delle Foto sparite.
func scrivi(dir string, foto []fotoYAML, rap *rapporto) error {
	cart := filepath.Join(dir, "importazione", "foto")
	if err := os.MkdirAll(cart, 0o755); err != nil {
		return err
	}
	presenti := map[string]bool{}
	for _, f := range foto {
		nome := strconv.Itoa(f.pid) + ".yaml"
		presenti[nome] = true
		p := filepath.Join(cart, nome)
		if vecchio, err := os.ReadFile(p); err == nil && bytes.Equal(vecchio, f.file) {
			continue
		}
		if err := os.WriteFile(p, f.file, 0o644); err != nil {
			return err
		}
		rap.scritte++
	}
	voci, err := os.ReadDir(cart)
	if err != nil {
		return err
	}
	for _, v := range voci {
		if presenti[v.Name()] || !strings.HasSuffix(v.Name(), ".yaml") {
			continue
		}
		if pid, err := strconv.Atoi(strings.TrimSuffix(v.Name(), ".yaml")); err == nil {
			rap.spariti = append(rap.spariti, pid)
		}
		if err := os.Remove(filepath.Join(cart, v.Name())); err != nil {
			return err
		}
	}
	slices.Sort(rap.spariti)
	return os.WriteFile(filepath.Join(dir, "importazione", "anomalie.md"), rap.markdown(), 0o644)
}

// markdown non contiene date né altro che cambi da solo: il diff mostra solo i cambiamenti.
func (r *rapporto) markdown() []byte {
	var b bytes.Buffer
	p := func(f string, a ...any) { fmt.Fprintf(&b, f, a...) }
	elenco := func(v []string) {
		for _, s := range v {
			p("- %s\n", s)
		}
	}
	p("# Anomalie dell'Importazione\n\nFoto importate: %d.\n\n## Formato\n\n", r.totale)
	for _, f := range []string{"immagine", "pdf", "altro"} {
		p("- %s: %d\n", f, r.formati[f])
	}
	p("\nFoto `altro` (pid, file): %d.\n\n", len(r.altro))
	elenco(r.altro)
	p("\n## Encoding\n\nFoto con campi decodificati in cp1252: %d.\n\n", len(r.cp1252))
	if len(r.cp1252) > 0 {
		p("%s\n", unisci(r.cp1252))
	}
	p("\nCampi non decodificabili (pid, campo): %d.\n\n", len(r.nonDecodificati))
	elenco(r.nonDecodificati)
	p("\nCampi con sequenze `Ã`, `Â`, `â€` dopo la decodifica (pid, campo): %d.\n\n", len(r.mojibake))
	elenco(r.mojibake)
	n := 0
	for _, c := range r.cancellati {
		n += c
	}
	p("\n## Contributore\n\nFoto senza Contributore risolto: %d.\n\n- ospiti (owner_id 0 e 1): %d\n- utenti cancellati dal forum: %d\n\n",
		r.ospiti+n, r.ospiti, n)
	ids := make([]int, 0, len(r.cancellati))
	for id := range r.cancellati {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	if len(ids) > 0 {
		p("| owner_id | Foto |\n|---:|---:|\n")
		for _, id := range ids {
			p("| %d | %d |\n", id, r.cancellati[id])
		}
		p("\n")
	}
	p("## Autore\n\nFoto senza Autore (`user1` vuoto): %d.\n\n", r.senzaAutore)
	// Elenco per scrivere a mano arricchimento/autori.yaml (#19): i valori con alias sono segnati.
	valori := make([]string, 0, len(r.autori))
	coperti := 0
	for v := range r.autori {
		valori = append(valori, v)
		if _, ok := r.alias[v]; ok {
			coperti++
		}
	}
	slices.SortFunc(valori, func(a, b string) int {
		return cmp.Or(r.autori[b]-r.autori[a], strings.Compare(a, b))
	})
	p("Valori distinti di `user1`: %d, di cui %d con alias in `arricchimento/autori.yaml`.\n\n", len(valori), coperti)
	if len(valori) > 0 {
		p("| Autore | Foto | Alias |\n|---|---:|---|\n")
		for _, v := range valori {
			a := ""
			if x, ok := r.alias[v]; ok {
				a = "→ " + strings.Join(x, ", ")
				if len(x) == 0 {
					a = "scartato"
				}
			}
			p("| %s | %d | %s |\n", cella(v), r.autori[v], cella(a))
		}
		p("\n")
	}
	p("## Datazione\n\nFoto con Datazione dall'EXIF: %d. Senza Datazione, con `user3` mostrato come testo: %d. Senza nessuna delle due: %d. Candidate, con la Datazione da `user3`: sotto.\n\n",
		r.datazioneExif, r.soloUser3, r.senzaData)
	p("EXIF senza una data valida, scartati (pid, valore): %d.\n\n", len(r.exifScartati))
	elenco(r.exifScartati)
	if len(r.exifScartati) > 0 {
		p("\n")
	}
	p("### Datazione da user3\n\nFoto candidate (#39): senza EXIF con `user3` più di un anno prima del caricamento (storiche) ")
	p("e con l'EXIF più di un anno dopo `user3` (scansioni). Per queste la Datazione viene da `user3`.\n\n")
	p("| Tipo | giorno | mese | anno | decennio | intervallo | incerto | residui |\n|---|---:|---:|---:|---:|---:|---:|---:|\n")
	for _, t := range []string{candStorica, candScansione} {
		c := r.candidate[t]
		p("| %s | %d | %d | %d | %d | %d | %d | %d |\n", t, c[precGiorno], c[precMese], c[precAnno], c[precDecennio], c[precIntervallo], c[precIncerto], c["residui"])
	}
	p("\nResidui, da sistemare a mano in `arricchimento/foto/<pid>.yaml` con `datazione:`: %d.\n\n", len(r.residui))
	if len(r.residui) > 0 {
		p("| pid | user3 | EXIF | Caricata |\n|---:|---|---|---|\n%s\n\n", strings.Join(r.residui, "\n"))
	}
	p("Scansioni da controllare: %d.\n\n", len(r.scansioni))
	if len(r.scansioni) > 0 {
		p("| pid | user3 | EXIF | Caricata | Datazione |\n|---:|---|---|---|---|\n%s\n\n", strings.Join(r.scansioni, "\n"))
	}
	p("## Riservate\n\nFoto in album non pubblici di Coppermine, con `riservata: true` e fuori dalla Gallery pubblica: %d.\n\n", r.riservate)
	p("Album riservati senza decisione in `arricchimento/riservati.yaml` (`pubblica` o `cancella`): %d.\n\n", len(r.senzaDecisione))
	if len(r.senzaDecisione) > 0 {
		p("| aid | Album | Visibilità | Proprietario | Foto |\n|---:|---|---|---|---:|\n%s\n\n", strings.Join(r.senzaDecisione, "\n"))
	}
	p("## Spariti\n\nYAML spariti rispetto all'Importazione precedente: %d.\n", len(r.spariti))
	if len(r.spariti) > 0 {
		p("\n%s\n", unisci(r.spariti))
	}
	if r.mappature != nil {
		p("\n")
		b.Write(r.mappature.markdown())
	}
	return b.Bytes()
}

// senzaDecisione elenca gli album riservati senza riga in riservati.yaml, con proprietario
// e numero di Foto: sono quelli da decidere in chat sui provini (#35).
func senzaDecisione(d *datiCoppermine, decisioni map[int]string) []string {
	var righe []string
	for _, aid := range chiaviOrdinate(d.riservati) {
		if decisioni[aid] != "" {
			continue
		}
		a, vis := d.album.Album[aid], d.riservati[aid]
		// Il proprietario è nella visibility (10000 + user_id) o nella categoria degli album personali.
		chi := "—"
		if id := max(vis, a.Categoria) - cidPersonali; id > 0 {
			chi = fmt.Sprintf("%d %s", id, d.utenti[id])
		}
		var v string
		switch {
		case vis >= cidPersonali:
			v = "solo il proprietario"
		case vis == 2:
			v = "utenti registrati"
		case vis == 102:
			v = "amministratori"
		default:
			v = fmt.Sprintf("gruppo %d", vis)
		}
		righe = append(righe, fmt.Sprintf("| %d | %s | %s | %s | %d |", aid, cella(a.Title), v, cella(strings.TrimSpace(chi)), a.Foto))
	}
	return righe
}

// cella rende un testo libero sicuro dentro una tabella Markdown.
func cella(s string) string {
	return strings.NewReplacer("|", "\\|", "\n", " ").Replace(s)
}

func unisci(v []int) string {
	s := make([]string, len(v))
	for i, n := range v {
		s[i] = strconv.Itoa(n)
	}
	return strings.Join(s, ", ")
}

// registra fa il commit di percorso nel Repository dei contenuti, che è solo locale.
// autore ("Nome <email>") è chi ha fatto la modifica; il committer è sempre gerbillo.
func registra(dir, percorso, msg, autore string) error {
	git := func(a ...string) *exec.Cmd {
		c := exec.Command("git", append([]string{"-C", dir, "-c", "user.name=gerbillo", "-c", "user.email=gerbillo@localhost"}, a...)...)
		c.Stderr = os.Stderr
		return c
	}
	if _, err := os.Stat(filepath.Join(dir, ".git")); errors.Is(err, os.ErrNotExist) {
		if err := git("init", "-q").Run(); err != nil {
			return err
		}
	}
	if err := git("add", "-A", percorso).Run(); err != nil {
		return err
	}
	if git("diff", "--cached", "--quiet").Run() == nil {
		fmt.Println("nessun cambiamento da registrare")
		return nil
	}
	a := []string{"commit", "-q", "-m", msg}
	if autore != "" {
		a = append(a, "--author", autore)
	}
	return git(a...).Run()
}
