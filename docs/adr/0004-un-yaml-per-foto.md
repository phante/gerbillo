# Un file YAML per Foto come fonte di verità, in un repo git locale sul server

I metadati pubblicati di ogni Foto stanno in un file YAML proprio nel Repository dei contenuti. Il formato è disperso ma atomico: ogni modifica tocca un file solo, si versiona con git e si rivede nei diff. Non c'è un database, per ridurre le superfici dinamiche. Se un giorno servisse un file unico, si passerà a un vero database invece di un YAML monolitico.

Rivisto il 15/10/2026: tolto il remoto `gerbillo-contenuti` con la build in Woodpecker.

Rivisto il 28/11/2026, nel grilling sulla gestione del repository: il Repository dei contenuti ha due fasi, separate dall'Avvio della produzione.

## Prima dell'Avvio: dati usa e getta, preparati sul Mac

- I contenuti si generano sul Mac con l'Importazione dal DB di produzione, senza anonimizzazione, in una cartella configurabile fuori dal repo del codice.
- Git nei contenuti è uno strumento di lavoro, non un registro: un commit dopo ogni Importazione, così `git diff` mostra cosa è cambiato a mano o dall'interfaccia di un'istanza locale.
- Il lavoro a mano (Mappature, alias degli Autori, riservati, correzioni delle Foto) sta tutto in `arricchimento/` dei contenuti. Il ciclo è: Importazione, analisi dei YAML, giro sull'interfaccia, `git diff`, regole nuove nelle Mappature, nuova Importazione.
- Fino all'Avvio, una copia temporanea dei contenuti sta su un repo privato in rete locale. Si cancella dopo l'Avvio.
- I contenuti e le Varianti si copiano sul server con `rsync`, anche distruttivo: prima dell'Avvio sul server non c'è niente da proteggere.

## Dall'Avvio: il repo del server è il database

- L'Avvio lo decide l'admin. Prima servono almeno: la sessione del bridge corretta (#120), la prova sul forum vero (#44), una revisione di sicurezza del Backend, niente Varianti per le Foto riservate (#119).
- All'Avvio si congelano i valori risolti nei YAML delle Foto (#40), si fa `git init` e il primo commit nella cartella persistente dell'istanza di produzione.
- Da lì il repository è **solo locale sul server**, con un solo scrittore: il Backend (Pubblicazioni, Correzioni approvate) e la Sincronizzazione da Coppermine. Git serve da storia, diff e annullamento, al posto di un log.
- Nessuna sincronizzazione con il Mac. Per lavorare in locale con i dati veri, si copia con `rsync` dal server al Mac.
- Il backup è quello del server: snapshot serale della VM, più un eventuale `git bundle` su S3 negli script di `progettodighe-server`, fuori da Gerbillo.

## Considered Options

- **XMP incorporato nelle immagini**: scartato come fonte di verità. Le immagini restano fuori da git e i metadati non sarebbero versionabili né facili da modificare in blocco. Un convertitore YAML → XMP si farà solo se servirà davvero.
- **Repo remoto `gerbillo-contenuti` con build in Woodpecker**: scartato il 15/10/2026. Funziona per il Sito principale, dove i contenuti nascono sul Mac; qui aggiungerebbe un viaggio inutile e una dipendenza esterna.
- **SQLite**: scartato. Si perdono il diff leggibile e la storia gratuita di git.
- **Mac e server come cloni sincronizzati via git**: scartato il 28/11/2026. Due scrittori portano conflitti, e il Mac non ha nulla da proteggere: i dati importati si rigenerano.
- **Backup su un Forgejo di casa, raggiunto dal server con una VPN**: scartato il 28/11/2026. Apre un percorso dal server di produzione verso la rete di casa.
