"""E2E P2-03: tienda (parcelas y Silo con 💧), selección de parcela, dock por parcela, 16 parcelas, móvil.

Se ejecuta con tools/e2e/run_isolated.sh.
"""
import json, os, re, subprocess, sys, urllib.request
from datetime import datetime, timedelta, timezone
from playwright.sync_api import sync_playwright

if "POMOFARM_URL" not in os.environ:
    sys.exit("Estos tests resetean la BD y no deben tocar tu partida. Ejecútalos con: tools/e2e/run_isolated.sh tools/e2e/p2_03_shop.py")
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
btn = lambda pg, n: pg.get_by_role("button", name=n, exact=True)
def state(): return call("GET", "/api/state")
def focus(n): sql(f"UPDATE players SET focus_points={n}")
PROJECT = """([x, z]) => { const t = window.__three; const v = new t.camera.position.constructor(x, 0.45, z).project(t.camera);
  const r = t.gl.domElement.getBoundingClientRect(); return [r.left + (v.x + 1) / 2 * r.width, r.top + (1 - v.y) / 2 * r.height] }"""
RINGS = "(() => { let n = 0; window.__three.scene.traverse(o => { if (o.isMesh && o.geometry.type === 'RingGeometry') n++ }); return n })()"
def click_plot(pg, x, z):
    px, py = pg.evaluate(PROJECT, [x, z]); pg.mouse.click(px, py)
def force_mature(plot_id, hours_ago, harvested):
    t = datetime.now(timezone.utc) - timedelta(hours=hours_ago)
    sql(f"UPDATE plots SET state='mature', plant_type='daisy', grow_s=600, life_s=86400, harvested={harvested}, matured_at='{iso(t)}', wilts_at='{iso(t+timedelta(hours=24))}', collected_to='{iso(t)}' WHERE id={plot_id}")

