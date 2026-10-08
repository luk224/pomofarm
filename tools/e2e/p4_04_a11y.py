"""E2E P4-04: accesibilidad (reducir animaciones, sin audio con avisos visuales, paleta para daltonismo, planta marchita sin depender
del color, atajo M). La vista 2D ligera se descartó (decisión del usuario).

Se ejecuta con tools/e2e/run_isolated.sh.
"""
import json, os, sqlite3, sys, urllib.request
from datetime import datetime, timedelta, timezone
from playwright.sync_api import sync_playwright

if "POMOFARM_URL" not in os.environ:
    sys.exit("Estos tests resetean la BD y no deben tocar tu partida. Ejecútalos con: tools/e2e/run_isolated.sh tools/e2e/p4_04_a11y.py")
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

call("POST", "/api/pomodoros/active/cancel")
for q in ("DELETE FROM structures", "DELETE FROM unlocks WHERE kind='animal'", "DELETE FROM plots WHERE NOT (x=1 AND y=1)", "DELETE FROM pomodoro_events", "DELETE FROM pomodoros", "DELETE FROM tags",
          "UPDATE plots SET state='empty', plant_type=NULL, planted_at=NULL, grow_s=NULL, matured_at=NULL, harvested=0, life_s=NULL, wilts_at=NULL, collected_to=NULL",
          "UPDATE players SET silo_micro=0, silo_peak_micro_h=0, silo_level=0, season=1, biome='spring', coins_milli=0, focus_points=0, rest_started_at=NULL, rest_until=NULL"):
    sql(q)
sql("INSERT INTO settings (player_id,key,value) VALUES (1,'tutorial_done','1') ON CONFLICT(player_id,key) DO UPDATE SET value='1'")
sql("INSERT INTO settings (player_id,key,value) VALUES (1,'rest_enabled','0') ON CONFLICT(player_id,key) DO UPDATE SET value='0'")

PROBE_MASTER = """() => { const a = window.__audioEngine; if (!a || !a.master) return false; if (a.mprobe) return true;
  const an = a.ctx.createAnalyser(); an.fftSize = 1024; a.master.connect(an); const buf = new Float32Array(1024); a.mprobe = { max: 0 };
  setInterval(() => { an.getFloatTimeDomainData(buf); for (const v of buf) a.mprobe.max = Math.max(a.mprobe.max, Math.abs(v)) }, 20); return true }"""
def master_max(pg, ms=1500):
    pg.wait_for_function("window.__audioEngine && window.__audioEngine.ctx.state === 'running' && window.__audioEngine.master", timeout=8000); pg.evaluate(PROBE_MASTER)
    pg.evaluate("window.__audioEngine.mprobe.max = 0"); pg.wait_for_timeout(ms); return pg.evaluate("window.__audioEngine.mprobe.max")
def open_settings(pg):
    if pg.locator("[data-testid=a11y-settings]").count() == 0: pg.get_by_role("button", name="Ajustes de avisos").click()
    pg.wait_for_selector("[data-testid=a11y-settings]")
BODY_ROT = "(() => { const g = window.__three.scene.getObjectByName('plant-%s'); return g ? [g.children[0].rotation.z, g.children[0].scale.y] : null })()"
MOCK_API = """
window.__yt = { calls: [] };
window.YT = { PlayerState: { PLAYING: 1, PAUSED: 2 }, Player: function (el, opts) { const f = document.createElement('iframe'); f.src = 'about:blank'; el.replaceWith(f); const me = this;
  me.playVideo = () => setTimeout(() => opts.events.onStateChange({ data: 1 }), 20); me.pauseVideo = () => {}; me.setVolume = () => {}; me.loadVideoById = () => {}; me.destroy = () => { window.__yt.calls.push('destroy'); f.remove() };
  setTimeout(() => opts.events.onReady(), 30) } };
setTimeout(() => window.onYouTubeIframeAPIReady && window.onYouTubeIframeAPIReady(), 30);
"""

