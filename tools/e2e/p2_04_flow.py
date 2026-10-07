"""E2E P2-04: Manzano, Roble mágico y modo Flow (deslizante 60–120 min contra la tabla del servidor).

Se ejecuta con tools/e2e/run_isolated.sh.
"""
import json, os, re, subprocess, sys, urllib.request
from playwright.sync_api import sync_playwright

if "POMOFARM_URL" not in os.environ:
    sys.exit("Estos tests resetean la BD y no deben tocar tu partida. Ejecútalos con: tools/e2e/run_isolated.sh tools/e2e/p2_04_flow.py")
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
btn = lambda pg, n: pg.get_by_role("button", name=n, exact=True)
def num(t): return float(re.search(r"[\d]+(?:,\d+)?", t.replace(".", "")).group().replace(",", "."))

sql("INSERT OR IGNORE INTO unlocks (player_id,kind,key,at) VALUES (1,'seed','apple','t'),(1,'seed','oak','t');"
    "INSERT INTO settings (player_id,key,value) VALUES (1,'tutorial_done','1') ON CONFLICT(player_id,key) DO UPDATE SET value='1'")
table = {r["duration_min"]: r for r in call("GET", "/api/flow")}
check("el servidor publica la tabla Flow de 61 minutos (60–120)", len(table) == 61 and min(table) == 60 and max(table) == 120, str(len(table)))

