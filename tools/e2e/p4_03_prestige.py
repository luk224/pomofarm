"""E2E P4-03: prestigio por estaciones (requisitos, doble confirmación, qué se reinicia y qué se conserva, bioma nuevo, móvil).

Se ejecuta con tools/e2e/run_isolated.sh.
"""
import json, os, sqlite3, sys, urllib.request
from datetime import datetime, timedelta, timezone
from playwright.sync_api import sync_playwright

if "POMOFARM_URL" not in os.environ:
    sys.exit("Estos tests resetean la BD y no deben tocar tu partida. Ejecútalos con: tools/e2e/run_isolated.sh tools/e2e/p4_03_prestige.py")
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
GROUND = "(() => { let r = null; window.__three.scene.traverse(o => { if (o.isInstancedMesh && o.instanceColor && !r && o.geometry.type === 'BoxGeometry' && o.count >= 36) { const c = new o.instanceColor.constructor.prototype.constructor(0); r = o } }); if (!r) return null; const c = new window.__three.scene.background.constructor(); r.getColorAt(0, c); return c.getHexString() })()"
SKY = "window.__three.scene.background.getHexString()"

call("POST", "/api/pomodoros/active/cancel")
for q in ("DELETE FROM structures", "DELETE FROM unlocks", "DELETE FROM plots WHERE NOT (x=1 AND y=1)", "DELETE FROM pomodoro_events", "DELETE FROM pomodoros", "DELETE FROM tags",
          "UPDATE plots SET state='empty', plant_type=NULL, planted_at=NULL, grow_s=NULL, matured_at=NULL, harvested=0, life_s=NULL, wilts_at=NULL, collected_to=NULL",
          "UPDATE players SET silo_micro=0, silo_peak_micro_h=0, silo_level=0, season=1, biome='spring', coins_milli=0, focus_points=0, rest_started_at=NULL, rest_until=NULL"):
    sql(q)
sql("INSERT INTO settings (player_id,key,value) VALUES (1,'tutorial_done','1') ON CONFLICT(player_id,key) DO UPDATE SET value='1'")

