package main

// Varianti delle Foto (G3, #11; ADR 0005): WebP generati dagli Originali, serviti a
// /gallery/f/<pid>/<variante>.webp. Il comando è idempotente e genera solo ciò che manca.
// Le Varianti non portano mai EXIF: si codificano i pixel, non il file.

import (
	"bytes"
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/gen2brain/webp"
	"golang.org/x/image/draw"
	"gopkg.in/yaml.v3"
)

type variante struct {
	nome    string
	lato    int // lato lungo massimo in pixel; mai ingrandire
	qualita int
}

// metodoWebP 2 invece del 4 predefinito: 2,6 volte più veloce, file più grandi dell'1,5% circa.
const metodoWebP = 2

var varianti = []variante{
	{"mini", 400, 75},     // griglie e Copia per il forum
	{"schermo", 1600, 80}, // visualizzatore
}

type configVarianti struct {
	Contenuti string
	Originali string // cartella albums/ di Coppermine, oppure il suo URL pubblico
	Varianti  string // cartella servita come /gallery/f/
}

type fotoOriginale struct {
	pid                int
	formato            string
	filepath, filename string
}

func generaVarianti(args []string) error {
	fs := flag.NewFlagSet("varianti", flag.ExitOnError)
	conf := fs.String("config", "gerbillo.toml", "file di configurazione TOML")
	campione := fs.Int("campione", 0, "solo una Foto ogni N (0: tutte)")
	lavori := fs.Int("j", 1, "Foto in parallelo")
	fs.Parse(args)
	var c configVarianti
	if _, err := toml.DecodeFile(*conf, &c); err != nil {
		return err
	}
	if c.Contenuti == "" || c.Originali == "" || c.Varianti == "" {
		return errors.New("servono contenuti, originali e varianti nella configurazione")
	}
	foto, err := leggiFotoImportate(c.Contenuti)
	if err != nil {
		return err
	}
	if *campione > 1 {
		var s []fotoOriginale
		for i, f := range foto {
			if i%*campione == 0 {
				s = append(s, f)
			}
		}
		foto = s
	}
	var (
		mu         sync.Mutex
		fatte, gia atomic.Int64
		altri      atomic.Int64
		byteOrig   atomic.Int64
		byteVar    = make([]atomic.Int64, len(varianti))
		tempo      atomic.Int64 // nanosecondi di decodifica + codifica, sommati sui lavori
		errori     []string
		coda       = make(chan fotoOriginale)
		wg         sync.WaitGroup
		inizio     = time.Now()
		client     = &http.Client{Timeout: time.Minute}
	)
	for range max(*lavori, 1) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for f := range coda {
				if f.formato != "immagine" {
					altri.Add(1) // PDF e altro: icona generica nella Gallery pubblica
					continue
				}
				dir := filepath.Join(c.Varianti, strconv.Itoa(f.pid))
				if presenti(dir) {
					gia.Add(1)
					continue
				}
				b, err := leggiOriginale(client, c.Originali, f)
				if err == nil {
					t := time.Now()
					var dim []int
					dim, err = scriviVarianti(dir, b)
					tempo.Add(int64(time.Since(t)))
					for i, d := range dim {
						byteVar[i].Add(int64(d))
					}
				}
				if err != nil {
					mu.Lock()
					errori = append(errori, fmt.Sprintf("%d %s%s: %v", f.pid, f.filepath, f.filename, err))
					mu.Unlock()
					continue
				}
				byteOrig.Add(int64(len(b)))
				fatte.Add(1)
			}
		}()
	}
	for _, f := range foto {
		coda <- f
	}
	close(coda)
	wg.Wait()

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	n := max(fatte.Load(), 1)
	fmt.Printf("Foto: %d, generate %d, già presenti %d, non immagini %d, errori %d\n",
		len(foto), fatte.Load(), gia.Load(), altri.Load(), len(errori))
	fmt.Printf("tempo totale %s, per Foto %s di elaborazione; memoria Go massima %d MiB\n",
		time.Since(inizio).Round(time.Second), (time.Duration(tempo.Load()) / time.Duration(n)).Round(time.Millisecond), m.Sys>>20)
	fmt.Printf("Originali %.1f MiB (media %d KiB)\n", float64(byteOrig.Load())/(1<<20), byteOrig.Load()/n>>10)
	for i, v := range varianti {
		fmt.Printf("%s: %.1f MiB (media %d KiB)\n", v.nome, float64(byteVar[i].Load())/(1<<20), byteVar[i].Load()/n>>10)
	}
	slices.Sort(errori)
	for _, e := range errori {
		fmt.Fprintln(os.Stderr, "errore:", e)
	}
	return nil
}

func presenti(dir string) bool {
	for _, v := range varianti {
		if _, err := os.Stat(filepath.Join(dir, v.nome+".webp")); err != nil {
			return false
		}
	}
	return true
}

// leggiFotoImportate legge pid, Formato e percorso dell'Originale dai YAML di G1.
func leggiFotoImportate(dir string) ([]fotoOriginale, error) {
	file, err := filepath.Glob(filepath.Join(dir, "importazione", "foto", "*.yaml"))
	if err != nil {
		return nil, err
	}
	var out []fotoOriginale
	for _, p := range file {
		b, err := os.ReadFile(p)
		if err != nil {
			return nil, err
		}
		var y struct {
			Pid        int
			Formato    string
			Coppermine struct{ Filepath, Filename string }
		}
		if err := yaml.Unmarshal(b, &y); err != nil {
			return nil, fmt.Errorf("%s: %w", p, err)
		}
		out = append(out, fotoOriginale{y.Pid, y.Formato, y.Coppermine.Filepath, y.Coppermine.Filename})
	}
	slices.SortFunc(out, func(a, b fotoOriginale) int { return a.pid - b.pid })
	return out, nil
}

