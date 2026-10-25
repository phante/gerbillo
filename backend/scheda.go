package main

// La Foto nella scheda del Backend (#55). La scheda è la stessa tra anteprima e definitivo:
// usa mini(), schermo() e originale() del build, che con -media restituiscono i file di
// Coppermine e altrimenti le Varianti WebP (l'Originale su S3 non è pubblico). Dimensioni
// e peso si leggono dal file vero, con una richiesta HTTP parziale, e si ricordano.

import (
	"bufio"
	"flag"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "golang.org/x/image/webp"
)

// leggiArgsBuild legge le opzioni del build da un file con un argomento per riga (seme/build.args).
func leggiArgsBuild(p string) ([]string, opzioniBuild, error) {
	var o opzioniBuild
	f, err := os.Open(p)
	if err != nil {
		return nil, o, err
	}
	defer f.Close()
	var args []string
	for sc := bufio.NewScanner(f); sc.Scan(); {
		args = append(args, sc.Text())
	}
	fs := flag.NewFlagSet("build.args", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	o.flag(fs)
	if err := fs.Parse(args); err != nil {
		return nil, o, fmt.Errorf("%s: %w", p, err)
	}
	return args, o, nil
}

// fileMedia è un file della Foto, come si mostra nella scheda.
type fileMedia struct {
	Nome, URL          string
	Larghezza, Altezza int
	Peso               string
	Errore             string
}

type infoMedia struct {
	larghezza, altezza int
	byte               int64
	errore             string
}

var (
	muInfo      sync.Mutex
	cacheInfo   = map[string]infoMedia{}
	clientMedia = &http.Client{Timeout: 15 * time.Second}
)

// leggiInfoMedia scarica l'inizio del file: bastano poche decine di kB per le dimensioni, e
// il peso viene da Content-Range o Content-Length.
func leggiInfoMedia(u string) infoMedia {
	muInfo.Lock()
	if i, ok := cacheInfo[u]; ok {
		muInfo.Unlock()
		return i
	}
	muInfo.Unlock()
	var i infoMedia
	req, err := http.NewRequest("GET", u, nil)
	if err == nil {
		req.Header.Set("Range", "bytes=0-262143")
		var r *http.Response
		if r, err = clientMedia.Do(req); err == nil {
			defer r.Body.Close()
			switch {
			case r.StatusCode == http.StatusPartialContent:
				if _, tot, ok := strings.Cut(r.Header.Get("Content-Range"), "/"); ok {
					i.byte, _ = strconv.ParseInt(tot, 10, 64)
				}
			case r.StatusCode == http.StatusOK:
				i.byte = r.ContentLength
			default:
				err = fmt.Errorf("HTTP %d", r.StatusCode)
			}
			if err == nil {
				if c, _, e := image.DecodeConfig(io.LimitReader(r.Body, 262144)); e == nil {
					i.larghezza, i.altezza = c.Width, c.Height
				}
			}
		}
	}
	if err != nil {
		i.errore = err.Error()
		return i // un errore non si ricorda: magari è passeggero
	}
	muInfo.Lock()
	if len(cacheInfo) > 10000 {
		cacheInfo = map[string]infoMedia{}
	}
	cacheInfo[u] = i
	muInfo.Unlock()
	return i
}

func peso(n int64) string {
	switch {
	case n <= 0:
		return ""
	case n < 1<<20:
		return fmt.Sprintf("%d kB", (n+512)/1024)
	}
	return strings.Replace(fmt.Sprintf("%.1f MB", float64(n)/(1<<20)), ".", ",", 1)
}

// assoluto rende assoluto un URL della Gallery: senza -media le Varianti hanno solo il percorso.
func (o opzioniBuild) assoluto(u string) string {
	if u == "" || strings.Contains(u, "://") {
		return u
	}
	return strings.TrimSuffix(o.pubblico, "/") + "/" + strings.TrimPrefix(strings.TrimPrefix(u, o.base), "/")
}

func (r *redazione) builderScheda() *builder {
	return &builder{opzioniBuild: r.opz}
}

// fileFoto risponde a /admin/foto/<pid>/file, che la scheda chiede dopo l'apertura: le
// richieste ai file possono richiedere qualche secondo.
func (r *redazione) fileDellaFoto(w http.ResponseWriter, req *http.Request, pid int) {
	pub, _, _, err := r.fotoDaModificare(pid)
	if err != nil {
		http.NotFound(w, req)
		return
	}
	b := r.builderScheda()
	var fs []fileMedia
	for _, x := range []struct{ nome, url string }{{"Originale", b.originale(pub)}, {"Schermo", b.schermo(pub)}, {"Mini", b.mini(pub)}} {
		f := fileMedia{Nome: x.nome, URL: r.opz.assoluto(x.url)}
		if f.URL == "" {
			f.Errore = "non pubblico"
			if x.nome == "Schermo" {
				f.Errore = "nessuno per questo Formato"
			}
			fs = append(fs, f)
			continue
		}
		i := leggiInfoMedia(f.URL)
		f.Larghezza, f.Altezza, f.Peso, f.Errore = i.larghezza, i.altezza, peso(i.byte), i.errore
		fs = append(fs, f)
	}
	mostraFrammento(w, "file-foto", fs)
}

// testata è la parte alta della scheda: la Foto e le informazioni tecniche.
type testata struct {
	Immagine, Formato, Caricata, Contributore string
	Larghezza, Altezza                        int
	Aid                                       int
	Album, LinkAlbum                          string
	Exif, User3                               string
}

func (r *redazione) testataDi(pub *fotoPubblicata) testata {
	b := r.builderScheda()
	t := testata{Immagine: r.opz.assoluto(b.schermo(pub)), Formato: pub.Formato, Larghezza: pub.Larghezza, Altezza: pub.Altezza,
		Aid: pub.Coppermine.Aid, Exif: pub.Coppermine.DataExif, User3: pub.Coppermine.Data}
	if t.Immagine == "" {
		t.Immagine = r.opz.assoluto(b.mini(pub))
	}
	if strings.HasPrefix(t.Immagine, r.opz.base+"statico/") || strings.HasPrefix(t.Immagine, "statico/") {
		t.Immagine = "" // l'icona del documento: nella scheda basta il Formato
	}
	if d, err := time.Parse(time.RFC3339, pub.Date); err == nil {
		t.Caricata = d.In(roma).Format("02/01/2006 15:04")
	}
	if c := pub.Contributore; c != nil {
		t.Contributore = fmt.Sprintf("%s (%d)", c.Nome, c.ID)
	}
	if idx := r.indiceCorrente(); idx.perPid != nil {
		if i, ok := idx.perPid[pub.Pid]; ok {
			t.Album = idx.Foto[i].Album
		}
	}
	t.LinkAlbum = fmt.Sprintf("https://progettodighe.it/gallery/thumbnails.php?album=%d", t.Aid)
	return t
}
