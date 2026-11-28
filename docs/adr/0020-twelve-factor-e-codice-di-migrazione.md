# Gerbillo segue twelve-factor; il codice di Migrazione è dietro un tag

Deciso il 28/11/2026, nel grilling sulla gestione del Repository dei contenuti.

## Configurazione nell'ambiente

Gerbillo segue le linee guida di [twelve-factor](https://12factor.net/). Per la configurazione (fattore III):

- **Unica fonte: variabili d'ambiente `GERBILLO_*`.** `gerbillo.toml` sparisce. Sul Mac un `.env` ignorato da git, caricato da `mise`; sul server `EnvironmentFile=` di systemd o `env_file:` di compose. I segreti (MySQL, token) seguono la stessa strada.
- **Nell'ambiente va tutto ciò che cambia da un deploy all'altro**: percorsi (contenuti, media, dist, Originali), MySQL, bridge, destinazioni degli Originali, intervallo della Sincronizzazione, Telegram. Restano flag solo le opzioni di una singola esecuzione (`-j` di `varianti`, un pid da provare), e il flag vince sulla variabile.
- **Una variabile obbligatoria che manca ferma il comando all'avvio**, con un messaggio che la nomina. Default solo per valori innocui (porta, intervallo); mai per percorsi e segreti, perché un default sul percorso dei contenuti scriverebbe nel posto sbagliato.

Chi può fare `docker inspect` vede i segreti: sul server chi ha accesso a Docker è già root, quindi il rischio non cambia.

Gli altri fattori (log su stdout, servizi esterni come risorse collegate, processi amministrativi della stessa release, parità fra dev e produzione, avvio e arresto rapidi) si verificano sul codice in un'issue a parte.

## Il codice di Migrazione è dietro il tag `migrazione`

L'Importazione di massa e la Migrazione (Mappature, alias degli Autori, abbinamento automatico, riscrittura di `importazione/`) stanno in file con `//go:build migrazione`. Sul Mac si compila con `-tags migrazione`; l'immagine di produzione no, quindi dopo l'Avvio quei comandi non esistono nel binario. I test girano con e senza il tag.

Il lettore di Coppermine (query, campi, regola di encoding di #9) non ha il tag: serve alla Sincronizzazione finché c'è Coppermine. Allo Switch il codice con il tag si cancella.

## Istanze

Sul server ci sono due istanze di Gerbillo:

- **dev**, su `dev.progettodighe.it/gerbillo/`, solo con dati di test, accesa quando serve (ADR 0011);
- **produzione**, su `progettodighe.it/gerbillo/` durante la convivenza e su `/gallery/` dallo Switch, con un'immagine con tag di rilascio, mai `:dev`, aggiornata a mano.

Nessuna pipeline riceve `docker.sock` o volumi dell'host: il server scarica le immagini, come in ADR 0011. Dove si fanno test e build (GitHub Actions o Woodpecker) si decide in un'issue a parte.

## Considered Options

- **File di configurazione TOML**: scartato. Gerbillo gira in container, dove l'ambiente è il modo più semplice di passare la configurazione, e due fonti costringono a cercare dove sta un valore.
- **Comando di Migrazione disattivato da un flag a runtime**: scartato. Il codice resterebbe nel binario di produzione, e un flag si riattiva.
- **Cancellare tutto il codice dell'Importazione all'Avvio**: scartato. La Sincronizzazione ne usa il lettore di Coppermine fino allo Switch.