func leggiOriginale(client *http.Client, base string, f fotoOriginale) ([]byte, error) {
	if !strings.HasPrefix(base, "http://") && !strings.HasPrefix(base, "https://") {
		return os.ReadFile(filepath.Join(base, f.filepath, f.filename))
	}
	var parti []string
	for _, s := range strings.Split(strings.Trim(f.filepath, "/")+"/"+f.filename, "/") {
		parti = append(parti, url.PathEscape(s))
	}
	r, err := client.Get(strings.TrimSuffix(base, "/") + "/" + strings.Join(parti, "/"))
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()
	if r.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", r.StatusCode)
	}
	return io.ReadAll(r.Body)
}

// scriviVarianti decodifica l'Originale, lo raddrizza secondo l'EXIF e scrive le
// Varianti. Restituisce la dimensione in byte di ciascuna.
func scriviVarianti(dir string, b []byte) ([]int, error) {
	img, err := decodificaImmagine(b)
	if err != nil {
		return nil, err
	}
	img = raddrizza(img, orientamento(b))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	var dim []int
	for _, v := range varianti {
		var buf bytes.Buffer
		if err := webp.Encode(&buf, ridimensiona(img, v.lato), webp.Options{Quality: v.qualita, Method: metodoWebP}); err != nil {
			return nil, err
		}
		p := filepath.Join(dir, v.nome+".webp")
		if err := os.WriteFile(p+".tmp", buf.Bytes(), 0o644); err != nil {
			return nil, err
		}
		if err := os.Rename(p+".tmp", p); err != nil {
			return nil, err
		}
		dim = append(dim, buf.Len())
	}
	return dim, nil
}

func decodificaImmagine(b []byte) (image.Image, error) {
	switch {
	case bytes.HasPrefix(b, []byte("\xff\xd8")):
		return jpeg.Decode(bytes.NewReader(b))
	case bytes.HasPrefix(b, []byte("\x89PNG")):
		return png.Decode(bytes.NewReader(b))
	case bytes.HasPrefix(b, []byte("GIF8")):
		return gif.Decode(bytes.NewReader(b)) // solo il primo fotogramma
	}
	return nil, errors.New("formato immagine non riconosciuto")
}

func ridimensiona(img image.Image, lato int) image.Image {
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	if max(w, h) <= lato {
		return img
	}
	if w >= h {
		w, h = lato, max(1, h*lato/w)
	} else {
		w, h = max(1, w*lato/h), lato
	}
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, img.Bounds(), draw.Src, nil)
	return dst
}

// orientamento legge il tag EXIF Orientation (0x0112) da un JPEG; 1 se assente.
func orientamento(b []byte) int {
	if !bytes.HasPrefix(b, []byte("\xff\xd8")) {
		return 1
	}
	for i := 2; i+4 <= len(b) && b[i] == 0xff; {
		marker, n := b[i+1], int(binary.BigEndian.Uint16(b[i+2:]))
		if marker == 0xda || i+2+n > len(b) { // inizio dei dati: niente più segmenti
			break
		}
		seg := b[i+4 : i+2+n]
		if marker == 0xe1 && bytes.HasPrefix(seg, []byte("Exif\x00\x00")) {
			return orientamentoTIFF(seg[6:])
		}
		i += 2 + n
	}
	return 1
}

func orientamentoTIFF(t []byte) int {
	if len(t) < 8 {
		return 1
	}
	var o binary.ByteOrder = binary.BigEndian
	if string(t[:2]) == "II" {
		o = binary.LittleEndian
	}
	ifd := int(o.Uint32(t[4:]))
	if ifd+2 > len(t) {
		return 1
	}
	for k, n := 0, int(o.Uint16(t[ifd:])); k < n; k++ {
		e := ifd + 2 + 12*k
		if e+12 > len(t) {
			break
		}
		if o.Uint16(t[e:]) == 0x0112 {
			if v := int(o.Uint16(t[e+8:])); v >= 1 && v <= 8 {
				return v
			}
		}
	}
	return 1
}

// raddrizza applica una delle otto orientazioni EXIF.
func raddrizza(img image.Image, o int) image.Image {
	if o <= 1 || o > 8 {
		return img
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	dw, dh := w, h
	if o >= 5 {
		dw, dh = h, w
	}
	dst := image.NewNRGBA(image.Rect(0, 0, dw, dh))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var dx, dy int
			switch o {
			case 2:
				dx, dy = w-1-x, y
			case 3:
				dx, dy = w-1-x, h-1-y
			case 4:
				dx, dy = x, h-1-y
			case 5:
				dx, dy = y, x
			case 6:
				dx, dy = h-1-y, x
			case 7:
				dx, dy = h-1-y, w-1-x
			case 8:
				dx, dy = y, w-1-x
			}
			dst.Set(dx, dy, img.At(b.Min.X+x, b.Min.Y+y))
		}
	}
	return dst
}
