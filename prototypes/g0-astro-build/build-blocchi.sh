#!/bin/sh
# PROTOTYPE G0: build a blocchi di K Foto, memoria indipendente dal numero totale di Foto.
# Uso: ./build-blocchi.sh [K=5000]
set -eu
K=${1:-5000}
node indice.mjs
MAX=$(node -e "const i=require('./indice.json'); console.log(i.at(-1)[0])")
rm -rf dist dist-*
da=1
while [ $da -le "$MAX" ]; do
  a=$((da + K))
  echo "blocco $da-$a"
  BLOCCO=$da-$a OUT=dist-$da npx astro build > /dev/null
  da=$a
done
echo "blocco indici"
BLOCCO=indici OUT=dist-indici npx astro build > /dev/null
mkdir dist
# dist-indici viene per ultimo in ordine alfabetico e sovrascrive home e ricerca.json
for d in dist-*; do cp -a "$d"/. dist/ && rm -rf "$d"; done
