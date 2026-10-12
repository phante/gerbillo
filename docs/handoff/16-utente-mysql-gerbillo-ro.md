# Handoff a progettodighe-server: utente MySQL `gerbillo_ro`

Origine: phante/gerbillo#16. Questo file va copiato in `progettodighe-server` come ticket
(`.scratch/gerbillo-ro/issues/01-utente-gerbillo-ro.md`, `Status: ready-for-agent`).

## Perché

Gerbillo è la nuova gallery di ProgettoDighe e sostituirà Coppermine. La sua Importazione
legge sempre dal MySQL di produzione, in due modi:

- dal Mac, con un tunnel SSH verso `/run/mysqld/mysqld.sock`;
- dal server, collegandosi direttamente al socket.

Il modello è quello di `prdi-mcp sync` (`internal/forumsync`). Il driver Go si collega come
`unix`, quindi l'host MySQL dell'utente è `localhost`.

Gerbillo non deve poter leggere dati personali, nemmeno se il suo codice sbaglia query. Per
questo i permessi sono `GRANT SELECT` **per colonna**, non per tabella.

## Cosa fare

Crea l'utente `gerbillo_ro@localhost` con un playbook Ansible idempotente. Non crearlo a mano.
La password va nel vault, per esempio `vault_gerbillo_ro_password`, cifrata con
`ansible-vault encrypt_string`. Usa `caching_sha2_password`, perché il ticket
`.scratch/mysql-native-password/` porta via le utenze da `mysql_native_password`.

Concedi solo questi permessi, e nient'altro (niente `USAGE` su altri database, niente `SHOW VIEW`,
niente `PROCESS`):

| database | tabella | colonne in `SELECT` |
|---|---|---|
| `progettodighe_it_gallery` | `cpg1410_pictures` | `pid, aid, filepath, filename, filesize, pwidth, pheight, hits, ctime, owner_id, title, caption, keywords, approved, user1, user2, user3` |
| `progettodighe_it_gallery` | `cpg1410_albums` | `aid, title, description, category, pos, visibility, owner, keyword` |
| `progettodighe_it_gallery` | `cpg1410_categories` | `cid, owner_id, name, description, pos, parent` |
| `progettodighe_it_gallery` | `cpg1410_exif` | `pid, exifData` |
| `progettodighe_it_forum` | `phpbb_users` | `user_id, username` |

Le colonne di `cpg1410_albums` e `cpg1410_categories` sono una proposta di Gerbillo per la
Mappatura (G2). Restano fuori di proposito `alb_password`, `alb_password_hint`,
`moderator_group` e i contatori. Se una colonna non esiste in Coppermine 1.6.29, il `GRANT`
fallisce: in quel caso toglila dalla lista e segnalalo in phante/gerbillo#16. Non sostituirla
con un permesso sulla tabella intera.

Per costruzione restano illeggibili, tra le altre:

- `cpg1410_pictures.pic_raw_ip`, `pic_hdr_ip`, `lasthit_ip`, `guest_token`;
- ogni colonna di `phpbb_users` diversa da `user_id` e `username`, email comprese.

## Indicazioni di implementazione

- Mettilo nel ruolo `progettodighe_it`, accanto a `prepare_gallery.yaml`, con un tag proprio
  (per esempio `gerbillo`). Ricorda la trappola numero 1 di `CLAUDE.md`: se usi
  `include_tasks`, serve `apply: tags:`.
- `ansible.mysql.mysql_user` accetta privilegi per colonna nella forma a dizionario, per esempio
  `"progettodighe_it_forum.phpbb_users": "SELECT (user_id, username)"`. Usa `append_privs: false`,
  così ogni privilegio in più fatto a mano viene revocato.
- Verifica l'idempotenza: la seconda esecuzione deve dare `changed=0`. Con i privilegi per
  colonna `mysql_user` può segnare `changed` a ogni giro, perché confronta male ordine e spazi
  delle colonne. Se succede, prova a scrivere le colonne nell'ordine in cui MySQL le restituisce
  in `SHOW GRANTS`. Se non basta, usa `mysql_query` con un controllo su
  `information_schema.COLUMN_PRIVILEGES`, ma tieni `mysql_user` per creare l'utente.
- Prova prima con `--check`, poi applica.
- Non serve toccare nginx, php-fpm o i permessi del socket: il socket è già raggiungibile
  dall'utente SSH che usa `prdi-mcp`.

## Fatto quando

- Il playbook crea l'utente in modo idempotente: seconda esecuzione con `changed=0`.
- `SHOW GRANTS FOR 'gerbillo_ro'@'localhost'` mostra solo le colonne della tabella sopra.
- Con `gerbillo_ro`, `SELECT pid, title FROM progettodighe_it_gallery.cpg1410_pictures LIMIT 1`
  funziona.
- Con `gerbillo_ro`, queste query falliscono con `ERROR 1143` (column access denied) o `1142`:
  - `SELECT pic_raw_ip FROM progettodighe_it_gallery.cpg1410_pictures LIMIT 1`
  - `SELECT * FROM progettodighe_it_gallery.cpg1410_pictures LIMIT 1`
  - `SELECT user_email FROM progettodighe_it_forum.phpbb_users LIMIT 1`
  - `SELECT * FROM progettodighe_it_gallery.cpg1410_comments LIMIT 1`
- Un ADR breve in `progettodighe-server/docs/adr/` registra l'utenza e il perché dei permessi
  per colonna.

## Cosa restituire a Gerbillo

Commenta phante/gerbillo#16 con:

- il commit del playbook;
- il nome della variabile vault con la password (non la password);
- l'output di `SHOW GRANTS` e delle query di verifica;
- le colonne eventualmente tolte dalla lista.
