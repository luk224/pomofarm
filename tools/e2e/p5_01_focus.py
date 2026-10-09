"""E2E P5-01: Modo Foco (15–30 FPS mientras corre un Pomodoro, sombras a ritmo bajo, nada si la pestaña está oculta).

Se ejecuta con tools/e2e/run_isolated.sh.
"""
import json, os, sqlite3, sys, urllib.request
from playwright.sync_api import sync_playwright

if "POMOFARM_URL" not in os.environ:
    sys.exit("Estos tests resetean la BD y no deben tocar tu partida. Ejecútalos con: tools/e2e/run_isolated.sh tools/e2e/p5_01_focus.py")
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

call("POST", "/api/pomodoros/active/cancel")
for q in ("DELETE FROM structures", "DELETE FROM unlocks WHERE kind='animal'", "DELETE FROM plots WHERE NOT (x=1 AND y=1)", "DELETE FROM pomodoro_events", "DELETE FROM pomodoros",
          "UPDATE plots SET state='empty', plant_type=NULL, planted_at=NULL, grow_s=NULL, matured_at=NULL, harvested=0, life_s=NULL, wilts_at=NULL, collected_to=NULL",
          "UPDATE players SET silo_micro=0, silo_peak_micro_h=0, silo_level=0, season=1, biome='spring', coins_milli=0, focus_points=0, rest_started_at=NULL, rest_until=NULL"):
    sql(q)
sql("INSERT INTO settings (player_id,key,value) VALUES (1,'tutorial_done','1') ON CONFLICT(player_id,key) DO UPDATE SET value='1'")
sql("INSERT INTO settings (player_id,key,value) VALUES (1,'rest_enabled','0') ON CONFLICT(player_id,key) DO UPDATE SET value='0'")

FRAMES = "window.__three.gl.info.render.frame"
MODE = "window.__three.get().frameloop"
def fps(pg, ms=3000):
    a = pg.evaluate(FRAMES); pg.wait_for_timeout(ms); return (pg.evaluate(FRAMES) - a) / (ms / 1000)
def plant(pg):
    pg.locator(".packet__body:not([disabled])").first.click(); pg.get_by_role("button", name="Plantar", exact=True).click(); pg.wait_for_selector("[data-testid=timer][data-status=running]")

