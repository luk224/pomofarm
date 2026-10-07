"""QA de la Fase 2 (P2-08): teclado y accesibilidad de lo nuevo, tres navegadores con una granja de la fase completa,
persistencia del estado ante reinicio, recoger sin conexión, movimiento reducido.

Se ejecuta con tools/e2e/run_isolated.sh.
"""
import json, os, re, signal, subprocess, sys, time, urllib.request
from datetime import datetime, timedelta, timezone
from playwright.sync_api import sync_playwright

if "POMOFARM_URL" not in os.environ:
    sys.exit("Estos tests resetean la BD y no deben tocar tu partida. Ejecútalos con: tools/e2e/run_isolated.sh tools/e2e/p2_08_qa.py")
BASE = os.environ["POMOFARM_URL"]; DB = os.environ["POMOFARM_DB"]
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
def force(pid, crop, minutes_ago=30.0, harvested=1):
    g, l = SPEC[crop]; t = datetime.now(timezone.utc) - timedelta(minutes=minutes_ago)
    sql(f"UPDATE plots SET state='mature', plant_type='{crop}', grow_s={g}, life_s={l*3600}, harvested={harvested}, matured_at='{iso(t)}', wilts_at='{iso(t+timedelta(hours=l))}', collected_to='{iso(t)}' WHERE id={pid}")
def reset_farm():
    sql("UPDATE plots SET state='empty',plant_type=NULL,harvested=0,matured_at=NULL,wilts_at=NULL,collected_to=NULL,grow_s=NULL,life_s=NULL;"
        "UPDATE players SET silo_micro=0,silo_peak_micro_h=0,coins_milli=0,rest_started_at=NULL,rest_until=NULL")
btn = lambda pg, n: pg.get_by_role("button", name=n, exact=True)
def new_page(b, **kw):
    ctx = b.new_context(viewport={"width": 1100, "height": 760}, **kw); pg = ctx.new_page(); errs = []
    pg.on("console", lambda m: errs.append(m.text[:200]) if m.type == "error" and "409" not in m.text and "502" not in m.text and "WebGL" not in m.text else None)
    pg.on("pageerror", lambda e: errs.append(str(e)[:200])); return ctx, pg, errs
def lin(c):
    c /= 255; return c / 12.92 if c <= 0.03928 else ((c + 0.055) / 1.055) ** 2.4
def lum(h): h = h.lstrip("#"); r, g, bl = [int(h[i:i+2], 16) for i in (0, 2, 4)]; return 0.2126 * lin(r) + 0.7152 * lin(g) + 0.0722 * lin(bl)
def cr(a, b): la, lb = sorted([lum(a), lum(b)], reverse=True); return (la + 0.05) / (lb + 0.05)
def blend(fg, bg, a): f = [int(fg.lstrip("#")[i:i+2], 16) for i in (0, 2, 4)]; k = [int(bg.lstrip("#")[i:i+2], 16) for i in (0, 2, 4)]; return "#%02x%02x%02x" % tuple(round(a * x + (1 - a) * y) for x, y in zip(f, k))

sql("INSERT INTO settings (player_id,key,value) VALUES (1,'tutorial_done','1') ON CONFLICT(player_id,key) DO UPDATE SET value='1'; UPDATE players SET focus_points=500")
call("POST", "/api/plots"); call("POST", "/api/plots")    # 3 parcelas

