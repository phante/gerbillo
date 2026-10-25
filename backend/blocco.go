package main

// Modifica in blocco di più Foto (#50). Un'operazione in blocco equivale a N modifiche
// singole, con la stessa logica (prepara): in arricchimento/foto/<pid>.yaml va solo ciò che
// differisce dall'Importazione. mappature.yaml resta solo a mano. Azioni:
//   - liste (slug, Categorie, Tag compresi gli interni, Autori): aggiungi e togli voci,
//     anche insieme; niente «sostituisci l'intera lista»;
//   - valori singoli (Datazione, Fonte, Punto geografico): imposta o svuota.
// Titolo, descrizione e alt restano fuori. L'anteprima è obbligatoria e fissa l'elenco dei
// pid: la conferma si applica a quelle Foto, ricalcolando ciascuna sul suo file attuale.
// Un commit per operazione, con il trailer «Operazione:» che #59 riconosce, e un build.

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"
)

var (
	campiListaBlocco   = []string{"slug", "categories", "tags", "authors"}
	campiSingoliBlocco = []string{"datazione", "fonte", "geolocation.coords"}
)

// sogliaConferma: oltre queste Foto serve scrivere il numero per confermare.
const sogliaConferma = 500

type modificaBlocco struct {
	aggiungi, togli map[string][]string // campo lista → voci
	imposta         map[string]string   // campo singolo → valore; "" = svuota
}

func leggiModificaBlocco(v url.Values) modificaBlocco {
	m := modificaBlocco{aggiungi: map[string][]string{}, togli: map[string][]string{}, imposta: map[string]string{}}
	pulisci := func(x []string) []string {
		var out []string
		for _, s := range x {
			for _, r := range strings.Split(s, "\n") {
				if r = strings.TrimSpace(r); r != "" && !slices.Contains(out, r) {
					out = append(out, r)
				}
			}
		}
		return out
	}
	for _, c := range campiListaBlocco {
		if a := pulisci(v["aggiungi."+c]); len(a) > 0 {
			m.aggiungi[c] = a
		}
		if t := pulisci(v["togli."+c]); len(t) > 0 {
			m.togli[c] = t
		}
	}
	for _, c := range campiSingoliBlocco {
		switch v.Get("modo." + c) {
		case "imposta":
			m.imposta[c] = strings.TrimSpace(v.Get("valore." + c))
		case "svuota":
			m.imposta[c] = ""
		}
	}
	return m
}

func (m modificaBlocco) vuota() bool { return len(m.aggiungi)+len(m.togli)+len(m.imposta) == 0 }

// valori rimanda la modifica come campi nascosti del modulo di conferma.
func (m modificaBlocco) valori() url.Values {
	v := url.Values{}
	for c, x := range m.aggiungi {
		v["aggiungi."+c] = x
	}
	for c, x := range m.togli {
		v["togli."+c] = x
	}
	for c, x := range m.imposta {
		if x == "" {
			v.Set("modo."+c, "svuota")
		} else {
			v.Set("modo."+c, "imposta")
			v.Set("valore."+c, x)
		}
	}
	return v
}

// applica ricava i valori nuovi di una Foto dai suoi attuali.
func (m modificaBlocco) applica(attuali map[string]string) map[string]string {
	nuovi := map[string]string{}
	for k, v := range attuali {
		nuovi[k] = v
	}
	for _, c := range campiListaBlocco {
		var voci []string
		if attuali[c] != "" {
			voci = strings.Split(attuali[c], "\n")
		}
		voci = slices.DeleteFunc(voci, func(s string) bool { return slices.Contains(m.togli[c], s) })
		for _, a := range m.aggiungi[c] {
			if !slices.Contains(voci, a) {
				voci = append(voci, a)
			}
		}
		nuovi[c] = strings.Join(voci, "\n")
	}
	for c, v := range m.imposta {
		nuovi[c] = v
	}
	return nuovi
}

