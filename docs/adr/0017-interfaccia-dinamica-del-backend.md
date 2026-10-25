# Interfaccia dinamica del Backend

Deciso il 25/10/2026, nel grill di #56. Rivede la parte dell'ADR 0007 sul JavaScript.

Il Backend è uno strumento di redazione, anche per la sistemazione massiva prima dello Switch. Deve essere comodo e moderno, altrimenti non si usa: dove si lavora sui dati, upload compreso, salvataggi e aggiornamenti parziali avvengono senza ricaricare la pagina.

- **Stack.** HTML generato dal server Go con `html/template`, **htmx** per gli aggiornamenti parziali, **Alpine.js** per lo stato locale dei componenti. Il server risponde con frammenti HTML: niente API JSON per disegnare le pagine e niente build JS. Restano JSON i dati che servono ai componenti, come lo stato del widget o i suggerimenti dei campi a chip.
- **Librerie nel repo.** Sono salvate in `backend/modelli/backend/librerie/` e incorporate nel binario, mai caricate da un CDN. `scripts/librerie-backend.sh` le scarica alla versione esatta. Regola per una libreria di componenti: versione esatta, nessuna dipendenza a runtime, licenza permissiva (MIT o simile). Una libreria matura si preferisce al codice ad hoc quando copre tastiera, mobile e accessibilità: Tom Select per i chip (#53); l'upload si sceglie in #29.
- **Alpine nella versione standard**, senza CSP. Se un rischio concreto la richiederà, si passa alla versione CSP di Alpine.
- **Non serve funzionare senza JavaScript.** Il Backend lo usano i Moderatori, con browser moderni.
- **Aspetto.** Un solo modello di pagina, con la barra in alto. Tachyons, colori e font della Gallery pubblica, ma con un layout da strumento: non deve sembrare una sezione del sito. Pagine pensate prima per il telefono.
- **Gallery pubblica.** Nessuna riscrittura. Può usare htmx e Alpine quando una funzione nuova ne ha bisogno.
- **Test.** Test Go più test end-to-end con `@playwright/test` in `e2e/`, sui flussi chiave e a larghezza telefono. npm serve solo ai test, con la versione bloccata. Il job `test` di `.github/workflows/anteprima.yml` gira in parallelo all'immagine: se fallisce avvisa su Telegram, ma non blocca l'anteprima. Prima di ogni merge su `main` i test devono essere verdi.

## Considered Options

- **Poco JavaScript scritto a mano** (ADR 0007): scartato. Chip, mappa, selezione multipla e aggiornamenti parziali diventerebbero un piccolo framework fatto in casa, senza la cura di tastiera e accessibilità delle librerie mature.
- **htmx da solo**, con chip e mappa scritti a mano: scartato per la stessa ragione. Alpine costa 50 kB e copre lo stato locale dei componenti.
- **Svelte**, che c'era nella prima versione dell'ADR 0007: era stato tolto perché condivideva i componenti con la Gallery in Astro, che non c'è più (ADR 0012). Senza quel vantaggio resta solo il costo di Node e di un build JS.
- **Una SPA con API JSON**: scartata. Raddoppia il lavoro (API e client) e porta un build JS che il progetto non vuole.
