#!/usr/bin/env bash
# Restaura la BD desde un backup del volumen /backups.
# Uso: deploy/restore.sh [latest|pomofarm-YYYYMMDD-HHMMSS.db]
# Respeta DOCKER_HOST (p. ej. ssh://luk@100.102.106.119) para restaurar en wyse.
set -euo pipefail
cd "$(dirname "$0")/.."
which="${1:-latest}"

echo "Parando app..."
docker compose stop app web >/dev/null

docker run --rm -v pomofarm_data:/data -v pomofarm_backups:/backups alpine:3.21 sh -eu -c '
  if [ "$1" = latest ]; then f=$(ls /backups/pomofarm-*.db | sort | tail -1); else f="/backups/$1"; fi
  [ -f "$f" ] || { echo "No existe el backup: $f" >&2; exit 1; }
  echo "Restaurando $f"
  rm -f /data/pomofarm.db /data/pomofarm.db-wal /data/pomofarm.db-shm
  cp "$f" /data/pomofarm.db
  chown 10001 /data/pomofarm.db
' sh "$which"

docker compose up -d
echo "Restauración completada."
