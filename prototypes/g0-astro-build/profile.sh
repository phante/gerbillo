#!/bin/sh
# PROTOTYPE G0: profilo memoria della build. Separa anon (processo) da file (page cache).
# Uso: ./profile.sh [N=25000] [MEM=2g] [CMD="npx astro build"]
set -eu
N=${1:-25000} MEM=${2:-2g} CMD=${3:-npx astro build}
VOL=g0-astro-work
IMG=docker.io/library/node:22-bookworm-slim
cd "$(dirname "$0")"
podman volume rm -f $VOL >/dev/null
podman run --rm -v "$PWD":/src:ro -v $VOL:/work $IMG sh -c "
  cd /src && tar cf - --exclude=node_modules --exclude=dist --exclude=.astro --exclude=content . | tar xf - -C /work
  cd /work && npm install --no-audit --no-fund --loglevel=error >/dev/null && node gen.mjs $N" | grep -v 'npm notice'
podman run --rm --cpus 1 --memory "$MEM" --memory-swap "$MEM" -v $VOL:/work -w /work $IMG sh -c "
  (while sleep 0.5; do echo \$(date +%s) \$(grep -E '^(anon|file) ' /sys/fs/cgroup/memory.stat | awk '{printf \"%d \", \$2/1048576}'); done) > mem.log &
  s=\$(date +%s); $CMD > build.log 2>&1; rc=\$?; e=\$(date +%s)
  grep -E 'Completed|generating|Building|Collecting|built in|Error' build.log | head -20
  awk -v s=\$s '{t=\$1-s; if(\$2>a){a=\$2; ta=t} if(\$3>f)f=\$3} END{print \"anon_max_MB=\"a\" a t=\"ta\"s  file_max_MB=\"f}' mem.log
  echo \"exit=\$rc secondi=\$((e-s)) pagine=\$(find dist -name index.html | wc -l)\"
  awk -v s=\$s 'NR%6==0{print \$1-s\"s anon=\"\$2\" file=\"\$3}' mem.log"
