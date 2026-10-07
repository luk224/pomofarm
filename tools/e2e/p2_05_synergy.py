"""E2E P2-05: sinergias (anillo de compatibilidad, adyacencia, Huerto completo) en la interfaz.

Se ejecuta con tools/e2e/run_isolated.sh. Fuerza plantas maduras en la BD temporal.
"""
import json, os, re, subprocess, sys, urllib.request
from datetime import datetime, timedelta, timezone
from playwright.sync_api import sync_playwright

if "POMOFARM_URL" not in os.environ:
    sys.exit("Estos tests resetean la BD y no deben tocar tu partida. Ejecútalos con: tools/e2e/run_isolated.sh tools/e2e/p2_05_synergy.py")
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
CROPS = {"daisy": (600, 24), "tomato": (1500, 36), "sunflower": (2100, 54), "apple": (2700, 72), "oak": (3600, 108)}
def force(plot_id, crop, matured_h_ago=1.0, life_h=None, harvested=1):
    grow, life = CROPS[crop]; life = life_h or life
    t = datetime.now(timezone.utc) - timedelta(hours=matured_h_ago)
    sql(f"UPDATE plots SET state='mature', plant_type='{crop}', grow_s={grow}, life_s={life*3600}, harvested={harvested}, matured_at='{iso(t)}', "
        f"wilts_at='{iso(t + timedelta(hours=life))}', collected_to='{iso(t)}' WHERE id={plot_id}")
def reset_farm():
    sql("UPDATE plots SET state='empty',plant_type=NULL,harvested=0,matured_at=NULL,wilts_at=NULL,collected_to=NULL,grow_s=NULL,life_s=NULL;"
        "UPDATE players SET silo_micro=0,silo_peak_micro_h=0,coins_milli=0")
BADGES = "[...(() => { const out = []; if (!window.__three) return out; window.__three.scene.traverse(o => { if (o.name === 'bonus-badge') out.push([o.userData.text, o.userData.gold]) }); return out })()].sort()"
PROJECT = """([x, z]) => { const t = window.__three; const v = new t.camera.position.constructor(x, 0.45, z).project(t.camera);
  const r = t.gl.domElement.getBoundingClientRect(); return [r.left + (v.x + 1) / 2 * r.width, r.top + (1 - v.y) / 2 * r.height] }"""
def badges(pg): return pg.evaluate(BADGES)

sql("INSERT INTO settings (player_id,key,value) VALUES (1,'tutorial_done','1') ON CONFLICT(player_id,key) DO UPDATE SET value='1';"
    "UPDATE players SET focus_points=100, silo_level=4")
for _ in range(3): call("POST", "/api/plots")
check("la granja tiene las 4 parcelas del 2×2 central", [(p["x"], p["y"]) for p in call("GET", "/api/state")["plots"]] == [(1, 1), (2, 1), (1, 2), (2, 2)])

