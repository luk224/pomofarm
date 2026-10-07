#!/usr/bin/env bash
# Despliega PomoFarm en wyse (Tailscale). Construye allí con Docker por SSH.
# Uso: deploy/deploy.sh          (pide confirmación)
#      deploy/deploy.sh -y       (sin preguntar)
set -euo pipefail
cd "$(dirname "$0")/.."

# Dirección por la que se llega a wyse por SSH/Docker: la de Tailscale, o la de la red local (p. ej.
# WYSE_HOST=luk@192.168.1.11 cuando estás en casa). NO afecta a dónde se publica el juego.
WYSE_HOST="${WYSE_HOST:-luk@100.102.106.119}"
# IP de Tailscale de wyse: el puerto se publica SOLO ahí (nunca en la LAN ni en internet), vengas por donde vengas.
WYSE_TS_IP="${WYSE_TS_IP:-100.102.106.119}"

echo "Destino: $WYSE_HOST  (el puerto ${POMOFARM_PORT:-8080} se publica solo en la IP de Tailscale $WYSE_TS_IP)"
if [ "${1:-}" != "-y" ]; then
  read -r -p "¿Desplegar ahora en wyse? [s/N] " ans
  [ "$ans" = "s" ] || { echo "Cancelado."; exit 1; }
fi

export DOCKER_HOST="ssh://$WYSE_HOST"
export POMOFARM_BIND="$WYSE_TS_IP"
docker compose up -d --build
docker compose ps

echo "Comprobando..."
for _ in $(seq 1 20); do
  if out=$(curl -fsS "http://$WYSE_TS_IP:${POMOFARM_PORT:-8080}/api/health" 2>/dev/null); then
    echo "OK: $out"; exit 0
  fi
  sleep 2
done
echo "La app no respondió a tiempo" >&2; exit 1
