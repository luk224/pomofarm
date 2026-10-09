#!/usr/bin/env bash
# QA de P5-05b: «restauración probada en wyse». Levanta en wyse un stack APARTE (proyecto pfqa, puerto 8091 solo en 127.0.0.1 de wyse),
# le copia la última copia REAL de producción (solo lectura), la restaura con deploy/restore.sh y compara con producción.
# No toca la partida real ni sus volúmenes, y al terminar borra todo lo del proyecto pfqa (volúmenes e imágenes incluidos).
# Uso: WYSE_HOST=luk@192.168.1.11 tools/e2e/qa_restore_remote.sh
set -uo pipefail
cd "$(dirname "$0")/../.."
WYSE_HOST="${WYSE_HOST:-luk@100.102.106.119}"; WYSE_TS_IP="${WYSE_TS_IP:-100.102.106.119}"
export DOCKER_HOST="ssh://$WYSE_HOST"
P=pfqa; PORT=8091; fail=0
ok()  { echo "PASS $1"; }
bad() { echo "FAIL $1"; fail=1; }
cleanup() { POMOFARM_PORT=$PORT POMOFARM_BIND=127.0.0.1 docker compose -p $P down -v --rmi local >/dev/null 2>&1; }
trap cleanup EXIT
if docker ps -a --format '{{.Names}}' | grep -q "^$P-"; then echo "ya existe un stack $P en wyse"; exit 1; fi
api() { docker compose -p $P exec -T app wget -qO- "http://127.0.0.1:8080$1" 2>/dev/null; }
wait_health() { for _ in $(seq 1 40); do api /api/health >/dev/null && return 0; sleep 1; done; return 1; }

export POMOFARM_PORT=$PORT POMOFARM_BIND=127.0.0.1
docker compose -p $P up -d --build >/dev/null 2>&1 && wait_health && ok "stack aparte (pfqa) levantado en wyse, instalación limpia" || { bad "el stack aparte no arranca"; exit 1; }
[ "$(api /api/state | python3 -c "import sys,json; print(json.load(sys.stdin)['player']['name'])")" = Granjero ] && ok "empieza vacío (jugador nuevo)" || bad "no empieza vacío"

# una copia RECIENTE de producción: se hace una copia ahora mismo (como la diaria; no modifica la partida) y se lee del volumen en solo lectura
docker compose -p pomofarm exec -T app pomofarm backup >/dev/null 2>&1 && ok "copia fresca hecha en producción (equivale a la diaria)" || bad "no se pudo hacer la copia en producción"
NAME=$(docker run --rm -v pomofarm_backups:/from:ro -v ${P}_backups:/to alpine:3.21 sh -c 'f=$(ls /from/pomofarm-*.db | sort | tail -1); cp "$f" /to/; basename "$f"' | tr -d '\r')
[ -n "$NAME" ] && ok "copiada a pfqa la copia real de producción ($NAME)" || { bad "no hay copia en producción"; exit 1; }
prod=$(curl -fsS -m 5 "http://$WYSE_TS_IP:8080/api/state") || { bad "no se puede leer producción"; exit 1; }

COMPOSE_PROJECT_NAME=$P deploy/restore.sh "$NAME" >/dev/null 2>&1 && ok "deploy/restore.sh restaura la copia en wyse" || bad "restore.sh falló en wyse"
wait_health && ok "tras restaurar la app responde" || bad "la app no responde tras restaurar"
restored=$(api /api/state)
python3 - "$prod" "$restored" <<'PYEOF' && ok "los datos coinciden con producción (nombre, 💧 de por vida, plantas y semillas)" || bad "los datos restaurados no coinciden con producción"
import sys, json
prod, rest = json.loads(sys.argv[1]), json.loads(sys.argv[2])
a = (prod['player']['name'], prod['player']['lifetime_focus'], sorted(s['key'] for s in prod['seeds'] if s['unlocked']), len(prod['plots']), prod['player']['silo_level'], prod['player']['season'])
b = (rest['player']['name'], rest['player']['lifetime_focus'], sorted(s['key'] for s in rest['seeds'] if s['unlocked']), len(rest['plots']), rest['player']['silo_level'], rest['player']['season'])
print('producción:', a); print('restaurada:', b)
sys.exit(0 if a == b else 1)
PYEOF
api /api/health | grep -q '"status":"ok"' && ok "y el juego responde con el esquema actual ($(api /api/health))" || bad "salud tras restaurar"
# una copia estropeada se rechaza también en wyse
docker run --rm -v ${P}_backups:/backups alpine:3.21 sh -c 'head -c 3000 /dev/urandom > /backups/pomofarm-29990101-000000.db'
COMPOSE_PROJECT_NAME=$P deploy/restore.sh latest >/dev/null 2>&1 && bad "wyse aceptó una copia estropeada" || ok "en wyse una copia estropeada se rechaza sin tocar nada"
api /api/health | grep -q '"status":"ok"' && ok "y la app sigue funcionando" || bad "la app dejó de funcionar"
# y comprobar que producción no se ha tocado
now=$(curl -fsS -m 5 "http://$WYSE_TS_IP:8080/api/health") && ok "producción sigue respondiendo ($now)" || bad "producción no responde"
[ $fail -eq 0 ] && echo "TODO OK" || echo "HAY FALLOS"
exit $fail
