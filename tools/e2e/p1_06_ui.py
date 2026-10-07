"""E2E P1-06: UI de juego completa (tutorial, plantar con etiqueta, atajos, cancelar, cosechar con clic, retirar, desbloquear, errores, móvil).

Se ejecuta con tools/e2e/run_isolated.sh (backend y Vite propios, BD temporal). Resetea la BD de desarrollo (backend/data/pomofarm.db).
Uso: python3 tools/e2e/p1_06_ui.py
"""
import json, os, re, subprocess, sys, urllib.request
from playwright.sync_api import sync_playwright

if "POMOFARM_URL" not in os.environ:
    sys.exit("Estos tests resetean la BD y no deben tocar tu partida. Ejecútalos con: tools/e2e/run_isolated.sh tools/e2e/<script>.py")

BASE = os.environ["POMOFARM_URL"]
DB = os.environ.get("POMOFARM_DB", "")
OUT = "/tmp/pomofarm-e2e"; os.makedirs(OUT, exist_ok=True)
fails = []
def check(name, ok, detail=""):
    print(("PASS " if ok else "FAIL ") + name + (f"  [{detail}]" if detail else ""))
    if not ok: fails.append(name)
def call(m, path, body=None):
    r = urllib.request.Request(BASE + path, method=m, data=json.dumps(body).encode() if body is not None else None, headers={"Content-Type": "application/json"})
    try: return json.load(urllib.request.urlopen(r))
    except urllib.error.HTTPError as e: return json.load(e)
def sql(q): subprocess.run(["sqlite3", DB, q], check=True)
def reset():
    if call("GET", "/api/state").get("pomodoro"): call("POST", "/api/pomodoros/active/cancel")
    sql("UPDATE plots SET state='empty',plant_type=NULL,harvested=0,matured_at=NULL,wilts_at=NULL,planted_at=NULL,grow_s=NULL,life_s=NULL;"
        "UPDATE players SET focus_points=0,lifetime_focus=0; DELETE FROM settings; DELETE FROM unlocks WHERE key<>'daisy'; DELETE FROM pomodoro_events; DELETE FROM pomodoros; DELETE FROM tags;")

PROJECT = """() => { const t = window.__three; const v = new t.camera.position.constructor(1, 0.5, 1).project(t.camera);
  const r = t.gl.domElement.getBoundingClientRect(); return [r.left + (v.x + 1) / 2 * r.width, r.top + (1 - v.y) / 2 * r.height] }"""
btn = lambda pg, name: pg.get_by_role("button", name=name, exact=True)

