#!/bin/sh
# PROTOTYPE G0: build Astro di N Foto sintetiche in un container con le risorse del server.
# Uso: ./bench.sh [N=25000] [CPU=2] [MEM=2g] [SWAP=MEM, memoria+swap totale]
set -eu
N=${1:-25000} CPU=${2:-2} MEM=${3:-2g}; SWAP=${4:-$MEM}
VOL=g0-astro-work
IMG=docker.io/library/node:22-bookworm-slim
cd "$(dirname "$0")"

# preparazione senza limiti: install + YAML (non misurata)
podman volume rm -f $VOL >/dev/null
podman run --rm -v "$PWD":/src:ro -v $VOL:/work $IMG sh -c "
  cd /src && tar cf - --exclude=node_modules --exclude=dist --exclude=.astro --exclude=content . | tar xf - -C /work
  cd /work && npm install --no-audit --no-fund --loglevel=error && node gen.mjs $N"

# build misurata, swap disattivato salvo SWAP > MEM
podman run --rm --cpus "$CPU" --memory "$MEM" --memory-swap "$SWAP" -e NODE_OPTIONS="${NODE_OPTIONS:-}" -v $VOL:/work -w /work $IMG sh -c '
  s=$(date +%s); npx astro build > build.log 2>&1; rc=$?; e=$(date +%s)
  tail -5 build.log
  echo "exit=$rc secondi=$((e-s)) picco_MB=$(( $(cat /sys/fs/cgroup/memory.peak) / 1048576 )) pagine=$(find dist -name index.html | wc -l)"'