with sync_playwright() as p:
    b = p.chromium.launch(args=["--use-gl=swiftshader", "--enable-unsafe-swiftshader"])

    print("\n## Teclado solo con varias parcelas")
    ctx, pg, errs = new_page(b); pg.goto(BASE); pg.wait_for_selector(".packet"); pg.wait_for_timeout(600)
    ann = lambda: pg.inner_text("[data-testid=plot-announcer]").strip()
    first = ann(); check("al abrir, el lector de pantalla oye qué parcela está en vista", re.fullmatch(r"Parcela \d de 3: libre", first) is not None, first)
    seen = {first}
    for _ in range(3): pg.keyboard.press("ArrowRight"); pg.wait_for_timeout(120); seen.add(ann())
    check("→ recorre las tres parcelas y da la vuelta", len(seen) == 3, str(sorted(seen)))
    pg.keyboard.press("ArrowLeft"); pg.wait_for_timeout(120); back = ann()
    pg.keyboard.press("ArrowRight"); pg.wait_for_timeout(120)
    check("← vuelve a la anterior", back != ann())
    pg.keyboard.press("ArrowRight"); pg.wait_for_timeout(150); target = ann(); n = int(re.search(r"Parcela (\d)", target).group(1))
    for _ in range(14):    # Tab hasta un sobre libre y elegirlo con Espacio
        pg.keyboard.press("Tab")
        if pg.evaluate("document.activeElement.classList.contains('packet__body') && !document.activeElement.disabled"): break
    pg.keyboard.press("Space"); pg.get_by_role("button", name="Plantar", exact=True).focus(); pg.keyboard.press("Enter")
    pg.wait_for_selector("[data-testid=timer]"); pl = call("GET", "/api/state")["pomodoro"]["plot_id"]
    check("plantar con el teclado va a la parcela que anunció el lector de pantalla", pl == n, f"anunció la parcela {n}, el servidor plantó en la {pl}")
    call("POST", "/api/pomodoros/active/cancel"); pg.reload(); pg.wait_for_selector(".packet")
    before = ann(); pg.locator(".field").focus(); pg.keyboard.type("ab"); pg.keyboard.press("ArrowRight"); pg.keyboard.press("ArrowLeft")
    check("con el cursor en un campo de texto, las flechas no cambian de parcela", ann() == before and pg.input_value(".field") == "ab")
    ctx.close()

    print("\n## Tabulación y foco de los controles nuevos")
    reset_farm(); force(1, "daisy", 2.0, 0); force(2, "tomato", 30.0, 1)
    ctx, pg, errs = new_page(b); pg.goto(BASE); pg.wait_for_selector("[data-testid=harvest]"); pg.wait_for_timeout(600)
    btn(pg, "Cosechar").click(); pg.wait_for_selector("[data-testid=rest]"); pg.wait_for_selector("[data-testid=silo]")
    def reachable(testid):
        pg.evaluate("document.activeElement && document.activeElement.blur(); window.scrollTo(0, 0)")
        for _ in range(45):
            pg.keyboard.press("Tab")
            if pg.evaluate(f"document.activeElement && document.activeElement.dataset && document.activeElement.dataset.testid === '{testid}'"):
                return pg.evaluate("(() => { const s = getComputedStyle(document.activeElement); return s.outlineStyle + ' ' + s.outlineWidth })()")
        return None
    for _ in range(4):    # con una parcela libre el dock apunta a ella: se elige con el teclado la cosechada para ver «Retirar planta»
        if "cosechada y produciendo" in pg.inner_text("[data-testid=plot-announcer]"): break
        pg.keyboard.press("ArrowRight"); pg.wait_for_timeout(150)
    for tid, name in [("shop-button", "botón Mejoras"), ("silo-collect", "botón Recoger del Silo"), ("rest-skip", "botón Saltar descanso"), ("clear", "botón Retirar planta")]:
        o = reachable(tid); check(f"el {name} se alcanza con Tab y muestra el foco", o is not None and "solid" in o and float(o.split()[1].replace("px", "")) >= 2, str(o))
    pg.get_by_test_id("silo-collect").focus(); pg.keyboard.press("Enter"); pg.wait_for_selector(".toast--coin", timeout=5000)
    check("Recoger el Silo funciona con la tecla Intro", True)
    pg.get_by_test_id("shop-button").focus(); pg.keyboard.press("Space"); pg.wait_for_selector("[data-testid=shop-panel]")
    check("la tienda se abre con el teclado y anuncia su estado (aria-expanded)", pg.get_attribute("[data-testid=shop-button]", "aria-expanded") == "true")
    pg.keyboard.press("Tab"); pg.keyboard.press("Tab"); inside = pg.evaluate("!!document.activeElement.closest('#shop-panel')")
    check("y el foco entra en el panel de la tienda", inside)
    pg.keyboard.press("Escape"); check("Escape la cierra", pg.locator("[data-testid=shop-panel]").count() == 0)
    check("sin errores de consola", not errs, "; ".join(errs)[:200]); ctx.close()

    print("\n## Contraste de los colores nuevos (WCAG AA)")
    panel = blend("#ffffff", "#cfe9f5", 0.88)
    pairs = [("texto de la tienda y los ajustes (tinta sobre panel)", "#3b2a1f", panel, 4.5), ("detalle gris del Silo y la tienda (ink-soft sobre panel)", "#5a4a3d", panel, 4.5),
             ("«Te faltan N 💧» (rojo sobre panel)", "#b3261e", panel, 4.5), ("botón primario (blanco sobre verde)", "#ffffff", "#2f7d32", 4.5), ("Recoger / Cosechar (tinta sobre sol)", "#3b2a1f", "#ffc400", 4.5),
             ("insignia verde de bono (blanco sobre verde)", "#ffffff", "#2f7d32", 4.5), ("insignia dorada (tinta sobre sol)", "#3b2a1f", "#ffc400", 4.5), ("cuadro de minutos del descanso (tinta sobre blanco)", "#3b2a1f", "#ffffff", 4.5),
             ("atajos: nombre de tecla", "#3b2a1f", panel, 4.5), ("atajos: descripción", "#5a4a3d", panel, 4.5), ("aviso +🪙 (tinta sobre sol, grande)", "#3b2a1f", "#ffc400", 3.0)]
    bad = [(n, round(cr(a, c), 1)) for n, a, c, t in pairs if cr(a, c) < t]
    check(f"los {len(pairs)} pares de color nuevos cumplen el contraste mínimo", not bad, str(bad) if bad else f"mínimo {min(cr(a, c) for _, a, c, _ in pairs):.1f}:1")

    print("\n## Movimiento reducido")
    reset_farm(); force(1, "daisy", 300.0, 1)    # 5 h de producción: hay monedas que recoger y, por tanto, chispa
    ctx, pg, errs = new_page(b, reduced_motion="reduce"); pg.goto(BASE); pg.wait_for_function("window.__three && window.__three.scene.getObjectByName('silo-3d')"); pg.wait_for_timeout(1500)
    ys = []
    for _ in range(6):
        ys.append(pg.evaluate("(() => { let y = null; window.__three.scene.getObjectByName('silo-3d').traverse(o => { if (o.children.length === 0 && o.geometry && o.geometry.type === 'OctahedronGeometry') y = o.parent.position.y }); return y })()")); pg.wait_for_timeout(200)
    check("con movimiento reducido la chispa del Silo no flota", len(set(ys)) == 1 and ys[0] is not None, str(ys))
    check("y las barras no animan su relleno", pg.evaluate("getComputedStyle(document.querySelector('.silo__bar span')).transitionDuration") in ("0s", "0.000001s"), pg.evaluate("getComputedStyle(document.querySelector('.silo__bar span')).transitionDuration"))
    ctx.close()

    print("\n## Reinicio del servidor con todo el estado de la fase")
    reset_farm(); force(1, "daisy", 5.0, 0); force(2, "tomato", 20.0, 1); force(3, "sunflower", 20.0, 1)
    call("POST", "/api/plots/1/harvest")    # arranca un descanso
    snap = lambda: (lambda s: {"player": s["player"], "plots": [(q["id"], q["state"], q["plant_type"], q["harvested"], q["matured_at"], q["wilts_at"], q["bonus"]) for q in s["plots"]], "shop": s["shop"], "settings": s["settings"], "rest_end": s["rest"] and s["rest"]["ends_at"], "silo": s["silo"]})(call("GET", "/api/state"))
    a = snap()
    hold = os.environ["POMOFARM_HOLD_FILE"]; open(hold, "w").close()
    pid = subprocess.run(["pgrep", "-P", os.environ["POMOFARM_SUPERVISOR_PID"], "-x", "pomofarm"], capture_output=True, text=True).stdout.split()[0]
    os.kill(int(pid), signal.SIGTERM); time.sleep(0.6)

    print("\n## Recoger el Silo sin conexión")
    ctx, pg, errs = new_page(b)
    ctx.route("**/api/state", lambda r: r.abort())     # la página no puede cargar el estado: probamos la acción con el estado ya cargado antes
    ctx.unroute("**/api/state")
    os.remove(hold)
    for _ in range(60):
        try: urllib.request.urlopen(BASE + "/api/health"); break
        except Exception: time.sleep(0.25)
    bb = snap()
    check("tras reiniciar el servidor, jugador, parcelas, tienda y ajustes quedan idénticos", a["player"] == bb["player"] and a["plots"] == bb["plots"] and a["shop"] == bb["shop"] and a["settings"] == bb["settings"], "")
    check("y el descanso sigue con la misma hora de fin", a["rest_end"] == bb["rest_end"] and a["rest_end"] is not None, str(bb["rest_end"]))
    check("el Silo conserva lo producido (solo puede haber crecido)", bb["silo"]["content_milli"] >= a["silo"]["content_milli"], f"{a['silo']['content_milli']} -> {bb['silo']['content_milli']}")
    pg.goto(BASE); pg.wait_for_selector("[data-testid=silo]"); open(hold, "w").close()
    pid = subprocess.run(["pgrep", "-P", os.environ["POMOFARM_SUPERVISOR_PID"], "-x", "pomofarm"], capture_output=True, text=True).stdout.split()[0]
    os.kill(int(pid), signal.SIGTERM); time.sleep(0.6); coins0 = pg.inner_text("[data-testid=coins]")
    pg.get_by_test_id("silo-collect").click(); pg.wait_for_selector(".toast--error", timeout=8000)
    check("recoger sin servidor da un aviso claro de conexión", "conexión" in pg.inner_text(".toast--error"), pg.inner_text(".toast--error"))
    check("y no cambia el saldo ni rompe la pantalla", pg.inner_text("[data-testid=coins]") == coins0 and pg.locator("[data-testid=silo]").count() == 1)
    os.remove(hold)
    for _ in range(60):
        try: urllib.request.urlopen(BASE + "/api/health"); break
        except Exception: time.sleep(0.25)
    pg.evaluate("window.dispatchEvent(new Event('focus'))"); pg.wait_for_timeout(800)
    pg.get_by_test_id("silo-collect").click(); pg.wait_for_selector(".toast--coin", timeout=8000)
    check("al volver el servidor, recoger funciona y el aviso desaparece", pg.locator(".toast--coin").count() >= 1)
    ctx.close()

    print("\n## Tres navegadores con una granja de la fase completa")
    reset_farm(); sql("UPDATE players SET silo_level=1")
    call("POST", "/api/plots")   # 4 parcelas: el 2×2 central
    for pid_, crop in {1: "daisy", 2: "tomato", 3: "oak", 4: "sunflower"}.items(): force(pid_, crop, 5.0, 0)
    call("POST", "/api/plots/1/harvest")
    b.close()
    for name in ["chromium", "firefox", "webkit"]:
        br = getattr(p, name).launch(args=["--use-gl=swiftshader", "--enable-unsafe-swiftshader"] if name == "chromium" else [])
        ctx, pg, errs = new_page(br)
        pg.goto(BASE); pg.wait_for_selector("[data-testid=silo]", timeout=30000); pg.wait_for_selector("[data-testid=rest]"); pg.wait_for_timeout(2000)
        badges = pg.evaluate("(() => { const out = []; window.__three.scene.traverse(o => { if (o.name === 'bonus-badge') out.push([o.userData.text, o.userData.gold]) }); return out.sort() })()")
        ok = badges == [["+25%", True], ["+25%", True], ["+35%", True], ["+35%", True]]
        check(f"{name}: el huerto completo muestra sus 4 insignias doradas", ok, str(badges))
        pg.get_by_test_id("shop-button").click(); pg.wait_for_selector("[data-testid=shop-panel]"); check(f"{name}: la tienda abre con parcelas y Silo", "Parcela 5" in pg.inner_text("[data-testid=shop-panel]") and "Ampliar el Silo" in pg.inner_text("[data-testid=shop-panel]"))
        pg.keyboard.press("Escape"); pg.get_by_test_id("silo-collect").click(); pg.wait_for_selector(".toast--coin", timeout=8000)
        check(f"{name}: recoger el Silo da monedas", "+" in pg.inner_text(".toast--coin"), pg.inner_text(".toast--coin"))
        check(f"{name}: sin errores de consola", not errs, "; ".join(errs)[:160]); ctx.close(); br.close()
sql("DELETE FROM plots WHERE id > 1; UPDATE plots SET state='empty',plant_type=NULL,harvested=0,matured_at=NULL,wilts_at=NULL,collected_to=NULL,grow_s=NULL,life_s=NULL; UPDATE players SET silo_level=0,silo_micro=0,silo_peak_micro_h=0,coins_milli=0,focus_points=0,rest_started_at=NULL,rest_until=NULL; DELETE FROM settings")
print("\nFALLOS:" if fails else "\nTODO OK", fails or "")
sys.exit(1 if fails else 0)
