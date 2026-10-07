#!/usr/bin/env bash
# Despliega PomoFarm en wyse (Tailscale). Construye allí con Docker por SSH.
# Uso: deploy/deploy.sh          (pide confirmación)
#      deploy/deploy.sh -y       (sin preguntar)
set -euo pipefail
cd "$(dirname "$0")/.."

WYSE_HOST="${WYSE_HOST:-luk@100.102.106.119}"
WYSE_TS_IP="${WYSE_HOST#*@}"

echo "Destino: $WYSE_HOST  (puerto ${POMOFARM_PORT:-8080} ligado a $WYSE_TS_IP)"
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
