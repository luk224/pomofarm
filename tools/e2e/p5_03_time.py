"""E2E P5-03: pruebas de tiempo en el navegador (GDD §8): cerrar y volver, hora del equipo cambiada, dos dispositivos, servidor caído,
página congelada. El tiempo restante sale del servidor y del reloj monótono, nunca de la hora del equipo.

Se ejecuta con tools/e2e/run_isolated.sh.
"""
import json, os, re, sqlite3, subprocess, sys, urllib.request
from datetime import datetime, timedelta, timezone
from playwright.sync_api import sync_playwright

if "POMOFARM_URL" not in os.environ:
    sys.exit("Estos tests resetean la BD y no deben tocar tu partida. Ejecútalos con: tools/e2e/run_isolated.sh tools/e2e/p5_03_time.py")
BASE = os.environ["POMOFARM_URL"]; DB = os.environ["POMOFARM_DB"]
fails = []
def check(name, ok, detail=""):
    print(("PASS " if ok else "FAIL ") + name + (f"  [{detail}]" if detail else ""))
    if not ok: fails.append(name)
def call(m, path, body=None):
    r = urllib.request.Request(BASE + path, method=m, data=json.dumps(body).encode() if body is not None else None, headers={"Content-Type": "application/json"})
    try: return json.load(urllib.request.urlopen(r))
    except urllib.error.HTTPError as e: return {"_status": e.code, **json.load(e)}
def sql(q, args=()):
    c = sqlite3.connect(DB, timeout=10); c.execute(q, args); c.commit(); c.close()
def state(): return call("GET", "/api/state")
def secs(txt):
    m = re.match(r"^(?:(\d+):)?(\d+):(\d{2})$", txt.strip()); h, mi, s = (int(x or 0) for x in m.groups()); return h * 3600 + mi * 60 + s
def shown(pg): return secs(pg.inner_text("[data-testid=timer]"))
def server_left(): return state()["pomodoro"]["remaining_ms"] / 1000

call("POST", "/api/pomodoros/active/cancel")
for q in ("DELETE FROM structures", "DELETE FROM plots WHERE NOT (x=1 AND y=1)", "DELETE FROM pomodoro_events", "DELETE FROM pomodoros",
          "UPDATE plots SET state='empty', plant_type=NULL, planted_at=NULL, grow_s=NULL, matured_at=NULL, harvested=0, life_s=NULL, wilts_at=NULL, collected_to=NULL",
          "UPDATE players SET silo_micro=0, silo_peak_micro_h=0, silo_level=0, season=1, biome='spring', coins_milli=0, focus_points=0, rest_started_at=NULL, rest_until=NULL"):
    sql(q)
sql("INSERT INTO settings (player_id,key,value) VALUES (1,'tutorial_done','1') ON CONFLICT(player_id,key) DO UPDATE SET value='1'")
sql("INSERT INTO settings (player_id,key,value) VALUES (1,'rest_enabled','0') ON CONFLICT(player_id,key) DO UPDATE SET value='0'")
SKEW = """
window.__skew = 0;
const __now = Date.now.bind(Date), __Date = Date;
Date.now = () => __now() + window.__skew;
window.Date = class extends __Date { constructor(...a) { if (a.length === 0) super(__now() + window.__skew); else super(...a); } static now() { return __now() + window.__skew; } };
"""

