#!/usr/bin/env bash
# Trae a ESTE equipo una copia de los backups de wyse (volumen /backups), para no depender de un solo disco.
# Solo lee en wyse. Verifica cada archivo traído con `pomofarm verify` (compilado aquí) y avisa de los que no valgan.
# Uso: WYSE_HOST=luk@192.168.1.11 deploy/fetch-backups.sh [directorio]     (por defecto ~/pomofarm-backups/wyse)
set -euo pipefail
cd "$(dirname "$0")/.."
WYSE_HOST="${WYSE_HOST:-luk@100.102.106.119}"
dest="${1:-$HOME/pomofarm-backups/wyse}"
project="${COMPOSE_PROJECT_NAME:-pomofarm}"
mkdir -p "$dest"
echo "Trayendo backups de $WYSE_HOST a $dest ..."
DOCKER_HOST="ssh://$WYSE_HOST" docker run --rm -v "${project}_backups":/b:ro alpine:3.21 tar c -C /b . | tar x -C "$dest"
bin=$(mktemp -d)/pomofarm; (cd backend && go build -o "$bin" ./cmd/pomofarm)
n=0; bad=0
while IFS= read -r f; do
  n=$((n+1))
  "$bin" verify "$f" >/dev/null 2>&1 || { echo "INVÁLIDO: $f"; bad=$((bad+1)); }
done < <(find "$dest" -name 'pomofarm-*.db' -type f | sort)
latest=$(find "$dest" -maxdepth 1 -name 'pomofarm-*.db' -type f | sort | tail -1)
echo "$n copias en $dest ($bad inválidas). La más reciente: ${latest:-ninguna}"
[ "$bad" -eq 0 ]
