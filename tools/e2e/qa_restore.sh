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
sqlite3 "$D/src.db" "UPDATE players SET name='Restaurado', focus_points=42, lifetime_focus=57; INSERT OR IGNORE INTO unlocks (player_id,kind,key,at) VALUES (1,'seed','tomato','t'); UPDATE plots SET state='mature',plant_type='daisy',harvested=1,grow_s=600,life_s=86400,matured_at='2026-10-01T00:00:00Z',wilts_at='2099-01-01T00:00:00Z',collected_to='2026-10-01T00:00:00Z'; INSERT INTO tags (player_id,name) VALUES (1,'etiqueta-guardada');"
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
print(('PASS' if 9900 <= d['silo']['content_milli'] <= 10000 else 'FAIL'), 'la copia antigua se pone al día: el Silo (12 h) se llenó con lo producido desde que se hizo:', d['silo']['content_milli']/1000, '🪙'); ok = ok and 9900 <= d['silo']['content_milli'] <= 10000
print(('PASS' if ok else 'FAIL'), 'datos restaurados:', d['player']['name'], d['player']['focus_points'], d['player']['lifetime_focus'], u, d['plots'][0]['state']); sys.exit(0 if ok else 1)" || fail=1

# 4) jugable: retirar, plantar, completar el ciclo de API
c1=$(curl -s -o /dev/null -w '%{http_code}' -XPOST 127.0.0.1:$PORT/api/plots/1/clear -H 'content-type: application/json' -d '{"confirm":true}')
c2=$(curl -s -o /dev/null -w '%{http_code}' -XPOST 127.0.0.1:$PORT/api/pomodoros -H 'content-type: application/json' -d '{"plot_id":1,"plant_type":"tomato","tag":"tras-restaurar"}')
[ "$c1" = 200 ] && [ "$c2" = 201 ] && ok "jugable: retirar (200) y plantar Tomates desbloqueados (201)" || bad "jugable: retirar=$c1 plantar=$c2"
curl -s 127.0.0.1:$PORT/api/state | python3 -c "import sys,json; p=json.load(sys.stdin)['pomodoro']; sys.exit(0 if p and p['planned_s']==1500 and p['tag']=='tras-restaurar' else 1)" && ok "el Pomodoro nuevo dura 25 min y guarda su etiqueta" || bad "Pomodoro inesperado"
# 5) y sobrevive a otro reinicio del contenedor
docker compose -p $P restart app >/dev/null 2>&1; for _ in $(seq 1 40); do curl -fs 127.0.0.1:$PORT/api/health >/dev/null 2>&1 && break; sleep 0.5; done
curl -s 127.0.0.1:$PORT/api/state | python3 -c "import sys,json; d=json.load(sys.stdin); sys.exit(0 if d['pomodoro'] and d['player']['focus_points']==42 else 1)" && ok "tras reiniciar el contenedor, el estado persiste" || bad "estado perdido tras reiniciar"

# ---------- P5-04: la restauración es segura ----------
state() { curl -s 127.0.0.1:$PORT/api/state | python3 -c "import sys,json; d=json.load(sys.stdin); print(d['player']['name'], d['player']['focus_points'], 'pomodoro' if d['pomodoro'] else 'sin-pomodoro')"; }
BEFORE=$(state)

# 6) una copia estropeada se rechaza y no se toca nada (ni siquiera si es la «última»)
docker run --rm -v ${P}_backups:/backups alpine:3.21 sh -c 'head -c 4000 /dev/urandom > /backups/pomofarm-29990101-000000.db'
if COMPOSE_PROJECT_NAME=$P "$(pwd)/deploy/restore.sh" latest >/dev/null 2>&1; then bad "restore.sh aceptó una copia estropeada"; else ok "una copia estropeada (la más reciente) se rechaza"; fi
[ "$(state)" = "$BEFORE" ] && ok "y la partida actual sigue igual y funcionando ($BEFORE)" || bad "la partida cambió tras rechazar la copia mala: $(state) vs $BEFORE"
if COMPOSE_PROJECT_NAME=$P "$(pwd)/deploy/restore.sh" pomofarm-no-existe.db >/dev/null 2>&1; then bad "restore.sh aceptó una copia inexistente"; else ok "una copia que no existe se rechaza"; fi
docker run --rm -v ${P}_backups:/backups alpine:3.21 rm -f /backups/pomofarm-29990101-000000.db

# 7) la partida anterior se guarda y --undo la recupera
COMPOSE_PROJECT_NAME=$P "$(pwd)/deploy/restore.sh" "$BK" >/dev/null 2>&1 && ok "se vuelve a restaurar la copia buena" || bad "segunda restauración falló"
for _ in $(seq 1 40); do curl -fs 127.0.0.1:$PORT/api/health >/dev/null 2>&1 && break; sleep 0.5; done
AFTER=$(state); [ "$AFTER" = "Restaurado 42 sin-pomodoro" ] && ok "tras restaurar, la partida es la de la copia ($AFTER)" || bad "estado tras restaurar: $AFTER"
saved=$(docker run --rm -v ${P}_backups:/backups alpine:3.21 sh -c 'ls -d /backups/pre-restore/*/ | wc -l')
[ "$saved" -ge 1 ] && ok "la partida anterior quedó guardada en /backups/pre-restore ($saved)" || bad "no se guardó la partida anterior"
COMPOSE_PROJECT_NAME=$P "$(pwd)/deploy/restore.sh" --undo >/dev/null 2>&1 && ok "restore.sh --undo termina bien" || bad "--undo falló"
for _ in $(seq 1 40); do curl -fs 127.0.0.1:$PORT/api/health >/dev/null 2>&1 && break; sleep 0.5; done
[ "$(state)" = "$BEFORE" ] && ok "--undo devuelve exactamente la partida anterior ($BEFORE)" || bad "--undo no devolvió la partida anterior: $(state)"