reset()
with sync_playwright() as p:
    b = p.chromium.launch(args=["--use-gl=swiftshader", "--enable-unsafe-swiftshader"])
    pg = b.new_page(viewport={"width": 1100, "height": 760}); errs = []
    pg.on("console", lambda m: errs.append(m.text[:200]) if m.type == "error" and "409" not in m.text else None); pg.on("pageerror", lambda e: errs.append(str(e)[:200]))
    pg.goto(BASE); pg.wait_for_selector(".packet")

    # --- tutorial y plantar ---
    check("tutorial paso 1 al empezar", pg.get_attribute("[data-testid=tutorial]", "data-step") == "1")
    check("solo la Margarita está desbloqueada", pg.locator(".packet__body:not([disabled])").count() == 1 and pg.locator(".packet__unlock").count() == 4)
    check("Plantar desactivado sin semilla", btn(pg, "Plantar").is_disabled())
    pg.locator(".packet__body").first.click()
    check("tutorial paso 2 al elegir semilla", pg.get_attribute("[data-testid=tutorial]", "data-step") == "2")
    check("semilla elegida marcada (aria-checked)", pg.get_attribute(".packet__body >> nth=0", "aria-checked") == "true")
    pg.fill(".field", "emails")
    pg.keyboard.press("h")   # con el foco en el campo, H se escribe: no cosecha ni dispara atajos
    btn(pg, "Plantar").click(); pg.wait_for_selector("[data-testid=timer][data-status=running]")
    check("tutorial paso 3 en marcha", pg.get_attribute("[data-testid=tutorial]", "data-step") == "3")
    check("el dock en marcha muestra planta y etiqueta", "Margarita · emailsh" in pg.inner_text(".readout__what"), pg.inner_text(".readout__what"))
    check("título de pestaña con tiempo", "⏱" in pg.title(), pg.title())
    sizes = {n: btn(pg, n).bounding_box()["height"] for n in ["Pausar", "Cancelar"]}
    check("botones principales ≥ 44 px de alto", all(h >= 44 for h in sizes.values()), str(sizes))

    # --- atajos ---
    pg.mouse.click(30, 300); pg.keyboard.press("Space"); pg.wait_for_selector("[data-testid=timer][data-status=paused]")
    check("Espacio pausa", True); t1 = pg.inner_text("[data-testid=timer]"); pg.wait_for_timeout(2200)
    check("en pausa el tiempo no avanza", pg.inner_text("[data-testid=timer]") == t1)
    pg.keyboard.press("Space"); pg.wait_for_selector("[data-testid=timer][data-status=running]")
    check("Espacio reanuda", True)

    # --- cancelar en dos pasos ---
    btn(pg, "Cancelar").click(); check("primer clic en Cancelar pide confirmación", "¿Cancelar?" in pg.inner_text(".dock"))
    check("sigue en marcha tras el primer clic", pg.locator("[data-testid=timer]").count() == 1)
    pg.get_by_role("button", name="¿Cancelar? Se pierde la planta").click(); pg.wait_for_selector(".packet")
    check("segundo clic cancela y vuelve a elegir semilla", True)
    pg.screenshot(path=f"{OUT}/ui_after_cancel.png")
    check("la etiqueta usada queda como sugerencia", pg.locator("#recent-tags option").count() == 1 and pg.get_attribute("#recent-tags option", "value") == "emailsh")

    # --- error legible: otro dispositivo ya plantó ---
    pg.locator(".packet__body").first.click()
    call("POST", "/api/pomodoros", {"plot_id": 1, "plant_type": "daisy"})   # "otro dispositivo"
    btn(pg, "Plantar").click(); pg.wait_for_selector(".toast--error")
    check("error de otro dispositivo en lenguaje llano", "otro dispositivo" in pg.inner_text(".toast--error"), pg.inner_text(".toast--error"))
    pg.wait_for_selector("[data-testid=timer]"); check("la UI se resincroniza con el Pomodoro existente", True)
    call("POST", "/api/pomodoros/active/cancel")

    # --- cosechar con clic en la planta (estado maduro forzado) ---
    sql("UPDATE plots SET state='mature',plant_type='daisy',harvested=0,matured_at=strftime('%Y-%m-%dT%H:%M:%SZ','now'),wilts_at=strftime('%Y-%m-%dT%H:%M:%SZ','now','+1 day') WHERE id=1;"
        "INSERT INTO pomodoros (player_id,plot_id,plant_type,planned_s,started_at,status,ended_at) VALUES (1,1,'daisy',600,'t','completed','t')")
    pg.reload(); pg.wait_for_selector("[data-testid=harvest]"); pg.wait_for_timeout(1200)
    check("dock ofrece Cosechar con planta lista", True)
    x, y = pg.evaluate(PROJECT); pg.mouse.click(x, y - 10)
    pg.wait_for_selector(".toast--reward"); check("clic en la planta cosecha: +1 💧", "+1" in pg.inner_text(".toast--reward"), pg.inner_text(".toast--reward"))
    check("el contador de 💧 sube", "1" in pg.inner_text("[data-testid=focus]"))
    pg.wait_for_function("document.querySelectorAll('.toast').length >= 2")
    hint = pg.locator(".toast:not(.toast--reward)").first.inner_text()
    check("primera cosecha explica para qué sirven los 💧", "8" in hint and "Tomates" in hint, hint)
    check("el tutorial termina tras la primera cosecha", pg.locator("[data-testid=tutorial]").count() == 0)
    pg.wait_for_selector("[data-testid=clear]"); check("tras cosechar aparece Retirar planta", True)
    pg.screenshot(path=f"{OUT}/ui_harvested.png")

    # --- retirar en dos pasos ---
    btn(pg, "Retirar planta").click(); check("Retirar pide confirmación", "¿Seguro?" in pg.inner_text(".dock"))
    pg.get_by_role("button", name="¿Seguro? Deja de producir").click(); pg.wait_for_selector(".packet")
    check("tras retirar vuelve a poder sembrar", btn(pg, "Plantar").is_visible())

    # --- desbloquear con 💧 y plantar tomates ---
    sql("UPDATE players SET focus_points=10")
    pg.reload(); pg.wait_for_selector(".packet__unlock:not([disabled])")
    pg.get_by_role("button", name="Desbloquear Tomates por 8 gotas").click()
    pg.wait_for_function("document.querySelectorAll('.packet__unlock').length === 3")
    check("desbloquear descuenta 8 💧", pg.inner_text("[data-testid=focus]").strip() == "2", pg.inner_text("[data-testid=focus]"))
    pg.get_by_role("radio", name=re.compile("^Tomates")).click(); btn(pg, "Plantar").click()
    pg.wait_for_selector("[data-testid=timer][data-status=running]")
    check("Tomates dura 25 min", pg.inner_text("[data-testid=timer]").startswith("24:") or pg.inner_text("[data-testid=timer]").startswith("25:"), pg.inner_text("[data-testid=timer]"))
    call("POST", "/api/pomodoros/active/cancel")

    # --- móvil ---
    pg.set_viewport_size({"width": 390, "height": 800}); pg.reload(); pg.wait_for_selector(".packet"); pg.wait_for_timeout(800)
    overflow = pg.evaluate("document.documentElement.scrollWidth > document.documentElement.clientWidth")
    check("móvil: sin scroll horizontal de la página", not overflow)
    box = pg.locator(".dock").bounding_box(); check("móvil: el dock cabe en la pantalla", box["x"] >= 0 and box["x"] + box["width"] <= 390.5, str(box))
    pg.screenshot(path=f"{OUT}/ui_mobile.png")
    check("sin errores de consola", not errs, "; ".join(errs)[:240])
    b.close()
reset()
print("\nFALLOS:" if fails else "\nTODO OK", fails or "")
sys.exit(1 if fails else 0)
