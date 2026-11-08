package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"net/http/httptest"
	"regexp"
	"slices"
	"strconv"
	"testing"
)

// Rotte che rispondono senza utente: i reindirizzamenti alla barra, i file statici e l'uscita.
var rotteSenzaUtente = []string{"/admin", "/admin/foto", "/admin/statico/", "/admin/esci"}

// rotteRegistrate legge da backend.go i percorsi che rotteBackend registra: una rotta
// nuova entra nel test da sola (#92).
func rotteRegistrate(t *testing.T) []string {
	f, err := parser.ParseFile(token.NewFileSet(), "backend.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	var rotte []string
	ast.Inspect(f, func(n ast.Node) bool {
		c, ok := n.(*ast.CallExpr)
		if !ok || len(c.Args) == 0 {
			return true
		}
		s, ok := c.Fun.(*ast.SelectorExpr)
		if !ok || (s.Sel.Name != "Handle" && s.Sel.Name != "HandleFunc") {
			return true
		}
		if x, _ := s.X.(*ast.Ident); x == nil || x.Name != "mux" {
			return true
		}
		lit, ok := c.Args[0].(*ast.BasicLit)
		if !ok {
			t.Fatalf("rotta registrata senza un percorso letterale: %#v", c.Args[0])
		}
		p, _ := strconv.Unquote(lit.Value)
		if !slices.Contains(rotte, p) {
			rotte = append(rotte, p)
		}
		return true
	})
	if len(rotte) < 10 {
		t.Fatalf("trovate solo %d rotte: %v", len(rotte), rotte)
	}
	return rotte
}

// Ogni rotta del Backend, sul mux vero, chiede l'accesso a un ospite, rifiuta chi non è
// Moderatore e rifiuta un POST da un altro sito (#92).
func TestPermessiRotte(t *testing.T) {
	rotte := rotteRegistrate(t)
	moderatore := &utente{ID: 2, Nome: "admin", Gruppi: []string{gruppoAdmin}}
	for _, red := range []*redazione{redazioneProva(t), nil} {
		for _, caso := range []struct {
			nome   string
			u      *utente
			metodo string
			sito   string // Sec-Fetch-Site
			atteso int
		}{
			{"ospite", nil, "GET", "", http.StatusUnauthorized},
			{"senza permessi", &utente{ID: 3, Nome: "utente"}, "GET", "", http.StatusForbidden},
			{"POST da un altro sito", moderatore, "POST", "cross-site", http.StatusForbidden},
		} {
			mux := http.NewServeMux()
			rotteBackend(mux, &utenteFinto{u: caso.u}, red)
			for _, p := range rotte {
				req := httptest.NewRequest(caso.metodo, p, nil)
				if caso.sito != "" {
					req.Header.Set("Sec-Fetch-Site", caso.sito)
				}
				if _, pattern := mux.Handler(req); pattern != p || slices.Contains(rotteSenzaUtente, p) {
					continue // non registrata senza redazione, o volutamente aperta
				}
				w := httptest.NewRecorder()
				mux.ServeHTTP(w, req)
				if w.Code != caso.atteso {
					t.Errorf("%s, redazione %v, %s %s: HTTP %d, atteso %d", caso.nome, red != nil, caso.metodo, p, w.Code, caso.atteso)
				}
			}
		}
	}
}

// Le pagine del Backend, come le genera il server, non caricano codice o stili da altri domini:
// solo immagini (tessere della mappa, Foto della scheda). TestStaticiBackend guarda i modelli (#93).
func TestPagineSenzaCodiceEsterno(t *testing.T) {
	r := redazioneConIndice(t)
	mux := http.NewServeMux()
	rotteBackend(mux, &utenteFinto{u: &utente{ID: 2, Nome: "admin", Gruppi: []string{gruppoAdmin}}}, r)
	esterno := regexp.MustCompile(`<(script|link)[^>]*(src|href)="(https?:)?//`)
	for _, p := range []string{"/admin/", "/admin/foto/", "/admin/foto/1", "/admin/foto/2", "/admin/operazioni"} {
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest("GET", p, nil))
		if w.Code != http.StatusOK {
			t.Errorf("%s: HTTP %d", p, w.Code)
		}
		if m := esterno.FindString(w.Body.String()); m != "" {
			t.Errorf("%s carica da un altro dominio: %s", p, m)
		}
	}
}
