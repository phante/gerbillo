#!/usr/bin/env bash
# Avvia in podman un MySQL con testdata/coppermine.sql.gz, per usare scripts/anteprima-gallery.sh
# e Playwright sul Mac. Stessa versione e stessi permessi della CI (.github/workflows/anteprima.yml).
# Ricrea il container a ogni giro: i dati sono solo quelli del dump. Porta solo su 127.0.0.1.
#
# Uso: scripts/mysql-locale.sh && scripts/anteprima-gallery.sh 127.0.0.1:3306
set -euo pipefail
cd "$(dirname "$0")/.."
nome=gerbillo-mysql
podman rm -f "$nome" >/dev/null
podman run -d --name "$nome" -p 127.0.0.1:3306:3306 -e MYSQL_ALLOW_EMPTY_PASSWORD=yes docker.io/library/mysql:8.0 >/dev/null
# Con -h si passa dal TCP: il server provvisorio dell'inizializzazione ascolta solo sul socket.
sql() { podman exec -i "$nome" mysql -h 127.0.0.1 -u root "$@"; }
for _ in {1..60}; do
  sql -e 'SELECT 1' >/dev/null 2>&1 && break
  sleep 2
done
gunzip -c testdata/coppermine.sql.gz | sql
sql -e "CREATE USER 'gerbillo_ro'@'%' IDENTIFIED BY 'gerbillo';
  GRANT SELECT ON progettodighe_it_gallery.* TO 'gerbillo_ro'@'%';
  GRANT SELECT ON progettodighe_it_forum.* TO 'gerbillo_ro'@'%';"
echo "MySQL pronto su 127.0.0.1:3306 (utente gerbillo_ro, password gerbillo)"
