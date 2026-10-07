#!/usr/bin/env bash
# QA GDD §8: "Restaurar una copia de seguridad en una instalación limpia deja el juego jugable".
# Crea datos distintivos en una BD temporal (NO la de desarrollo), saca una copia con `pomofarm backup`,
# levanta un stack Docker nuevo y vacío (proyecto pfqa, puerto 8091), restaura y comprueba que se puede jugar.
set -uo pipefail
cd "$(dirname "$0")/../.."
P=pfqa; PORT=8091; D=$(mktemp -d /tmp/pomofarm-restore.XXXXXX); fail=0
ok()  { echo "PASS $1"; }
bad() { echo "FAIL $1"; fail=1; }
cleanup() { POMOFARM_PORT=$PORT docker compose -p $P down -v >/dev/null 2>&1; rm -rf "$D"; }
trap cleanup EXIT
if docker ps -a --format '{{.Names}}' | grep -q "^$P-"; then echo "ya existe un stack $P"; exit 1; fi

# 1) BD de origen con datos reconocibles (arrancar el binario crea y migra la BD)
(cd backend && go build -o "$D/pomofarm" ./cmd/pomofarm) || exit 1
POMOFARM_DB="$D/src.db" POMOFARM_BACKUPS="$D/none" POMOFARM_ADDR=127.0.0.1:8192 "$D/pomofarm" >/dev/null 2>&1 & SRC=$!
for _ in $(seq 1 40); do curl -fs 127.0.0.1:8192/api/health >/dev/null 2>&1 && break; sleep 0.25; done
kill $SRC; wait $SRC 2>/dev/null
sqlite3 "$D/src.db" "UPDATE players SET name='Restaurado', focus_points=42, lifetime_focus=57; INSERT OR IGNORE INTO unlocks (player_id,kind,key,at) VALUES (1,'seed','tomato','t'); UPDATE plots SET state='mature',plant_type='daisy',harvested=1,matured_at='2026-10-01T00:00:00Z',wilts_at='2026-10-02T00:00:00Z'; INSERT INTO tags (player_id,name) VALUES (1,'etiqueta-guardada');"
POMOFARM_DB="$D/src.db" POMOFARM_BACKUPS="$D/bk" "$D/pomofarm" backup >/dev/null; BK=$(basename "$(ls "$D"/bk/*.db | tail -1)")
ok "copia de seguridad creada ($BK)"

# 2) instalación limpia: stack nuevo, volúmenes vacíos
export POMOFARM_PORT=$PORT
docker compose -p $P up -d --build >/dev/null 2>&1 || { bad "el stack limpio no arranca"; exit 1; }
for _ in $(seq 1 40); do curl -fs 127.0.0.1:$PORT/api/health >/dev/null 2>&1 && break; sleep 0.5; done
n=$(curl -s 127.0.0.1:$PORT/api/state | python3 -c "import sys,json; print(json.load(sys.stdin)['player']['name'])")
[ "$n" = "Granjero" ] && ok "instalación limpia: jugador nuevo ($n)" || bad "instalación limpia inesperada ($n)"

# 3) restaurar
docker run --rm -v ${P}_backups:/backups -v "$D/bk":/src alpine:3.21 cp "/src/$BK" /backups/
COMPOSE_PROJECT_NAME=$P "$(pwd)/deploy/restore.sh" "$BK" >/dev/null 2>&1 && ok "deploy/restore.sh termina bien" || bad "restore.sh falló"
for _ in $(seq 1 40); do curl -fs 127.0.0.1:$PORT/api/health >/dev/null 2>&1 && break; sleep 0.5; done
S=$(curl -s 127.0.0.1:$PORT/api/state)
echo "$S" | python3 -c "
import sys,json; d=json.load(sys.stdin); u=[s['key'] for s in d['seeds'] if s['unlocked']]
ok = d['player']['name']=='Restaurado' and d['player']['focus_points']==42 and d['player']['lifetime_focus']==57 and sorted(u)==['daisy','tomato'] and d['plots'][0]['state']=='mature' and 'etiqueta-guardada' not in d['recent_tags']
print(('PASS' if ok else 'FAIL'), 'datos restaurados:', d['player']['name'], d['player']['focus_points'], d['player']['lifetime_focus'], u, d['plots'][0]['state']); sys.exit(0 if ok else 1)" || fail=1

# 4) jugable: retirar, plantar, completar el ciclo de API
c1=$(curl -s -o /dev/null -w '%{http_code}' -XPOST 127.0.0.1:$PORT/api/plots/1/clear -H 'content-type: application/json' -d '{"confirm":true}')
c2=$(curl -s -o /dev/null -w '%{http_code}' -XPOST 127.0.0.1:$PORT/api/pomodoros -H 'content-type: application/json' -d '{"plot_id":1,"plant_type":"tomato","tag":"tras-restaurar"}')
[ "$c1" = 200 ] && [ "$c2" = 201 ] && ok "jugable: retirar (200) y plantar Tomates desbloqueados (201)" || bad "jugable: retirar=$c1 plantar=$c2"
curl -s 127.0.0.1:$PORT/api/state | python3 -c "import sys,json; p=json.load(sys.stdin)['pomodoro']; sys.exit(0 if p and p['planned_s']==1500 and p['tag']=='tras-restaurar' else 1)" && ok "el Pomodoro nuevo dura 25 min y guarda su etiqueta" || bad "Pomodoro inesperado"
# 5) y sobrevive a otro reinicio del contenedor
docker compose -p $P restart app >/dev/null 2>&1; for _ in $(seq 1 40); do curl -fs 127.0.0.1:$PORT/api/health >/dev/null 2>&1 && break; sleep 0.5; done
curl -s 127.0.0.1:$PORT/api/state | python3 -c "import sys,json; d=json.load(sys.stdin); sys.exit(0 if d['pomodoro'] and d['player']['focus_points']==42 else 1)" && ok "tras reiniciar el contenedor, el estado persiste" || bad "estado perdido tras reiniciar"
[ $fail -eq 0 ] && echo "TODO OK" || echo "HAY FALLOS"
exit $fail
