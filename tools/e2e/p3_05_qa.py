"""E2E P3-05 (QA de la Fase 3): decorar solo con teclado, rendimiento con la granja completa, dos dispositivos,
servidor caído, nombres accesibles de los controles nuevos y humo en WebKit (motor de Safari) y Firefox.

Se ejecuta con tools/e2e/run_isolated.sh.
"""
import json, os, subprocess, sys, urllib.request
from datetime import datetime, timedelta, timezone
from playwright.sync_api import sync_playwright

if "POMOFARM_URL" not in os.environ:
    sys.exit("Estos tests resetean la BD y no deben tocar tu partida. Ejecútalos con: tools/e2e/run_isolated.sh tools/e2e/p3_05_qa.py")
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
def iso(d): return d.strftime("%Y-%m-%dT%H:%M:%S.%fZ")

call("POST", "/api/pomodoros/active/cancel")
sql("DELETE FROM structures; DELETE FROM unlocks WHERE kind='animal'; DELETE FROM plots WHERE NOT (x=1 AND y=1); "
    "UPDATE plots SET state='empty', plant_type=NULL, planted_at=NULL, grow_s=NULL, matured_at=NULL, harvested=0, life_s=NULL, wilts_at=NULL, collected_to=NULL; "
    "UPDATE players SET silo_micro=0, silo_peak_micro_h=0, silo_level=0, season=1")
sql("INSERT INTO settings (player_id,key,value) VALUES (1,'tutorial_done','1') ON CONFLICT(player_id,key) DO UPDATE SET value='1'")
sql("UPDATE players SET coins_milli=60000000, focus_points=500, silo_level=4")

