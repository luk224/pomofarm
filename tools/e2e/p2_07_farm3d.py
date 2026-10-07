"""E2E P2-07: granja 3D completa (presupuesto de dibujo con 16 plantas, Silo 3D, parcelas instanciadas, móvil).

Se ejecuta con tools/e2e/run_isolated.sh. Fuerza plantas maduras en la BD temporal.
"""
import json, os, re, subprocess, sys, urllib.request
from datetime import datetime, timedelta, timezone
from playwright.sync_api import sync_playwright

if "POMOFARM_URL" not in os.environ:
    sys.exit("Estos tests resetean la BD y no deben tocar tu partida. Ejecútalos con: tools/e2e/run_isolated.sh tools/e2e/p2_07_farm3d.py")
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
SPEC = {"daisy": (600, 24), "tomato": (1500, 36), "sunflower": (2100, 54), "apple": (2700, 72), "oak": (3600, 108)}
CROPS = list(SPEC)
def force(pid, crop, hours_ago=2.0, harvested=1):
    g, l = SPEC[crop]; t = datetime.now(timezone.utc) - timedelta(hours=hours_ago)
    sql(f"UPDATE plots SET state='mature', plant_type='{crop}', grow_s={g}, life_s={l*3600}, harvested={harvested}, matured_at='{iso(t)}', wilts_at='{iso(t+timedelta(hours=l))}', collected_to='{iso(t)}' WHERE id={pid}")
def clear_farm():
    sql("UPDATE plots SET state='empty',plant_type=NULL,harvested=0,matured_at=NULL,wilts_at=NULL,collected_to=NULL,grow_s=NULL,life_s=NULL; UPDATE players SET silo_micro=0,silo_peak_micro_h=0,coins_milli=0")
STATS = "(() => { const i = window.__three.gl.info; let inst = 0; window.__three.scene.traverse(o => { if (o.isInstancedMesh) inst++ }); return {calls: i.render.calls, tris: i.render.triangles, geos: i.memory.geometries, inst} })()"
SCREEN = """([x, y, z]) => { const t = window.__three; const v = new t.camera.position.constructor(x, y, z).project(t.camera);
  const r = t.gl.domElement.getBoundingClientRect(); return [r.left + (v.x + 1) / 2 * r.width, r.top + (1 - v.y) / 2 * r.height] }"""
FILL = "(() => { const o = window.__three.scene.getObjectByName('silo-fill'); return o ? o.scale.y : null })()"
BADGES = "(() => { const out = []; window.__three.scene.traverse(o => { if (o.name === 'bonus-badge') out.push(o.userData.text) }); return out })()"
SILO = [-1.15, 0.5, 4.15]
def wait_scene(pg): pg.wait_for_function("window.__three && window.__three.scene.getObjectByName('silo-3d')"); pg.wait_for_timeout(1200)
def num(t): return float(re.search(r"[\d]+(?:,\d+)?", t.replace(".", "")).group().replace(",", "."))

