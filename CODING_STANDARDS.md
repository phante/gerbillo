# Regole di scrittura del codice

La base sono [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments) ed [Effective Go](https://go.dev/doc/effective_go). Le regole qui sotto coprono quello che quelle guide lasciano aperto per questo repo. Dove una regola di questo file contraddice quelle guide, vince questo file.

Formattazione e segnalazioni di `go vet` e `staticcheck` sono compito degli strumenti: la CI li esegue (vedi «Strumenti»). Non si discutono nelle revisioni.

## Lingua

- Identificatori Go, commenti, messaggi di commit, issue e documenti sono in italiano.
- Un concetto ha un nome solo, e quel nome è il termine di `CONTEXT.md`, con la sua maiuscola: Foto, Entità, Variante, Formato. Un termine fra gli _Avoid_ del glossario non si usa né nel codice né nei testi per l'utente.
- I messaggi d'errore sono testi per chi usa il Backend: possono cominciare con un termine del glossario, che ha la maiuscola. Per questo `staticcheck.conf` esclude ST1005.
- Un commento cita la issue che ha portato il comportamento, per esempio `(#87)`.

## Errori

- Un errore non si perde mai in silenzio. Si sceglie una di queste strade:
  - lo si restituisce, con il contesto: `fmt.Errorf("%s: %w", rel, err)`;
  - lo si scrive nel log con `log.Printf`, quando il Backend può andare avanti senza quel risultato;
  - nei test ci si ferma con `t.Fatal`.
- `_ =`, `x, _ :=` o una chiamata che restituisce un errore e il cui risultato si ignora hanno bisogno di un commento che dica perché l'errore non può capitare o non conta.
- Se perdere un errore può lasciare il Repository dei contenuti diverso da quello che l'utente legge («Salvato», «Non salvato»), si riportano i file com'erano prima di rispondere.

## Ambito

- Si costruisce solo quello che la issue chiede. Campi, parametri o funzioni per una issue successiva aspettano quella issue.
- Una scorciatoia o un rinvio voluti hanno un commento `ponytail:`. Il commento dice il limite e la issue o la condizione che lo toglie, per esempio `// ponytail: due rename, non atomico; uno symlink da scambiare se il 404 di un istante dà fastidio`.
- Il codice usato solo dai test sta nei file `_test.go`.

## Nomi e tipi

- Un concetto del dominio che passa per più funzioni ha un suo tipo, non una `string` nuda (per esempio il Formato).
- Un letterale che indica uno stato o un tag interno (`"da-classificare"`) si scrive una volta sola, come costante.
- I nomi dei tipi sono sostantivi che dicono che cosa contiene il valore.
- Una cosa che si fa in più posti nello stesso modo (scrivere un file in modo atomico, costruire il percorso del YAML di una Foto) ha una funzione sola.

## Test

- I test del Backend passano dagli handler HTTP (`httptest`) o dai metodi della redazione. Leggono i file o git solo per controllare che cosa è stato salvato.
- Un commento prima di ogni test dice il comportamento che controlla, come nei test esistenti.
- Per le chiamate HTTP si usa `httptest`. Mai una rete vera, nemmeno nei test end-to-end.
- La logica non banale (un ramo, un ciclo, un parser, tutto ciò che scrive nel Repository dei contenuti o pubblica) ha almeno un test che fallisce quando la logica si rompe.

## Strumenti

Dalla cartella `backend/`:

```bash
go vet ./... && mise exec -- staticcheck ./... && go test ./...
```

`mise.toml` fissa le versioni di Go e di `staticcheck`; la prima volta serve `mise trust`. La CI esegue gli stessi tre comandi nel job `test`.
