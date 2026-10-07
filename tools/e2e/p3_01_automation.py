"""E2E P3-01: Abejas y Perro Pastor (desbloqueo con 💧, compra con 🪙, colocación con vista previa, mover gratis, móvil).

Se ejecuta con tools/e2e/run_isolated.sh.
"""
import json, os, subprocess, sys, urllib.request
from datetime import datetime, timedelta, timezone
from playwright.sync_api import sync_playwright

if "POMOFARM_URL" not in os.environ:
    sys.exit("Estos tests resetean la BD y no deben tocar tu partida. Ejecútalos con: tools/e2e/run_isolated.sh tools/e2e/p3_01_automation.py")
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
def state(): return call("GET", "/api/state")
PROJECT = """([x, y, z]) => { const t = window.__three; const v = new t.camera.position.constructor(x, y, z).project(t.camera);
  const r = t.gl.domElement.getBoundingClientRect(); return [r.left + (v.x + 1) / 2 * r.width, r.top + (1 - v.y) / 2 * r.height] }"""
def click_at(pg, x, y, z):
    px, py = pg.evaluate(PROJECT, [x, y, z]); pg.mouse.click(px, py)
COVER = "(() => { let n = 0; window.__three.scene.traverse(o => { if (o.isMesh && o.geometry.type === 'PlaneGeometry') n++ }); return n })()"
HAS = "(name) => !!window.__three.scene.getObjectByName(name)"
def hive_world(pg, name):
    return pg.evaluate("(n) => { const o = window.__three.scene.getObjectByName(n); return o ? [o.position.x, o.position.y, o.position.z] : null }", name)

# Estado limpio aunque otros scripts hayan usado la misma BD (Perro que recoge solo, parcelas, etc.)
sql("DELETE FROM structures; DELETE FROM unlocks WHERE kind='animal'; DELETE FROM plots WHERE NOT (x=1 AND y=1); "
    "UPDATE plots SET state='empty', plant_type=NULL, planted_at=NULL, grow_s=NULL, matured_at=NULL, harvested=0, life_s=NULL, wilts_at=NULL, collected_to=NULL; "
    "UPDATE players SET silo_micro=0, silo_peak_micro_h=0, silo_level=0, season=1")
# a 3×3 farm, every plant mature and producing
t = datetime.now(timezone.utc) - timedelta(hours=2)
sql("INSERT INTO settings (player_id,key,value) VALUES (1,'tutorial_done','1') ON CONFLICT(player_id,key) DO UPDATE SET value='1'")
for x, y in [(0,0),(1,0),(2,0),(0,1),(1,1),(2,1),(0,2),(1,2),(2,2)]:
    sql(f"INSERT OR IGNORE INTO plots (player_id,x,y) VALUES (1,{x},{y})")
sql(f"UPDATE plots SET state='mature', plant_type='daisy', grow_s=600, life_s=86400, harvested=1, matured_at='{iso(t)}', wilts_at='{iso(t+timedelta(hours=24))}', collected_to='{iso(t)}'")
sql("UPDATE players SET focus_points=200, coins_milli=6000000")

