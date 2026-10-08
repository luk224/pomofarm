"""E2E P2-06: descansos (banner, cuenta atrás, saltar, fin por sí solo con aviso, ajustes, ventana de 15 min).

Se ejecuta con tools/e2e/run_isolated.sh. Fuerza plantas maduras en la BD temporal.
"""
import json, os, re, subprocess, sys, urllib.request
from datetime import datetime, timedelta, timezone
from playwright.sync_api import sync_playwright

if "POMOFARM_URL" not in os.environ:
    sys.exit("Estos tests resetean la BD y no deben tocar tu partida. Ejecútalos con: tools/e2e/run_isolated.sh tools/e2e/p2_06_rest.py")
BASE = os.environ["POMOFARM_URL"]; DB = os.environ["POMOFARM_DB"]
OUT = "/tmp/pomofarm-e2e"; os.makedirs(OUT, exist_ok=True)
fails = []
def check(name, ok, detail=""):
    print(("PASS " if ok else "FAIL ") + name + (f"  [{detail}]" if detail else ""))
    if not ok: fails.append(name)
def call(m, path, body=None):
    r = urllib.request.Request(BASE + path, method=m, data=json.dumps(body).encode() if body is not None else None, headers={"Content-Type": "application/json"})
    try: return json.load(urllib.request.urlopen(r))
    except urllib.error.HTTPError as e: return {"_status": e.code, **json.load(e)}
def sql(q): subprocess.run(["sqlite3", DB, q], check=True)
def iso(dt): return dt.strftime("%Y-%m-%dT%H:%M:%S.%fZ")
CROPS = {"daisy": (600, 24), "sunflower": (2100, 54), "oak90": (5400, 162)}
def force(crop="daisy", matured_min_ago=1.0, harvested=0):
    grow, life = CROPS[crop]; t = datetime.now(timezone.utc) - timedelta(minutes=matured_min_ago); plant = "oak" if crop == "oak90" else crop
    sql(f"UPDATE plots SET state='mature', plant_type='{plant}', grow_s={grow}, life_s={life*3600}, harvested={harvested}, matured_at='{iso(t)}', "
        f"wilts_at='{iso(t + timedelta(hours=life))}', collected_to='{iso(t)}' WHERE id=1")
def reset():
    sql("UPDATE plots SET state='empty',plant_type=NULL,harvested=0,matured_at=NULL,wilts_at=NULL,collected_to=NULL,grow_s=NULL,life_s=NULL;"
        "UPDATE players SET rest_started_at=NULL, rest_until=NULL, silo_micro=0, silo_peak_micro_h=0, coins_milli=0;"
        "DELETE FROM settings WHERE key LIKE 'rest_%'; DELETE FROM pomodoro_events; DELETE FROM pomodoros")
def secs(t):
    m = re.search(r"(\d+):(\d{2})", t); return int(m.group(1)) * 60 + int(m.group(2))
btn = lambda pg, n: pg.get_by_role("button", name=n, exact=True)
INSTRUMENT = """
window.__audio = { ctxs: 0, osc: 0 }; const AC = window.AudioContext;
window.AudioContext = class extends AC { constructor() { super(); window.__audio.ctxs++ } createOscillator() { window.__audio.osc++; return super.createOscillator() } };
window.__notif = { created: [] }; window.__away = false; Document.prototype.hasFocus = function () { return !window.__away };
class N { constructor(t, o) { window.__notif.created.push(Object.assign({ title: t }, o)) } close() {} static requestPermission() { N.permission = 'granted'; return Promise.resolve('granted') } }
N.permission = 'granted'; window.Notification = N;
"""

