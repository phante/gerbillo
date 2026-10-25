#!/usr/bin/env bash
# Avvia il Backend per i test end-to-end: gli stessi dati dell'anteprima, generati da
# scripts/anteprima-gallery.sh in dist/ e seme/, sotto /gerbillo/ come dietro nginx, con una
# password fissa e un Repository dei contenuti nuovo a ogni avvio.
set -euo pipefail
cd "$(dirname "$0")/.."
if [ ! -f seme/indice.json ] || [ ! -f dist/index.html ]; then
  echo "mancano dist/ e seme/: lancia prima scripts/anteprima-gallery.sh" >&2
  exit 1
fi
lavoro=$(mktemp -d)
(cd backend && go build -o "$lavoro/gerbillo" .)
export GERBILLO_SEME=$PWD/seme GERBILLO_DIST=$PWD/dist \
  GERBILLO_CONTENUTI=$lavoro/contenuti GERBILLO_DIST_MODIFICHE=$lavoro/dist \
  GERBILLO_ADDR=127.0.0.1:${PORTA:-8091} GERBILLO_PASSWORD=prova GERBILLO_PREFISSO=/gerbillo/ \
  GERBILLO_STATO_HOST=$lavoro/stato.json
exec "$lavoro/gerbillo" serve