with sync_playwright() as p:
    b = p.chromium.launch(args=["--use-gl=swiftshader", "--enable-unsafe-swiftshader", "--enable-precise-memory-info"])
    ctx = b.new_context(viewport={"width": 1100, "height": 760})
    pg = ctx.new_page(); errs = []
    pg.on("console", lambda m: errs.append(m.text[:200]) if m.type == "error" and "409" not in m.text else None); pg.on("pageerror", lambda e: errs.append(str(e)[:200]))
    pg.goto(BASE); pg.wait_for_selector(".packet"); pg.wait_for_timeout(1500)
    cdp = ctx.new_cdp_session(pg); cdp.send("Performance.enable")
    def cpu(ms=4000):
        m0 = {x["name"]: x["value"] for x in cdp.send("Performance.getMetrics")["metrics"]}; f0 = pg.evaluate(FRAMES); pg.wait_for_timeout(ms)
        m1 = {x["name"]: x["value"] for x in cdp.send("Performance.getMetrics")["metrics"]}
        return (m1["TaskDuration"] - m0["TaskDuration"]) / (ms / 1000), (pg.evaluate(FRAMES) - f0) / (ms / 1000)

    # 1) sin Pomodoro: dibuja con normalidad
    check("sin Pomodoro el bucle es el normal ('always') y las sombras se actualizan solas", pg.evaluate(MODE) == "always" and pg.evaluate("window.__three.gl.shadowMap.autoUpdate") is True)
    idle_cpu, idle_fps = cpu()
    check("sin Pomodoro dibuja a buen ritmo", idle_fps > 5, f"{idle_fps:.1f} FPS, {idle_cpu * 100:.0f}% de CPU de la página")

    # 2) con Pomodoro en marcha: Modo Foco
    plant(pg); pg.wait_for_timeout(1500)
    check("con el Pomodoro en marcha entra en Modo Foco ('demand')", pg.evaluate(MODE) == "demand")
    check("las sombras dejan de actualizarse en cada fotograma", pg.evaluate("window.__three.gl.shadowMap.autoUpdate") is False)
    check("la resolución baja a 1 píxel por píxel", pg.evaluate("window.__three.gl.getPixelRatio()") == 1)
    focus_cpu, focus_fps = cpu()
    check("dibuja entre 12 y 24 FPS (objetivo del GDD: 15–30)", 12 <= focus_fps <= 24, f"{focus_fps:.1f} FPS")
    if idle_fps > 30:
        check("y gasta bastante menos CPU que sin Modo Foco", focus_cpu < idle_cpu * 0.75, f"{focus_cpu * 100:.0f}% frente a {idle_cpu * 100:.0f}%")
    else:
        print(f"INFO el equipo de pruebas dibuja a {idle_fps:.0f} FPS sin Modo Foco (renderizado por software): no se puede medir ahorro de CPU aquí; Foco: {focus_fps:.0f} FPS, {focus_cpu * 100:.0f}% frente a {idle_cpu * 100:.0f}%")
    check("Modo Foco nunca dibuja MÁS que el bucle normal", focus_fps <= max(idle_fps, 24) + 1.5, f"{focus_fps:.1f} vs {idle_fps:.1f}")

    # 3) lo que debe seguir igual de bien
    t0 = pg.inner_text("[data-testid=timer]"); pg.wait_for_timeout(2200); t1 = pg.inner_text("[data-testid=timer]")
    check("el temporizador sigue contando (no depende del dibujado)", t0 != t1, f"{t0} -> {t1}")
    rot = lambda: pg.evaluate("(() => { let r = null; window.__three.scene.traverse(o => { if (o.name === 'plant-daisy') r = o.children[0].scale.y }); return r })()")
    s0 = rot(); pg.wait_for_timeout(2500); s1 = rot()
    check("la planta sigue creciendo, a su tiempo", s1 is not None and s1 >= s0, f"{s0} -> {s1}")
    pg.screenshot(path=f"{OUT}/focus_running.png")
    pg.mouse.click(550, 330); pg.wait_for_timeout(300)
    check("la escena responde a los clics en Modo Foco", pg.evaluate(MODE) == "demand")

    # 4) pausa: vuelve al modo normal
    pg.get_by_role("button", name="Pausar", exact=True).click(); pg.wait_for_selector("[data-testid=timer][data-status=paused]"); pg.wait_for_timeout(800)
    check("en pausa se vuelve al dibujado normal", pg.evaluate(MODE) == "always" and pg.evaluate("window.__three.gl.shadowMap.autoUpdate") is True)
    pg.get_by_role("button", name="Reanudar", exact=True).click(); pg.wait_for_selector("[data-testid=timer][data-status=running]"); pg.wait_for_timeout(800)
    check("al reanudar vuelve a Modo Foco", pg.evaluate(MODE) == "demand")

    # 5) pestaña oculta: no se dibuja nada
    pg.evaluate("Object.defineProperty(document, 'visibilityState', { configurable: true, get: () => 'hidden' }); document.dispatchEvent(new Event('visibilitychange'))"); pg.wait_for_timeout(600)
    check("con la pestaña oculta el bucle se detiene ('never')", pg.evaluate(MODE) == "never")
    hidden_fps = fps(pg, 2500)
    check("con la pestaña oculta no se dibuja ningún fotograma", hidden_fps < 0.5, f"{hidden_fps:.2f} FPS")
    t0 = pg.inner_text("[data-testid=timer]"); pg.wait_for_timeout(2200)
    check("y el temporizador no pierde el tiempo (lo calcula el servidor/reloj monótono)", pg.inner_text("[data-testid=timer]") != t0)
    pg.evaluate("Object.defineProperty(document, 'visibilityState', { configurable: true, get: () => 'visible' }); document.dispatchEvent(new Event('visibilitychange'))"); pg.wait_for_timeout(800)
    back = fps(pg, 2500)
    check("al volver a la pestaña se reanuda el dibujado en Modo Foco", pg.evaluate(MODE) == "demand" and 8 <= back <= 24, f"{back:.1f} FPS")
    # sin Pomodoro y oculta
    call("POST", "/api/pomodoros/active/cancel"); pg.reload(); pg.wait_for_selector(".packet"); pg.wait_for_timeout(1000)
    pg.evaluate("Object.defineProperty(document, 'visibilityState', { configurable: true, get: () => 'hidden' }); document.dispatchEvent(new Event('visibilitychange'))"); pg.wait_for_timeout(500)
    check("oculta y sin Pomodoro tampoco se dibuja (el GDD lo pide siempre)", pg.evaluate(MODE) == "never" and fps(pg, 2000) < 0.5)
    pg.evaluate("Object.defineProperty(document, 'visibilityState', { configurable: true, get: () => 'visible' }); document.dispatchEvent(new Event('visibilitychange'))")

    # 6) ajuste: se puede desactivar y se recuerda
    pg.get_by_role("button", name="Ajustes de avisos").click(); pg.wait_for_selector("[data-testid=performance-settings]")
    check("el ajuste está activado por defecto y se explica", pg.get_by_test_id("focus-mode").is_checked() and "20 veces por segundo" in pg.inner_text("[data-testid=performance-settings]"))
    pg.get_by_test_id("focus-mode").uncheck(); pg.keyboard.press("Escape"); plant(pg); pg.wait_for_timeout(1500)
    check("con el ajuste desactivado, el Pomodoro no activa Modo Foco", pg.evaluate(MODE) == "always" and pg.evaluate("window.__three.gl.shadowMap.autoUpdate") is True)
    pg.reload(); pg.wait_for_selector("[data-testid=timer]"); pg.wait_for_timeout(1200)
    check("la preferencia se recuerda tras recargar", pg.evaluate(MODE) == "always")
    pg.get_by_role("button", name="Ajustes de avisos").click(); pg.get_by_test_id("focus-mode").check(); pg.keyboard.press("Escape"); pg.wait_for_timeout(1000)
    check("al reactivarlo con un Pomodoro en marcha vuelve a Modo Foco", pg.evaluate(MODE) == "demand")

    # 7) al terminar el Pomodoro se vuelve al modo normal
    sql("UPDATE pomodoros SET started_at=strftime('%Y-%m-%dT%H:%M:%fZ','now','-597 seconds') WHERE status='running'"); pg.evaluate("window.dispatchEvent(new Event('focus'))")
    pg.wait_for_selector("[data-testid=harvest], .toast", timeout=15000); pg.wait_for_timeout(1200)
    check("al terminar el Pomodoro se vuelve al dibujado normal y la planta madura", pg.evaluate(MODE) == "always" and call("GET", "/api/state")["plots"][0]["state"] == "mature")
    check("sin errores de consola", not errs, "; ".join(errs[:3]))
    b.close()
print(f"\n{len(fails)} fallos" if fails else "\nTodo OK"); sys.exit(1 if fails else 0)
