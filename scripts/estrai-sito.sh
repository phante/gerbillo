#!/usr/bin/env bash
# Estrae dal Sito principale le pagine delle sezioni date come dati di prova:
# solo il front matter di index.md (il corpo resta fuori) e geo.geojson.
# Uso: scripts/estrai-sito.sh [percorso/progettodighe] [sezione…]
# Senza sezioni: dighe centrali reportage tecnica.
set -euo pipefail
src="${1:-$HOME/Siti/progettodighe}/content"
shift || true
sezioni=("$@")
[ $# -eq 0 ] && sezioni=(dighe centrali reportage tecnica)
for sez in "${sezioni[@]}"; do
  dst="$(dirname "$0")/../testdata/sito/$sez"
  rm -rf "$dst" && mkdir -p "$dst"
  for d in "$src/$sez"/*/; do
    slug=$(basename "$d")
    [ -f "$d/index.md" ] || continue
    mkdir -p "$dst/$slug"
    awk 'NR==1 && $0=="---" {print; fm=1; next} fm && $0=="---" {print; exit} fm {print}' \
      "$d/index.md" > "$dst/$slug/index.md"
    [ -f "$d/geo.geojson" ] && cp "$d/geo.geojson" "$dst/$slug/"
  done
  echo "$(ls "$dst" | wc -l | tr -d ' ') pagine $sez in $dst"
done