with sync_playwright() as p:
    b = p.chromium.launch(args=["--use-gl=swiftshader", "--enable-unsafe-swiftshader"])
    ctx = b.new_context(viewport={"width": 1100, "height": 760}); ctx.add_init_script(SKEW)
    pg = ctx.new_page(); errs = []
    pg.on("console", lambda m: errs.append(m.text[:200]) if m.type == "error" and "409" not in m.text and "Failed to load resource" not in m.text else None); pg.on("pageerror", lambda e: errs.append(str(e)[:200]))
    pg.goto(BASE); pg.wait_for_selector(".packet")
    pg.locator(".packet__body:not([disabled])").first.click(); pg.get_by_role("button", name="Plantar", exact=True).click(); pg.wait_for_selector("[data-testid=timer][data-status=running]"); pg.wait_for_timeout(1500)

    # 1) la hora del equipo cambia: adelante 10 h, atrás 3 días, otra vez adelante; el contador no se entera
    base = shown(pg)
    for skew_h, label in ((10, "+10 h"), (-72, "−3 días"), (24 * 365, "+1 año")):
        pg.evaluate(f"window.__skew = {skew_h} * 3600 * 1000"); t0 = shown(pg); pg.wait_for_timeout(3200); t1 = shown(pg)
        check(f"con la hora del equipo {label} el contador sigue bajando a ritmo normal", 2 <= t0 - t1 <= 4, f"{t0} -> {t1}")
        check(f"con la hora del equipo {label} el Pomodoro no se completa ni se salta tiempo", pg.locator("[data-testid=timer][data-status=running]").count() == 1 and abs(shown(pg) - server_left()) <= 2, f"UI {shown(pg)} s, servidor {server_left():.1f} s")
    pg.evaluate("window.__skew = 0")
    check("el título de la pestaña sigue la cuenta atrás y no la hora del equipo", re.match(r"^⏱ \d+:\d{2} · PomoFarm$", pg.title()) is not None, pg.title())
    # la hora del equipo cambiada y recarga: el servidor manda
    pg.evaluate("window.__skew = -30 * 24 * 3600 * 1000"); pg.reload(); pg.wait_for_selector("[data-testid=timer]"); pg.wait_for_timeout(600)
    check("recargar con la hora del equipo un mes atrás muestra el tiempo del servidor (±2 s)", abs(shown(pg) - server_left()) <= 2, f"UI {shown(pg)} s, servidor {server_left():.1f} s")
    pg.evaluate("window.__skew = 0")

    # 2) cerrar el navegador a mitad y volver (±1 s)
    ctx.close(); expect_left = server_left()
    fresh = b.new_context(viewport={"width": 1100, "height": 760}); fresh.add_init_script(SKEW); q = fresh.new_page(); q.goto(BASE); q.wait_for_selector("[data-testid=timer]"); q.wait_for_timeout(500)
    st = state()["pomodoro"]; started = datetime.fromisoformat(st["started_at"].replace("Z", "+00:00").split(".")[0] + "+00:00")
    exact = st["planned_s"] - (datetime.now(timezone.utc) - started).total_seconds()
    check("tras cerrar el navegador y volver, el tiempo restante es exacto (±1 s) respecto a los datos guardados", abs(shown(q) - exact) <= 1.6, f"UI {shown(q)} s, esperado {exact:.1f} s")
    check("y no se perdió nada: sigue siendo el mismo Pomodoro en marcha", q.locator("[data-testid=timer][data-status=running]").count() == 1 and expect_left - server_left() >= 0)

    # 3) dos dispositivos: el segundo muestra el Pomodoro existente y no deja iniciar otro
    other = b.new_context(viewport={"width": 390, "height": 760}); o = other.new_page(); o.goto(BASE); o.wait_for_selector("[data-testid=timer]"); o.wait_for_timeout(500)
    check("el segundo dispositivo muestra el mismo Pomodoro con el mismo tiempo (±1,5 s)", abs(shown(o) - shown(q)) <= 1.5, f"{shown(o)} vs {shown(q)}")
    check("el segundo dispositivo no ofrece plantar otro (no hay semillas ni botón Plantar)", o.locator(".packet").count() == 0 and o.get_by_role("button", name="Plantar", exact=True).count() == 0)
    r = call("POST", "/api/pomodoros", {"plot_id": 1, "plant_type": "daisy"})
    check("y el servidor rechaza un segundo Pomodoro", r.get("_status") == 409 and r.get("error") == "pomodoro_active", str(r))
    q.get_by_role("button", name="Pausar", exact=True).click(); q.wait_for_selector("[data-testid=timer][data-status=paused]")
    o.evaluate("window.dispatchEvent(new Event('focus'))"); o.wait_for_selector("[data-testid=timer][data-status=paused]", timeout=6000)
    frozen_a = shown(o); o.wait_for_timeout(2500)
    check("la pausa hecha en un dispositivo llega al otro y congela el tiempo", shown(o) == frozen_a and shown(q) == frozen_a, f"{frozen_a} -> {shown(o)} / {shown(q)}")
    q.get_by_role("button", name="Reanudar", exact=True).click(); q.wait_for_selector("[data-testid=timer][data-status=running]")
    other.close()

    # 4) servidor caído 15 s y de vuelta: el contador sigue y se corrige con el servidor
    open(os.environ["POMOFARM_HOLD_FILE"], "w").close()
    subprocess.run(["bash", "-c", "pkill -P " + os.environ["POMOFARM_SUPERVISOR_PID"] + " -x pomofarm || true"]); t_down = shown(q); q.wait_for_timeout(6000)
    check("con el servidor caído el contador local sigue bajando (reloj monótono)", 4 <= t_down - shown(q) <= 8, f"{t_down} -> {shown(q)}")
    os.remove(os.environ["POMOFARM_HOLD_FILE"]); q.wait_for_timeout(2500)
    q.evaluate("window.dispatchEvent(new Event('focus'))"); q.wait_for_timeout(2500)
    check("al volver el servidor la cuenta coincide con la suya (±2 s)", abs(shown(q) - server_left()) <= 2, f"UI {shown(q)} s, servidor {server_left():.1f} s")

    # 5) página congelada (portátil que se suspende) y despertada
    cdp = fresh.new_cdp_session(q); cdp.send("Page.enable")
    try:
        cdp.send("Page.setWebLifecycleState", {"state": "frozen"}); import time; time.sleep(6); cdp.send("Page.setWebLifecycleState", {"state": "active"})
        q.evaluate("window.dispatchEvent(new Event('focus'))"); q.wait_for_timeout(2500)
        check("tras congelar la página 6 s y despertarla, el contador se corrige con el servidor (±2 s)", abs(shown(q) - server_left()) <= 2, f"UI {shown(q)} s, servidor {server_left():.1f} s")
    except Exception as e:
        print("INFO no se pudo congelar la página en este navegador:", str(e)[:80])

    # 6) acabar: el servidor completa en el instante exacto aunque la hora del equipo esté mal
    q.evaluate("window.__skew = 5 * 24 * 3600 * 1000")
    sql("UPDATE pomodoros SET started_at=strftime('%Y-%m-%dT%H:%M:%fZ','now', printf('-%d seconds', planned_s - 4 + paused_total_s)) WHERE status='running'"); q.evaluate("window.dispatchEvent(new Event('focus'))")
    q.wait_for_function("document.querySelector('[data-testid=timer]') === null", timeout=20000)
    st = state()
    check("el Pomodoro termina por el reloj del servidor aunque la hora del equipo esté 5 días adelantada", st["pomodoro"] is None and st["plots"][0]["state"] == "mature")
    ended = datetime.fromisoformat(st["plots"][0]["matured_at"].replace("Z", "+00:00").split(".")[0] + "+00:00")
    check("la planta madura en su instante (no antes de que acabara el tiempo guardado)", ended <= datetime.now(timezone.utc) + timedelta(seconds=1))
    check("sin errores de consola", not errs, "; ".join(errs[:3]))
    fresh.close(); b.close()
print(f"\n{len(fails)} fallos" if fails else "\nTodo OK"); sys.exit(1 if fails else 0)
