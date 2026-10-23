package main

import (
	"html"
	"strings"
	"unicode/utf8"
)

// Esito della decodifica di un campo testuale di Coppermine.
type esitoTesto int

const (
	inUTF8          esitoTesto = iota
	inCP1252                   // non era UTF-8 valido: letto come cp1252
	nonDecodificato            // cp1252 con byte non definiti, sostituiti da U+FFFD
)

// decodifica applica la regola di G1 (#9) ai byte grezzi di una colonna latin1:
// UTF-8 se valido, altrimenti cp1252, poi html.unescape. Non c'è mai doppio encoding.
func decodifica(b []byte) (string, esitoTesto) {
	var s string
	esito := inUTF8
	if utf8.Valid(b) {
		s = string(b)
	} else {
		s, esito = daCP1252(b), inCP1252
		if strings.ContainsRune(s, utf8.RuneError) {
			esito = nonDecodificato
		}
	}
	s = strings.ReplaceAll(html.UnescapeString(s), "\r\n", "\n")
	return strings.TrimSpace(s), esito
}

// cp1252 fra 0x80 e 0x9F; il resto coincide con latin1. Zero = byte non definito.
var cp1252 = [32]rune{
	0x20AC, 0, 0x201A, 0x0192, 0x201E, 0x2026, 0x2020, 0x2021,
	0x02C6, 0x2030, 0x0160, 0x2039, 0x0152, 0, 0x017D, 0,
	0, 0x2018, 0x2019, 0x201C, 0x201D, 0x2022, 0x2013, 0x2014,
	0x02DC, 0x2122, 0x0161, 0x203A, 0x0153, 0, 0x017E, 0x0178,
}

func daCP1252(b []byte) string {
	var sb strings.Builder
	for _, c := range b {
		switch {
		case c >= 0x80 && c <= 0x9F && cp1252[c-0x80] == 0:
			sb.WriteRune(utf8.RuneError)
		case c >= 0x80 && c <= 0x9F:
			sb.WriteRune(cp1252[c-0x80])
		default:
			sb.WriteRune(rune(c))
		}
	}
	return sb.String()
}

// mojibake segnala UTF-8 decodificato due volte: il gate di G1 ne vuole zero.
func mojibake(s string) bool {
	return strings.Contains(s, "Ã") || strings.Contains(s, "Â") || strings.Contains(s, "â€")
}
