package main

// Operazioni recenti e Annulla (#59): ogni modifica del Backend è un commit nel Repository
// dei contenuti, con autore l'utente (autoreCommit, @gerbillo.invalid). La pagina li elenca
// dal più recente; «Annulla» ne fa il git revert, con autore l'utente corrente, poi un build.
// Si annulla solo se i file toccati non sono cambiati dopo: altrimenti si elencano le Foto
// cambiate nel frattempo e non si tocca niente. Le operazioni in blocco (#50) hanno il
// trailer «Operazione:», gli annullamenti il trailer «Annulla:».

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

const maxOperazioni = 100

type operazione struct {
	Hash, Breve   string
	Quando        string
	Autore        string
	Tipo          string // «Modifica», «In blocco», «Annullamento»
	Campi         string
	Pid           []int
	Annulla       string // l'hash che questa operazione annulla
	AnnullataDa   string // l'hash dell'annullamento, se c'è
	Esito, Errore string
	CambiateDopo  []int
	annullabile   bool
}

// Mostrate è quante Foto si elencano per un'operazione; le altre si contano.
func (o operazione) Mostrate() []int   { return o.Pid[:min(len(o.Pid), 12)] }
func (o operazione) Altre() int        { return max(len(o.Pid)-12, 0) }
func (o operazione) Annullabile() bool { return o.annullabile && o.AnnullataDa == "" }

// git esegue git nel Repository dei contenuti e ne restituisce l'uscita.
func (r *redazione) git(a ...string) (string, error) {
	c := exec.Command("git", append([]string{"-C", r.contenuti, "-c", "user.name=gerbillo", "-c", "user.email=gerbillo@localhost"}, a...)...)
	var out, errb bytes.Buffer
	c.Stdout, c.Stderr = &out, &errb
	if err := c.Run(); err != nil {
		return out.String(), fmt.Errorf("git %s: %v: %s", a[0], err, strings.TrimSpace(errb.String()))
	}
	return out.String(), nil
}

var (
	reFileFoto  = regexp.MustCompile(`^arricchimento/foto/(\d+)\.yaml$`)
	reTrailer   = regexp.MustCompile(`(?m)^(Operazione|Annulla): (\S+)$`)
	reCampiDopo = regexp.MustCompile(`: ([a-z., ]+)$`)
)

// operazioni legge i commit del Backend, dal più recente.
func (r *redazione) operazioni() ([]operazione, error) {
	out, err := r.git("log", "-n", strconv.Itoa(maxOperazioni), "--author=@gerbillo.invalid",
		"--format=%x1e%H%x00%an%x00%aI%x00%s%x00%b%x00", "--name-only", "--", "arricchimento")
	if err != nil {
		return nil, err
	}
	var ops []operazione
	annullate := map[string]string{}
	for _, blocco := range strings.Split(out, "\x1e") {
		parti := strings.SplitN(blocco, "\x00", 6)
		if len(parti) < 6 {
			continue
		}
		o := operazione{Hash: parti[0], Breve: parti[0][:min(7, len(parti[0]))], Autore: parti[1], Tipo: "Modifica"}
		if t, err := time.Parse(time.RFC3339, parti[2]); err == nil {
			o.Quando = t.In(roma).Format("02/01 15:04")
		}
		oggetto, corpo := parti[3], parti[4]
		if m := reCampiDopo.FindStringSubmatch(oggetto); m != nil {
			o.Campi = m[1]
		}
		for _, t := range reTrailer.FindAllStringSubmatch(corpo, -1) {
			switch t[1] {
			case "Operazione":
				o.Tipo = "In blocco"
			case "Annulla":
				o.Tipo, o.Annulla = "Annullamento", t[2]
				annullate[t[2]] = o.Hash
			}
		}
		for _, f := range strings.Fields(parti[5]) {
			if m := reFileFoto.FindStringSubmatch(f); m != nil {
				p, _ := strconv.Atoi(m[1])
				o.Pid = append(o.Pid, p)
			}
		}
		slices.Sort(o.Pid)
		o.annullabile = len(o.Pid) > 0
		ops = append(ops, o)
	}
	for i := range ops {
		ops[i].AnnullataDa = annullate[ops[i].Hash]
	}
	return ops, nil
}

