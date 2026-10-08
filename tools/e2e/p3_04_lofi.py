"""E2E P3-04: reproductor Lofi (iframe de YouTube), emisiones configurables, enlace propio y fallback al ambiente local.

La API de YouTube se sustituye por una simulada (route), así que el test no depende de la red ni de que una emisión
siga en directo. Al final hay una prueba informativa contra YouTube real (no falla si no hay red).
Se ejecuta con tools/e2e/run_isolated.sh.
"""
import json, os, subprocess, sys, urllib.request
from urllib.parse import urlparse
from playwright.sync_api import sync_playwright

if "POMOFARM_URL" not in os.environ:
    sys.exit("Estos tests resetean la BD y no deben tocar tu partida. Ejecútalos con: tools/e2e/run_isolated.sh tools/e2e/p3_04_lofi.py")
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
call("POST", "/api/pomodoros/active/cancel")
sql("DELETE FROM structures; DELETE FROM plots WHERE NOT (x=1 AND y=1); UPDATE plots SET state='empty', plant_type=NULL, planted_at=NULL, grow_s=NULL, matured_at=NULL, harvested=0, life_s=NULL, wilts_at=NULL, collected_to=NULL")
sql("INSERT INTO settings (player_id,key,value) VALUES (1,'tutorial_done','1') ON CONFLICT(player_id,key) DO UPDATE SET value='1'")

MOCK_API = """
window.__yt = { creations: [], calls: [], silent: false, current: null, fail: (code) => window.__yt.current && window.__yt.current.opts.events.onError({ data: code }) };
window.YT = { PlayerState: { PLAYING: 1, PAUSED: 2, ENDED: 0, BUFFERING: 3 }, Player: function (el, opts) {
  const iframe = document.createElement('iframe'); iframe.setAttribute('data-mock-yt', opts.videoId); iframe.src = 'about:blank'; el.replaceWith(iframe);
  const me = this; window.__yt.current = me; me.opts = opts; window.__yt.creations.push({ videoId: opts.videoId, host: opts.host, vars: opts.playerVars });
  const fire = (s) => setTimeout(() => opts.events.onStateChange({ data: s }), 20);
  me.playVideo = () => { window.__yt.calls.push(['play']); fire(1) }; me.pauseVideo = () => { window.__yt.calls.push(['pause']); fire(2) };
  me.setVolume = (v) => window.__yt.calls.push(['volume', v]); me.loadVideoById = (id) => { window.__yt.calls.push(['load', id]); iframe.setAttribute('data-mock-yt', id); fire(1) };
  me.destroy = () => { window.__yt.calls.push(['destroy']); iframe.remove() };
  if (!window.__yt.silent) setTimeout(() => opts.events.onReady(), 30);
} };
setTimeout(() => window.onYouTubeIframeAPIReady && window.onYouTubeIframeAPIReady(), 30);
"""
PROBE = """() => { const a = window.__audioEngine; if (!a) return false; if (a.probes) return true;
  a.probes = {}; for (const l of ['ambient']) { const an = a.ctx.createAnalyser(); an.fftSize = 1024; a.buses[l].connect(an);
    const buf = new Float32Array(1024); const st = { max: 0 }; a.probes[l] = st;
    setInterval(() => { an.getFloatTimeDomainData(buf); for (const v of buf) st.max = Math.max(st.max, Math.abs(v)) }, 20) } return true }"""
def ambient_signal(pg, ms=1800):
    pg.wait_for_function("window.__audioEngine && window.__audioEngine.ctx.state === 'running'", timeout=8000); pg.evaluate(PROBE)
    pg.evaluate("window.__audioEngine.probes.ambient.max = 0"); pg.wait_for_timeout(ms); return pg.evaluate("window.__audioEngine.probes.ambient.max")
def status(pg): return pg.get_attribute("[data-testid=lofi-status]", "data-status")
def wait_status(pg, s, t=8000): pg.wait_for_function(f"document.querySelector('[data-testid=lofi-status]') && document.querySelector('[data-testid=lofi-status]').dataset.status === '{s}'", timeout=t)
def calls(pg): return pg.evaluate("window.__yt.calls")
def open_player(pg):
    if pg.locator("[data-testid=lofi-player]").count() == 0: pg.get_by_test_id("music-button").click()
    pg.wait_for_selector("[data-testid=lofi-player]")

