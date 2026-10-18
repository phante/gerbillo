# Lavoro remoto

Come si sviluppa Gerbillo senza computer: sessioni cloud di Claude Code, avviate dall'app Claude (scheda Code) su iPhone o iPad. Decisioni prese il 18/10/2026. L'anteprima è descritta in ADR 0011.

## Decisioni già prese (non ridiscuterle)

- **Il server di produzione non si tocca da remoto.** Nessuna sessione apre SSH, lancia comandi sull'host o propone leve per farlo. Se serve un intervento sul server, lo si annota e lo si fa al rientro.
- **`testdata/` è pubblico per scelta.** Le dighe vengono dal Sito principale, che è pubblico. Coppermine è anonimizzato (username e Autore pseudonimizzati, EXIF ridotto alla data di scatto; Coppermine non salva il GPS). Si usa liberamente in test, commit e anteprima.
- **Claude fa merge su `main`** quando glielo chiedi. Una sessione alla volta. Ogni push su qualunque branch pubblica l'anteprima, e vince l'ultimo push.
- **Upload attivo sull'anteprima.** Lo protegge la password usa e getta, che arriva su Telegram.

## Come lavorare con Elvis

- **Ogni misura di sicurezza va motivata da un rischio concreto.** Niente blocchi né protezioni a priori su dati anonimizzati o di test. La prudenza vale per il server di produzione e per le credenziali vere.
- **Risposte concise, in italiano.** Dal telefono, quando serve un'azione sua, chiedi un punto alla volta.

## Cosa c'è nella sessione cloud

- Il repo, `CLAUDE.md`, `.claude/settings.json`. **Non ci sono** skill e plugin dell'utente, né `~/.claude`.
- MySQL 8.0, come in produzione. Lo installa il setup script dell'ambiente. L'hook `SessionStart` (`scripts/sessione-cloud.sh`) lo avvia e carica `testdata/coppermine.sql.gz`. Database e tabelle hanno gli stessi nomi della produzione. Credenziali: utente `gerbillo_ro`, password `gerbillo`, socket locale.
- Gli Originali si scaricano da `https://progettodighe.it/gallery/albums/` + `filepath` + `filename`.
- **GitHub**: `gh issue` e `gh pr` non funzionano, perché il proxy blocca GraphQL. Usa `gh api repos/phante/gerbillo/...` (REST) oppure gli strumenti GitHub integrati. Esempi:
  - `gh api repos/phante/gerbillo/issues?state=open`
  - `gh api repos/phante/gerbillo/issues/9/comments -f body=...`
  - `gh api -X PUT repos/phante/gerbillo/pulls/N/merge`

## Anteprima

- Ogni push avvia GitHub Actions (`.github/workflows/anteprima.yml`), che pubblica `ghcr.io/phante/gerbillo:dev` e manda una notifica Telegram.
- Entro circa un minuto, l'anteprima è su `https://dev.progettodighe.it/gerbillo/`. Branch e commit sono nella striscia gialla.
- L'admin è su `/gerbillo/admin/`. Utente `gerbillo`, password su Telegram, nuova a ogni avvio e ogni 24 ore.
- Lo stato è su `/gerbillo/admin/stato`: memoria del container e dell'host, swap, disco, riavvii. Un messaggio Telegram con una password nuova, che non arriva dopo un deploy, significa che il container è ripartito: controlla lo stato.
- Woodpecker (`deploy.prdi.it`) esegue solo `.woodpecker/simulazione.yml`, su push su `main` o a mano. Non builda e non pubblica nulla.

## Ambiente cloud su claude.ai (una volta)

- Rete: **Custom**, domini Trusted di default + `progettodighe.it` + `*.progettodighe.it`.
- Setup script:

  ```bash
  #!/bin/bash
  apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y mysql-server-8.0 || true
  ```

## Setup del server (una volta, dal Mac)

Dalla root del repo:

```bash
scp server/gerbillo-dev-aggiorna.sh server/gerbillo-dev.service server/gerbillo-dev.timer \
    server/nginx-gerbillo.conf ubuntu@prdi.it:/tmp/
ssh ubuntu@prdi.it
```

Sul server:

```bash
sudo install -m 755 /tmp/gerbillo-dev-aggiorna.sh /usr/local/bin/
sudo install -m 644 /tmp/gerbillo-dev.service /tmp/gerbillo-dev.timer /etc/systemd/system/
sudo install -m 600 /dev/null /etc/gerbillo-dev.env
sudoedit /etc/gerbillo-dev.env          # TELEGRAM_TOKEN=…  TELEGRAM_CHAT_ID=21449581
sudo install -m 644 /tmp/nginx-gerbillo.conf /etc/nginx/snippets/gerbillo.conf
# in /etc/nginx/sites-available/dev.progettodighe.it, nel server block 443 di
# dev.progettodighe.it, accanto a "include snippets/location-acme-challenge.conf;":
#   include snippets/gerbillo.conf;
sudo nginx -t && sudo systemctl reload nginx
sudo systemctl daemon-reload && sudo systemctl enable --now gerbillo-dev.timer
```

Verifica:
- `systemctl list-timers gerbillo-dev.timer`;
- `sudo /usr/local/bin/gerbillo-dev-aggiorna.sh` (al primo giro avvia il container);
- `curl -s localhost:8090/`.

Il package `ghcr.io/phante/gerbillo` deve essere **pubblico**, altrimenti il server non riesce a scaricarlo: GitHub → Packages → gerbillo → Package settings → Change visibility.

Al rientro: riportare in Ansible (`progettodighe-server`) la `location`, le unit e l'env file.
