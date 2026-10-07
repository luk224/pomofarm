"""E2E P3-02: decoración (caminos, vallas, farolillos en el terreno de alrededor, mover/quitar, sombrero del Perro, móvil).

Se ejecuta con tools/e2e/run_isolated.sh.
"""
import json, os, subprocess, sys, urllib.request
from playwright.sync_api import sync_playwright

if "POMOFARM_URL" not in os.environ:
    sys.exit("Estos tests resetean la BD y no deben tocar tu partida. Ejecútalos con: tools/e2e/run_isolated.sh tools/e2e/p3_02_decor.py")
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
def state(): return call("GET", "/api/state")
PROJECT = """([x, y, z]) => { const t = window.__three; const v = new t.camera.position.constructor(x, y, z).project(t.camera);
  const r = t.gl.domElement.getBoundingClientRect(); return [r.left + (v.x + 1) / 2 * r.width, r.top + (1 - v.y) / 2 * r.height] }"""
def click_at(pg, x, y, z):
    px, py = pg.evaluate(PROJECT, [x, y, z]); pg.mouse.click(px, py)
HAS = "(name) => !!window.__three.scene.getObjectByName(name)"
def coins(): return state()["player"]["coins_milli"]

# Estado limpio aunque otros scripts hayan usado la misma BD (Perro que recoge solo, parcelas, etc.)
sql("DELETE FROM structures; DELETE FROM unlocks WHERE kind='animal'; DELETE FROM plots WHERE NOT (x=1 AND y=1); "
    "UPDATE plots SET state='empty', plant_type=NULL, planted_at=NULL, grow_s=NULL, matured_at=NULL, harvested=0, life_s=NULL, wilts_at=NULL, collected_to=NULL; "
    "UPDATE players SET silo_micro=0, silo_peak_micro_h=0, silo_level=0, season=1")
