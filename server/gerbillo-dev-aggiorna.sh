#!/usr/bin/env bash
# Gira ogni minuto sul server (gerbillo-dev.timer), come root. ADR 0011.
# Scarica l'immagine dev, ricrea il container se è cambiata, scrive lo stato
# per /gerbillo/admin/stato e manda allarmi Telegram. Nessun input dall'esterno
# oltre all'immagine.
set -uo pipefail
IMG=ghcr.io/phante/gerbillo:dev
NOME=gerbillo-dev
DIR=/srv/gerbillo-dev
ENV=/etc/gerbillo-dev.env
SOGLIA_SWAP_MB=1500
mkdir -p "$DIR/stato"
# shellcheck disable=SC1090
. "$ENV"

avvisa() {
  curl -s -o /dev/null -X POST "https://api.telegram.org/bot$TELEGRAM_TOKEN/sendMessage" \
    -d chat_id="$TELEGRAM_CHAT_ID" --data-urlencode text="gerbillo-dev: $1"
}

avvia() {
  docker rm -f "$NOME" >/dev/null 2>&1
  docker run -d --name "$NOME" --restart unless-stopped \
    -p 127.0.0.1:8090:8080 \
    --memory 384m --memory-swap 384m --cpus 0.5 \
    --read-only --tmpfs /tmp:size=200m \
    --log-opt max-size=10m --log-opt max-file=2 \
    --env-file "$ENV" \
    -v "$DIR/stato:/stato:ro" \
    "$IMG" >/dev/null
}

prima=$(docker image inspect -f '{{.Id}}' "$IMG" 2>/dev/null)
docker pull -q "$IMG" >/dev/null 2>&1
dopo=$(docker image inspect -f '{{.Id}}' "$IMG" 2>/dev/null)
in_uso=$(docker inspect -f '{{.Image}}' "$NOME" 2>/dev/null)
if [ -n "$dopo" ] && [ "$dopo" != "$in_uso" ]; then
  avvia && docker image prune -f >/dev/null
fi

# Stato: numeri soltanto, letti dal container in sola lettura.
running=$(docker inspect -f '{{.State.Running}}' "$NOME" 2>/dev/null || echo false)
oom=$(docker inspect -f '{{.State.OOMKilled}}' "$NOME" 2>/dev/null || echo false)
restart=$(docker inspect -f '{{.RestartCount}}' "$NOME" 2>/dev/null || echo 0)
swap_mb=$(free -m | awk '/^Swap:/ {print $3}')
{
  echo "aggiornato: $(date -Is)"
  echo "container: running=$running oom_killed=$oom restart=$restart"
  docker stats --no-stream --format 'stats: cpu {{.CPUPerc}} mem {{.MemUsage}} ({{.MemPerc}})' "$NOME" 2>/dev/null
  free -m
  df -h / | tail -1
} > "$DIR/stato/stato.json.tmp" && mv "$DIR/stato/stato.json.tmp" "$DIR/stato/stato.json"

# Allarmi: solo quando la condizione cambia, per non ripetere ogni minuto.
allarme=""
[ "$running" = true ] || allarme="container fermo"
[ "$oom" = true ] && allarme="OOM kill (supera 384 MiB)"
[ "${swap_mb:-0}" -gt "$SOGLIA_SWAP_MB" ] && allarme="${allarme:+$allarme; }swap host ${swap_mb} MiB"
if [ "$allarme" != "$(cat "$DIR/ultimo-allarme" 2>/dev/null)" ]; then
  if [ -n "$allarme" ]; then avvisa "$allarme"; else avvisa "tutto ok"; fi
  printf '%s' "$allarme" > "$DIR/ultimo-allarme"
fi
