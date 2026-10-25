package main

// Punto geografico nel Backend (#54): il campo testo accetta i formati che si copiano più
// spesso, e la mappa del modulo (Leaflet, lo stesso di mappa.js della diga, #27) lo
// imposta con un pin. Si salva sempre come «latitudine, longitudine» in decimali.

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

var (
	// Gradi, minuti e secondi: 46°17'26"N 8°17'55"E, anche con i primi e secondi tipografici
	// o in decimali, e con l'emisfero prima o dopo.
	reGMS = regexp.MustCompile(`(?i)([NSEWO]?)\s*(\d{1,3}(?:[.,]\d+)?)\s*°\s*(?:(\d{1,2}(?:[.,]\d+)?)\s*['′’]\s*)?(?:(\d{1,2}(?:[.,]\d+)?)\s*(?:"|″|”|''|′′)\s*)?([NSEWO]?)`)
	// Link di Google Maps: !3d…!4d… (il punto del luogo), @lat,lon, q= o ll=.
	reGoogle3d4d = regexp.MustCompile(`!3d(-?\d+(?:\.\d+)?)!4d(-?\d+(?:\.\d+)?)`)
	reGoogleAt   = regexp.MustCompile(`@(-?\d+(?:\.\d+)?),(-?\d+(?:\.\d+)?)`)
	reCoppia     = regexp.MustCompile(`^(-?\d+(?:\.\d+)?),\s*(-?\d+(?:\.\d+)?)$`)
)

// interpretaPunto legge latitudine e longitudine da uno dei formati ammessi.
func interpretaPunto(s string) (lat, lon float64, err error) {
	s = strings.TrimSpace(s)
	errore := fmt.Errorf("punto geografico %q: scrivi latitudine, longitudine (es. 46.2905, 8.2987), gradi e minuti (46°17'26\"N 8°17'55\"E) o incolla un link di Google Maps", s)
	num := func(x string) float64 { v, _ := strconv.ParseFloat(strings.ReplaceAll(x, ",", "."), 64); return v }
	ok := false
	switch {
	case strings.Contains(s, "://") || strings.HasPrefix(s, "maps."):
		lat, lon, ok = puntoDaLink(s)
	case strings.Contains(s, "°"):
		lat, lon, ok = puntoGMS(s, num)
	default:
		// La virgola decimale vale solo con il punto e virgola tra i due numeri:
		// «46,2905, 8,2987» è ambiguo e si rifiuta.
		var parti []string
		switch {
		case strings.Count(s, ";") == 1:
			parti = strings.Split(s, ";")
		case strings.Count(s, ",") == 1:
			parti = strings.Split(s, ",")
		case !strings.Contains(s, ","):
			parti = strings.Fields(s)
		}
		if len(parti) == 2 {
			var e1, e2 error
			lat, e1 = strconv.ParseFloat(strings.ReplaceAll(strings.TrimSpace(parti[0]), ",", "."), 64)
			lon, e2 = strconv.ParseFloat(strings.ReplaceAll(strings.TrimSpace(parti[1]), ",", "."), 64)
			ok = e1 == nil && e2 == nil
		}
	}
	if !ok || math.IsNaN(lat) || lat < -90 || lat > 90 || lon < -180 || lon > 180 {
		return 0, 0, errore
	}
	return lat, lon, nil
}

func puntoDaLink(s string) (float64, float64, bool) {
	f := func(a, b string) (float64, float64, bool) {
		x, e1 := strconv.ParseFloat(a, 64)
		y, e2 := strconv.ParseFloat(b, 64)
		return x, y, e1 == nil && e2 == nil
	}
	if m := reGoogle3d4d.FindStringSubmatch(s); m != nil {
		return f(m[1], m[2])
	}
	if u, err := url.Parse(s); err == nil {
		for _, k := range []string{"q", "query", "ll", "center", "destination", "daddr"} {
			if m := reCoppia.FindStringSubmatch(strings.TrimSpace(u.Query().Get(k))); m != nil {
				return f(m[1], m[2])
			}
		}
	}
	if m := reGoogleAt.FindStringSubmatch(s); m != nil {
		return f(m[1], m[2])
	}
	return 0, 0, false
}

func puntoGMS(s string, num func(string) float64) (float64, float64, bool) {
	var v [2]float64
	var emisferi [2]string
	n := 0
	for _, m := range reGMS.FindAllStringSubmatch(s, -1) {
		if n == 2 || m[2] == "" {
			break
		}
		v[n] = num(m[2]) + num(m[3])/60 + num(m[4])/3600
		emisferi[n] = strings.ToUpper(m[1] + m[5])
		n++
	}
	if n != 2 {
		return 0, 0, false
	}
	// L'ordine è latitudine e longitudine, salvo emisferi che dicono il contrario.
	if strings.ContainsAny(emisferi[0], "EWO") && strings.ContainsAny(emisferi[1], "NS") {
		v[0], v[1] = v[1], v[0]
		emisferi[0], emisferi[1] = emisferi[1], emisferi[0]
	}
	if strings.Contains(emisferi[0], "S") {
		v[0] = -v[0]
	}
	if strings.ContainsAny(emisferi[1], "WO") {
		v[1] = -v[1]
	}
	return v[0], v[1], true
}

// testoPunto è il Punto geografico come lo mostra il campo: decimali, al più 6 cifre (10 cm).
func testoPunto(lat, lon float64) string {
	r := func(x float64) string { return strconv.FormatFloat(math.Round(x*1e6)/1e6, 'f', -1, 64) }
	return r(lat) + ", " + r(lon)
}

// punto risponde a /admin/punto?v=…: il campo lo chiede per spostare il pin mentre si scrive.
func punto(w http.ResponseWriter, req *http.Request) {
	lat, lon, err := interpretaPunto(req.URL.Query().Get("v"))
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(map[string]string{"errore": err.Error()})
		return
	}
	json.NewEncoder(w).Encode(map[string]any{"lat": lat, "lon": lon, "testo": testoPunto(lat, lon)})
}