with sync_playwright() as p:
    b = p.chromium.launch(args=["--use-gl=swiftshader", "--enable-unsafe-swiftshader"])
    pg = b.new_page(viewport={"width": 1100, "height": 760}); errs = []
    pg.on("console", lambda m: errs.append(m.text[:200]) if m.type == "error" and "409" not in m.text else None); pg.on("pageerror", lambda e: errs.append(str(e)[:200]))
    pg.goto(BASE); pg.wait_for_selector(".packet")

    check("el Manzano y el Roble están desbloqueados", pg.locator(".packet__unlock").count() == 2 and pg.locator(".packet__body:not([disabled])").count() == 3, f"{pg.locator('.packet__body:not([disabled])').count()} libres, {pg.locator('.packet__unlock').count()} por desbloquear")
    oak = pg.get_by_role("radio", name=re.compile("Roble")); apple = pg.get_by_role("radio", name=re.compile("Manzano"))
    check("el sobre del Roble anuncia el rango 60–120 min", "60–120 min" in oak.inner_text(), oak.inner_text().replace("\n", " "))
    check("sin elegir el Roble no hay deslizante", pg.locator("[data-testid=flow]").count() == 0)
    apple.click(); check("el Manzano tiene duración fija: sin deslizante", pg.locator("[data-testid=flow]").count() == 0)

    oak.click(); pg.wait_for_selector("[data-testid=flow]")
    rng = pg.locator("input[type=range]")
    check("deslizante de 60 a 120 de uno en uno", rng.get_attribute("min") == "60" and rng.get_attribute("max") == "120" and rng.get_attribute("step") == "1")
    check("empieza en 60 min (1:00:00)", rng.input_value() == "60" and "1:00:00" in pg.inner_text("[data-testid=flow-minutes]"))
    for m in (60, 75, 90, 91, 97, 105, 120):
        rng.fill(str(m)); pg.wait_for_timeout(120)
        txt = pg.inner_text("[data-testid=flow-summary]"); want = table[m]
        reward = int(re.search(r"\+(\d+)", txt).group(1)); life = num(re.search(r"vive ([\d.,]+) h", txt).group(1))
        check(f"{m} min: la vista previa coincide con el servidor (+{want['reward']} 💧, {want['life_h']:.1f} h)", reward == want["reward"] and abs(life - want["life_h"]) < 0.06, txt)
    rng.fill("90"); rng.focus(); pg.keyboard.press("ArrowRight"); pg.keyboard.press("ArrowRight"); pg.keyboard.press("ArrowLeft")
    check("el teclado mueve el deslizante de minuto en minuto", rng.input_value() == "91", rng.input_value())
    check("el texto accesible dice minutos y recompensa", "91 minutos" in rng.get_attribute("aria-valuetext") and "gotas" in rng.get_attribute("aria-valuetext"), rng.get_attribute("aria-valuetext"))
    pg.screenshot(path="/tmp/pomofarm-e2e/flow_slider.png")
    apple.click(); oak.click(); check("al volver al Roble recuerda la duración elegida", pg.locator("input[type=range]").input_value() == "91")

    # plantar un Roble de 75 min
    pg.locator("input[type=range]").fill("75"); btn(pg, "Plantar").click(); pg.wait_for_selector("[data-testid=timer][data-status=running]")
    st = call("GET", "/api/state")["pomodoro"]
    check("el servidor planta un Pomodoro de 75 min (4500 s)", st["plant_type"] == "oak" and st["planned_s"] == 4500, f"{st['plant_type']} {st['planned_s']} s")
    t = pg.inner_text("[data-testid=timer]"); check("el temporizador muestra horas (1:14:xx o 1:15:00)", re.match(r"1:1[45]:\d\d", t) is not None, t)
    ring = pg.inner_text("[data-testid=timer-ring]"); check("el anillo también (1:1x:xx)", re.match(r"1:1[45]:\d\d", ring.strip()) is not None, ring.strip())
    check("el título de la pestaña muestra horas", re.search(r"1:1[45]:\d\d", pg.title()) is not None, pg.title())
    pg.screenshot(path="/tmp/pomofarm-e2e/flow_running.png")
    btn(pg, "Cancelar").click(); pg.get_by_role("button", name="¿Cancelar? Se pierde la planta").click(); pg.wait_for_selector(".packet")

    # el Manzano: 45 min fijos
    pg.get_by_role("radio", name=re.compile("Manzano")).click(); btn(pg, "Plantar").click(); pg.wait_for_selector("[data-testid=timer][data-status=running]")
    st = call("GET", "/api/state")["pomodoro"]; check("el Manzano planta 45 min (2700 s)", st["planned_s"] == 2700, str(st["planned_s"]))
    call("POST", "/api/pomodoros/active/cancel"); pg.reload(); pg.wait_for_selector(".packet")

    # móvil
    pg.set_viewport_size({"width": 360, "height": 640}); pg.reload(); pg.wait_for_selector(".packet"); pg.get_by_role("radio", name=re.compile("Roble")).click(); pg.wait_for_selector("[data-testid=flow]")
    fb = pg.locator("[data-testid=flow]").bounding_box(); db = pg.locator(".dock").bounding_box()
    check("360 px: el deslizante cabe en el dock", fb["x"] >= db["x"] and fb["x"] + fb["width"] <= db["x"] + db["width"] + 0.5, f"{fb['x']:.0f}-{fb['x']+fb['width']:.0f} en dock {db['x']:.0f}-{db['x']+db['width']:.0f}")
    check("360 px: el deslizante usa todo el ancho útil (≥ 280 px)", fb["width"] >= 280, f"{fb['width']:.0f} px")
    check("360 px: el dock sigue dentro de la pantalla y sin scroll horizontal", db["x"] >= 0 and db["x"] + db["width"] <= 360.5 and not pg.evaluate("document.documentElement.scrollWidth > document.documentElement.clientWidth"))
    check("360 px: Plantar sigue visible", btn(pg, "Plantar").is_visible() and btn(pg, "Plantar").bounding_box()["y"] + 44 <= 640, str(btn(pg, "Plantar").bounding_box()))
    check("el deslizante tiene un área táctil ≥ 36 px", pg.locator("input[type=range]").bounding_box()["height"] >= 36)
    pg.screenshot(path="/tmp/pomofarm-e2e/flow_mobile.png")
    check("sin errores de consola", not errs, "; ".join(errs)[:240])
    b.close()
sql("DELETE FROM unlocks WHERE key<>'daisy'; DELETE FROM settings; DELETE FROM pomodoro_events; DELETE FROM pomodoros")
print("\nFALLOS:" if fails else "\nTODO OK", fails or "")
sys.exit(1 if fails else 0)