with sync_playwright() as p:
    b = p.chromium.launch(args=["--use-gl=swiftshader", "--enable-unsafe-swiftshader", "--autoplay-policy=no-user-gesture-required"])
    ctx = b.new_context(viewport={"width": 1100, "height": 760}); yt_requests = []
    ctx.route("**/iframe_api", lambda r: (yt_requests.append(r.request.url), r.fulfill(status=200, content_type="application/javascript", body=MOCK_API)))
    pg = ctx.new_page(); errs = []
    pg.on("request", lambda r: yt_requests.append(r.url) if "youtube" in (urlparse(r.url).hostname or "") and "iframe_api" not in r.url else None)
    pg.on("console", lambda m: errs.append(m.text[:200]) if m.type == "error" and "409" not in m.text else None); pg.on("pageerror", lambda e: errs.append(str(e)[:200]))
    pg.goto(BASE); pg.wait_for_selector(".packet")

    # 1) privacidad: nada de YouTube hasta pulsar Reproducir
    pg.get_by_test_id("music-button").click(); pg.wait_for_selector("[data-testid=lofi-player]")
    check("el chip abre el reproductor y anuncia su estado", pg.get_attribute("[data-testid=music-button]", "aria-expanded") == "true")
    check("abrir el reproductor no pide nada a YouTube", len(yt_requests) == 0, str(yt_requests))
    check("hay 3 emisiones por defecto y un campo para enlace propio", pg.locator("[data-testid=lofi-station] option").count() == 3 and pg.locator("[data-testid=lofi-link]").count() == 1)
    check("el vídeo está oculto hasta reproducir", not pg.is_visible("[data-testid=lofi-video]"))

    # 2) reproducir
    pg.get_by_test_id("lofi-play").click(); wait_status(pg, "playing")
    cr = pg.evaluate("window.__yt.creations")
    check("se carga la API una sola vez y se crea un reproductor", len(yt_requests) == 1 and len(cr) == 1, str(yt_requests))
    check("usa la emisión elegida, el dominio sin cookies y reproducción en línea", cr[0]["videoId"] == "lTRiuFIWV54" and "nocookie" in cr[0]["host"] and cr[0]["vars"]["playsinline"] == 1, str(cr[0]))
    check("el iframe se ve y arranca con el volumen guardado (50)", pg.is_visible("[data-testid=lofi-video] iframe") and ["volume", 50] in calls(pg), str(calls(pg)))
    check("el botón pasa a 'Pausa'", pg.inner_text("[data-testid=lofi-play]") == "Pausa")

    # 3) pausa, reanudar, volumen propio
    pg.get_by_test_id("lofi-play").click(); wait_status(pg, "paused"); check("pausa", ["pause"] in calls(pg) and pg.inner_text("[data-testid=lofi-play]") == "Reanudar")
    pg.get_by_test_id("lofi-play").click(); wait_status(pg, "playing")
    pg.evaluate("""() => { const el = document.querySelector('[data-testid=lofi-volume]'); Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, 'value').set.call(el, '80'); el.dispatchEvent(new Event('input', { bubbles: true })) }""")
    pg.wait_for_timeout(200)
    check("el volumen propio llega al reproductor (80)", ["volume", 80] in calls(pg))
    check("y no toca las tres capas de audio local", pg.evaluate("Object.values(window.__audioEngine ? window.__audioEngine.volumes : {}).join()") in ("0.5,0.7,0.8", ""), pg.evaluate("JSON.stringify(window.__audioEngine && window.__audioEngine.volumes)"))

    # 3b) REGRESIÓN: cerrar el panel con el vídeo sonando y volver a abrirlo (antes el vídeo desaparecía y no se podía reproducir)
    pg.get_by_test_id("music-button").click(); pg.wait_for_timeout(400)
    check("cerrar el panel no detiene la música: el reproductor y su iframe siguen vivos", pg.locator("iframe[data-mock-yt]").count() == 1 and "destroy" not in [c[0] if isinstance(c, list) else c for c in calls(pg)] and pg.evaluate("document.querySelector('[data-testid=music-button]').getAttribute('aria-expanded')") == "false")
    check("con el panel cerrado el chip sigue avisando de que suena", "chip--attention" in (pg.get_attribute("[data-testid=music-button]", "class") or ""))
    check("y el panel cerrado está fuera de pantalla e inerte (ni se ve, ni se tabula, ni lo leen los lectores de pantalla)", pg.evaluate("document.querySelector('[data-testid=lofi-player]').inert === true") and pg.get_by_test_id("lofi-player").bounding_box()["x"] < -1000)
    pg.get_by_test_id("music-button").click(); pg.wait_for_selector("[data-testid=lofi-player]:visible")
    check("al reabrirlo el vídeo se ve y el estado sigue siendo 'sonando'", pg.is_visible("[data-testid=lofi-video] iframe") and status(pg) == "playing" and pg.inner_text("[data-testid=lofi-play]") == "Pausa")
    pg.get_by_test_id("lofi-play").click(); wait_status(pg, "paused"); pg.get_by_test_id("lofi-play").click(); wait_status(pg, "playing")
    check("tras cerrar y reabrir, Pausa y Reanudar siguen funcionando", len(pg.evaluate("window.__yt.creations")) == 1)
    for _ in range(3):  # varias veces seguidas, como al jugar de verdad
        pg.get_by_test_id("music-button").click(); pg.get_by_test_id("music-button").click()
    pg.wait_for_selector("[data-testid=lofi-player]:visible")
    check("abrir y cerrar varias veces no pierde el vídeo ni crea reproductores nuevos", pg.is_visible("[data-testid=lofi-video] iframe") and len(pg.evaluate("window.__yt.creations")) == 1 and status(pg) == "playing")

    # 4) cambiar de emisión y enlace propio
    pg.select_option("[data-testid=lofi-station]", "4xDzrJKXOOY"); pg.wait_for_timeout(200)
    check("cambiar de emisión carga la nueva sin recrear el reproductor", ["load", "4xDzrJKXOOY"] in calls(pg) and len(pg.evaluate("window.__yt.creations")) == 1)
    pg.get_by_test_id("lofi-link").fill("https://example.com/watch?v=abcdefghijk"); pg.get_by_test_id("lofi-add").click()
    check("un enlace que no es de YouTube se rechaza con un aviso", pg.locator("[role=alert]").count() == 1 and pg.locator("[data-testid=lofi-station] option").count() == 3)
    pg.get_by_test_id("lofi-link").fill("https://youtu.be/abcdefghijk?si=x"); pg.keyboard.press("Enter"); pg.wait_for_timeout(300)
    check("un enlace de YouTube se añade, se selecciona y se reproduce", pg.locator("[data-testid=lofi-station] option").count() == 4 and ["load", "abcdefghijk"] in calls(pg) and pg.input_value("[data-testid=lofi-station]") == "abcdefghijk")
    check("sin aviso de error tras un enlace bueno", pg.locator("[role=alert]").count() == 0)

    # 5) parar cierra el iframe
    pg.get_by_test_id("lofi-stop").click()
    check("Parar destruye el reproductor y cierra el widget", ["destroy"] in calls(pg) and pg.locator("[data-testid=lofi-player]").count() == 0 and pg.locator("iframe[data-mock-yt]").count() == 0)

    # 6) persistencia
    pg.reload(); pg.wait_for_selector(".packet"); open_player(pg)
    check("el enlace propio, la emisión y el volumen persisten", pg.locator("[data-testid=lofi-station] option").count() == 4 and pg.input_value("[data-testid=lofi-station]") == "abcdefghijk" and pg.input_value("[data-testid=lofi-volume]") == "80")
    check("tras recargar no se reproduce solo", status(pg) == "idle" and pg.locator("iframe[data-mock-yt]").count() == 0)
    pg.get_by_test_id("lofi-remove-abcdefghijk").click()
    check("quitar el enlace propio vuelve a la primera emisión", pg.locator("[data-testid=lofi-station] option").count() == 3 and pg.input_value("[data-testid=lofi-station]") == "lTRiuFIWV54")

    # 7) fallo del vídeo (150 = no se puede incrustar) -> ambiente local
    pg.get_by_test_id("lofi-play").click(); wait_status(pg, "playing")
    check("antes del fallo no hay ambiente local", ambient_signal(pg, 600) < 0.0005)
    pg.evaluate("window.__yt.fail(150)"); wait_status(pg, "fallback")
    check("si el vídeo falla, lo dice y se cierra el iframe", "no se puede reproducir" in pg.inner_text("[data-testid=lofi-status]") and pg.locator("iframe[data-mock-yt]").count() == 0)
    check("y suena el ambiente local (lluvia) por la capa de ambiente", ambient_signal(pg) > 0.005 and pg.evaluate("JSON.parse(localStorage.getItem('pomofarm.prefs')).ambient") == "rain")
    check("el botón ofrece Reintentar", pg.inner_text("[data-testid=lofi-play]") == "Reintentar")
    pg.get_by_test_id("lofi-play").click(); wait_status(pg, "playing")
    check("Reintentar vuelve a YouTube y apaga el ambiente de reserva", pg.evaluate("JSON.parse(localStorage.getItem('pomofarm.prefs')).ambient") == "off" and len(pg.evaluate("window.__yt.creations")) == 2)
    pg.evaluate("window.__yt.fail(2)"); wait_status(pg, "fallback")
    pg.get_by_test_id("lofi-stop").click(); pg.wait_for_timeout(1200)
    check("Parar tras un fallo apaga el ambiente que puso el fallback", pg.evaluate("JSON.parse(localStorage.getItem('pomofarm.prefs')).ambient") == "off" and ambient_signal(pg, 1200) < 0.0005)

    # 8) si el jugador ya había elegido un ambiente, el fallback lo respeta
    open_player(pg); pg.select_option("[data-testid=ambient-kind]", "fire")
    pg.get_by_test_id("lofi-play").click(); wait_status(pg, "playing"); pg.evaluate("window.__yt.fail(100)"); wait_status(pg, "fallback"); pg.get_by_test_id("lofi-stop").click(); pg.wait_for_timeout(500)
    check("un ambiente elegido por el jugador no se apaga al parar", pg.evaluate("JSON.parse(localStorage.getItem('pomofarm.prefs')).ambient") == "fire")
    open_player(pg); pg.select_option("[data-testid=ambient-kind]", "off")

    # 9) la API no arranca a tiempo (el vídeo nunca avisa de estar listo)
    open_player(pg); pg.evaluate("window.__yt.silent = true"); pg.get_by_test_id("lofi-play").click(); wait_status(pg, "loading", 3000)
    check("mientras conecta lo dice", "Conectando" in pg.inner_text("[data-testid=lofi-status]"))
    wait_status(pg, "fallback", 14000)
    check("tras 10 s sin respuesta pasa al ambiente local", "No se pudo conectar" in pg.inner_text("[data-testid=lofi-status]") and ambient_signal(pg) > 0.005)
    pg.get_by_test_id("lofi-stop").click(); pg.evaluate("window.__yt.silent = false")

    # 10) móvil
    m = ctx.new_page(); m.set_viewport_size({"width": 390, "height": 700}); m.goto(BASE); m.wait_for_selector(".chip"); m.get_by_test_id("music-button").click(); m.wait_for_selector("[data-testid=lofi-player]")
    m.get_by_test_id("lofi-play").click(); wait_status(m, "playing")
    box = m.get_by_test_id("lofi-player").bounding_box()
    check("móvil: el widget cabe en pantalla", box and box["x"] >= 0 and box["x"] + box["width"] <= 390 and box["y"] + box["height"] <= 700, str(box))
    check("móvil: sin desbordamiento horizontal", m.evaluate("document.documentElement.scrollWidth <= window.innerWidth"))
    m.screenshot(path="/tmp/pomofarm-e2e/lofi_mobile.png"); m.close()
    pg.screenshot(path="/tmp/pomofarm-e2e/lofi_desktop.png")

    # 11) sin red hacia YouTube: la carga de la API falla -> ambiente local
    off = b.new_context(viewport={"width": 1100, "height": 760}); off.route("**/iframe_api", lambda r: r.abort())
    q = off.new_page(); q.goto(BASE); q.wait_for_selector(".packet"); q.get_by_test_id("music-button").click(); q.get_by_test_id("lofi-play").click(); wait_status(q, "fallback")
    check("sin conexión con YouTube: avisa y suena el ambiente local", "No se pudo conectar" in q.inner_text("[data-testid=lofi-status]") and ambient_signal(q) > 0.005)
    off.close()

    check("sin errores de consola", not errs, "; ".join(errs[:3]))
    ctx.close()

    # 12) informativo: YouTube real (no falla el test si no hay red o la emisión no se puede incrustar)
    real = b.new_context(viewport={"width": 1100, "height": 760}); r = real.new_page(); r.goto(BASE); r.wait_for_selector(".packet")
    r.get_by_test_id("music-button").click(); r.get_by_test_id("lofi-play").click()
    try:
        r.wait_for_function("['playing','fallback'].includes(document.querySelector('[data-testid=lofi-status]').dataset.status)", timeout=20000)
        st = status(r)
        if st == "playing":
            check("YouTube real: la emisión por defecto suena en un iframe", r.locator("iframe").count() == 1)
            r.get_by_test_id("music-button").click(); r.wait_for_timeout(4000)
            check("YouTube real: cerrar el panel no la pausa (sigue 'sonando' tras 4 s)", status(r) == "playing" and r.locator("iframe").count() == 1, status(r))
            r.get_by_test_id("music-button").click(); r.wait_for_selector("[data-testid=lofi-player]:visible"); r.wait_for_timeout(500)
            check("YouTube real: al reabrirlo el vídeo se ve y se puede pausar y reanudar", r.is_visible("[data-testid=lofi-video] iframe"))
            r.get_by_test_id("lofi-play").click(); wait_status(r, "paused", 8000); r.get_by_test_id("lofi-play").click(); wait_status(r, "playing", 8000)
            check("YouTube real: pausa y reanuda tras cerrar y reabrir", status(r) == "playing")
        else: print("INFO YouTube real:", st, "-", r.inner_text("[data-testid=lofi-status]"), "| iframes:", r.locator("iframe").count(), "(¿sin red o emisión retirada?)")
    except Exception as e:
        print("INFO YouTube real: sin resultado en 20 s", str(e)[:80])
    real.close(); b.close()
print(f"\n{len(fails)} fallos" if fails else "\nTodo OK"); sys.exit(1 if fails else 0)
