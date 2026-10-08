"""E2E P4-05 (QA de la Fase 4): recorrido completo (Libro → métricas → prestigio → conservación), dos dispositivos, servidor caído,
zona horaria del navegador, teclado, nombres accesibles, rendimiento y humo en WebKit y Firefox.

Se ejecuta con tools/e2e/run_isolated.sh.
"""
import csv, io, json, os, sqlite3, subprocess, sys, urllib.request
from datetime import datetime, timedelta, timezone
from playwright.sync_api import sync_playwright

if "POMOFARM_URL" not in os.environ:
    sys.exit("Estos tests resetean la BD y no deben tocar tu partida. Ejecútalos con: tools/e2e/run_isolated.sh tools/e2e/p4_05_qa.py")
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
def sql(q, args=()):
    c = sqlite3.connect(DB, timeout=10); c.execute(q, args); c.commit(); c.close()
def iso(d): return d.strftime("%Y-%m-%dT%H:%M:%S.%fZ")
def state(): return call("GET", "/api/state")

call("POST", "/api/pomodoros/active/cancel")
for q in ("DELETE FROM structures", "DELETE FROM unlocks WHERE kind='animal'", "DELETE FROM plots WHERE NOT (x=1 AND y=1)", "DELETE FROM pomodoro_events", "DELETE FROM pomodoros", "DELETE FROM tags",
          "DELETE FROM settings WHERE key IN ('strict_mode')",
          "UPDATE plots SET state='empty', plant_type=NULL, planted_at=NULL, grow_s=NULL, matured_at=NULL, harvested=0, life_s=NULL, wilts_at=NULL, collected_to=NULL",
          "UPDATE players SET silo_micro=0, silo_peak_micro_h=0, silo_level=0, season=1, biome='spring', coins_milli=0, focus_points=0, rest_started_at=NULL, rest_until=NULL"):
    sql(q)
sql("INSERT INTO settings (player_id,key,value) VALUES (1,'tutorial_done','1') ON CONFLICT(player_id,key) DO UPDATE SET value='1'")
sql("INSERT INTO settings (player_id,key,value) VALUES (1,'rest_enabled','0') ON CONFLICT(player_id,key) DO UPDATE SET value='0'")

# historial: 40 días con 1–3 Pomodoros al día, algunos con etiqueta y algunos estrictos limpios
now = datetime.now(timezone.utc).replace(hour=11, minute=0, second=0, microsecond=0)
c = sqlite3.connect(DB); total = 0; clean = 0
for d in range(40):
    for k in range(1 + d % 3):
        end = now - timedelta(days=d, hours=k)
        tag = ["tesis", "inglés", None][(d + k) % 3]
        tid = None
        if tag:
            c.execute("INSERT OR IGNORE INTO tags (player_id,name) VALUES (1,?)", (tag,)); tid = c.execute("SELECT id FROM tags WHERE name=?", (tag,)).fetchone()[0]
        strict = 1 if (d + k) % 4 == 0 else 0
        c.execute("INSERT INTO pomodoros (player_id,plot_id,plant_type,tag_id,planned_s,started_at,paused_total_s,ended_at,reward_focus,status,strict) VALUES (1,NULL,'daisy',?,1500,?,0,?,1,'completed',?)",
                  (tid, iso(end - timedelta(minutes=25)), iso(end), strict)); total += 1; clean += strict
c.commit(); c.close()

def full_farm():
    t = datetime.now(timezone.utc) - timedelta(hours=1)
    for x in range(4):
        for y in range(4): sql("INSERT OR IGNORE INTO plots (player_id,x,y) VALUES (1,?,?)", (x, y))
    crops = ["daisy", "tomato", "sunflower", "apple", "oak"]
    for i in range(1, 17):
        sql("UPDATE plots SET state='mature', plant_type=?, grow_s=600, life_s=86400, harvested=1, matured_at=?, wilts_at=?, collected_to=? WHERE id=?", (crops[i % 5], iso(t), iso(t + timedelta(hours=24)), iso(t), i))
    sql("UPDATE players SET silo_level=4, coins_milli=75000000, focus_points=300")
    for k in ("bees", "dog"): sql("INSERT OR IGNORE INTO unlocks (player_id,kind,key,at) VALUES (1,'animal',?, 't')", (k,))
    for x, y in ((1, 1), (2, 1), (1, 2), (2, 2)): sql("INSERT INTO structures (player_id,kind,x,y) VALUES (1,'hive',?,?)", (x, y))
    sql("INSERT INTO structures (player_id,kind,x,y) VALUES (1,'dog',0,0),(1,'hat',0,0),(1,'lantern',4,1)")
