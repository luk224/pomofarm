#!/usr/bin/env bash
# Ejecuta scripts E2E contra un backend y un Vite PROPIOS (BD temporal, puertos 8180/5273),
# sin tocar la partida de desarrollo (puertos 8080/5173).
# Uso: tools/e2e/run_isolated.sh tools/e2e/p1_06_ui.py [otro.py ...]
set -uo pipefail
cd "$(dirname "$0")/../.."
[ $# -gt 0 ] || { echo "uso: $0 script.py [script.py ...]" >&2; exit 2; }

API_PORT=8180; WEB_PORT=5273
for port in $API_PORT $WEB_PORT; do
  if ss -ltn 2>/dev/null | grep -q ":$port "; then
    echo "El puerto $port está ocupado (¿queda un proceso de una ejecución anterior?):" >&2
    ss -ltnp 2>/dev/null | grep ":$port " >&2
    exit 1
  fi
done
D=$(mktemp -d /tmp/pomofarm-qa.XXXXXX)
cleanup() { [ -n "${WEB_PID:-}" ] && kill -- "-$WEB_PID" 2>/dev/null; kill "${API_PID:-}" 2>/dev/null; rm -rf "$D"; }
trap cleanup EXIT

(cd backend && go build -o "$D/pomofarm" ./cmd/pomofarm) || exit 1
POMOFARM_DB="$D/data/pomofarm.db" POMOFARM_BACKUPS="$D/backups" POMOFARM_ADDR="127.0.0.1:$API_PORT" "$D/pomofarm" >"$D/api.log" 2>&1 &
API_PID=$!
# setsid: Vite y sus hijos forman un grupo de procesos propio, que se mata entero al terminar.
setsid bash -c "cd frontend && POMOFARM_API=http://127.0.0.1:$API_PORT exec npx vite --port $WEB_PORT --strictPort --host 127.0.0.1" >"$D/web.log" 2>&1 &
WEB_PID=$!

for _ in $(seq 1 60); do
  curl -fs "http://127.0.0.1:$WEB_PORT/api/health" >/dev/null 2>&1 && break
  sleep 0.5
done
curl -fs "http://127.0.0.1:$WEB_PORT/api/health" >/dev/null || { echo "el entorno aislado no arrancó"; tail "$D"/*.log; exit 1; }

export POMOFARM_URL="http://127.0.0.1:$WEB_PORT" POMOFARM_DB="$D/data/pomofarm.db"
# Para los tests que reinician el backend (persistencia, recuperación sin conexión):
export POMOFARM_BIN="$D/pomofarm" POMOFARM_BACKUPS="$D/backups" POMOFARM_ADDR="127.0.0.1:$API_PORT" POMOFARM_API_PID="$API_PID"
rc=0
for s in "$@"; do
  echo "=== $s"
  python3 -I "$s" || rc=1
done
if [ $rc -ne 0 ]; then echo "--- log del backend (últimas líneas)"; tail -n 25 "$D/api.log"; fi
exit $rc