with sync_playwright() as p:
    b = p.chromium.launch(args=["--use-gl=swiftshader", "--enable-unsafe-swiftshader"])
    ctx = b.new_context(viewport={"width": 1100, "height": 760}, timezone_id="UTC")
    pg = ctx.new_page(); errs = []
    pg.on("console", lambda m: errs.append(m.text[:200]) if m.type == "error" and "409" not in m.text else None); pg.on("pageerror", lambda e: errs.append(str(e)[:200]))
    pg.goto(BASE); pg.wait_for_selector(".packet"); pg.wait_for_timeout(800)

    # 1) granja incompleta: requisitos visibles y botón desactivado
    pg.get_by_test_id("shop-button").click(); pg.wait_for_selector("[data-testid=season]")
    txt = pg.inner_text("[data-testid=season]")
    check("la sección ofrece la estación 2 con su bioma y +10%", "Estación 2: Verano dorado" in txt and "+10%" in txt, txt.replace("\n", " | ")[:140])
    check("lista los 4 requisitos con lo que llevas", pg.locator("[data-testid^=season-req-]").count() == 4 and "Parcelas: 1 de 16" in txt and "Silo: 0 de 4" in txt and "Colmenas: 0 de 4" in txt and "Perro Pastor: 0 de 1" in txt)
    check("el botón está desactivado hasta que la granja esté completa", pg.get_by_test_id("start-season").is_disabled())
    check("explica qué se reinicia y qué se conserva", "Empiezan de cero" in pg.inner_text("[data-testid=season]") or pg.locator("details summary").count() == 1)
    pg.keyboard.press("Escape")
    check("el cielo y el césped son los de primavera", pg.evaluate(SKY) == "cfe9f5" and pg.evaluate(GROUND) == "8fc65a", f"{pg.evaluate(SKY)} {pg.evaluate(GROUND)}")

    # 2) granja completa con cosas por reiniciar y por conservar
    t = datetime.now(timezone.utc) - timedelta(hours=1)
    for x in range(4):
        for y in range(4): sql("INSERT OR IGNORE INTO plots (player_id,x,y) VALUES (1,?,?)", (x, y))
    crops = ["daisy", "tomato", "sunflower", "apple", "oak"]
    for i in range(1, 17):
        harvested = 1 if i <= 8 else 0   # 8 plantas listas SIN cosechar
        sql("UPDATE plots SET state='mature', plant_type=?, grow_s=600, life_s=86400, harvested=?, matured_at=?, wilts_at=?, collected_to=? WHERE id=?",
            (crops[i % 5], harvested, iso(t), iso(t + timedelta(hours=24)), iso(t), i))
    sql("UPDATE players SET silo_level=4, coins_milli=75000000, focus_points=40, lifetime_focus=300")
    for k in ("daisy", "tomato", "sunflower"): sql("INSERT INTO unlocks (player_id,kind,key,at) VALUES (1,'seed',?, 't')", (k,))
    for k in ("bees", "dog"): sql("INSERT INTO unlocks (player_id,kind,key,at) VALUES (1,'animal',?, 't')", (k,))
    for x, y in ((1, 1), (2, 1), (1, 2), (2, 2)): sql("INSERT INTO structures (player_id,kind,x,y) VALUES (1,'hive',?,?)", (x, y))
    sql("INSERT INTO structures (player_id,kind,x,y) VALUES (1,'dog',0,0),(1,'hat',0,0),(1,'lantern',4,1),(1,'path',4,2),(1,'fence',5,1)")
    c = sqlite3.connect(DB)
    for d in range(1, 4):
        end = datetime.now(timezone.utc) - timedelta(days=d)
        c.execute("INSERT INTO pomodoros (player_id,plot_id,plant_type,planned_s,started_at,paused_total_s,ended_at,reward_focus,status,strict) VALUES (1,NULL,'daisy',1500,?,0,?,1,'completed',0)", (iso(end - timedelta(minutes=25)), iso(end)))
    c.commit(); c.close()
    before = state(); unharvested = [pl for pl in before["plots"] if not pl["harvested"]]
    pg.reload(); pg.wait_for_selector("[data-testid=silo]"); pg.wait_for_timeout(800)
    pg.get_by_test_id("shop-button").click(); pg.wait_for_selector("[data-testid=season]")
    check("con la granja completa los 4 requisitos se cumplen", all("season__req--met" in (pg.get_attribute(f"[data-testid=season-req-{k}]", "class") or "") for k in ("plots", "silo", "hives", "dog")))
    check("y el botón se activa", pg.get_by_test_id("start-season").is_enabled())
    check("avisa de las plantas listas que se cosecharán solas", f"{len(unharvested)} plantas listas" in pg.inner_text("[data-testid=season]") or pg.locator("[data-testid=season] details").count() == 1)
    pg.screenshot(path=f"{OUT}/season_panel.png")

    # 3) doble confirmación
    pg.get_by_test_id("start-season").click()
    check("el primer toque solo pide confirmar y no cambia nada", "¿Seguro?" in pg.inner_text("[data-testid=start-season]") and state()["player"]["season"] == 1)
    pg.wait_for_timeout(6500)
    check("si no confirmas, se desarma a los 6 s", "¿Seguro?" not in pg.inner_text("[data-testid=start-season]") and state()["player"]["season"] == 1)
    pg.get_by_test_id("start-season").focus(); pg.keyboard.press("Enter"); pg.keyboard.press("Enter")   # con teclado
    pg.wait_for_function("document.querySelector('.toast') && document.querySelector('.toast').innerText.includes('estación 2')", timeout=8000)
    st = state()
    check("la segunda pulsación empieza la estación 2 con bioma de verano", st["player"]["season"] == 2 and st["player"]["biome"] == "summer", f"{st['player']['season']} {st['player']['biome']}")
    check("el aviso dice el bioma y el +10%", "Verano dorado" in pg.inner_text(".toast") and "10%" in pg.inner_text(".toast"), pg.inner_text(".toast"))

    # 4) se reinicia lo que dice el GDD
    check("todas las parcelas quedan vacías (16 se conservan)", len(st["plots"]) == 16 and all(pl["state"] == "empty" and not pl["plant_type"] for pl in st["plots"]))
    check("sin colmenas, Perro, sombrero ni decoración", not st["automation"]["bees"]["hives"] and not st["automation"]["dog"]["owned"] and not st["decor"]["items"] and not st["decor"]["hat"]["owned"])
    check("las 🪙 vuelven a 0", st["player"]["coins_milli"] == 0 and st["silo"]["content_milli"] == 0)
    # 5) se conserva lo demás
    check("el Silo conserva su nivel máximo", st["player"]["silo_level"] == 4)
    check("las semillas y los animales desbloqueados se conservan", sum(1 for s in st["seeds"] if s["unlocked"]) == 3 and st["automation"]["bees"]["unlocked"] and st["automation"]["dog"]["unlocked"])
    check("las 💧 se conservan y las plantas sin cosechar se cosecharon antes", st["player"]["focus_points"] > 40, f"{st['player']['focus_points']} 💧 (antes 40)")
    book = call("GET", "/api/book?tz=UTC")
    check("el Libro de Cosechas conserva sus 3 Pomodoros", sum(w["pomodoros"] for w in call("GET", "/api/stats?tz=UTC")["weeks"]) == 3 and call("GET", "/api/stats?tz=UTC")["total"] == 3, str(call("GET", "/api/stats?tz=UTC")["total"]))

    # 6) el bioma cambia en la escena
    pg.wait_for_timeout(800)
    check("el cielo y el césped pasan a verano", pg.evaluate(SKY) == "fbe6b4" and pg.evaluate(GROUND) == "d9c45c", f"{pg.evaluate(SKY)} {pg.evaluate(GROUND)}")
    pg.screenshot(path=f"{OUT}/season_summer.png")
    pg.reload(); pg.wait_for_selector(".packet, [data-testid=silo]"); pg.wait_for_timeout(800)
    check("tras recargar sigue siendo verano", pg.evaluate(SKY) == "fbe6b4")
    pg.get_by_test_id("shop-button").click(); pg.wait_for_selector("[data-testid=season]")
    txt = pg.inner_text("[data-testid=season]")
    check("la siguiente oferta es la estación 3 (Otoño, +20%) y está desactivada", "Estación 3: Otoño" in txt and "+20%" in txt and pg.get_by_test_id("start-season").is_disabled(), txt.replace("\n", " | ")[:100])
    pg.keyboard.press("Escape")

    # 7) la nueva estación produce +10%
    sql("UPDATE players SET coins_milli=0"); st = state()
    check("el servidor anuncia el bono permanente del 10%", st["prestige"]["bonus_pct"] == 10 and st["prestige"]["next_bonus_pct"] == 20)

    # 8) otras estaciones (solo paleta), y móvil
    for season, biome, sky, grass in ((3, "autumn", "f2d8cc", "c98a45"), (4, "winter", "dce6ee", "eef3f7")):
        sql("UPDATE players SET season=?, biome=?", (season, biome)); pg.reload(); pg.wait_for_selector(".chip"); pg.wait_for_timeout(900)
        check(f"paleta de {biome}", pg.evaluate(SKY) == sky and pg.evaluate(GROUND) == grass, f"{pg.evaluate(SKY)} {pg.evaluate(GROUND)}")
        pg.screenshot(path=f"{OUT}/season_{biome}.png")
    m = ctx.new_page(); m.set_viewport_size({"width": 390, "height": 700}); m.goto(BASE); m.wait_for_selector(".chip")
    m.get_by_test_id("shop-button").click(); m.get_by_test_id("start-season").scroll_into_view_if_needed()
    box = m.get_by_test_id("shop-panel").bounding_box()
    check("móvil: la sección de estaciones cabe en pantalla (con scroll)", box and box["x"] >= 0 and box["x"] + box["width"] <= 390 and box["y"] + box["height"] <= 700 and m.evaluate("document.documentElement.scrollWidth <= window.innerWidth"), str(box))
    m.close()
    check("sin errores de consola", not errs, "; ".join(errs[:3]))
    b.close()
print(f"\n{len(fails)} fallos" if fails else "\nTodo OK"); sys.exit(1 if fails else 0)
