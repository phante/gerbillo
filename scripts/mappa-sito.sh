#!/usr/bin/env bash
# Rigenera backend/modelli/sito/mappa.js e leaflet.css dal componente mappa del Sito
# principale (#27): stessi sorgenti (themes/progettodighe/assets/asset/js/map*.js),
# stesse versioni di Leaflet del package-lock del sito, stesse opzioni di esbuild
# di js-bundle.html in produzione. Va rilanciato quando TestCopieSito segnala che
# le copie divergono.
#
# Uso: scripts/mappa-sito.sh [cartella del repo del sito, predefinita ../progettodighe]
set -euo pipefail
cd "$(dirname "$0")/.."
sito=$(cd "${1:-../progettodighe}" && pwd)
js="$sito/themes/progettodighe/assets/asset/js"
dest=backend/modelli/sito
lavoro=$(mktemp -d)
trap 'rm -rf "$lavoro"' EXIT

versione() { node -e "console.log(require('$sito/package-lock.json').packages['node_modules/$1'].version)"; }
(cd "$lavoro" && npm install --silent --no-save --no-audit --no-fund \
  "leaflet@$(versione leaflet)" "leaflet.markercluster@$(versione leaflet.markercluster)" esbuild >/dev/null)

# Stessa disposizione dei mount Hugo del sito (config/_default/module.yml).
mkdir -p "$lavoro/js/leaflet"
cp "$js"/map*.js "$lavoro/js/"
cp "$lavoro/node_modules/leaflet/dist/leaflet.js" "$lavoro/js/leaflet/leaflet.js"
cp "$lavoro/node_modules/leaflet.markercluster/dist/leaflet.markercluster.js" "$lavoro/js/leaflet/leaflet.markercluster.js"

"$lavoro/node_modules/.bin/esbuild" "$lavoro/js/map.js" --bundle --minify --format=iife \
  --define:PD_DEBUG=false --drop:console --log-level=warning --outfile="$dest/mappa.js"
cp "$lavoro/node_modules/leaflet/dist/leaflet.css" "$dest/leaflet.css"
cp "$sito/themes/progettodighe/assets/asset/css/map.control.verticalexpand.css" "$dest/"
echo "mappa.js $(wc -c < "$dest/mappa.js") byte, Leaflet $(versione leaflet), markercluster $(versione leaflet.markercluster)"