// cambiateDopo sono le Foto di op cambiate da un commit successivo.
func (r *redazione) cambiateDopo(op operazione) ([]int, error) {
	var file []string
	for _, p := range op.Pid {
		file = append(file, filepath.ToSlash(filepath.Join("arricchimento", fileFoto(p))))
	}
	out, err := r.git(append([]string{"log", "--format=", "--name-only", op.Hash + "..HEAD", "--"}, file...)...)
	if err != nil {
		return nil, err
	}
	var pid []int
	for _, f := range strings.Fields(out) {
		if m := reFileFoto.FindStringSubmatch(f); m != nil {
			p, _ := strconv.Atoi(m[1])
			if !slices.Contains(pid, p) {
				pid = append(pid, p)
			}
		}
	}
	slices.Sort(pid)
	return pid, nil
}

var errCambiate = errors.New("alcune Foto sono cambiate dopo")

// annulla fa il revert dell'operazione hash con autore u. Con errCambiate restituisce le
// Foto cambiate nel frattempo, e non tocca niente.
func (r *redazione) annulla(hash string, u *utente) ([]int, error) {
	r.salvataggio.Lock()
	defer r.salvataggio.Unlock()
	ops, err := r.operazioni()
	if err != nil {
		return nil, err
	}
	i := slices.IndexFunc(ops, func(o operazione) bool { return o.Hash == hash })
	if i < 0 || !ops[i].Annullabile() {
		return nil, fmt.Errorf("operazione %s non annullabile", hash)
	}
	op := ops[i]
	if dopo, err := r.cambiateDopo(op); err != nil {
		return nil, err
	} else if len(dopo) > 0 {
		return dopo, errCambiate
	}
	if _, err := r.git("revert", "--no-commit", op.Hash); err != nil {
		r.git("revert", "--abort")
		return nil, err
	}
	oggetto, _ := r.git("log", "-1", "--format=%s", op.Hash)
	msg := fmt.Sprintf("annulla: %s\n\nAnnullata dal Backend (#59).\n\nAnnulla: %s", strings.TrimSpace(oggetto), op.Hash)
	if _, err := r.git("commit", "-q", "-m", msg, "--author", autoreCommit(u)); err != nil {
		r.git("revert", "--abort")
		return nil, err
	}
	return nil, nil
}

type corpoOperazioni struct {
	Operazioni []operazione
	Errore     string
}

// paginaOperazioni risponde a /admin/operazioni, e ad /admin/operazioni/<hash>/annulla in POST.
func (r *redazione) paginaOperazioni(w http.ResponseWriter, req *http.Request) {
	resto := strings.Trim(strings.TrimPrefix(req.URL.Path, "/admin/operazioni"), "/")
	if hash, ok := strings.CutSuffix(resto, "/annulla"); ok && req.Method == http.MethodPost {
		dopo, err := r.annulla(hash, utenteDi(req))
		ops, _ := r.operazioni()
		i := slices.IndexFunc(ops, func(o operazione) bool { return o.Hash == hash })
		if i < 0 {
			http.NotFound(w, req)
			return
		}
		op := ops[i]
		switch {
		case errors.Is(err, errCambiate):
			op.Errore, op.CambiateDopo = "Non annullata: queste Foto sono cambiate dopo, e l'annullamento le riporterebbe indietro.", dopo
		case err != nil:
			op.Errore = "Non annullata: " + err.Error()
		default:
			r.richiediBuild()
			op.Esito = "Annullata. La Gallery cambia dopo il build."
		}
		mostraFrammento(w, "operazione", op)
		return
	}
	if resto != "" {
		http.NotFound(w, req)
		return
	}
	var c corpoOperazioni
	ops, err := r.operazioni()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		c.Errore = err.Error()
	}
	c.Operazioni = ops
	mostraPagina(w, req, "operazioni", "Operazioni recenti", c)
}
