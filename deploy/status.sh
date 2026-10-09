#!/usr/bin/env bash
# Estado de PomoFarm en wyse: contenedores, salud, esquema y antigüedad de la última copia de seguridad.
# Uso: WYSE_HOST=luk@192.168.1.11 deploy/status.sh        (sale con error si algo va mal)
set -uo pipefail
cd "$(dirname "$0")/.."
WYSE_HOST="${WYSE_HOST:-luk@100.102.106.119}"
WYSE_TS_IP="${WYSE_TS_IP:-100.102.106.119}"
export DOCKER_HOST="ssh://$WYSE_HOST"
project="${COMPOSE_PROJECT_NAME:-pomofarm}"
bad=0
docker compose -p "$project" ps
health=$(curl -fsS -m 5 "http://$WYSE_TS_IP:${POMOFARM_PORT:-8080}/api/health" 2>/dev/null) && echo "Salud: $health" || { echo "Salud: SIN RESPUESTA"; bad=1; }
last=$(docker compose -p "$project" exec -T app sh -c 'ls /backups/pomofarm-*.db 2>/dev/null | sort | tail -1' | tr -d '\r')
if [ -z "$last" ]; then echo "Backups: NINGUNO"; bad=1; else
  stamp=$(basename "$last" .db | sed 's/pomofarm-//'); epoch=$(date -u -d "${stamp:0:4}-${stamp:4:2}-${stamp:6:2} ${stamp:9:2}:${stamp:11:2}:${stamp:13:2}" +%s)
  age_h=$(( ($(date -u +%s) - epoch) / 3600 ))
  count=$(docker compose -p "$project" exec -T app sh -c 'ls /backups/pomofarm-*.db | wc -l' | tr -d '\r')
  echo "Backups: $count; la última es de hace ${age_h} h ($(basename "$last"))"
  [ "$age_h" -le 36 ] || { echo "AVISO: la última copia tiene más de 36 h"; bad=1; }
  docker compose -p "$project" exec -T app pomofarm verify "$last" >/dev/null 2>&1 && echo "La última copia es válida." || { echo "AVISO: la última copia NO es válida"; bad=1; }
fi
docker compose -p "$project" exec -T app sh -c 'du -sh /data /backups' 2>/dev/null
exit $bad