full_farm()

with sync_playwright() as p:
    b = p.chromium.launch(args=["--use-gl=swiftshader", "--enable-unsafe-swiftshader", "--autoplay-policy=no-user-gesture-required"])
    ctx = b.new_context(viewport={"width": 1100, "height": 760}, timezone_id="Europe/Madrid", accept_downloads=True)
    pg = ctx.new_page(); errs = []
    pg.on("console", lambda m: errs.append(m.text[:200]) if m.type == "error" and "409" not in m.text and "Failed to load resource" not in m.text else None); pg.on("pageerror", lambda e: errs.append(str(e)[:200]))
    pg.goto(BASE); pg.wait_for_selector("[data-testid=silo]"); pg.wait_for_timeout(1000)

    # 1) el Libro con 40 días de historia, solo con teclado
    pg.get_by_test_id("book-button").focus(); pg.keyboard.press("Enter"); pg.wait_for_selector("[data-testid=personal-numbers]")
    check("el Libro se abre con el teclado", pg.get_attribute("[data-testid=book-button]", "aria-expanded") == "true")
    st = call("GET", "/api/stats?tz=Europe/Madrid")
    check("racha: 40 días seguidos, mejor y actual", st["best_streak_days"] == 40 and st["current_streak_days"] == 40, f"{st['best_streak_days']} / {st['current_streak_days']}")
    check("el Libro muestra esa racha en español", "40 días seguidos" in pg.inner_text("[data-testid=num-best]") and pg.inner_text("[data-testid=num-current]") == "40 días")
    check("limpios = Pomodoros estrictos completados (sin pausas)", st["clean"] == clean and st["strict_total"] == clean, f"{st['clean']} de {st['strict_total']} (esperado {clean})")
    check("el total de los números coincide con la base de datos", st["total"] == total, f"{st['total']} vs {total}")
    pg.get_by_test_id("book-table-toggle").focus(); pg.keyboard.press("Space"); pg.wait_for_selector("[data-testid=book-tables]")
    check("la tabla se activa con el teclado", pg.locator("[data-testid=book-tables]").count() == 1)
    pg.get_by_test_id("book-prev").focus(); pg.keyboard.press("Enter"); pg.wait_for_timeout(600)
    check("el mes anterior se alcanza con el teclado", pg.get_by_test_id("book-prev").is_enabled() or pg.get_by_test_id("book-next").is_enabled())
    pg.keyboard.press("Escape"); check("Escape cierra el Libro", pg.locator("[data-testid=harvest-book]").count() == 0)

    # 2) zona horaria del navegador: Auckland (UTC+13) mueve Pomodoros de día
    ak = b.new_context(viewport={"width": 1100, "height": 760}, timezone_id="Pacific/Auckland"); a = ak.new_page(); a.goto(BASE); a.wait_for_selector(".chip")
    a.get_by_test_id("book-button").click(); a.wait_for_selector("[data-testid=book-total]")
    tz_m = call("GET", "/api/book?tz=Pacific/Auckland")["time_zone"]
    check("el navegador de Auckland recibe sus días (el servidor usa su zona)", tz_m == "Pacific/Auckland" and "Pomodoro" in a.inner_text("[data-testid=book-total]"))
    ak.close()

    # 3) dos dispositivos: B hace el prestigio, A (con estado viejo) lo recibe sin romperse
    dev_b = b.new_context(viewport={"width": 390, "height": 760}); bb = dev_b.new_page(); bb.goto(BASE); bb.wait_for_selector("[data-testid=silo]"); bb.wait_for_timeout(800)
    bb.get_by_test_id("shop-button").click(); bb.wait_for_selector("[data-testid=season]")
    check("el dispositivo B ve la estación 2 lista", bb.get_by_test_id("start-season").is_enabled())
    bb.get_by_test_id("start-season").click(); bb.get_by_test_id("start-season").click()
    bb.wait_for_function("document.querySelector('.toast') && document.querySelector('.toast').innerText.includes('estación 2')", timeout=8000)
    s2 = state()
    check("B inició la estación 2 (servidor)", s2["player"]["season"] == 2 and s2["player"]["biome"] == "summer")
    # A todavía muestra la granja vieja (hasta su próximo refresco): intentar una acción vieja debe resolverse con aviso y recargar
    pg.get_by_test_id("shop-button").click(); pg.wait_for_selector("[data-testid=shop-panel]")
    old_hive_btn = pg.get_by_test_id("buy-hive").count()
    pg.keyboard.press("Escape")
    pg.evaluate("window.dispatchEvent(new Event('focus'))"); pg.wait_for_timeout(1500)
    check("A recibe la nueva estación al recuperar el foco (sin recargar a mano)", pg.evaluate("window.__three.scene.background.getHexString()") == "fbe6b4", pg.evaluate("window.__three.scene.background.getHexString()"))
    pg.get_by_test_id("book-button").click(); pg.wait_for_selector("[data-testid=personal-numbers]")
    check("el Libro y las métricas sobreviven al prestigio", call("GET", "/api/stats?tz=UTC")["total"] == total and "40 días seguidos" in pg.inner_text("[data-testid=num-best]"))
    pg.keyboard.press("Escape")
    # acción vieja contra el servidor ya reiniciado: comprar una colmena sin 🪙 → error amable, sin 500
    r = call("POST", "/api/structures", {"kind": "hive", "plot_id": 1})
    check("una compra con el estado viejo recibe un error claro (no un 500)", r.get("_status") in (409, 403) and r.get("error") in ("insufficient_coins", "animal_locked", "cell_taken"), str(r))
    dev_b.close()

    # 4) servidor caído al abrir el Libro: aviso amable, y se recupera
    open(os.environ["POMOFARM_HOLD_FILE"], "w").close()
    subprocess.run(["bash", "-c", "pkill -P " + os.environ["POMOFARM_SUPERVISOR_PID"] + " -x pomofarm || true"]); pg.wait_for_timeout(800)
    pg.get_by_test_id("book-button").click(); pg.wait_for_selector("[data-testid=harvest-book]"); pg.wait_for_timeout(1500)
    check("sin servidor, el Libro explica que no hay conexión (sin pantalla en blanco)", pg.locator("[role=alert]").count() >= 1 and "conexión" in pg.inner_text("[data-testid=harvest-book]").lower(), pg.inner_text("[data-testid=harvest-book]")[:90].replace("\n", " "))
    os.remove(os.environ["POMOFARM_HOLD_FILE"]); pg.wait_for_timeout(3000)
    pg.keyboard.press("Escape"); pg.get_by_test_id("book-button").click(); pg.wait_for_selector("[data-testid=book-total]", timeout=8000)
    check("al volver el servidor, el Libro carga", pg.locator("[data-testid=book-total]").count() == 1)
    pg.keyboard.press("Escape")

    # 5) nombres accesibles de los controles nuevos
    pg.get_by_test_id("book-button").click(); pg.wait_for_selector("[data-testid=harvest-book]")
    nameless = pg.evaluate("""() => [...document.querySelectorAll('[data-testid=harvest-book] button, [data-testid=harvest-book] input, [data-testid=harvest-book] a')]
      .filter(el => !(el.getAttribute('aria-label') || el.innerText || el.labels?.[0]?.innerText || '').trim()).length""")
    check("todos los controles del Libro tienen nombre accesible", nameless == 0, str(nameless))
    pg.get_by_test_id("book-table-toggle").uncheck(); pg.wait_for_selector(".harvest__icons")
    check("el gráfico de la cosecha y el de semanas son imágenes con texto alternativo", all(pg.get_attribute(s, "aria-label") for s in (".harvest__icons", "[data-testid=weeks-chart]")))
    pg.keyboard.press("Escape")
    pg.get_by_test_id("shop-button").click(); pg.wait_for_selector("[data-testid=season]")
    nameless = pg.evaluate("""() => [...document.querySelectorAll('[data-testid=season] button, [data-testid=season] summary')].filter(el => !(el.getAttribute('aria-label') || el.innerText || '').trim()).length""")
    check("la sección de estaciones tiene nombres accesibles y requisitos con texto (cumplido / falta)", nameless == 0 and pg.locator("[data-testid=season] .sr-only").count() == 4)
    pg.keyboard.press("Escape")

    # 6) rendimiento en la estación 2 con granja reconstruida y el Libro abierto
    full_farm_again = state()
    sql("UPDATE players SET coins_milli=75000000")
    for x, y in ((1, 1), (2, 1), (1, 2), (2, 2)): call("POST", "/api/structures", {"kind": "hive", "plot_id": [pl["id"] for pl in full_farm_again["plots"] if (pl["x"], pl["y"]) == (x, y)][0]})
    pg.reload(); pg.wait_for_selector(".chip"); pg.wait_for_timeout(1200); pg.get_by_test_id("book-button").click(); pg.wait_for_timeout(1200)
    calls = pg.evaluate("window.__three.gl.info.render.calls")
    check("draw calls razonables en la estación 2 con el Libro abierto (≤ 90)", calls <= 90, f"{calls}")
    pg.keyboard.press("Escape")

    # 7) CSV completo coincide con la BD tras el prestigio
    text = urllib.request.urlopen(BASE + "/api/book/export.csv?tz=Europe/Madrid").read().decode("utf-8-sig")
    rows = list(csv.reader(io.StringIO(text))); cc = sqlite3.connect(DB); n = cc.execute("SELECT COUNT(*) FROM pomodoros WHERE status IN ('completed','cancelled')").fetchone()[0]; cc.close()
    check("el CSV sigue teniendo una fila por Pomodoro tras el prestigio", len(rows) - 1 == n == total, f"{len(rows) - 1} / {n} / {total}")
    check("sin errores de consola en Chromium", not errs, "; ".join(errs[:3]))
    ctx.close()

    # 8) humo en WebKit (Safari) y Firefox: Libro, estaciones, accesibilidad y atajo M
    sql("UPDATE players SET season=2, biome='summer'")
    for name in ("webkit", "firefox"):
        try:
            br = getattr(p, name).launch()
        except Exception as e:
            print(f"INFO {name} no disponible: {str(e)[:70]}"); continue
        try:
            cx = br.new_context(viewport={"width": 390, "height": 760}, accept_downloads=True); q = cx.new_page(); e2 = []
            q.on("pageerror", lambda e: e2.append(str(e)[:150])); q.on("console", lambda m: e2.append(m.text[:150]) if m.type == "error" and "Failed to load resource" not in m.text else None)
            q.goto(BASE); q.wait_for_selector(".chip", timeout=15000); q.wait_for_timeout(1500)
            check(f"{name}: carga con el bioma de la estación", q.locator("canvas").count() == 1)
            q.get_by_test_id("book-button").click(); q.wait_for_selector("[data-testid=personal-numbers]", timeout=10000)
            check(f"{name}: el Libro y 'Tus números' se muestran", "40 días" in q.inner_text("[data-testid=num-best]"))
            q.get_by_test_id("book-table-toggle").check(); q.wait_for_selector("[data-testid=book-tables]")
            check(f"{name}: la vista de tabla funciona", q.locator("table").count() == 3)
            with q.expect_download() as dl: q.get_by_test_id("book-export").click()
            check(f"{name}: la descarga del CSV funciona", dl.value.suggested_filename == "libro-de-cosechas.csv")
            q.keyboard.press("Escape")
            q.get_by_test_id("shop-button").click(); q.wait_for_selector("[data-testid=season]")
            check(f"{name}: la sección de estaciones se muestra", q.locator("[data-testid=start-season]").count() == 1)
            q.keyboard.press("Escape")
            q.get_by_role("button", name="Ajustes de avisos").click(); q.wait_for_selector("[data-testid=a11y-settings]")
            q.get_by_test_id("a11y-palette").check(); q.wait_for_timeout(300)
            check(f"{name}: la paleta para daltonismo se aplica", q.evaluate("document.documentElement.dataset.palette") == "cb")
            q.keyboard.press("Escape"); q.keyboard.press("m"); q.wait_for_timeout(300)
            check(f"{name}: la tecla M silencia", q.evaluate("JSON.parse(localStorage.getItem('pomofarm.prefs')).muted") is True)
            q.screenshot(path=f"{OUT}/phase4_{name}.png")
            check(f"{name}: sin errores de JavaScript", not e2, "; ".join(e2[:2]))
        except Exception as e:
            check(f"{name}: humo completo", False, str(e)[:140])
        finally:
            br.close()
    b.close()
print(f"\n{len(fails)} fallos" if fails else "\nTodo OK"); sys.exit(1 if fails else 0)