with sync_playwright() as p:
    b = p.chromium.launch(args=["--use-gl=swiftshader", "--enable-unsafe-swiftshader"])
    pg = b.new_page(viewport={"width": 1100, "height": 760}); errs = []
    pg.on("console", lambda m: errs.append(m.text[:200]) if m.type == "error" and "409" not in m.text else None); pg.on("pageerror", lambda e: errs.append(str(e)[:200]))
    pg.goto(BASE); pg.wait_for_selector(".packet")

    # combinaciones en los sobres (anillo del GDD §4.6)
    pairs = pg.evaluate("[...document.querySelectorAll('.packet__pair')].map(e => e.title)")
    check("cada sobre dice con qué se combina (el anillo da la vuelta)", pairs == ["Combina con Roble mágico y Tomates", "Combina con Margarita y Girasol", "Combina con Tomates y Manzano", "Combina con Girasol y Roble mágico", "Combina con Manzano y Margarita"], str(pairs))
    check("y lo anuncia a lectores de pantalla", "combina con Roble mágico y Tomates" in pg.get_by_role("radio", name=re.compile("^Margarita")).get_attribute("aria-label"))
    check("sin plantas maduras no hay insignias", badges(pg) == [])

    # 1) dos vecinas compatibles
    force(1, "daisy"); force(2, "tomato"); pg.reload(); pg.wait_for_function("window.__three && window.__three.scene.getObjectByName('bonus-badge')"); pg.wait_for_timeout(800)
    check("margarita y tomate juntas: dos insignias +10% verdes", badges(pg) == [["+10%", False], ["+10%", False]], str(badges(pg)))
    pg.screenshot(path=f"{OUT}/syn_pair.png")

    # 2) Huerto completo: 2×2 con cuatro cultivos distintos
    force(1, "daisy"); force(2, "tomato"); force(3, "oak"); force(4, "sunflower"); pg.reload(); pg.wait_for_function("window.__three && window.__three.scene.getObjectByName('bonus-badge')"); pg.wait_for_timeout(1000)
    got = badges(pg)
    check("Huerto completo: +35%, +35%, +25%, +25% y todas doradas", got == [["+25%", True], ["+25%", True], ["+35%", True], ["+35%", True]], str(got))
    st = call("GET", "/api/state"); mult = {p["id"]: p["bonus"]["multiplier"] for p in st["plots"]}
    check("el servidor da los mismos multiplicadores (1,35 / 1,35 / 1,25 / 1,25)", [round(mult[i], 2) for i in (1, 2, 3, 4)] == [1.35, 1.35, 1.25, 1.25], str(mult))
    px, py = pg.evaluate(PROJECT, [1, 1]); pg.mouse.click(px, py + 8)
    pg.wait_for_selector("[data-testid=clear]")
    check("el dock de la parcela dice su multiplicador (×1,35)", "×1,35" in pg.inner_text(".dock__msg"), pg.inner_text(".dock__msg"))
    rate = st["silo"]["rate_milli_per_h"]
    base = lambda c: (CROPS[c][0] / 60) and 0
    from math import pow
    def y(d): return 2.0 * pow(d / 10, 0.8) * d
    expected = sum(y(CROPS[c][0] / 60) / CROPS[c][1] * m for c, m in (("daisy", 1.35), ("tomato", 1.35), ("oak", 1.25), ("sunflower", 1.25))) * 1000
    check("el ritmo del Silo suma los cuatro con sus bonos", abs(rate - expected) < expected * 0.01, f"{rate} vs ≈{expected:.0f} milésimas/h")
    pg.screenshot(path=f"{OUT}/syn_garden.png")

    # 3) un vecino distinto del mismo cultivo, o incompatible, no da nada
    force(1, "daisy"); force(2, "daisy"); force(3, "apple"); force(4, "daisy"); pg.reload(); pg.wait_for_selector(".packet,[data-testid=clear],[data-testid=harvest]"); pg.wait_for_timeout(800)
    check("margaritas juntas (o con un vecino incompatible) no dan bono", badges(pg) == [], str(badges(pg)))

    # 4) un vecino que aún crece no da bono (decisión 11): plantar tomate junto a una margarita madura
    reset_farm(); force(1, "daisy"); sql("INSERT OR IGNORE INTO unlocks (player_id,kind,key,at) VALUES (1,'seed','tomato','t')")
    r = call("POST", "/api/pomodoros", {"plot_id": 2, "plant_type": "tomato"}); pg.reload(); pg.wait_for_selector("[data-testid=timer]"); pg.wait_for_timeout(1000)
    check("mientras el tomate crece, la margarita no tiene insignia", badges(pg) == [], str(badges(pg)))
    call("POST", "/api/pomodoros/active/cancel")

    # 5) el bono se apaga cuando el vecino se marchita
    reset_farm(); force(1, "daisy"); force(2, "tomato", life_h=36)
    t = datetime.now(timezone.utc) - timedelta(hours=36) + timedelta(seconds=3)   # el tomate muere en 3 s
    sql(f"UPDATE plots SET matured_at='{iso(t)}', wilts_at='{iso(t + timedelta(hours=36))}', collected_to='{iso(t)}' WHERE id=2")
    pg.reload(); pg.wait_for_function("window.__three && window.__three.scene.getObjectByName('bonus-badge')"); pg.wait_for_timeout(500)
    check("recién cargado, ambos tienen insignia (+10%)", len(badges(pg)) == 2, str(badges(pg)))
    pg.wait_for_timeout(4000); pg.evaluate("window.dispatchEvent(new Event('focus'))"); pg.wait_for_function(f"({BADGES}).length === 0", timeout=8000)
    check("al marchitarse el tomate desaparecen las dos insignias", badges(pg) == [])
    check("la margarita queda sin bono en el servidor", call("GET", "/api/state")["plots"][0]["bonus"]["multiplier"] == 1)

    # 6) retirar al vecino
    reset_farm(); force(1, "daisy"); force(2, "tomato"); pg.reload(); pg.wait_for_function(f"({BADGES}).length === 2")
    call("POST", "/api/plots/2/clear", {"confirm": True}); pg.evaluate("window.dispatchEvent(new Event('focus'))"); pg.wait_for_function(f"({BADGES}).length === 0", timeout=8000)
    check("retirar al vecino apaga la insignia de la otra", True)

    # 7) móvil: los sobres más altos siguen cabiendo
    pg.set_viewport_size({"width": 360, "height": 640}); reset_farm(); pg.reload(); pg.wait_for_selector(".packet"); pg.wait_for_timeout(600)
    db = pg.locator(".dock").bounding_box(); check("360 px: el dock con los sobres altos cabe y no se sale", db["x"] >= 0 and db["x"] + db["width"] <= 360.5 and db["y"] >= 0, str(db))
    check("360 px: sin scroll horizontal", not pg.evaluate("document.documentElement.scrollWidth > document.documentElement.clientWidth"))
    pg.screenshot(path=f"{OUT}/syn_mobile.png")
    check("sin errores de consola", not errs, "; ".join(errs)[:240])
    b.close()
sql("DELETE FROM plots WHERE id > 1; UPDATE plots SET state='empty',plant_type=NULL,harvested=0,matured_at=NULL,wilts_at=NULL,collected_to=NULL,grow_s=NULL,life_s=NULL; UPDATE players SET silo_level=0,silo_micro=0,silo_peak_micro_h=0,coins_milli=0,focus_points=0; DELETE FROM settings; DELETE FROM unlocks WHERE key<>'daisy'; DELETE FROM pomodoro_events; DELETE FROM pomodoros")
print("\nFALLOS:" if fails else "\nTODO OK", fails or "")
sys.exit(1 if fails else 0)
