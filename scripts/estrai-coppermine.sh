#!/usr/bin/env bash
# Estrae da Coppermine (MySQL di produzione) le sole colonne concesse a gerbillo_ro (#16),
# in esadecimale per conservare i byte grezzi. Solo letture, nessuna scrittura sul server.
# L'output grezzo NON va committato: lo anonimizza scripts/anonimizza-coppermine.py.
# Uso: scripts/estrai-coppermine.sh [percorso/progettodighe-server]
set -euo pipefail
srv="${1:-$HOME/Siti/progettodighe-server}"
out="$(cd "$(dirname "$0")/.." && pwd)/grezzo"
mkdir -p "$out"
pw=$(cd "$srv" && .venv/bin/ansible prdi.it -m debug -a 'var=progettodighe_it_gerbillo_ro_password' 2>/dev/null \
  | sed -n 's/.*": "\(.*\)".*/\1/p') || true
[ -n "$pw" ] || { echo "password di gerbillo_ro non trovata nel vault" >&2; exit 1; }

g=progettodighe_it_gallery
f=progettodighe_it_forum
# Le colonne visibili in information_schema sono solo quelle concesse.
sql=$(cat <<SQL
SELECT 'META', TABLE_SCHEMA, TABLE_NAME, COLUMN_NAME, COLUMN_TYPE, IFNULL(COLLATION_NAME,''), IS_NULLABLE, ORDINAL_POSITION
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA IN ('$g','$f') ORDER BY TABLE_SCHEMA, TABLE_NAME, ORDINAL_POSITION;
SELECT 'cpg1410_pictures', HEX(pid), HEX(aid), HEX(filepath), HEX(filename), HEX(filesize), HEX(pwidth), HEX(pheight), HEX(hits), HEX(ctime), HEX(owner_id), HEX(title), HEX(caption), HEX(keywords), HEX(approved), HEX(user1), HEX(user2), HEX(user3) FROM $g.cpg1410_pictures;
SELECT 'cpg1410_albums', HEX(aid), HEX(title), HEX(description), HEX(category), HEX(pos), HEX(visibility), HEX(owner), HEX(keyword) FROM $g.cpg1410_albums;
SELECT 'cpg1410_categories', HEX(cid), HEX(owner_id), HEX(name), HEX(description), HEX(pos), HEX(parent) FROM $g.cpg1410_categories;
SELECT 'cpg1410_exif', HEX(pid), HEX(exifData) FROM $g.cpg1410_exif;
SELECT 'phpbb_users', HEX(user_id), HEX(username) FROM $f.phpbb_users
  WHERE user_id IN (SELECT owner_id FROM $g.cpg1410_pictures) OR user_id IN (SELECT owner FROM $g.cpg1410_albums);
SELECT 'VERSION', VERSION();
SQL
)
# La password viaggia su stdin, mai in argv né su disco.
{ printf '%s\n' "$pw"; printf '%s\n' "$sql"; } | ssh ubuntu@prdi.it \
  'read -r MYSQL_PWD; export MYSQL_PWD; mysql -u gerbillo_ro --batch --skip-column-names --default-character-set=binary' \
  | gzip > "$out/coppermine.tsv.gz"
echo "scritto $out/coppermine.tsv.gz ($(du -h "$out/coppermine.tsv.gz" | cut -f1))"
