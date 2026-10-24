#!/usr/bin/env bash
# Genera in ./dist la Gallery pubblica dai dati di test, per l'anteprima (ADR 0011, 0012).
# Richiede un MySQL con testdata/coppermine.sql.gz caricato e l'utente gerbillo_ro.
# Le immagini puntano agli albums/ pubblici di Coppermine: le Varianti non ci sono ancora.
#
# Uso: scripts/anteprima-gallery.sh [host:porta MySQL] [testo della striscia]
set -euo pipefail
cd "$(dirname "$0")/.."
indirizzo=${1:-}
striscia=${2:-anteprima locale}
lavoro=$(mktemp -d)
trap 'rm -rf "$lavoro"' EXIT

(cd backend && CGO_ENABLED=0 go build -tags nodynamic -o "$lavoro/gerbillo" .)
cat > "$lavoro/gerbillo.toml" <<EOF
contenuti = "$lavoro/contenuti"
[mysql]
utente = "gerbillo_ro"
password = "gerbillo"
indirizzo = "$indirizzo"
EOF
g() { "$lavoro/gerbillo" "$@"; }
g indice-entita -sito testdata/sito > "$lavoro/indice.json"
# dati/arricchimento/ fa da Livello di arricchimento finché non c'è il rientro (#38, #39).
g importa -config "$lavoro/gerbillo.toml" -indice "$lavoro/indice.json" -arricchimento dati/arricchimento -vocabolario dati/vocabolario.yaml
# Nell'anteprima nginx serve la Gallery sotto /gerbillo/ e toglie il prefisso.
g build -config "$lavoro/gerbillo.toml" -indice "$lavoro/indice.json" -arricchimento dati/arricchimento -dist dist \
  -base /gerbillo/ -anteprima -media https://progettodighe.it/gallery/albums/ -striscia "$striscia"
