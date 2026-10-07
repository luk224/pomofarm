"""E2E P1-05b: plantas por etapa, presupuesto de triángulos, anillo del temporizador, madurez y movimiento reducido.

Requiere backend (:8080) y Vite (:5173). Toca la BD de desarrollo (parcela 1) y la deja vacía.
Uso: python3 tools/e2e/p1_05b_scene.py
"""
import json, os, re, subprocess, sys, urllib.request
from playwright.sync_api import sync_playwright

BASE = os.environ.get("POMOFARM_URL", "http://localhost:5173")
DB = os.environ.get("POMOFARM_DB", "backend/data/pomofarm.db")
OUT = "/tmp/pomofarm-e2e"; os.makedirs(OUT, exist_ok=True)
MAX_TRIS = 2000  # GDD §2: presupuesto por planta
fails = []
def check(name, ok, detail=""):
    print(("PASS " if ok else "FAIL ") + name + (f"  [{detail}]" if detail else ""))
    if not ok: fails.append(name)

def call(method, path, body=None):
    req = urllib.request.Request(BASE + path, method=method, data=json.dumps(body).encode() if body else None, headers={"Content-Type": "application/json"})
    try:
        with urllib.request.urlopen(req) as r: return json.load(r)
    except urllib.error.HTTPError as e: return {"error": json.load(e).get("error")}

def sql(q): subprocess.run(["sqlite3", DB, q], check=True)

TRIS_JS = """(name) => { const o = window.__three.scene.getObjectByName(name); let t = 0;
  o.traverse(m => { if (m.isMesh) { const g = m.geometry; t += (g.index ? g.index.count : g.attributes.position.count) / 3 } }); return t }"""

def new_page(b, **kw):
    ctx = b.new_context(viewport={"width": 1100, "height": 760}, **kw); page = ctx.new_page()
    errs = []; page.on("pageerror", lambda e: errs.append(str(e)[:200])); page.on("console", lambda m: errs.append(m.text[:200]) if m.type == "error" else None)
    return ctx, page, errs

with sync_playwright() as p:
    b = p.chromium.launch(args=["--use-gl=swiftshader", "--enable-unsafe-swiftshader"])

    # 1) presupuesto de triángulos por planta y etapa (galería de desarrollo)
    ctx, page, errs = new_page(b); page.goto(BASE + "/?lab=plants"); page.wait_for_function("window.__three && window.__three.scene.getObjectByName('gallery-oak-mature')"); page.wait_for_timeout(800)
    worst = {}
    for kind in ["daisy", "tomato", "sunflower", "apple", "oak"]:
        t = page.evaluate(TRIS_JS, f"gallery-{kind}-mature"); worst[kind] = int(t)
    print("triángulos por planta madura:", worst)
    check(f"cada planta < {MAX_TRIS} triángulos", all(v < MAX_TRIS for v in worst.values()), str(max(worst.values())))
    page.screenshot(path=f"{OUT}/gallery.png"); check("galería sin errores", not errs, "; ".join(errs)); ctx.close()

    # 2) granja real con un Pomodoro en curso
    st = call("GET", "/api/state")
    if st.get("pomodoro"): call("POST", "/api/pomodoros/active/cancel")
    sql("UPDATE plots SET state='empty', plant_type=NULL, harvested=0, matured_at=NULL, wilts_at=NULL WHERE id=1")
    call("POST", "/api/pomodoros", {"plot_id": 1, "plant_type": "daisy"})
    ctx, page, errs = new_page(b); page.goto(BASE); page.wait_for_selector("[data-testid=timer-ring]"); page.wait_for_timeout(1500)
    ring = page.get_attribute("[data-testid=timer-ring]", "data-left"); txt = page.inner_text("[data-testid=timer-ring]")
    check("anillo flotante visible sobre la planta activa", float(ring) > 0.97, f"restante {ring}, texto '{txt}'")
    check("el anillo muestra mm:ss", re.fullmatch(r"\d+:\d{2}", txt.strip()) is not None, txt)
    sc = page.evaluate("window.__three.scene.getObjectByName('plant-daisy').children[0].scale.x")
    check("la planta recién plantada es un brote pequeño", 0.29 <= sc <= 0.4, f"escala {sc:.2f}")
    info = page.evaluate("({calls: window.__three.gl.info.render.calls, tris: window.__three.gl.info.render.triangles})")
    print("render real:", info)
    page.screenshot(path=f"{OUT}/farm_growing.png")
    page.get_by_role("button", name="Pausar", exact=True).click(); page.wait_for_selector("[data-testid=timer][data-status=paused]"); page.wait_for_timeout(600)
    check("en pausa el anillo muestra ⏸", "⏸" in page.inner_text("[data-testid=timer-ring]"))
    call("POST", "/api/pomodoros/active/cancel"); ctx.close()
    check("granja sin errores de consola", not errs, "; ".join(errs))

    # 3) planta madura lista para cosechar (se fuerza el estado en la BD de desarrollo)
    sql("UPDATE plots SET state='mature', plant_type='sunflower', harvested=0, matured_at=strftime('%Y-%m-%dT%H:%M:%SZ','now'), wilts_at=strftime('%Y-%m-%dT%H:%M:%SZ','now','+2 days') WHERE id=1")
    ctx, page, errs = new_page(b); page.goto(BASE); page.wait_for_function("window.__three && window.__three.scene.getObjectByName('plant-sunflower')"); page.wait_for_timeout(1500)
    sc = page.evaluate("window.__three.scene.getObjectByName('plant-sunflower').children[0].scale.x")
    check("planta madura a tamaño completo", sc > 0.97, f"escala {sc:.2f}")
    check("sin anillo cuando no hay Pomodoro", page.locator("[data-testid=timer-ring]").count() == 0)
    check("destello de 'lista para cosechar'", page.evaluate("window.__three.scene.getObjectByName('plant-sunflower').children.length") == 2)
    page.screenshot(path=f"{OUT}/farm_mature.png")
    rot = {page.evaluate("window.__three.scene.getObjectByName('plant-sunflower').children[0].rotation.z") for _ in range(6) if not page.wait_for_timeout(180)}
    check("con movimiento normal la planta se balancea", len(rot) > 2, f"{len(rot)} valores distintos")
    ctx.close()

    # 4) movimiento reducido
    ctx, page, errs = new_page(b, reduced_motion="reduce"); page.goto(BASE); page.wait_for_function("window.__three && window.__three.scene.getObjectByName('plant-sunflower')"); page.wait_for_timeout(1200)
    rot = {page.evaluate("window.__three.scene.getObjectByName('plant-sunflower').children[0].rotation.z") for _ in range(6) if not page.wait_for_timeout(180)}
    check("con prefers-reduced-motion no hay balanceo", rot == {0}, str(rot))
    ctx.close()
    sql("UPDATE plots SET state='empty', plant_type=NULL, harvested=0, matured_at=NULL, wilts_at=NULL WHERE id=1")
    b.close()
print("\nFALLOS:" if fails else "\nTODO OK", fails or "")
sys.exit(1 if fails else 0)
