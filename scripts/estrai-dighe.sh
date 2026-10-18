#!/usr/bin/env bash
# Estrae dal Sito principale le pagine delle dighe come dati di prova:
# solo il front matter di index.md (il corpo resta fuori) e geo.geojson.
# Uso: scripts/estrai-dighe.sh [percorso/progettodighe]
set -euo pipefail
src="${1:-$HOME/Siti/progettodighe}/content/dighe"
dst="$(dirname "$0")/../testdata/sito/dighe"
rm -rf "$dst" && mkdir -p "$dst"
for d in "$src"/*/; do
  slug=$(basename "$d")
  [ -f "$d/index.md" ] || continue
  mkdir -p "$dst/$slug"
  awk 'NR==1 && $0=="---" {print; fm=1; next} fm && $0=="---" {print; exit} fm {print}' \
    "$d/index.md" > "$dst/$slug/index.md"
  [ -f "$d/geo.geojson" ] && cp "$d/geo.geojson" "$dst/$slug/"
done
echo "$(ls "$dst" | wc -l | tr -d ' ') dighe in $dst"
