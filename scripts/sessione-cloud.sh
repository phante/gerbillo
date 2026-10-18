#!/usr/bin/env bash
# Hook SessionStart: nelle sessioni cloud avvia MySQL e carica i dati di test di
# Coppermine. MySQL lo installa il setup script dell'ambiente (docs/lavoro-remoto.md).
[ "${CLAUDE_CODE_REMOTE:-}" = true ] || exit 0
cd "${CLAUDE_PROJECT_DIR:-.}" || exit 0
dump=testdata/coppermine.sql.gz
command -v mysqld >/dev/null || { echo "MySQL assente: controlla il setup script dell'ambiente"; exit 0; }
service mysql start >/dev/null 2>&1 || true
[ -f "$dump" ] || exit 0
firma=$(sha256sum "$dump" | cut -c1-16)
if [ "$(mysql -N -e 'SELECT firma FROM gerbillo_test.caricato' 2>/dev/null)" != "$firma" ]; then
  gunzip -c "$dump" | mysql && mysql -e "
    CREATE DATABASE IF NOT EXISTS gerbillo_test;
    DROP TABLE IF EXISTS gerbillo_test.caricato;
    CREATE TABLE gerbillo_test.caricato (firma CHAR(16));
    INSERT INTO gerbillo_test.caricato VALUES ('$firma');
    CREATE USER IF NOT EXISTS 'gerbillo_ro'@'localhost' IDENTIFIED BY 'gerbillo';
    GRANT SELECT ON progettodighe_it_gallery.* TO 'gerbillo_ro'@'localhost';
    GRANT SELECT ON progettodighe_it_forum.* TO 'gerbillo_ro'@'localhost';"
  echo "dati di test Coppermine caricati in MySQL (utente gerbillo_ro, password gerbillo)"
fi
exit 0
