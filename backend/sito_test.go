package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCopieSito verifica che le copie in modelli/sito siano ancora identiche ai
// file del Sito principale (#25). Si salta se il repo del sito non è accanto a questo.
func TestCopieSito(t *testing.T) {
	tema := filepath.Join("..", "..", "progettodighe", "themes", "progettodighe")
	if _, err := os.Stat(tema); err != nil {
		t.Skip("repo del sito non trovato in ../progettodighe")
	}
	copie := map[string]string{
		"tachyons.css": "assets/asset/css/tachyons.css",
		"main.css":     "assets/asset/css/main.css",
		"map.css":      "assets/asset/css/map.css",
		"search.css":   "assets/asset/css/search.css",
		"PDLogo.svg":   "static/asset/brand/PDLogo.svg",
	}
	icone, _ := filepath.Glob("modelli/sito/icone/*.svg")
	for _, p := range icone {
		n := filepath.Base(p)
		switch {
		case n == "cc.svg" || n == "by.svg" || n == "sa.svg":
			copie["icone/"+n] = "static/asset/brand/cc/" + n
		case esiste(filepath.Join(tema, "layouts/_partials/svg/simbols", n)):
			copie["icone/"+n] = "layouts/_partials/svg/simbols/" + n
		default:
			copie["icone/"+n] = "layouts/_partials/svg/brands/" + strings.TrimSuffix(n, ".svg") + "-simpleicons.org.svg"
		}
	}
	for qui, la := range copie {
		a, err1 := os.ReadFile(filepath.Join("modelli/sito", qui))
		b, err2 := os.ReadFile(filepath.Join(tema, la))
		if err1 != nil || err2 != nil {
			t.Errorf("%s: %v %v", qui, err1, err2)
			continue
		}
		if !bytes.Equal(a, b) {
			t.Errorf("modelli/sito/%s diverge da %s: ricopialo dal sito", qui, la)
		}
	}
}