with sync_playwright() as p:
    b = p.chromium.launch(args=["--use-gl=swiftshader", "--enable-unsafe-swiftshader", "--autoplay-policy=no-user-gesture-required"])
    ctx = b.new_context(viewport={"width": 1100, "height": 760})
    ctx.route("**/iframe_api", lambda r: r.fulfill(status=200, content_type="application/javascript", body=MOCK_API))
    pg = ctx.new_page(); errs = []
    pg.on("console", lambda m: errs.append(m.text[:200]) if m.type == "error" and "409" not in m.text else None); pg.on("pageerror", lambda e: errs.append(str(e)[:200]))
    pg.goto(BASE); pg.wait_for_selector(".packet")

    # 1) ajustes y valores por defecto
    open_settings(pg)
    check("hay tres ajustes de accesibilidad, todos apagados por defecto", all(not pg.get_by_test_id(t).is_checked() for t in ("a11y-muted", "a11y-motion", "a11y-palette")))
    check("cada uno explica lo que hace", all(w in pg.inner_text("[data-testid=a11y-settings]") for w in ("Silencia todo", "balanceo", "Azul y naranja")))
    check("el atajo M aparece en la lista de atajos", "Silenciar todo" in pg.inner_text("[data-testid=shortcuts]"))
    check("sin cambios: el html no tiene la paleta ni el modo reducido", pg.evaluate("[document.documentElement.dataset.palette, document.documentElement.dataset.reduceMotion]") == ["default", "false"])
    pg.keyboard.press("Escape")

    # 2) paleta para daltonismo
    t = datetime.now(timezone.utc) - timedelta(hours=1)
    sql("INSERT OR IGNORE INTO plots (player_id,x,y) VALUES (1,2,1),(1,0,1)")
    sql("UPDATE plots SET state='mature', plant_type='daisy', grow_s=600, life_s=86400, harvested=1, matured_at=?, wilts_at=?, collected_to=? WHERE x=1 AND y=1", (iso(t), iso(t + timedelta(hours=24)), iso(t)))
    sql("UPDATE plots SET state='mature', plant_type='tomato', grow_s=1500, life_s=129600, harvested=1, matured_at=?, wilts_at=?, collected_to=? WHERE x=2 AND y=1", (iso(t), iso(t + timedelta(hours=36)), iso(t)))
    sql("UPDATE plots SET state='withered', plant_type='daisy', grow_s=600, life_s=3600, harvested=1, matured_at=?, wilts_at=?, collected_to=? WHERE x=0 AND y=1", (iso(t - timedelta(hours=5)), iso(t - timedelta(hours=4)), iso(t - timedelta(hours=4))))
    pg.reload(); pg.wait_for_selector(".chip"); pg.wait_for_timeout(1500)
    BADGE_PX = """() => { let px = null; window.__three.scene.traverse(o => { if (o.name === 'bonus-badge' && !px) { const c = o.material.map.userData.canvas; const d = c.getContext('2d').getImageData(14, 32, 1, 1).data; px = [d[0], d[1], d[2]] } }); return px }"""
    px = pg.evaluate(BADGE_PX)
    check("paleta normal: el distintivo de bono es verde", px is not None and px[1] > px[0] and px[1] > px[2], str(px))
    open_settings(pg); pg.get_by_test_id("a11y-palette").check(); pg.wait_for_timeout(500)
    check("el html pasa a la paleta para daltonismo", pg.evaluate("document.documentElement.dataset.palette") == "cb")
    btn_bg = pg.evaluate("getComputedStyle(document.querySelector('.btn--primary') || document.body).backgroundColor")
    check("los botones principales pasan de verde a azul", btn_bg in ("rgb(0, 114, 178)", "rgba(0, 114, 178, 1)") or pg.evaluate("getComputedStyle(document.documentElement).getPropertyValue('--leaf').trim()") == "#0072b2", btn_bg)
    pg.keyboard.press("Escape"); pg.wait_for_timeout(1200)
    px2 = pg.evaluate(BADGE_PX)
    check("el distintivo de bono pasa a azul (sin verde ni rojo)", px2 is not None and px2[2] > px2[0] and px2[2] > px2[1], str(px2))
    pg.screenshot(path=f"{OUT}/a11y_palette_cb.png")
    pg.reload(); pg.wait_for_selector(".chip")
    check("la paleta se recuerda tras recargar", pg.evaluate("document.documentElement.dataset.palette") == "cb")
    # anillo del temporizador
    sql("UPDATE plots SET state='empty', plant_type=NULL, harvested=0, matured_at=NULL, wilts_at=NULL, collected_to=NULL, grow_s=NULL, life_s=NULL WHERE x=0 AND y=1"); pg.reload(); pg.wait_for_selector(".packet")
    pg.locator(".packet__body:not([disabled])").first.click(); pg.get_by_role("button", name="Plantar", exact=True).click(); pg.wait_for_selector("[data-testid=timer][data-status=running]"); pg.wait_for_timeout(600)
    ring = pg.evaluate("document.querySelectorAll('[data-testid=timer-ring] circle')[1].getAttribute('stroke')")
    check("el anillo del temporizador es azul en esta paleta", ring.lower() == "#0072b2", ring)
    call("POST", "/api/pomodoros/active/cancel")
    open_settings(pg); pg.get_by_test_id("a11y-palette").uncheck(); pg.keyboard.press("Escape")
    check("y vuelve a la normal al desmarcar", pg.evaluate("document.documentElement.dataset.palette") == "default")

    # 3) planta marchita: distinta sin depender del color (más baja e inclinada)
    sql("UPDATE plots SET state='withered', plant_type='daisy', grow_s=600, life_s=3600, harvested=1, matured_at=?, wilts_at=?, collected_to=? WHERE x=0 AND y=1", (iso(t - timedelta(hours=5)), iso(t - timedelta(hours=4)), iso(t - timedelta(hours=4))))
    pg.reload(); pg.wait_for_selector(".chip"); pg.wait_for_timeout(1500)
    info = pg.evaluate("""() => { const out = []; window.__three.scene.traverse(o => { if (o.name === 'plant-daisy') out.push([o.parent.parent.position.x, o.children[0].rotation.z, o.children[0].scale.y]) }); return out }""")
    alive = [i for i in info if i[0] == 1]; dead = [i for i in info if i[0] == 0]
    check("la marchita está más baja y claramente inclinada, la viva no", bool(alive and dead) and dead[0][2] < alive[0][2] * 0.9 and abs(dead[0][1]) > 0.15 and abs(alive[0][1]) < 0.1, f"viva {alive} marchita {dead}")
    pg.screenshot(path=f"{OUT}/a11y_withered.png")

    # 4) reducir animaciones
    swing = lambda: pg.evaluate("(() => { const g = window.__three.scene.getObjectByName('plant-daisy'); return g.children[0].rotation.z })()")
    # la margarita viva está en (1,1): leer su balanceo en dos instantes
    def alive_rot():
        return pg.evaluate("""() => { let r = null; window.__three.scene.traverse(o => { if (o.name === 'plant-daisy' && o.parent.parent.position.x === 1) r = o.children[0].rotation.z }); return r }""")
    samples = []
    for _ in range(6): samples.append(alive_rot()); pg.wait_for_timeout(260)
    check("con las animaciones normales la planta viva se balancea", max(samples) - min(samples) > 0.004, f"{min(samples):.4f}..{max(samples):.4f}")
    open_settings(pg); pg.get_by_test_id("a11y-motion").check(); pg.keyboard.press("Escape"); pg.wait_for_timeout(500)
    check("el html lo refleja", pg.evaluate("document.documentElement.dataset.reduceMotion") == "true")
    samples = []
    for _ in range(6): samples.append(alive_rot()); pg.wait_for_timeout(260)
    check("con 'Reducir animaciones' la planta está quieta", max(samples) - min(samples) < 1e-6, f"{min(samples):.4f}..{max(samples):.4f}")
    pg.keyboard.press("m"); pg.wait_for_selector(".toast"); pg.keyboard.press("m")
    anim = pg.evaluate("getComputedStyle(document.querySelector('.toast')).animationName")
    check("y los avisos de la interfaz no se animan", anim == "none", anim)
    pg.reload(); pg.wait_for_selector(".chip")
    check("se recuerda tras recargar", pg.evaluate("document.documentElement.dataset.reduceMotion") == "true")
    open_settings(pg); pg.get_by_test_id("a11y-motion").uncheck(); pg.keyboard.press("Escape")

    # 5) sin audio: silencia todas las capas y la música
    open_settings(pg); pg.select_option("[data-testid=ambient-kind]", "rain"); pg.keyboard.press("Escape")
    loud = master_max(pg)
    check("antes de silenciar, el ambiente suena", loud > 0.01, f"{loud:.4f}")
    open_settings(pg); pg.get_by_test_id("a11y-muted").check(); pg.keyboard.press("Escape"); pg.wait_for_timeout(700)
    check("'Sin audio' deja la salida en silencio aunque haya ambiente", master_max(pg) < 0.0005, f"{master_max(pg):.5f}")
    check("el control maestro llega a 0 y el html no cambia de estado raro", pg.evaluate("window.__audioEngine.master.gain.value") < 0.001)
    # la música
    pg.get_by_test_id("music-button").click()
    check("con 'Sin audio' no se puede reproducir música y se explica", pg.get_by_test_id("lofi-play").is_disabled() and "Sin audio activado" in pg.inner_text("[data-testid=lofi-muted]"))
    pg.get_by_test_id("lofi-stop").click()
    pg.keyboard.press("m"); pg.wait_for_timeout(600)
    check("la tecla M quita el silencio y avisa", "Audio activado" in pg.inner_text(".toasts") and master_max(pg) > 0.01)
    pg.get_by_test_id("music-button").click(); pg.get_by_test_id("lofi-play").click(); pg.wait_for_function("document.querySelector('[data-testid=lofi-status]').dataset.status === 'playing'", timeout=8000)
    pg.keyboard.press("m"); pg.wait_for_timeout(700)
    st = pg.get_attribute("[data-testid=lofi-status]", "data-status") if pg.locator("[data-testid=lofi-status]").count() else "cerrado"
    check("silenciar con la música sonando la detiene", st == "idle" and "destroy" in pg.evaluate("window.__yt.calls"), st)
    pg.keyboard.press("m"); pg.wait_for_timeout(400)
    pg.get_by_test_id("lofi-stop").click() if pg.locator("[data-testid=lofi-stop]").count() else None
    open_settings(pg); pg.select_option("[data-testid=ambient-kind]", "off"); pg.keyboard.press("Escape")

    # 6) aviso visual al terminar con el audio apagado
    call("POST", "/api/plots/1/clear", {"confirm": True}); pg.reload(); pg.wait_for_selector(".packet")
    open_settings(pg); pg.get_by_test_id("a11y-muted").check(); pg.keyboard.press("Escape")
    pg.locator(".packet__body:not([disabled])").first.click(); pg.get_by_role("button", name="Plantar", exact=True).click(); pg.wait_for_selector("[data-testid=timer][data-status=running]")
    sql("UPDATE pomodoros SET started_at=strftime('%Y-%m-%dT%H:%M:%fZ','now','-597 seconds') WHERE status='running'"); pg.evaluate("window.dispatchEvent(new Event('focus'))")
    pg.wait_for_selector("[data-testid=alert-flash-label]", timeout=15000)
    check("sin audio, al terminar sale un marco dorado y un aviso con texto", "lista" in pg.inner_text("[data-testid=alert-flash-label]") and pg.locator("[data-testid=alert-flash]").count() == 1, pg.inner_text("[data-testid=alert-flash-label]"))
    check("el aviso es un 'alert' para lectores de pantalla", pg.get_attribute("[data-testid=alert-flash-label]", "role") == "alert")
    pg.screenshot(path=f"{OUT}/a11y_flash.png")
    pg.wait_for_function("document.querySelector('[data-testid=alert-flash-label]') === null", timeout=6000)
    check("el aviso desaparece solo a los pocos segundos", True)
    # y con audio, no hay marco
    call("POST", "/api/plots/1/harvest"); call("POST", "/api/plots/1/clear", {"confirm": True}); pg.reload(); pg.wait_for_selector(".packet")
    open_settings(pg); pg.get_by_test_id("a11y-muted").uncheck(); pg.keyboard.press("Escape")
    pg.locator(".packet__body:not([disabled])").first.click(); pg.get_by_role("button", name="Plantar", exact=True).click(); pg.wait_for_selector("[data-testid=timer][data-status=running]")
    sql("UPDATE pomodoros SET started_at=strftime('%Y-%m-%dT%H:%M:%fZ','now','-597 seconds') WHERE status='running'"); pg.evaluate("window.dispatchEvent(new Event('focus'))")
    pg.wait_for_selector(".toast", timeout=15000); pg.wait_for_timeout(800)
    check("con audio y campana activos no hay marco (basta el sonido)", pg.locator("[data-testid=alert-flash]").count() == 0)

    # 7) M no actúa mientras se escribe
    call("POST", "/api/plots/1/harvest"); call("POST", "/api/plots/1/clear", {"confirm": True}); pg.reload(); pg.wait_for_selector(".packet")
    before = pg.evaluate("JSON.parse(localStorage.getItem('pomofarm.prefs') || '{}').muted === true")
    pg.get_by_placeholder("Etiqueta opcional").fill("mates"); pg.get_by_placeholder("Etiqueta opcional").press("m")
    check("la tecla M no silencia mientras escribes en un campo", pg.evaluate("JSON.parse(localStorage.getItem('pomofarm.prefs') || '{}').muted === true") == before and pg.input_value("[placeholder='Etiqueta opcional']").endswith("m"))

    # 8) teclado: los ajustes se alcanzan con Tab y se ve el foco
    open_settings(pg)
    pg.get_by_label("Campana al terminar").focus()
    for _ in range(6):  # con el teclado de verdad, para que cuente como foco visible
        pg.keyboard.press("Tab")
        if pg.evaluate("document.activeElement.dataset.testid") == "a11y-muted": break
    check("Tab llega a 'Sin audio'", pg.evaluate("document.activeElement.dataset.testid") == "a11y-muted")
    outline = pg.evaluate("(() => { const s = getComputedStyle(document.activeElement); return [s.outlineStyle, s.outlineWidth, s.boxShadow] })()")
    check("el foco del teclado es visible en los ajustes nuevos", outline[0] != "none" or outline[2] != "none", str(outline))
    pg.keyboard.press("Space"); pg.wait_for_timeout(200)
    check("se activa con la barra espaciadora", pg.get_by_test_id("a11y-muted").is_checked())
    pg.keyboard.press("Space")
    pg.keyboard.press("Escape")

    # 9) móvil
    m = ctx.new_page(); m.set_viewport_size({"width": 390, "height": 700}); m.goto(BASE); m.wait_for_selector(".chip"); m.get_by_role("button", name="Ajustes de avisos").click(); m.wait_for_selector("[data-testid=a11y-settings]")
    box = m.get_by_test_id("a11y-settings").bounding_box(); panel = m.locator("#settings-panel").bounding_box()
    check("móvil: el panel de ajustes cabe en pantalla y se desplaza", panel and panel["x"] >= 0 and panel["x"] + panel["width"] <= 390 and panel["y"] + panel["height"] <= 700 and m.evaluate("document.documentElement.scrollWidth <= window.innerWidth"), str(panel))
    m.close()
    check("sin errores de consola", not errs, "; ".join(errs[:3]))
    b.close()
print(f"\n{len(fails)} fallos" if fails else "\nTodo OK"); sys.exit(1 if fails else 0)