// descrizione è la modifica in parole, per l'anteprima.
func (m modificaBlocco) descrizione() []string {
	var d []string
	for _, c := range campiListaBlocco {
		if a := m.aggiungi[c]; len(a) > 0 {
			d = append(d, fmt.Sprintf("%s: aggiungi %s", c, strings.Join(a, ", ")))
		}
		if t := m.togli[c]; len(t) > 0 {
			d = append(d, fmt.Sprintf("%s: togli %s", c, strings.Join(t, ", ")))
		}
	}
	for _, c := range campiSingoliBlocco {
		if v, ok := m.imposta[c]; ok {
			if v == "" {
				d = append(d, c+": svuota")
			} else {
				d = append(d, fmt.Sprintf("%s: imposta %s", c, v))
			}
		}
	}
	return d
}

type esitoBlocco struct {
	file     []fileNuovo // solo quelli che cambiano
	aPosto   []int
	errori   []string // «pid: messaggio»
	perCampo map[string]int
}

// calcola prepara la modifica per ogni pid, senza scrivere niente.
func (r *redazione) calcolaBlocco(pids []int, m modificaBlocco) (esitoBlocco, error) {
	st, idx, err := r.strati()
	if err != nil {
		return esitoBlocco{}, err
	}
	e := esitoBlocco{perCampo: map[string]int{}}
	for _, pid := range pids {
		f, err := r.prepara(st, idx, pid, m.applica)
		switch {
		case err != nil:
			e.errori = append(e.errori, fmt.Sprintf("%d: %v", pid, err))
		case len(f.cambiati) == 0:
			e.aPosto = append(e.aPosto, pid)
		default:
			e.file = append(e.file, f)
			for _, c := range f.cambiati {
				e.perCampo[c]++
			}
		}
	}
	return e, nil
}

// applicaBlocco scrive la modifica sulle Foto di pids e ne fa un commit solo, con autore u.
func (r *redazione) applicaBlocco(pids []int, m modificaBlocco, u *utente) (esitoBlocco, string, error) {
	r.salvataggio.Lock()
	defer r.salvataggio.Unlock()
	e, err := r.calcolaBlocco(pids, m)
	if err != nil {
		return e, "", err
	}
	if len(e.errori) > 0 {
		return e, "", fmt.Errorf("%d Foto non si possono salvare", len(e.errori))
	}
	if len(e.file) == 0 {
		return e, "", nil
	}
	for _, f := range e.file {
		if err := f.scrivi(r.contenuti); err != nil {
			return e, "", err
		}
	}
	id := idOperazione()
	var campi []string
	for _, c := range append(slices.Clone(campiListaBlocco), campiSingoliBlocco...) {
		if e.perCampo[c] > 0 {
			campi = append(campi, c)
		}
	}
	msg := fmt.Sprintf("blocco: %d Foto: %s\n\nModifica in blocco dal Backend (#50).\n%s\n\nOperazione: %s",
		len(e.file), strings.Join(campi, ", "), strings.Join(m.descrizione(), "\n"), id)
	if err := registra(r.contenuti, "arricchimento", msg, autoreCommit(u)); err != nil {
		return e, "", err
	}
	return e, id, nil
}

func idOperazione() string {
	b := make([]byte, 3)
	rand.Read(b)
	return time.Now().In(roma).Format("20060102-150405") + "-" + hex.EncodeToString(b)
}

func leggiPids(s string) []int {
	var p []int
	for _, x := range strings.FieldsFunc(s, func(r rune) bool { return r == ',' || r == ' ' }) {
		if n, err := strconv.Atoi(x); err == nil && !slices.Contains(p, n) {
			p = append(p, n)
		}
	}
	return p
}

func scriviPids(p []int) string {
	s := make([]string, len(p))
	for i, x := range p {
		s[i] = strconv.Itoa(x)
	}
	return strings.Join(s, ",")
}

type corpoBlocco struct {
	Pid        string // l'elenco fissato, separato da virgole
	N          int
	Elenco     string // per tornare all'elenco filtrato
	Campi      []campoBlocco
	TagInterni []tagInterno
	Gruppi     []gruppoCategorie
}

type campoBlocco struct {
	Chiave, Etichetta string
	Lista             bool
}