with sync_playwright() as p:
    b = p.chromium.launch(args=["--use-gl=swiftshader", "--enable-unsafe-swiftshader"])
    pg = b.new_page(viewport={"width": 1100, "height": 760}); errs = []
    pg.on("console", lambda m: errs.append(m.text[:200]) if m.type == "error" and "409" not in m.text else None); pg.on("pageerror", lambda e: errs.append(str(e)[:200]))
    pg.goto(BASE); pg.wait_for_selector("[data-testid=silo]"); pg.wait_for_timeout(800)

    # 1) desbloquear las Abejas con 💧
    pg.get_by_test_id("shop-button").click(); pg.wait_for_selector("[data-testid=shop-panel]")
    check("la tienda ofrece desbloquear las Abejas por 30 💧", "30" in pg.inner_text("[data-testid=unlock-bees]") and pg.get_by_test_id("buy-hive").count() == 0)
    check("y el Perro por 120 💧", "120" in pg.inner_text("[data-testid=unlock-dog]"))
    pg.get_by_test_id("unlock-bees").click(); pg.wait_for_selector("[data-testid=buy-hive]")
    st = state()
    check("desbloquear cobra 30 💧 y no toca las monedas", st["player"]["focus_points"] == 170 and st["player"]["coins_milli"] >= 6000000 and st["automation"]["bees"]["unlocked"])
    txt = pg.locator(".offer", has=pg.get_by_test_id("buy-hive")).inner_text()
    check("ofrece la colmena 1 por 4.000 🪙", "Colmena 1" in txt and "4.000" in txt, txt.replace("\n", " | "))

    # 2) colocar: vista previa, cancelar no cobra
    coins0 = st["player"]["coins_milli"]
    pg.get_by_test_id("buy-hive").click(); pg.wait_for_selector("[data-testid=placing-bar]")
    check("al comprar se cierra la tienda y aparece la barra de colocación", pg.locator("[data-testid=shop-panel]").count() == 0)
    click_at(pg, 1, 0.3, 1); pg.wait_for_timeout(300)
    txt = pg.inner_text("[data-testid=placing-text]")
    check("en el centro de la granja cubre 9 parcelas, todas produciendo", "Cubre 9 parcelas" in txt and "9 produciendo" in txt, txt)
    check("se dibujan 9 recuadros de cobertura", pg.evaluate(COVER) == 9, str(pg.evaluate(COVER)))
    pg.screenshot(path=f"{OUT}/hive_preview.png")
    click_at(pg, 0, 0.3, 0); pg.wait_for_timeout(300)
    check("en una esquina cubre solo 4", "Cubre 4 parcelas" in pg.inner_text("[data-testid=placing-text]") and pg.evaluate(COVER) == 4)
    pg.keyboard.press("Escape"); pg.wait_for_timeout(200)
    check("Escape cancela: sin barra, sin recuadros y sin cobro", pg.locator("[data-testid=placing-bar]").count() == 0 and pg.evaluate(COVER) == 0 and state()["player"]["coins_milli"] >= coins0 and not state()["automation"]["bees"]["hives"])

    # 3) comprar de verdad
    pg.get_by_test_id("shop-button").click(); pg.get_by_test_id("buy-hive").click(); pg.wait_for_selector("[data-testid=placing-bar]")
    click_at(pg, 1, 0.3, 1); pg.wait_for_timeout(200)
    c0 = state()["player"]["coins_milli"]
    pg.get_by_test_id("placing-confirm").click(); pg.wait_for_function("document.querySelector('[data-testid=placing-bar]') === null")
    st = state(); hives = st["automation"]["bees"]["hives"]
    check("la colmena se compra junto a la parcela del centro", len(hives) == 1 and (hives[0]["x"], hives[0]["y"]) == (1, 1), str(hives))
    check("cuesta 4.000 🪙", c0 - st["player"]["coins_milli"] >= 4000000 - 50000 and st["automation"]["bees"]["next_cost"] == 6000, f"{(c0 - st['player']['coins_milli'])/1000}")
    check("las 9 plantas reciben las abejas (+25% al menos)", sum(1 for q in st["plots"] if q["bonus"] and q["bonus"]["bees"]) == 9)
    check("la colmena existe en la escena 3D y no tapa la planta", pg.evaluate(HAS, "hive-%d" % hives[0]["id"]) and hive_world(pg, "hive-%d" % hives[0]["id"])[0] < 1)
    pg.wait_for_timeout(600); pg.screenshot(path=f"{OUT}/hive_placed.png")

    # 4) mover gratis tocando la colmena
    hid = hives[0]["id"]; wx, wy, wz = hive_world(pg, f"hive-{hid}")
    c1 = state()["player"]["coins_milli"]
    click_at(pg, wx, wy + 0.25, wz); pg.wait_for_selector("[data-testid=placing-bar]", timeout=5000)
    check("tocar la colmena ofrece moverla gratis", "gratis" in pg.inner_text("[data-testid=placing-confirm]"))
    click_at(pg, 2, 0.3, 2); pg.wait_for_timeout(200)
    pg.get_by_test_id("placing-confirm").click(); pg.wait_for_function("document.querySelector('[data-testid=placing-bar]') === null")
    st = state(); h = st["automation"]["bees"]["hives"][0]
    check("la colmena se mueve a la esquina sin cobrar", (h["x"], h["y"]) == (2, 2) and st["player"]["coins_milli"] >= c1, str(h))
    check("ahora solo 4 plantas tienen abejas", sum(1 for q in st["plots"] if q["bonus"] and q["bonus"]["bees"]) == 4)

    # 5) Perro
    pg.get_by_test_id("shop-button").click()
    check("sin 120 💧 no se puede desbloquear... con 170 sí", pg.get_by_test_id("unlock-dog").is_enabled())
    pg.get_by_test_id("unlock-dog").click(); pg.wait_for_selector("[data-testid=buy-dog]")
    check("tras desbloquear, el Perro cuesta 30.000 🪙 y faltan monedas", "30.000" in pg.inner_text("[data-testid=buy-dog]") and pg.get_by_test_id("buy-dog").is_disabled() and "Te faltan" in pg.locator(".offer", has=pg.get_by_test_id("buy-dog")).inner_text())
    pg.screenshot(path=f"{OUT}/shop_animals.png")
    sql("UPDATE players SET coins_milli=40000000"); pg.reload(); pg.wait_for_selector("[data-testid=silo]")
    pg.get_by_test_id("shop-button").click(); pg.get_by_test_id("buy-dog").click(); pg.wait_for_function("document.querySelector('.toast') && document.querySelector('.toast').innerText.includes('Perro')")
    st = state()
    check("el Perro se compra por 30.000 🪙 y vigila el Silo", st["automation"]["dog"]["owned"] and st["player"]["coins_milli"] < 10000000 + 5000000)
    pg.wait_for_timeout(500)
    check("el Perro aparece en la escena", pg.evaluate(HAS, "dog-3d"))
    check("la tienda ya no lo ofrece", pg.get_by_test_id("buy-dog").count() == 0 and "vigila" in pg.inner_text("[data-testid=shop-panel]"))
    pg.keyboard.press("Escape")
    pg.screenshot(path=f"{OUT}/dog.png")

    # 6) móvil: la barra de colocación cabe
    m = b.new_page(viewport={"width": 390, "height": 760}); m.goto(BASE); m.wait_for_selector("[data-testid=silo]")
    m.get_by_test_id("shop-button").click(); m.get_by_test_id("shop-panel").wait_for()
    box = m.get_by_test_id("shop-panel").bounding_box()
    check("móvil: el panel de la tienda cabe en 390 px", box and box["x"] >= 0 and box["x"] + box["width"] <= 390, str(box))
    m.keyboard.press("Escape")
    check("móvil: sin desbordamiento horizontal", m.evaluate("document.documentElement.scrollWidth <= window.innerWidth"))
    m.close()
    check("sin errores de consola", not errs, "; ".join(errs[:3]))
    b.close()
print(f"\n{len(fails)} fallos" if fails else "\nTodo OK"); sys.exit(1 if fails else 0)