sql("INSERT INTO settings (player_id,key,value) VALUES (1,'tutorial_done','1') ON CONFLICT(player_id,key) DO UPDATE SET value='1'")
with sync_playwright() as p:
    b = p.chromium.launch(args=["--use-gl=swiftshader", "--enable-unsafe-swiftshader"])
    pg = b.new_page(viewport={"width": 1100, "height": 760}); errs = []
    pg.on("console", lambda m: errs.append(m.text[:200]) if m.type == "error" and "409" not in m.text else None); pg.on("pageerror", lambda e: errs.append(str(e)[:200]))
    pg.goto(BASE); pg.wait_for_selector(".packet")

    # 1) la tienda sin 💧
    pg.get_by_test_id("shop-button").click(); pg.wait_for_selector("[data-testid=shop-panel]")
    txt = pg.inner_text("[data-testid=shop-panel]")
    check("ofrece la parcela 2 y dice cuántas tienes", "Parcela 2" in txt and "1 de 16" in txt, txt.replace("\n", " | "))
    check("ofrece ampliar el Silo de 12 h a 24 h", "12 h → 24 h" in txt)
    check("sin 💧 los botones están desactivados y dice cuántas faltan", pg.get_by_test_id("buy-plot").is_disabled() and "Te faltan 3" in txt and "Te faltan 25" in txt)
    check("el botón de la tienda no llama la atención sin 💧", "chip--attention" not in (pg.get_attribute("[data-testid=shop-button]", "class") or ""))
    pg.keyboard.press("Escape"); check("Escape cierra la tienda", pg.locator("[data-testid=shop-panel]").count() == 0)

    # 2) con 💧: comprar parcela
    focus(40); pg.reload(); pg.wait_for_selector(".packet")
    check("con 💧 suficientes el botón Mejoras se resalta", "chip--attention" in (pg.get_attribute("[data-testid=shop-button]", "class") or ""))
    pg.get_by_test_id("shop-button").click(); pg.get_by_test_id("buy-plot").click(); pg.wait_for_function("document.querySelectorAll('.toast').length > 0")
    st = state()
    check("la parcela 2 se compra por 3 💧 y aparece en el estado", len(st["plots"]) == 2 and st["player"]["focus_points"] == 37, f"{len(st['plots'])} parcelas, {st['player']['focus_points']} 💧")
    check("la siguiente cuesta 5 💧 (tabla del GDD)", st["shop"]["next_plot"]["cost"] == 5)
    check("avisa con un mensaje", "parcela" in pg.inner_text(".toast").lower(), pg.inner_text(".toast"))
    pg.keyboard.press("Escape"); pg.wait_for_timeout(600)
    check("tras comprar, el juego mira la parcela nueva (marco de selección)", pg.evaluate(RINGS) == 1)
    check("y el dock ofrece sembrar ahí", btn(pg, "Plantar").is_visible())
    pg.screenshot(path=f"{OUT}/shop_two_plots.png")

    # 3) sembrar en la parcela nueva
    pg.locator(".packet__body:not([disabled])").first.click(); btn(pg, "Plantar").click(); pg.wait_for_selector("[data-testid=timer][data-status=running]")
    check("el Pomodoro crece en la parcela 2, no en la 1", state()["pomodoro"]["plot_id"] == 2, str(state()["pomodoro"]["plot_id"]))
    check("mientras corre un Pomodoro no se muestra marco de selección", pg.evaluate(RINGS) == 0)

    # 4) la parcela 1 madura mientras tanto; elegir parcelas con el clic
    call("POST", "/api/pomodoros/active/cancel"); force_mature(1, 2, 0)
    pg.reload(); pg.wait_for_selector("[data-testid=harvest]"); pg.wait_for_timeout(1200)
    check("sin elección, el dock apunta a lo que necesita atención (planta lista)", "lista" in pg.inner_text(".dock"))
    click_plot(pg, 2, 1); pg.wait_for_selector(".packet", timeout=5000)
    check("clic en una parcela vacía: el dock pasa a sembrar", btn(pg, "Plantar").is_visible() and pg.evaluate(RINGS) == 1)
    click_plot(pg, 1, 1); pg.wait_for_selector(".toast--reward", timeout=5000)
    check("clic en la planta lista: la cosecha (+1 💧)", "+1" in pg.inner_text(".toast--reward"), pg.inner_text(".toast--reward"))
    pg.wait_for_selector("[data-testid=clear]")
    check("y el dock ofrece retirarla (parcela seleccionada)", "produciendo" in pg.inner_text(".dock__msg"), pg.inner_text(".dock__msg"))
    btn(pg, "Sembrar en otra parcela").click(); pg.wait_for_selector(".packet")
    check("'Sembrar en otra parcela' salta a la libre", btn(pg, "Plantar").is_visible())
    pg.screenshot(path=f"{OUT}/shop_select.png")

    # 5) ampliar el Silo
    pg.wait_for_selector("[data-testid=silo]"); cap0 = int(state()["silo"]["capacity_hours"])
    pg.get_by_test_id("shop-button").click(); pg.get_by_test_id("upgrade-silo").click(); pg.wait_for_function("document.querySelector('.toast') && document.querySelector('.toast').innerText.includes('Silo')")
    st = state()
    check("el Silo pasa de 12 h a 24 h y cobra 25 💧", cap0 == 12 and st["silo"]["capacity_hours"] == 24 and st["player"]["focus_points"] == 37 + 1 - 25, f"{st['silo']['capacity_hours']} h, {st['player']['focus_points']} 💧")
    pg.keyboard.press("Escape")

    # 6) las 16 parcelas, con precios de la tabla
    focus(2000)
    for _ in range(14): call("POST", "/api/plots")
    pg.reload(); pg.wait_for_selector(".packet"); pg.wait_for_timeout(1500)
    st = state(); cells = {(q["x"], q["y"]) for q in st["plots"]}
    check("16 parcelas distintas en la cuadrícula 4×4", len(st["plots"]) == 16 and len(cells) == 16 and all(0 <= x <= 3 and 0 <= y <= 3 for x, y in cells), f"{len(cells)} celdas")
    pg.get_by_test_id("shop-button").click(); check("sin parcelas que comprar lo dice", "Tienes las 16 parcelas" in pg.inner_text("[data-testid=shop-panel]"))
    pg.keyboard.press("Escape"); pg.screenshot(path=f"{OUT}/shop_16_plots.png")
    r = call("POST", "/api/plots"); check("la 17ª se rechaza", r.get("_status") == 409 and r.get("error") == "maxed_out", str(r))

    # 7) móvil
    pg.set_viewport_size({"width": 360, "height": 640}); pg.reload(); pg.wait_for_selector(".packet"); pg.wait_for_timeout(800)
    tb = pg.locator(".topbar").bounding_box()
    check("360 px: la barra superior cabe", tb["x"] + tb["width"] <= 360.5, str(tb))
    pg.get_by_test_id("shop-button").click(); pg.wait_for_selector("[data-testid=shop-panel]"); sb = pg.locator("[data-testid=shop-panel]").bounding_box()
    check("360 px: la tienda cabe en la pantalla", sb["x"] >= 0 and sb["x"] + sb["width"] <= 360.5, str(sb))
    check("360 px: sin scroll horizontal", not pg.evaluate("document.documentElement.scrollWidth > document.documentElement.clientWidth"))
    pg.screenshot(path=f"{OUT}/shop_mobile.png")
    check("sin errores de consola", not errs, "; ".join(errs)[:240])
    b.close()
sql("DELETE FROM plots WHERE id > 1; UPDATE plots SET state='empty',plant_type=NULL,harvested=0,matured_at=NULL,wilts_at=NULL,collected_to=NULL,grow_s=NULL,life_s=NULL; UPDATE players SET silo_level=0,silo_micro=0,silo_peak_micro_h=0,coins_milli=0,focus_points=0,lifetime_focus=0; DELETE FROM settings")
print("\nFALLOS:" if fails else "\nTODO OK", fails or "")
sys.exit(1 if fails else 0)