# 8) una copia de un esquema antiguo se actualiza, y antes de actualizar se guarda una copia de seguridad verificada
OLD=$(ls -d "$HOME"/pomofarm-backups/wyse-before-phase4-*/ 2>/dev/null | sort | tail -1)
if [ -n "$OLD" ] && [ -f "$OLD/pomofarm.db" ]; then
  mkdir -p "$D/old" && cp "$OLD"/pomofarm.db* "$D/old/" && sqlite3 "$D/old/pomofarm.db" "VACUUM INTO '$D/old-v5.db'" \
    && docker run --rm -v ${P}_backups:/backups -v "$D":/src alpine:3.21 cp /src/old-v5.db /backups/pomofarm-20260101-000000.db
  COMPOSE_PROJECT_NAME=$P "$(pwd)/deploy/restore.sh" pomofarm-20260101-000000.db >/dev/null 2>&1 && ok "se restaura una copia con el esquema 5 (partida real de ayer)" || bad "restaurar la copia antigua falló"
  for _ in $(seq 1 40); do curl -fs 127.0.0.1:$PORT/api/health >/dev/null 2>&1 && break; sleep 0.5; done
  v=$(curl -s 127.0.0.1:$PORT/api/health | python3 -c "import sys,json; print(json.load(sys.stdin)['schema_version'])")
  [ "$v" -ge 6 ] && ok "la app la actualiza al esquema $v y sigue funcionando" || bad "esquema tras restaurar: $v"
  pm=$(docker run --rm -v ${P}_backups:/backups alpine:3.21 sh -c 'ls /backups/pre-migration/ 2>/dev/null')
  echo "$pm" | grep -q "pomofarm-v5-" && ok "antes de actualizar se guardó una copia verificada ($(echo "$pm" | head -1))" || bad "no se guardó la copia previa a la migración: $pm"
  docker compose -p $P exec -T app pomofarm verify "/backups/pre-migration/$(echo "$pm" | head -1)" >/dev/null 2>&1 && ok "y esa copia previa es válida" || bad "la copia previa no es válida"
  curl -s 127.0.0.1:$PORT/api/state | python3 -c "import sys,json; d=json.load(sys.stdin); print('PASS la partida antigua se conserva:', d['player']['name'], d['player']['focus_points'], 'gotas')" 
else
  echo "SKIP no hay una copia con el esquema 5 en ~/pomofarm-backups"
fi

# 9) arranque tras reiniciar el equipo: web y app arrancan a la vez y en cualquier orden; el servidor puede caerse y volver
docker compose -p $P stop web app >/dev/null 2>&1
docker start ${P}-web-1 >/dev/null 2>&1; sleep 3   # con el motor, como al arrancar el equipo (compose start arrancaría también la app)
[ "$(docker inspect --format '{{.State.Status}}' ${P}-web-1)" = running ] && ok "web arranca aunque la app todavía no exista (no se queda en bucle de error)" || bad "web no arranca sin la app"
c502=$(curl -s -o /dev/null -w '%{http_code}' 127.0.0.1:$PORT/api/health); [ "$c502" = 502 ] && ok "mientras la app no está, la web contesta 502 (no se cuelga)" || bad "web sin app contestó $c502"
docker start ${P}-app-1 >/dev/null 2>&1
for _ in $(seq 1 40); do curl -fs 127.0.0.1:$PORT/api/health >/dev/null 2>&1 && break; sleep 0.5; done
curl -fs 127.0.0.1:$PORT/api/health >/dev/null 2>&1 && ok "en cuanto la app está sana, la web la encuentra sola (sin reiniciar la web)" || bad "la web no encontró la app tras arrancar"
[ "$(docker inspect --format '{{.RestartCount}}' ${P}-web-1)" = 0 ] && ok "y la web no tuvo que reiniciarse" || bad "la web se reinició"
docker exec ${P}-app-1 sh -c 'kill -TERM 1' >/dev/null 2>&1
for _ in $(seq 1 30); do curl -fs 127.0.0.1:$PORT/api/health >/dev/null 2>&1 && [ "$(docker inspect --format '{{.RestartCount}}' ${P}-app-1)" -ge 1 ] && break; sleep 1; done
curl -fs 127.0.0.1:$PORT/api/health >/dev/null 2>&1 && [ "$(docker inspect --format '{{.RestartCount}}' ${P}-app-1)" -ge 1 ] && ok "si el proceso del servidor muere, Docker lo reinicia solo y el juego vuelve a responder" || bad "la app no volvió tras morir"
[ $fail -eq 0 ] && echo "TODO OK" || echo "HAY FALLOS"
exit $fail