sql("INSERT INTO settings (player_id,key,value) VALUES (1,'tutorial_done','1') ON CONFLICT(player_id,key) DO UPDATE SET value='1'")
sql("UPDATE players SET coins_milli=500000, focus_points=500")
with sync_playwright() as p:
    b = p.chromium.launch(args=["--use-gl=swiftshader", "--enable-unsafe-swiftshader"])
    pg = b.new_page(viewport={"width": 1100, "height": 760}); errs = []
    pg.on("console", lambda m: errs.append(m.text[:200]) if m.type == "error" and "409" not in m.text else None); pg.on("pageerror", lambda e: errs.append(str(e)[:200]))
    pg.goto(BASE); pg.wait_for_selector("[data-testid=top-coins], .chip"); pg.wait_for_timeout(800)

    # 1) catálogo con precios del GDD
    pg.get_by_test_id("shop-button").click(); pg.wait_for_selector("[data-testid=shop-panel]")
    for k, price in [("path", "15"), ("fence", "25"), ("lantern", "80")]:
        check(f"la tienda ofrece {k} por {price} 🪙", price in pg.inner_text(f"[data-testid=decor-{k}]"))
    check("sin Perro no se ofrece el sombrero", pg.get_by_test_id("buy-hat").count() == 0)
    check("avisa de que no cuenta para el prestigio", "prestigio" in pg.inner_text("[data-testid=shop-panel]"))

    # 2) colocar caminos con un toque cada uno
    pg.get_by_test_id("decor-path").click(); pg.wait_for_selector("[data-testid=decor-bar]")
    check("la tienda se cierra y aparece la barra de decorar", pg.locator("[data-testid=shop-panel]").count() == 0 and "15" in pg.inner_text("[data-testid=decor-text]"))
    pg.wait_for_timeout(300)
    c0 = coins()
    click_at(pg, 4, 0.03, 1); pg.wait_for_function("document.querySelectorAll('canvas').length>0 && window.__three.scene.getObjectByName('decor-path')")
    click_at(pg, 4, 0.03, 2); pg.wait_for_timeout(500)
    st = state()
    check("dos toques colocan dos caminos y cobran 30 🪙", len(st["decor"]["items"]) == 2 and c0 - st["player"]["coins_milli"] == 30000, f"{len(st['decor']['items'])} piezas, -{(c0 - st['player']['coins_milli'])/1000}")
    check("se siguen pudiendo colocar más sin reabrir la tienda", pg.locator("[data-testid=decor-bar]").count() == 1)
    click_at(pg, 1, 0.03, 1); pg.wait_for_timeout(400)
    check("tocar una parcela en modo decorar no coloca nada ahí", len(state()["decor"]["items"]) == 2)
    click_at(pg, 4, 0.03, 1); pg.wait_for_timeout(500)
    check("tocar una celda ya ocupada no cobra de nuevo", len(state()["decor"]["items"]) == 2 and coins() == st["player"]["coins_milli"])
    pg.screenshot(path=f"{OUT}/decor_placing.png")
    pg.keyboard.press("Escape"); pg.wait_for_timeout(200)
    check("Escape termina el modo decorar", pg.locator("[data-testid=decor-bar]").count() == 0)

    # 3) vallas y farolillos
    pg.get_by_test_id("shop-button").click(); pg.get_by_test_id("decor-fence").click(); pg.wait_for_selector("[data-testid=decor-bar]"); pg.wait_for_timeout(300)
    click_at(pg, -1, 0.1, 0); pg.wait_for_timeout(500); click_at(pg, -1, 0.1, 1); pg.wait_for_timeout(500)
    pg.get_by_test_id("decor-done").click()
    pg.get_by_test_id("shop-button").click(); pg.get_by_test_id("decor-lantern").click(); pg.wait_for_selector("[data-testid=decor-bar]"); pg.wait_for_timeout(300)
    click_at(pg, 4, 0.3, 3); pg.wait_for_timeout(500); pg.get_by_test_id("decor-done").click(); pg.wait_for_timeout(300)
    kinds = sorted(i["kind"] for i in state()["decor"]["items"])
    check("hay 2 caminos, 2 vallas y 1 farolillo", kinds == ["fence", "fence", "lantern", "path", "path"], str(kinds))
    check("existen las tres mallas instanciadas en la escena", all(pg.evaluate(HAS, f"decor-{k}") for k in ("path", "fence", "lantern")))
    pg.screenshot(path=f"{OUT}/decor_placed.png")

    # 4) mover gratis tocando una pieza
    lan = [i for i in state()["decor"]["items"] if i["kind"] == "lantern"][0]; c1 = coins()
    click_at(pg, lan["x"], 0.72, lan["y"]); pg.wait_for_selector("[data-testid=decor-remove]", timeout=5000)
    check("tocar una pieza ofrece moverla (gratis) o quitarla", "gratis" in pg.inner_text("[data-testid=decor-text]"))
    click_at(pg, -2, 0.03, 2); pg.wait_for_timeout(600)
    lan2 = [i for i in state()["decor"]["items"] if i["kind"] == "lantern"][0]
    check("la pieza se mueve sin cobrar y se sale del modo", (lan2["x"], lan2["y"]) == (-2, 2) and coins() == c1 and pg.locator("[data-testid=decor-bar]").count() == 0, f"{lan2} coins {coins()} vs {c1} bar {pg.locator('[data-testid=decor-bar]').count()}")

    # 5) quitar
    click_at(pg, -2, 0.72, 2); pg.wait_for_selector("[data-testid=decor-remove]", timeout=5000); pg.get_by_test_id("decor-remove").click(); pg.wait_for_timeout(500)
    check("quitar elimina la pieza sin tocar las monedas", len(state()["decor"]["items"]) == 4 and coins() == c1)

    # 6) la decoración no cambia producción ni prestigio
    st = state()
    check("no cambia el nivel de prestigio ni las parcelas", st["player"]["season"] == 1 and len(st["plots"]) == 1)

    # 7) sombrero: necesita el Perro
    sql("INSERT OR IGNORE INTO unlocks (player_id,kind,key,at) VALUES (1,'animal','dog','t')"); sql("UPDATE players SET coins_milli=40000000")
    pg.reload(); pg.wait_for_selector("[data-testid=silo], .chip"); pg.wait_for_timeout(500)
    pg.get_by_test_id("shop-button").click(); pg.get_by_test_id("buy-dog").click(); pg.wait_for_timeout(600)
    check("tras comprar el Perro se ofrece el sombrero por 600 🪙", "600" in pg.inner_text("[data-testid=buy-hat]"))
    c2 = coins(); pg.get_by_test_id("buy-hat").click(); pg.wait_for_function("document.querySelector('.toast') && document.querySelector('.toast').innerText.includes('sombrero')")
    st = state()
    check("el sombrero cuesta 600 🪙 y deja de ofrecerse", c2 - st["player"]["coins_milli"] >= 600000 and st["decor"]["hat"]["owned"] and pg.get_by_test_id("buy-hat").count() == 0)
    pg.keyboard.press("Escape"); pg.wait_for_timeout(500)
    pg.screenshot(path=f"{OUT}/decor_hat.png")

    # 8) móvil
    m = b.new_page(viewport={"width": 390, "height": 700}); m.goto(BASE); m.wait_for_selector(".chip")
    m.get_by_test_id("shop-button").click(); m.get_by_test_id("shop-panel").wait_for()
    box = m.get_by_test_id("shop-panel").bounding_box()
    check("móvil: el panel de la tienda cabe en pantalla (con scroll)", box and box["x"] >= 0 and box["x"] + box["width"] <= 390 and box["y"] + box["height"] <= 700, str(box))
    m.get_by_test_id("decor-path").scroll_into_view_if_needed(); m.get_by_test_id("decor-path").click(); m.get_by_test_id("decor-bar").wait_for()
    bar = m.get_by_test_id("decor-bar").bounding_box()
    check("móvil: la barra de decorar cabe", bar and bar["x"] >= 0 and bar["x"] + bar["width"] <= 390, str(bar))
    check("móvil: sin desbordamiento horizontal", m.evaluate("document.documentElement.scrollWidth <= window.innerWidth"))
    m.screenshot(path=f"{OUT}/decor_mobile.png"); m.close()
    check("sin errores de consola", not errs, "; ".join(errs[:3]))
    b.close()
print(f"\n{len(fails)} fallos" if fails else "\nTodo OK"); sys.exit(1 if fails else 0)
