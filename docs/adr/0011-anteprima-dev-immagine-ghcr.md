# Anteprima dev: immagine su ghcr scaricata dal server, container limitato

L'anteprima di Gerbillo su `dev.progettodighe.it/gerbillo/` è un'immagine Docker unica (Backend, admin e Gallery pubblica), costruita da GitHub Actions a ogni push e pubblicata su `ghcr.io/phante/gerbillo:dev`. Sul server un timer systemd la scarica ogni minuto e ricrea il container se è cambiata. Dall'esterno nessuno scrive sul server: il server legge e basta. Il server ospita la produzione di ProgettoDighe (sito, forum, Coppermine), ha 1,9 GiB di RAM e non è raggiungibile da remoto.

Il container ha limiti rigidi (`--memory 384m --memory-swap 384m --cpus 0.5`, sola lettura, upload su `tmpfs` da 200 MB), ascolta solo su `127.0.0.1:8090` dietro nginx e non è root. L'admin è protetto da una password usa e getta generata all'avvio, inviata via Telegram e valida 24 ore, dopo la quale si sceglie con quale utente entrare: è l'implementazione dev di "utente corrente" (ADR 0008). Ogni pagina mostra branch e commit.

**Criterio:** ogni gate del pilota dichiara la RAM consumata, letta da `/gerbillo/admin/stato`. Se Gerbillo non sta nelle risorse del server accanto alla produzione, si cambia strada.

## Considered Options

- **Woodpecker che pubblica sul server** (volumi o `docker.sock`): scartato. Un repo *trusted* può montare qualunque path dell'host, e un errore colpirebbe la produzione.
- **Build su Woodpecker**: scartata, perché consumerebbe la RAM della produzione. Se GitHub Actions non fosse più disponibile si riconsidera: `.woodpecker/simulazione.yml` prova solo webhook e clone, non builda.
- **Pipeline di emergenza con shell root sull'host**: scartata. Da remoto il server non si tocca.

## Consequences

- Il setup sul server (container, timer, nginx, `/etc/gerbillo-dev.env`) si fa una volta a mano. File in `server/`, istruzioni in `docs/lavoro-remoto.md`.
- Lo stato dell'admin è effimero: ogni deploy riparte dai dati di test.
- Il timer scrive `stato.json` (container, memoria, swap, disco) e manda allarmi Telegram su OOM, container fermo o swap alto.