sql("INSERT INTO settings (player_id,key,value) VALUES (1,'tutorial_done','1') ON CONFLICT(player_id,key) DO UPDATE SET value='1'")
with sync_playwright() as p:
    b = p.chromium.launch(args=["--use-gl=swiftshader", "--enable-unsafe-swiftshader"])
    ctx = b.new_context(viewport={"width": 1100, "height": 760}); ctx.add_init_script(INSTRUMENT)
    pg = ctx.new_page(); errs = []
    pg.on("console", lambda m: errs.append(m.text[:200]) if m.type == "error" and "409" not in m.text else None); pg.on("pageerror", lambda e: errs.append(str(e)[:200]))
    pg.goto(BASE); pg.wait_for_selector(".packet")
    # un Pomodoro cancelado desbloquea el audio (gesto del usuario) sin dar descanso
    pg.locator(".packet__body:not([disabled])").first.click(); btn(pg, "Plantar").click(); pg.wait_for_selector("[data-testid=timer]")
    btn(pg, "Cancelar").click(); pg.get_by_role("button", name="¿Cancelar? Se pierde la planta").click(); pg.wait_for_selector(".packet")
    check("cancelar un Pomodoro no da descanso", pg.locator("[data-testid=rest]").count() == 0)

    # 1) cosechar a los pocos minutos de terminar -> descanso de 5 min
    reset(); force("daisy", 1.0); pg.reload(); pg.wait_for_selector("[data-testid=harvest]"); pg.wait_for_timeout(800)
    check("antes de cosechar no hay descanso", pg.locator("[data-testid=rest]").count() == 0)
    btn(pg, "Cosechar").click(); pg.wait_for_selector("[data-testid=rest]")
    t0 = secs(pg.inner_text("[data-testid=rest-time]"))
    check("al cosechar aparece el descanso de 5 min (≤25 min de Pomodoro)", 295 <= t0 <= 300, f"{t0}s")
    check("el título de la pestaña muestra el descanso", "☕" in pg.title(), pg.title())
    check("el descanso no bloquea nada: sigue el dock de la parcela", pg.locator("[data-testid=clear]").count() == 1)
    pg.wait_for_timeout(2300); t1 = secs(pg.inner_text("[data-testid=rest-time]")); check("la cuenta atrás avanza sola", 1 <= t0 - t1 <= 4, f"{t0}->{t1}")
    pg.screenshot(path=f"{OUT}/rest_banner.png")
    two = ctx.new_page(); two.goto(BASE); two.wait_for_selector("[data-testid=rest]"); t2 = secs(two.inner_text("[data-testid=rest-time]")); two.close()
    check("otro dispositivo ve el mismo descanso (±3 s)", abs(t2 - t1) <= 3 + 1, f"{t1}s / {t2}s")

    # 2) saltar
    btn(pg, "Saltar descanso").click(); pg.wait_for_function("document.querySelectorAll('[data-testid=rest]').length === 0")
    check("saltar descanso lo termina", call("GET", "/api/state")["rest"] is None)
    check("y el título vuelve a normal", "☕" not in pg.title(), pg.title())
    before = pg.evaluate("window.__audio.osc"); pg.wait_for_timeout(500)
    check("saltar no suena ni avisa", pg.evaluate("window.__audio.osc") == before and not pg.evaluate("window.__notif.created.length"))

    # 3) el descanso termina solo: aviso con cuenco y notificación (pestaña sin foco)
    reset(); force("daisy", 1.0); pg.reload(); pg.wait_for_selector("[data-testid=harvest]"); btn(pg, "Cosechar").click(); pg.wait_for_selector("[data-testid=rest]")
    now = datetime.now(timezone.utc)
    sql(f"UPDATE players SET rest_started_at='{iso(now - timedelta(seconds=297))}', rest_until='{iso(now + timedelta(seconds=3))}'")
    pg.evaluate("window.__away = true"); pg.evaluate("window.dispatchEvent(new Event('focus'))"); osc0 = pg.evaluate("window.__audio.osc")
    pg.wait_for_function("document.querySelectorAll('[data-testid=rest]').length === 0", timeout=15000)
    pg.wait_for_selector(".toast:has-text('Descanso terminado')", timeout=5000)
    check("al acabar el descanso sale un aviso amable", True, pg.inner_text(".toast:has-text('Descanso terminado')"))
    check("suena el cuenco (8 osciladores)", pg.evaluate("window.__audio.osc") - osc0 == 8, str(pg.evaluate("window.__audio.osc") - osc0))
    n = pg.evaluate("window.__notif.created[window.__notif.created.length - 1]")
    check("y notificación del sistema silenciosa si no miras la pestaña", n["title"] == "Descanso terminado" and n["silent"] is True, str(n))
    pg.evaluate("window.__away = false")

    # 4) sembrar durante el descanso lo termina
    reset(); force("daisy", 1.0); pg.reload(); pg.wait_for_selector("[data-testid=harvest]"); btn(pg, "Cosechar").click(); pg.wait_for_selector("[data-testid=rest]")
    btn(pg, "Retirar planta").click(); pg.get_by_role("button", name="¿Seguro? Deja de producir").click(); pg.wait_for_selector(".packet")
    check("retirar la planta no termina el descanso", pg.locator("[data-testid=rest]").count() == 1)
    osc0 = pg.evaluate("window.__audio.osc")
    pg.locator(".packet__body:not([disabled])").first.click(); btn(pg, "Plantar").click(); pg.wait_for_selector("[data-testid=timer]"); pg.wait_for_timeout(400)
    check("empezar otro Pomodoro termina el descanso", pg.locator("[data-testid=rest]").count() == 0 and call("GET", "/api/state")["rest"] is None)
    pg.wait_for_timeout(500); check("terminarlo así no se anuncia como 'descanso terminado' (solo suena el efecto de plantar)", pg.evaluate("window.__audio.osc") == osc0 + 1 and pg.locator(".toast:has-text('Descanso terminado')").count() == 0)
    call("POST", "/api/pomodoros/active/cancel")

    # 5) duraciones por tramo: Girasol (35 min) -> 10; Roble 90 min -> 15
    reset(); force("sunflower", 1.0); pg.reload(); pg.wait_for_selector("[data-testid=harvest]"); btn(pg, "Cosechar").click(); pg.wait_for_selector("[data-testid=rest]")
    t = secs(pg.inner_text("[data-testid=rest-time]")); check("Pomodoro de 35 min: descanso de 10 min", 595 <= t <= 600, f"{t}s")
    reset(); force("oak90", 1.0); pg.reload(); pg.wait_for_selector("[data-testid=harvest]"); btn(pg, "Cosechar").click(); pg.wait_for_selector("[data-testid=rest]")
    t = secs(pg.inner_text("[data-testid=rest-time]")); check("Flow de 90 min: descanso de 15 min", 895 <= t <= 900, f"{t}s")

    # 6) cosechar tarde: no hay descanso
    reset(); force("daisy", 40.0); pg.reload(); pg.wait_for_selector("[data-testid=harvest]"); btn(pg, "Cosechar").click(); pg.wait_for_selector("[data-testid=clear]"); pg.wait_for_timeout(600)
    check("cosechar 40 min después de terminar no propone descanso", pg.locator("[data-testid=rest]").count() == 0)

    # 7) ajustes
    reset(); pg.reload(); pg.wait_for_selector(".packet"); pg.get_by_role("button", name="Ajustes de avisos").click(); pg.wait_for_selector("[data-testid=rest-enabled]")
    check("los ajustes ofrecen descansos con 5, 10 y 15 min por defecto", pg.get_by_test_id("rest-enabled").is_checked() and [pg.get_by_test_id(f"rest-input-rest_{k}_min").input_value() for k in ("short", "medium", "long")] == ["5", "10", "15"])
    inp = pg.get_by_test_id("rest-input-rest_short_min"); inp.fill("2"); inp.press("Enter"); pg.wait_for_timeout(500)
    check("cambiar la duración corta a 2 min se guarda en el servidor", call("GET", "/api/state")["settings"].get("rest_short_min") == "2")
    inp.fill("99"); pg.get_by_test_id("rest-input-rest_long_min").click(); pg.wait_for_timeout(500)
    check("un valor fuera de rango se limita a 60", call("GET", "/api/state")["settings"].get("rest_short_min") == "60", str(call("GET", "/api/state")["settings"]))
    inp.fill("2"); inp.press("Enter"); pg.wait_for_timeout(400); pg.keyboard.press("Escape")
    force("daisy", 1.0); pg.reload(); pg.wait_for_selector("[data-testid=harvest]"); btn(pg, "Cosechar").click(); pg.wait_for_selector("[data-testid=rest]")
    t = secs(pg.inner_text("[data-testid=rest-time]")); check("el siguiente descanso dura lo configurado (2 min)", 115 <= t <= 120, f"{t}s")
    btn(pg, "Saltar descanso").click(); pg.wait_for_function("document.querySelectorAll('[data-testid=rest]').length === 0")
    pg.get_by_role("button", name="Ajustes de avisos").click(); pg.get_by_test_id("rest-enabled").uncheck(); pg.wait_for_timeout(500)
    check("desactivar descansos oculta las duraciones", pg.get_by_test_id("rest-input-rest_short_min").count() == 0 and call("GET", "/api/state")["settings"]["rest_enabled"] == "0")
    pg.keyboard.press("Escape"); btn(pg, "Retirar planta").click(); pg.get_by_role("button", name="¿Seguro? Deja de producir").click(); pg.wait_for_selector(".packet")
    force("daisy", 1.0); pg.reload(); pg.wait_for_selector("[data-testid=harvest]"); btn(pg, "Cosechar").click(); pg.wait_for_selector("[data-testid=clear]"); pg.wait_for_timeout(600)
    check("con los descansos desactivados no aparece ninguno", pg.locator("[data-testid=rest]").count() == 0)
    sql("DELETE FROM settings WHERE key LIKE 'rest_%'")

    # 8) móvil
    reset(); force("daisy", 1.0); pg.set_viewport_size({"width": 360, "height": 640}); pg.reload(); pg.wait_for_selector("[data-testid=harvest]"); btn(pg, "Cosechar").click(); pg.wait_for_selector("[data-testid=rest]"); pg.wait_for_timeout(500)
    rb, db = pg.locator("[data-testid=rest]").bounding_box(), pg.locator(".dock").bounding_box()
    check("360 px: el banner cabe y queda encima del dock", rb["x"] >= 0 and rb["x"] + rb["width"] <= 360.5 and rb["y"] + rb["height"] <= db["y"] + 1, f"banner {rb['y']:.0f}-{rb['y']+rb['height']:.0f}, dock desde {db['y']:.0f}")
    check("360 px: sin scroll horizontal", not pg.evaluate("document.documentElement.scrollWidth > document.documentElement.clientWidth"))
    check("el botón Saltar descanso mide ≥ 44 px", btn(pg, "Saltar descanso").bounding_box()["height"] >= 44)
    pg.screenshot(path=f"{OUT}/rest_mobile.png")
    check("sin errores de consola", not errs, "; ".join(errs)[:240])
    b.close()
reset(); sql("DELETE FROM settings")
print("\nFALLOS:" if fails else "\nTODO OK", fails or "")
sys.exit(1 if fails else 0)