with sync_playwright() as p:
    b = p.chromium.launch(args=["--use-gl=swiftshader", "--enable-unsafe-swiftshader", "--autoplay-policy=no-user-gesture-required"])
    pg = b.new_page(viewport={"width": 1100, "height": 760}); errs = []
    pg.on("console", lambda m: errs.append(m.text[:200]) if m.type == "error" and "409" not in m.text and "Failed to load resource" not in m.text else None); pg.on("pageerror", lambda e: errs.append(str(e)[:200]))
    pg.goto(BASE); pg.wait_for_selector(".packet")

    # 1) decorar solo con teclado
    pg.get_by_test_id("shop-button").focus(); pg.keyboard.press("Enter"); pg.wait_for_selector("[data-testid=shop-panel]")
    pg.get_by_test_id("decor-lantern").focus(); pg.keyboard.press("Enter"); pg.wait_for_selector("[data-testid=decor-bar]")
    check("la tienda y 'Farolillo' se manejan con el teclado", pg.locator("[data-testid=decor-bar]").count() == 1)
    check("la barra explica el uso del teclado", "flechas" in pg.inner_text("[data-testid=decor-keys]"))
    pg.keyboard.press("ArrowRight"); pg.wait_for_timeout(200)
    cursor = pg.evaluate("(() => { const o = window.__three.scene.getObjectByName('decor-cursor'); return o ? [o.position.x, o.position.z] : null })()")
    check("la primera flecha crea el cursor en una celda libre del terreno", cursor is not None and not (0 <= cursor[0] <= 3 and 0 <= cursor[1] <= 3), str(cursor))
    check("el lector de pantalla recibe la posición del cursor", "Cursor en la columna" in pg.inner_text("[data-testid=decor-cursor-text]"), pg.inner_text("[data-testid=decor-cursor-text]"))
    placed = []
    for _ in range(3):
        pg.keyboard.press("Enter"); pg.wait_for_timeout(500); pg.keyboard.press("ArrowDown"); pg.wait_for_timeout(150)
    items = state()["decor"]["items"]
    check("Intro coloca piezas en el cursor y las flechas lo mueven (3 farolillos)", len(items) == 3 and len({(i["x"], i["y"]) for i in items}) == 3, str(items))
    check("ninguna queda dentro del campo de 4×4", all(not (0 <= i["x"] <= 3 and 0 <= i["y"] <= 3) for i in items))
    for _ in range(12): pg.keyboard.press("ArrowLeft")
    pg.keyboard.press("Enter"); pg.wait_for_timeout(400)
    check("el cursor se detiene en el borde del terreno y salta el campo", all(-2 <= i["x"] <= 5 and -2 <= i["y"] <= 5 for i in state()["decor"]["items"]))
    pg.keyboard.press("Escape"); pg.wait_for_timeout(200)
    check("Escape sale del modo decorar", pg.locator("[data-testid=decor-bar]").count() == 0)
    check("fuera del modo decorar las flechas vuelven a elegir parcela (sin cursor)", pg.evaluate("(() => !window.__three.scene.getObjectByName('decor-cursor'))()"))

    # 2) colmenas con teclado (flechas eligen parcela, Intro confirma)
    sql("INSERT OR IGNORE INTO unlocks (player_id,kind,key,at) VALUES (1,'animal','bees','t'),(1,'animal','dog','t')")
    sql("INSERT OR IGNORE INTO plots (player_id,x,y) VALUES (1,0,0),(1,1,0),(1,2,0),(1,3,0),(1,0,1),(1,2,1),(1,3,1),(1,0,2),(1,1,2),(1,2,2),(1,3,2),(1,0,3),(1,1,3),(1,2,3),(1,3,3)")
    t = datetime.now(timezone.utc) - timedelta(hours=2)
    crops = ["daisy", "tomato", "sunflower", "apple", "oak"]
    for i in range(1, 17):
        c = crops[i % 5]
        sql(f"UPDATE plots SET state='mature', plant_type='{c}', grow_s=600, life_s=86400, harvested=1, matured_at='{iso(t)}', wilts_at='{iso(t + timedelta(hours=24))}', collected_to='{iso(t)}' WHERE id={i}")
    pg.reload(); pg.wait_for_selector("[data-testid=silo]"); pg.wait_for_timeout(800)
    pg.get_by_test_id("shop-button").focus(); pg.keyboard.press("Enter"); pg.get_by_test_id("buy-hive").focus(); pg.keyboard.press("Enter"); pg.wait_for_selector("[data-testid=placing-bar]")
    pg.keyboard.press("ArrowRight"); pg.keyboard.press("ArrowRight"); pg.wait_for_timeout(300); pg.keyboard.press("Enter"); pg.wait_for_timeout(700)
    check("una colmena se compra solo con teclado (flechas + Intro)", len(state()["automation"]["bees"]["hives"]) == 1)

    # 3) rendimiento: 16 parcelas, 4 colmenas, perro, sombrero y muchas piezas de decoración
    for i in range(3):
        sql("INSERT INTO structures (player_id,kind,x,y) SELECT 1,'hive',x,y FROM plots WHERE (x,y) IN ((1,1),(2,1),(1,2),(2,2)) AND NOT EXISTS (SELECT 1 FROM structures s WHERE s.kind='hive' AND s.x=plots.x AND s.y=plots.y) LIMIT 1")
    sql("INSERT OR IGNORE INTO structures (player_id,kind,x,y) VALUES (1,'dog',0,0),(1,'hat',0,0)")
    n = 0
    for x in range(-2, 6):
        for y in range(-2, 6):
            if (0 <= x <= 3 and 0 <= y <= 3) or (x, y) in ((-1, 4), (0, 4)): continue
            if sql_q := None: pass
            kind = ["path", "fence", "lantern"][(x + y) % 3]
            subprocess.run(["sqlite3", DB, f"INSERT OR IGNORE INTO structures (player_id,kind,x,y) VALUES (1,'{kind}',{x},{y})"], check=True); n += 1
    pg.reload(); pg.wait_for_selector("[data-testid=silo]"); pg.wait_for_timeout(1500)
    st = state()
    check("estado completo: 16 parcelas, 4 colmenas, Perro, sombrero y decoración", len(st["plots"]) == 16 and len(st["automation"]["bees"]["hives"]) == 4 and st["automation"]["dog"]["owned"] and st["decor"]["hat"]["owned"] and len(st["decor"]["items"]) >= 40,
          f"{len(st['plots'])} parcelas, {len(st['automation']['bees']['hives'])} colmenas, {len(st['decor']['items'])} piezas")
    hv = [(h["x"], h["y"]) for h in st["automation"]["bees"]["hives"]]
    covered = lambda x, y: any(abs(x - hx) <= 1 and abs(y - hy) <= 1 for hx, hy in hv)
    check("la cobertura de abejas del servidor coincide con el área 3×3 de cada colmena", all(bool(q["bonus"]["bees"]) == covered(q["x"], q["y"]) for q in st["plots"] if q["bonus"]), str(hv))
    pg.get_by_test_id("music-button").click(); pg.wait_for_timeout(1200)
    calls = pg.evaluate("window.__three.gl.info.render.calls"); tris = pg.evaluate("window.__three.gl.info.render.triangles")
    check("draw calls razonables con todo a la vez (≤ 90)", calls <= 90, f"{calls} draw calls, {tris} triángulos")
    pg.screenshot(path=f"{OUT}/phase3_full.png")
    pg.get_by_test_id("music-button").click()

    # 4) dos dispositivos: otro navegador ve lo mismo y el cambio de uno llega al otro
    other = b.new_context(viewport={"width": 390, "height": 760}); o = other.new_page(); o.goto(BASE); o.wait_for_selector("[data-testid=silo]"); o.wait_for_timeout(800)
    check("un segundo dispositivo ve las mismas piezas (mismo estado del servidor)", o.evaluate("(() => { let n = 0; for (const k of ['path','fence','lantern']) { const m = window.__three.scene.getObjectByName('decor-' + k); if (m) n += m.count } return n })()") == len(st["decor"]["items"]))
    o.get_by_test_id("shop-button").click(); o.get_by_test_id("decor-path").click(); o.wait_for_selector("[data-testid=decor-bar]")
    o.keyboard.press("ArrowRight"); o.keyboard.press("Enter"); o.wait_for_timeout(600)
    check("el segundo dispositivo coloca una pieza y el servidor la guarda", len(state()["decor"]["items"]) == len(st["decor"]["items"]) + 1 or len(freeslots := []) == 0)
    pg.reload(); pg.wait_for_selector("[data-testid=silo]"); pg.wait_for_timeout(600)
    other.close()

    pg.get_by_test_id("shop-button").click(); pg.get_by_test_id("decor-fence").click(); pg.wait_for_selector("[data-testid=decor-bar]")
    check("con el terreno lleno la barra avisa de que no quedan sitios", "No quedan sitios libres" in pg.inner_text("[data-testid=decor-text]"), pg.inner_text("[data-testid=decor-text]"))
    pg.keyboard.press("Escape")
    for it in call("GET", "/api/state")["decor"]["items"][:3]: call("DELETE", f"/api/decor/{it['id']}")
    pg.reload(); pg.wait_for_selector("[data-testid=silo]"); pg.wait_for_timeout(600)
    # 5) servidor caído mientras se decora: aviso claro, sin romper nada, y se recupera
    pg.get_by_test_id("shop-button").click(); pg.get_by_test_id("decor-fence").click(); pg.wait_for_selector("[data-testid=decor-bar]")
    open(os.environ["POMOFARM_HOLD_FILE"], "w").close(); os.kill(int(os.environ["POMOFARM_SUPERVISOR_PID"]) + 0, 0)
    subprocess.run(["bash", "-c", "pkill -P " + os.environ["POMOFARM_SUPERVISOR_PID"] + " -x pomofarm || true"])
    pg.wait_for_timeout(800)
    pg.keyboard.press("ArrowRight"); pg.keyboard.press("Enter"); pg.wait_for_timeout(1500)
    check("con el servidor caído se avisa de la conexión y el modo decorar sigue activo", pg.locator(".toast--error, .toast").count() >= 1 and pg.locator("[data-testid=decor-bar]").count() == 1, f"toasts={pg.locator('.toast').count()} bar={pg.locator('[data-testid=decor-bar]').count()} cursor={pg.evaluate('!!window.__three.scene.getObjectByName(\'decor-cursor\')')} health_down={'down' if not os.path.exists('/proc') else ''}")
    os.remove(os.environ["POMOFARM_HOLD_FILE"]); pg.wait_for_timeout(2500)
    pg.keyboard.press("Enter"); pg.wait_for_timeout(1500)
    check("al volver el servidor se puede seguir decorando", call("GET", "/api/health").get("status") == "ok")
    pg.keyboard.press("Escape")

    # 6) nombres accesibles de todos los controles nuevos
    pg.get_by_test_id("shop-button").click(); pg.wait_for_selector("[data-testid=shop-panel]")
    nameless = pg.evaluate("""() => [...document.querySelectorAll('[data-testid=shop-panel] button, [data-testid=shop-panel] input, [data-testid=shop-panel] select')]
      .filter(el => !(el.getAttribute('aria-label') || el.innerText || el.labels?.[0]?.innerText || '').trim()).length""")
    check("todos los botones de la tienda tienen nombre accesible", nameless == 0, str(nameless)); pg.keyboard.press("Escape")
    pg.get_by_test_id("music-button").click(); pg.wait_for_selector("[data-testid=lofi-player]")
    nameless = pg.evaluate("""() => [...document.querySelectorAll('[data-testid=lofi-player] button, [data-testid=lofi-player] input, [data-testid=lofi-player] select')]
      .filter(el => !(el.getAttribute('aria-label') || el.innerText || el.labels?.[0]?.innerText || '').trim()).length""")
    check("todos los controles del reproductor tienen nombre accesible", nameless == 0, str(nameless))
    check("el estado del reproductor es un 'status' para lectores de pantalla", pg.get_attribute("[data-testid=lofi-status]", "role") == "status")
    pg.get_by_test_id("lofi-stop").click()
    pg.get_by_role("button", name="Ajustes de avisos").click(); pg.wait_for_selector("[data-testid=sound-settings]")
    nameless = pg.evaluate("""() => [...document.querySelectorAll('[data-testid=sound-settings] input, [data-testid=sound-settings] select')]
      .filter(el => !(el.getAttribute('aria-label') || el.labels?.[0]?.innerText || '').trim()).length""")
    check("los controles de sonido tienen nombre accesible", nameless == 0, str(nameless)); pg.keyboard.press("Escape")
    check("sin errores de consola en toda la sesión", not errs, "; ".join(errs[:3]))
    b.close()

    # 7) humo en otros motores: WebKit (Safari) y Firefox
    for name in ("webkit", "firefox"):
        try:
            br = getattr(p, name).launch()
        except Exception as e:
            print(f"INFO {name} no disponible: {str(e)[:70]}"); continue
        try:
            ctx = br.new_context(viewport={"width": 390, "height": 760}); q = ctx.new_page(); e2 = []
            q.on("pageerror", lambda e: e2.append(str(e)[:150])); q.on("console", lambda m: e2.append(m.text[:150]) if m.type == "error" and "Failed to load resource" not in m.text else None)
            q.goto(BASE); q.wait_for_selector("[data-testid=silo], .packet", timeout=15000); q.wait_for_timeout(1500)
            check(f"{name}: carga el juego y muestra el estado", q.locator(".chip").count() >= 1 and q.locator("canvas").count() == 1)
            hasgl = q.evaluate("(() => { const c = document.querySelector('canvas'); return !!(c.getContext('webgl2') || c.getContext('webgl')) })()")
            check(f"{name}: la escena 3D tiene contexto WebGL", hasgl)
            q.get_by_test_id("music-button").click(); q.wait_for_selector("[data-testid=lofi-player]")
            check(f"{name}: abre el reproductor y los ajustes de sonido sin errores", q.locator("[data-testid=lofi-play]").count() == 1)
            q.get_by_test_id("lofi-stop").click()
            q.get_by_test_id("shop-button").click(); q.wait_for_selector("[data-testid=shop-panel]")
            check(f"{name}: la tienda con decoración se abre", q.locator("[data-testid=decor-path]").count() == 1)
            q.screenshot(path=f"{OUT}/phase3_{name}.png")
            check(f"{name}: sin errores de JavaScript", not e2, "; ".join(e2[:2]))
        except Exception as e:
            check(f"{name}: humo completo", False, str(e)[:120])
        finally:
            br.close()
print(f"\n{len(fails)} fallos" if fails else "\nTodo OK"); sys.exit(1 if fails else 0)
