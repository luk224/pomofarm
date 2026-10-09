#!/usr/bin/env bash
# Restaura la BD desde una copia del volumen /backups, sin arriesgar la partida actual.
#
# Uso: deploy/restore.sh [latest | pomofarm-YYYYMMDD-HHMMSS.db | pre-migration/pomofarm-vN-….db]
#      deploy/restore.sh --undo        vuelve a la partida que había justo antes de la última restauración
# Respeta DOCKER_HOST (p. ej. ssh://luk@192.168.1.11) para restaurar en wyse.
#
# Seguridad:
#  1. La copia se VERIFICA antes de tocar nada (integridad de SQLite y tablas del juego). Si no es válida, no se cambia nada.
#  2. La partida actual NO se borra: se guarda entera en /backups/pre-restore/<fecha>/ (se conservan las 5 últimas)
#     y `--undo` la recupera.
#  3. Al terminar se comprueba que la app responde; si no, se dice cómo deshacerlo.
set -euo pipefail
cd "$(dirname "$0")/.."
arg="${1:-latest}"
project="${COMPOSE_PROJECT_NAME:-pomofarm}"   # otro valor permite restaurar en un stack aislado (QA)
run_app() { docker compose -p "$project" run --rm --no-deps -T "$@"; }

if [ "$arg" = "--undo" ]; then
  set_dir=$(run_app --entrypoint sh app -c 'ls -d /backups/pre-restore/*/ 2>/dev/null | sort | tail -1' | tr -d '\r')
  [ -n "$set_dir" ] || { echo "No hay ninguna partida guardada de una restauración anterior." >&2; exit 1; }
  echo "Deshaciendo: volviendo a $set_dir"
  mode=undo; src="$set_dir"
else
  # 1) resolver y verificar la copia con la propia app, ANTES de parar nada
  src=$(run_app --entrypoint sh app -c 'if [ "$1" = latest ]; then ls /backups/pomofarm-*.db 2>/dev/null | sort | tail -1; else echo "/backups/$1"; fi' sh "$arg" | tr -d '\r')
  [ -n "$src" ] || { echo "No hay copias de seguridad en /backups." >&2; exit 1; }
  echo "Verificando $src ..."
  if ! run_app --entrypoint pomofarm app verify "$src"; then
    echo "La copia no es válida: NO se ha cambiado nada. Prueba con otra (docker compose run --rm app ls /backups)." >&2
    exit 1
  fi
  mode=restore
fi

echo "Parando app..."
docker compose -p "$project" stop app web >/dev/null

docker run --rm -v "${project}_data":/data -v "${project}_backups":/backups alpine:3.21 sh -eu -c '
  mode="$1"; src="$2"
  # guardar la partida actual entera (BD + WAL) antes de sustituirla
  if [ -f /data/pomofarm.db ]; then
    keep=/backups/pre-restore/$(date -u +%Y%m%d-%H%M%S); mkdir -p "$keep"
    for f in pomofarm.db pomofarm.db-wal pomofarm.db-shm; do [ -f /data/$f ] && cp /data/$f "$keep/"; done
    echo "Partida actual guardada en $keep"
    ls -d /backups/pre-restore/*/ | sort | head -n -5 | xargs -r rm -rf
  fi
  rm -f /data/pomofarm.db /data/pomofarm.db-wal /data/pomofarm.db-shm
  if [ "$mode" = undo ]; then
    for f in pomofarm.db pomofarm.db-wal pomofarm.db-shm; do [ -f "$src/$f" ] && cp "$src/$f" /data/; done
  else
    cp "$src" /data/pomofarm.db
  fi
  chown 10001 /data/pomofarm.db* 2>/dev/null || true
' sh "$mode" "$src"

docker compose -p "$project" up -d
port_check() { docker compose -p "$project" exec -T app wget -qO- http://127.0.0.1:8080/api/health 2>/dev/null; }
for _ in $(seq 1 40); do out=$(port_check) && break; sleep 1; done
if [ -n "${out:-}" ]; then
  echo "Restauración completada: $out"
  if [ "$mode" = restore ]; then echo "(Si algo no es lo esperado: deploy/restore.sh --undo)"; fi
else
  echo "La app no respondió tras restaurar. La partida anterior está en /backups/pre-restore/: deploy/restore.sh --undo" >&2
  exit 1
fi