sql("INSERT INTO settings (player_id,key,value) VALUES (1,'tutorial_done','1') ON CONFLICT(player_id,key) DO UPDATE SET value='1'; UPDATE players SET focus_points=3000")
for _ in range(15): call("POST", "/api/plots")
with sync_playwright() as p:
    b = p.chromium.launch(args=["--use-gl=swiftshader", "--enable-unsafe-swiftshader"])
    pg = b.new_page(viewport={"width": 1100, "height": 760}); errs = []
    pg.on("console", lambda m: errs.append(m.text[:200]) if m.type == "error" and "409" not in m.text else None); pg.on("pageerror", lambda e: errs.append(str(e)[:200]))

    # 1) presupuesto: 16 plantas maduras con bonos y un Silo
    sql("UPDATE players SET silo_level=4")
    for i in range(1, 17): force(i, CROPS[i % 5])
    pg.goto(BASE); wait_scene(pg); st = pg.evaluate(STATS)
    print("granja de 16 plantas:", st)
    check("16 plantas maduras: ≤ 90 llamadas de dibujo (antes: 409)", st["calls"] <= 90, str(st["calls"]))
    check("pocas geometrías distintas (se comparten): ≤ 30", st["geos"] <= 30, str(st["geos"]))
    check("suelo y parcelas van instanciados (≥ 4 mallas instanciadas)", st["inst"] >= 4, str(st["inst"]))
    check("la granja entera sigue por debajo de 14.000 triángulos", st["tris"] <= 14000, str(st["tris"]))
    pg.screenshot(path=f"{OUT}/farm3d_full.png")

    # 2) el Silo crece con las mejoras
    h = lambda: pg.evaluate("(() => { const g = window.__three.scene.getObjectByName('silo-3d').children[0].geometry; g.computeBoundingBox(); return g.boundingBox.max.y - g.boundingBox.min.y })()")
    h4 = h(); sql("UPDATE players SET silo_level=0"); pg.reload(); wait_scene(pg); h0 = h()
    check("el Silo de nivel 4 es más alto que el de nivel 0", h4 > h0 + 0.4, f"{h0:.2f} -> {h4:.2f}")

    # 3) indicador de llenado
    sql("UPDATE players SET silo_level=0"); clear_farm(); force(1, "daisy", hours_ago=6.0)    # 6 h × 0,83 = 5 🪙 de 10 de capacidad
    pg.reload(); wait_scene(pg); f = pg.evaluate(FILL); api = call("GET", "/api/state")["silo"]; want = api["content_milli"] / api["capacity_milli"]
    check("el indicador del Silo marca lo que hay (≈50%)", abs(f - want) < 0.05 and 0.4 < f < 0.6, f"escena {f:.2f}, servidor {want:.2f}")
    check("hay chispa de 'algo que recoger' y no hay aviso de lleno", pg.evaluate(BADGES).count("¡Lleno!") == 0)

    # 4) clic en el Silo recoge
    sx, sy = pg.evaluate(SCREEN, SILO); pg.mouse.click(sx, sy)
    pg.wait_for_selector(".toast--coin", timeout=6000)
    check("clic en el Silo 3D recoge las monedas", "+4," in pg.inner_text(".toast--coin") or "+5" in pg.inner_text(".toast--coin"), pg.inner_text(".toast--coin"))
    pg.wait_for_timeout(700); check("tras recogerlo, el indicador baja a ≈0", pg.evaluate(FILL) < 0.08, f"{pg.evaluate(FILL):.3f}")
    check("el saldo de 🪙 sube", num(pg.inner_text("[data-testid=coins]")) > 4, pg.inner_text("[data-testid=coins]"))
    pg.mouse.click(sx, sy); pg.wait_for_selector(".toast:has-text('aún no tiene monedas')", timeout=6000)
    check("clic en un Silo vacío lo dice sin dar error", pg.locator(".toast--error").count() == 0)

    # 5) Silo lleno
    clear_farm(); force(1, "daisy", hours_ago=40.0); pg.reload(); wait_scene(pg)
    check("Silo lleno: avisa con '¡Lleno!' sobre el edificio", "¡Lleno!" in pg.evaluate(BADGES), str(pg.evaluate(BADGES)))
    check("y el indicador está casi al máximo", pg.evaluate(FILL) > 0.95, f"{pg.evaluate(FILL):.2f}")
    pg.screenshot(path=f"{OUT}/farm3d_silo_full.png")
    pg.keyboard.press("c"); pg.wait_for_selector(".toast--coin"); pg.wait_for_timeout(600)
    check("recoger (tecla C) quita el aviso de lleno", "¡Lleno!" not in pg.evaluate(BADGES))

    # 6) clic en la tierra (malla instanciada) selecciona la parcela y cosecha la lista
    clear_farm(); force(2, "tomato", hours_ago=0.2, harvested=0)    # la parcela 2 está lista; la 1 libre
    pg.reload(); wait_scene(pg)
    ex, ey = pg.evaluate(SCREEN, [1.0, 0.06, 1.0]); pg.mouse.click(ex, ey)   # tierra de la parcela 1 (x=1,z=1), sin planta
    pg.wait_for_selector(".packet", timeout=5000)
    check("clic en la tierra de una parcela vacía: el dock pasa a sembrar", pg.get_by_role("button", name="Plantar", exact=True).is_visible())
    px, py = pg.evaluate(SCREEN, [2.0, 0.06, 1.0]); pg.mouse.click(px + 30, py + 8)   # un lado de la parcela 2 (con planta lista)
    pg.wait_for_selector(".toast--reward", timeout=6000)
    check("clic en la tierra de una planta lista la cosecha", "+5" in pg.inner_text(".toast--reward"), pg.inner_text(".toast--reward"))

    # 7) móvil: todo el campo y el Silo caben en pantalla
    sql("UPDATE players SET silo_level=2"); clear_farm()
    for i in range(1, 17): force(i, CROPS[i % 5], hours_ago=0.5)
    pg.set_viewport_size({"width": 360, "height": 640}); pg.reload(); wait_scene(pg)
    pts = {"silo": SILO, "esquina trasera (0,0)": [0, 0, 0], "esquina derecha (3,0)": [3, 0, 0], "esquina izquierda (0,3)": [0, 0, 3], "esquina delantera (3,3)": [3, 0, 3]}
    inside = {k: pg.evaluate(SCREEN, v) for k, v in pts.items()}
    ok = all(5 <= x <= 355 and 0 <= y <= 640 for x, y in inside.values())
    check("360×640: el campo 4×4 y el Silo caben en pantalla al abrir", ok, ", ".join(f"{k} ({x:.0f},{y:.0f})" for k, (x, y) in inside.items()))
    pg.screenshot(path=f"{OUT}/farm3d_mobile.png")
    check("sin errores de consola", not errs, "; ".join(errs)[:240])
    b.close()
sql("DELETE FROM plots WHERE id > 1; UPDATE plots SET state='empty',plant_type=NULL,harvested=0,matured_at=NULL,wilts_at=NULL,collected_to=NULL,grow_s=NULL,life_s=NULL; UPDATE players SET silo_level=0,silo_micro=0,silo_peak_micro_h=0,coins_milli=0,focus_points=0; DELETE FROM settings")
print("\nFALLOS:" if fails else "\nTODO OK", fails or "")
sys.exit(1 if fails else 0)