type corpoAnteprimaBlocco struct {
	Pid, Modifica         string // Modifica: la query dei campi, rimandata con la conferma
	Descrizione           []string
	N, Cambiano, APosto   int
	PerCampo              [][2]string
	Errori                []string
	Fotine                []int
	Soglia                bool // oltre sogliaConferma: si scrive il numero
	Campi                 url.Values
	Esito, ErroreConferma string
	Operazione            string
}

// blocco risponde a /admin/blocco/: in POST da l'elenco con la selezione apre il modulo; poi
// blocco/anteprima e blocco/applica, con htmx.
func (r *redazione) blocco(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		rimanda(w, "../foto/")
		return
	}
	req.ParseForm()
	azione := strings.Trim(strings.TrimPrefix(req.URL.Path, "/admin/blocco"), "/")
	var pids []int
	if req.PostForm.Get("tutte") == "1" {
		q, _ := url.ParseQuery(req.PostForm.Get("filtro"))
		for _, f := range leggiFiltro(q).applica(r.indiceCorrente()) {
			pids = append(pids, f.Pid)
		}
	} else {
		pids = leggiPids(req.PostForm.Get("pid"))
	}
	switch azione {
	case "":
		c := corpoBlocco{Pid: scriviPids(pids), N: len(pids), Elenco: "foto/?" + req.PostForm.Get("filtro")}
		for _, k := range campiEditoriali {
			if slices.Contains(campiListaBlocco, k.chiave) || slices.Contains(campiSingoliBlocco, k.chiave) {
				c.Campi = append(c.Campi, campoBlocco{k.chiave, k.etichetta, k.lista})
			}
		}
		if r.voc != nil {
			c.TagInterni, c.Gruppi = r.voc.TagsInterni, gruppiCategorie(r.voc, nil)
		}
		mostraPagina(w, req, "blocco", fmt.Sprintf("Modifica in blocco di %d Foto", len(pids)), c)
	case "anteprima", "applica":
		m := leggiModificaBlocco(req.PostForm)
		a := corpoAnteprimaBlocco{Pid: scriviPids(pids), N: len(pids), Descrizione: m.descrizione(), Soglia: len(pids) > sogliaConferma, Campi: m.valori()}
		a.Modifica = a.Campi.Encode()
		if m.vuota() {
			a.Errori = []string{"Nessuna modifica: scegli cosa aggiungere, togliere, impostare o svuotare."}
			mostraFrammento(w, "anteprima-blocco", a)
			return
		}
		if azione == "applica" {
			if a.Soglia && strings.TrimSpace(req.PostForm.Get("conferma")) != strconv.Itoa(len(pids)) {
				a.ErroreConferma = fmt.Sprintf("Per confermare su %d Foto scrivi %d.", len(pids), len(pids))
			} else {
				e, id, err := r.applicaBlocco(pids, m, utenteDi(req))
				r.riempiAnteprima(&a, e)
				if err != nil {
					a.ErroreConferma = "Non applicata: " + err.Error()
				} else {
					if len(e.file) > 0 {
						r.richiediBuild()
					}
					a.Operazione, a.Esito = id, fmt.Sprintf("Cambiate %d Foto con un commit solo. La Gallery cambia dopo il build.", len(e.file))
				}
				mostraFrammento(w, "anteprima-blocco", a)
				return
			}
		}
		e, err := r.calcolaBlocco(pids, m)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		r.riempiAnteprima(&a, e)
		mostraFrammento(w, "anteprima-blocco", a)
	default:
		http.NotFound(w, req)
	}
}

func (r *redazione) riempiAnteprima(a *corpoAnteprimaBlocco, e esitoBlocco) {
	a.Cambiano, a.APosto, a.Errori = len(e.file), len(e.aPosto), e.errori
	for _, c := range append(slices.Clone(campiListaBlocco), campiSingoliBlocco...) {
		if n := e.perCampo[c]; n > 0 {
			a.PerCampo = append(a.PerCampo, [2]string{c, strconv.Itoa(n)})
		}
	}
	for _, f := range e.file {
		a.Fotine = append(a.Fotine, f.pid)
	}
}
