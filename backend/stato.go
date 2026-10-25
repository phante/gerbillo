package main

// Stato del server (ADR 0011): la pagina /admin/stato con i dettagli, e /admin/widget (JSON)
// per il widget in tutte le pagine del Backend (#48), che si aggiorna ogni 5 secondi.

import (
	"bufio"
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

var dirCgroup = "/sys/fs/cgroup/"

func fileHost() string { return envOr("GERBILLO_STATO_HOST", "/stato/stato.json") }

type corpoStato struct {
	Avvio, Da            string
	Heap, Sys, Goroutine uint64
	Memoria              memoria
	Cgroup               [][2]string
	Host                 string
}

// stato mostra la memoria del processo, quella del container (cgroup v2)
// e il file scritto dall'host dal timer (montato in sola lettura).
func stato(w http.ResponseWriter, r *http.Request) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	c := corpoStato{Avvio: avvio.In(roma).Format("02/01/2006 15:04:05"), Da: time.Since(avvio).Round(time.Second).String(),
		Heap: m.HeapAlloc >> 20, Sys: m.Sys >> 20, Goroutine: uint64(runtime.NumGoroutine()), Memoria: memoriaContainer()}
	for _, f := range []string{"memory.current", "memory.peak", "memory.max", "memory.stat"} {
		v, _ := os.ReadFile(dirCgroup + f)
		c.Cgroup = append(c.Cgroup, [2]string{f, strings.TrimSpace(string(v))})
	}
	host, err := os.ReadFile(fileHost())
	if err != nil {
		host = []byte(err.Error())
	}
	c.Host = string(host)
	mostraPagina(w, r, "stato", "Stato del server", c)
}

// memoria del container in MiB. Vera è quella che il kernel non può liberare: il totale
// meno la cache dei file. I file in /tmp (tmpfs) sono cache ma restano in memoria, e contano.
// Durante un build -confronta rilegge la Gallery intera e la cache arriva quasi al limite.
type memoria struct {
	Vera  int64 `json:"vera"`
	Cache int64 `json:"cache"`
	Max   int64 `json:"max"` // 0 = senza limite
}

func memoriaContainer() memoria {
	leggi := func(f string) int64 {
		v, _ := os.ReadFile(dirCgroup + f)
		n, _ := strconv.ParseInt(strings.TrimSpace(string(v)), 10, 64)
		return n
	}
	stat := map[string]int64{}
	if b, err := os.ReadFile(dirCgroup + "memory.stat"); err == nil {
		for sc := bufio.NewScanner(bytes.NewReader(b)); sc.Scan(); {
			if k, v, ok := strings.Cut(sc.Text(), " "); ok {
				stat[k], _ = strconv.ParseInt(v, 10, 64)
			}
		}
	}
	cache := max(stat["file"]-stat["shmem"], 0)
	return memoria{Vera: max(leggi("memory.current")-cache, 0) >> 20, Cache: cache >> 20, Max: leggi("memory.max") >> 20}
}

var reRiavvii = regexp.MustCompile(`restart=(\d+)`)

// riavvii del container, dal file dell'host (gerbillo-dev-aggiorna.sh). -1 se non si sa.
func riavvii() int {
	b, err := os.ReadFile(fileHost())
	if err != nil {
		return -1
	}
	m := reRiavvii.FindSubmatch(b)
	if m == nil {
		return -1
	}
	n, _ := strconv.Atoi(string(m[1]))
	return n
}

type statoJSON struct {
	Build   *buildJSON `json:"build,omitempty"`
	Memoria memoria    `json:"memoria"`
	Riavvii int        `json:"riavvii"`
}

type buildJSON struct {
	InCorso bool   `json:"inCorso"`
	InCoda  bool   `json:"inCoda"`
	Esito   string `json:"esito,omitempty"` // "riuscito", "fallito" o vuoto se non ce n'è stato uno
	Fine    string `json:"fine,omitempty"`  // hh:mm
	Durata  string `json:"durata,omitempty"`
	Errore  string `json:"errore,omitempty"`
}

// statoWidget risponde a /admin/widget. Senza redazione non c'è il build.
func statoWidget(red *redazione) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s := statoJSON{Memoria: memoriaContainer(), Riavvii: riavvii()}
		if red != nil {
			inCorso, inCoda, e := red.statoDelBuild()
			s.Build = &buildJSON{InCorso: inCorso, InCoda: inCoda}
			if e != nil {
				s.Build.Esito, s.Build.Fine, s.Build.Durata, s.Build.Errore = "fallito", e.Fine.In(roma).Format("15:04"), e.Durata.String(), e.Errore
				if e.Riuscito {
					s.Build.Esito = "riuscito"
				}
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		json.NewEncoder(w).Encode(s)
	}
}
