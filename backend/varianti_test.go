package main

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"

	"github.com/gen2brain/webp"
)

// jpegConOrientamento crea un JPEG w×h con un segmento EXIF che dichiara l'orientamento o.
func jpegConOrientamento(t *testing.T, w, h, o int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	img.Set(0, 0, color.RGBA{255, 0, 0, 255})
	var j bytes.Buffer
	if err := jpeg.Encode(&j, img, nil); err != nil {
		t.Fatal(err)
	}
	tiff := []byte("MM\x00\x2a\x00\x00\x00\x08\x00\x01")
	tiff = append(tiff, 0x01, 0x12, 0x00, 0x03, 0, 0, 0, 1, byte(o>>8), byte(o), 0, 0)
	tiff = append(tiff, 0, 0, 0, 0)
	seg := append([]byte("Exif\x00\x00"), tiff...)
	app1 := []byte{0xff, 0xe1, 0, 0}
	binary.BigEndian.PutUint16(app1[2:], uint16(len(seg)+2))
	b := j.Bytes()
	return append(append(append([]byte{}, b[:2]...), append(app1, seg...)...), b[2:]...)
}

func TestOrientamento(t *testing.T) {
	for _, o := range []int{1, 3, 6, 8} {
		if got := orientamento(jpegConOrientamento(t, 8, 4, o)); got != o {
			t.Errorf("orientamento %d letto come %d", o, got)
		}
	}
	if orientamento([]byte("\x89PNG")) != 1 {
		t.Error("PNG senza EXIF")
	}
}

func TestRaddrizza(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 4, 2))
	img.Set(0, 0, color.RGBA{255, 0, 0, 255})
	r := raddrizza(img, 6) // 90° in senso orario: l'angolo in alto a sinistra va in alto a destra
	if b := r.Bounds(); b.Dx() != 2 || b.Dy() != 4 {
		t.Fatalf("dimensioni %v", b)
	}
	if c := color.RGBAModel.Convert(r.At(1, 0)).(color.RGBA); c.R != 255 {
		t.Errorf("pixel ruotato nel posto sbagliato: %v", c)
	}
}

func TestRidimensiona(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 2000, 1000))
	if b := ridimensiona(img, 400).Bounds(); b.Dx() != 400 || b.Dy() != 200 {
		t.Errorf("400: %v", b)
	}
	if b := ridimensiona(img, 4000).Bounds(); b.Dx() != 2000 {
		t.Errorf("non deve ingrandire: %v", b)
	}
}

func TestScriviVarianti(t *testing.T) {
	dir := t.TempDir()
	dim, err := scriviVarianti(dir, jpegConOrientamento(t, 1800, 900, 6))
	if err != nil {
		t.Fatal(err)
	}
	if len(dim) != len(varianti) || !presenti(dir) {
		t.Fatalf("varianti mancanti: %v", dim)
	}
	b, _ := os.ReadFile(filepath.Join(dir, "schermo.webp"))
	c, err := webp.DecodeConfig(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	if c.Width != 800 || c.Height != 1600 { // ruotata e ridotta a 1600 sul lato lungo
		t.Errorf("schermo %d×%d", c.Width, c.Height)
	}
	if bytes.Contains(b, []byte("EXIF")) || bytes.Contains(b, []byte("Exif")) {
		t.Error("la Variante contiene EXIF")
	}
}
