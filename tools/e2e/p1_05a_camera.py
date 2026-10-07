"""E2E P1-05a: cámara isométrica con zoom (rueda), paneo (arrastre) y límites. Se ejecuta con tools/e2e/run_isolated.sh."""
import os, sys
from playwright.sync_api import sync_playwright

if "POMOFARM_URL" not in os.environ:
    sys.exit("Estos tests resetean la BD y no deben tocar tu partida. Ejecútalos con: tools/e2e/run_isolated.sh tools/e2e/<script>.py")

BASE = os.environ["POMOFARM_URL"]
OUT = "/tmp/pomofarm-e2e"; os.makedirs(OUT, exist_ok=True)
fails = []
def check(name, ok, detail=""):
    print(("PASS " if ok else "FAIL ") + name + (f"  [{detail}]" if detail else ""))
    if not ok: fails.append(name)

with sync_playwright() as p:
    b = p.chromium.launch(args=["--use-gl=swiftshader", "--enable-unsafe-swiftshader"])
    page = b.new_page(viewport={"width": 1000, "height": 640})
    errors = []; page.on("pageerror", lambda e: errors.append(str(e)))
    page.on("console", lambda m: errors.append(m.text[:160]) if m.type == "error" else None)
    page.goto(BASE); page.wait_for_selector("canvas"); page.wait_for_timeout(1500)
    page.mouse.move(500, 320); page.mouse.wheel(0, -1); page.wait_for_timeout(400)   # despierta el estado
    iso = lambda: page.evaluate("window.__iso")
    s0 = iso(); check("la cámara reporta su estado", s0 is not None, str(s0))
    z0 = s0["zoom"]

    page.mouse.move(500, 320)
    for _ in range(5): page.mouse.wheel(0, -300); page.wait_for_timeout(60)
    page.wait_for_timeout(600); z1 = iso()["zoom"]
    check("rueda hacia arriba acerca", z1 > z0, f"{z0:.1f} -> {z1:.1f}")
    for _ in range(40): page.mouse.wheel(0, -400); page.wait_for_timeout(30)
    page.wait_for_timeout(600); zmax = iso()["zoom"]
    check("zoom máximo limitado (140)", zmax <= 140.01, f"{zmax:.1f}")
    for _ in range(80): page.mouse.wheel(0, 500); page.wait_for_timeout(30)
    page.wait_for_timeout(600); zmin = iso()["zoom"]
    check("zoom mínimo limitado (30)", zmin >= 29.99, f"{zmin:.1f}")

    # paneo moderado
    before = iso(); page.mouse.move(500, 320); page.mouse.down(); page.mouse.move(420, 300, steps=8); page.mouse.up(); page.wait_for_timeout(700)
    after = iso(); moved = abs(after["x"] - before["x"]) + abs(after["z"] - before["z"])
    check("arrastrar desplaza la vista", moved > 0.05, f"movido {moved:.2f}")
    check("el paneo no cambia el zoom", abs(after["zoom"] - before["zoom"]) < 0.01)

    # paneo extremo: no se sale del límite (6 unidades del centro 1.5,1.5)
    for _ in range(12):
        page.mouse.move(900, 600); page.mouse.down(); page.mouse.move(100, 50, steps=6); page.mouse.up()
    page.wait_for_timeout(900); e = iso()
    d = ((e["x"] - 1.5) ** 2 + (e["z"] - 1.5) ** 2) ** 0.5
    check("el paneo respeta el límite de 6 unidades", d <= 6.05, f"distancia {d:.2f}")

    page.screenshot(path=f"{OUT}/camera.png")
    check("sin errores de consola", not errors, "; ".join(errors)[:200])
    b.close()
print("\nFALLOS:" if fails else "\nTODO OK", fails or "")
sys.exit(1 if fails else 0)
