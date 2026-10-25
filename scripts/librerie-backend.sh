#!/usr/bin/env bash
# Scarica in backend/modelli/backend/librerie/ le librerie JavaScript del Backend (ADR 0017),
# alla versione esatta scritta qui sotto, dal registro npm. Il binario le incorpora: nessuna
# pagina del Backend le carica da un CDN. Per aggiornarne una si cambia la versione, si
# rilancia lo script e si fa il commit dei file e di questo script insieme.
#
# Regola per una libreria nuova: versione esatta, nessuna dipendenza a runtime (il file
# copiato è già completo), licenza permissiva (MIT o simile).
#
# Uso: scripts/librerie-backend.sh
set -euo pipefail
cd "$(dirname "$0")/.."
dest=backend/modelli/backend/librerie
lavoro=$(mktemp -d)
trap 'rm -rf "$lavoro"' EXIT

# pacchetto versione file-nel-pacchetto nome-nel-binario
librerie=(
  "htmx.org 2.0.11 dist/htmx.min.js htmx.min.js"
  "alpinejs 3.17.4 dist/cdn.min.js alpine.min.js"
)

for l in "${librerie[@]}"; do
  read -r pacchetto versione file nome <<<"$l"
  curl -fsSL "https://registry.npmjs.org/$pacchetto/-/${pacchetto##*/}-$versione.tgz" -o "$lavoro/p.tgz"
  rm -rf "$lavoro/package" && tar -xzf "$lavoro/p.tgz" -C "$lavoro"
  cp "$lavoro/package/$file" "$dest/$nome"
  licenza=$(node -p "require('$lavoro/package/package.json').license")
  echo "$nome: $pacchetto $versione ($licenza), $(wc -c < "$dest/$nome") byte"
done
