"""E2E: la ventana de música en el móvil. Sonido de ambiente (permisos estrictos de audio de Chrome y Safari en teléfonos), la ventana cabe
sin hacer scroll aunque suene el vídeo, el vídeo es pequeño y no se ofrece un control de volumen que el móvil ignora.

Se ejecuta con tools/e2e/run_isolated.sh.
"""
import json, os, sqlite3, sys, urllib.request
from playwright.sync_api import sync_playwright

if "POMOFARM_URL" not in os.environ:
    sys.exit("Estos tests resetean la BD y no deben tocar tu partida. Ejecútalos con: tools/e2e/run_isolated.sh tools/e2e/p5_07_mobile_music.py")
BASE = os.environ["POMOFARM_URL"]; DB = os.environ["POMOFARM_DB"]
OUT = "/tmp/pomofarm-e2e"; os.makedirs(OUT, exist_ok=True)
fails = []
def check(name, ok, detail=""):
    print(("PASS " if ok else "FAIL ") + name + (f"  [{detail}]" if detail else ""))
    if not ok: fails.append(name)
def sql(q, args=()):
    c = sqlite3.connect(DB, timeout=10); c.execute(q, args); c.commit(); c.close()
sql("INSERT INTO settings (player_id,key,value) VALUES (1,'tutorial_done','1') ON CONFLICT(player_id,key) DO UPDATE SET value='1'")

MOCK_API = """
window.YT = { PlayerState: { PLAYING: 1, PAUSED: 2 }, Player: function (el, opts) { const f = document.createElement('iframe'); f.src = 'about:blank'; el.replaceWith(f); const me = this;
  me.playVideo = () => setTimeout(() => opts.events.onStateChange({ data: 1 }), 20); me.pauseVideo = () => {}; me.setVolume = () => {}; me.loadVideoById = () => {}; me.destroy = () => f.remove();
  setTimeout(() => opts.events.onReady(), 30) } };
setTimeout(() => window.onYouTubeIframeAPIReady && window.onYouTubeIframeAPIReady(), 30);
"""
# Un teléfono estricto: los contextos de audio nacen suspendidos y resume() solo funciona dentro de un toque que TERMINA (touchend/click)
# o de una tecla; pointerdown no vale. Así se comportan Chrome en Android y Safari en iPhone.
STRICT = """
window.__g = false; window.__blockedResume = 0; window.__resumes = 0;
const mark = () => { window.__g = true; setTimeout(() => { window.__g = false }, 0) };
for (const e of ['touchend', 'click', 'keydown', 'mouseup']) window.addEventListener(e, mark, true);
const AC = window.AudioContext, orig = AC.prototype.resume;
AC.prototype.resume = function () { window.__resumes++; if (!window.__g) { window.__blockedResume++; return new Promise(() => {}) } return orig.call(this) };
window.AudioContext = class extends AC { constructor() { super(); this.suspend() } };
Object.defineProperty(navigator, 'audioSession', { value: { type: 'auto' }, configurable: true });
"""
PROBE = """() => { const a = window.__audioEngine; if (!a) return false; if (a.probes) return true;
  const an = a.ctx.createAnalyser(); an.fftSize = 1024; a.buses.ambient.connect(an); const buf = new Float32Array(1024); a.probes = { max: 0 };
  setInterval(() => { an.getFloatTimeDomainData(buf); for (const v of buf) a.probes.max = Math.max(a.probes.max, Math.abs(v)) }, 20); return true }"""

def open_music(pg):
    pg.get_by_test_id("music-button").tap(); pg.wait_for_selector("[data-testid=lofi-player]"); pg.wait_for_timeout(400)
def fits(pg, label, w, h):
    box = pg.get_by_test_id("lofi-player").bounding_box()
    scroll = pg.evaluate("(() => { const e = document.querySelector('[data-testid=lofi-player]'); return [e.scrollHeight, e.clientHeight] })()")
    check(f"{label}: la ventana de música cabe en pantalla sin scroll mientras suena el vídeo", box["y"] + box["height"] <= h - 6 and scroll[0] <= scroll[1] + 1, f"{w}x{h}: ocupa {box['height']:.0f} px de alto (de {box['y']:.0f} a {box['y'] + box['height']:.0f}); contenido {scroll[0]} / visible {scroll[1]}")
    check(f"{label}: la ventana no se sale de la pantalla por los lados", box["x"] >= 0 and box["x"] + box["width"] <= w, f"{box['x']:.0f}..{box['x'] + box['width']:.0f} de {w}")
    over = pg.evaluate("""() => { const p = document.querySelector('[data-testid=lofi-player]'); const r = p.getBoundingClientRect(); const bad = [];
      for (const el of p.querySelectorAll('button, input, select, a, iframe, p, [data-testid=lofi-video]')) { const b = el.getBoundingClientRect(); if (b.width && (b.right > r.right + 0.5 || b.left < r.left - 0.5)) bad.push(el.tagName + ':' + (el.dataset.testid || el.className || el.textContent.slice(0, 18)) + ' ' + Math.round(b.left) + '..' + Math.round(b.right)) }
      return [p.scrollWidth, p.clientWidth, Math.round(r.left) + '..' + Math.round(r.right), bad] }""")
    check(f"{label}: NADA queda cortado a la derecha: todo el contenido cabe en el ancho de la ventana", over[0] <= over[1] + 1 and not over[3], f"contenido {over[0]} / visible {over[1]} px, ventana {over[2]}, fuera: {over[3]}")
    v = pg.get_by_test_id("lofi-video").bounding_box()
    check(f"{label}: el vídeo es pequeño (≤ 190 px de ancho) y se ve", 120 <= v["width"] <= 190 and pg.is_visible("[data-testid=lofi-video] iframe"), f"{v['width']:.0f}x{v['height']:.0f}")
    for tid in ("lofi-play", "lofi-station", "lofi-link", "ambient-kind", "volume-ambient", "volume-effects", "volume-alerts"):
        el = pg.get_by_test_id(tid).bounding_box()
        if not (el and el["y"] + el["height"] <= h and el["y"] >= 0):
            check(f"{label}: «{tid}» está a la vista", False, str(el)); break
    else:
        check(f"{label}: todos los controles (reproducir, emisión, enlace, ambiente y tres volúmenes) están a la vista", True)

with sync_playwright() as p:
    # ---- 1) audio estricto de teléfono (Chromium emulando un Pixel 7)
    b = p.chromium.launch(args=["--use-gl=swiftshader", "--enable-unsafe-swiftshader", "--autoplay-policy=document-user-activation-required"])
    ctx = b.new_context(**p.devices["Pixel 7"]); ctx.add_init_script(STRICT); pg = ctx.new_page(); errs = []
    pg.on("console", lambda m: errs.append(m.text[:200]) if m.type == "error" and "Failed to load resource" not in m.text else None); pg.on("pageerror", lambda e: errs.append(str(e)[:200]))
    pg.goto(BASE); pg.wait_for_selector(".chip"); pg.wait_for_timeout(1200)
    check("antes de tocar nada el audio no se ha creado ni arrancado", pg.evaluate("!window.__audioEngine || window.__audioEngine.ctx.state !== 'running'"))
    open_music(pg)
    check("el primer toque (al terminar) desbloquea el audio en un teléfono estricto", pg.evaluate("window.__audioEngine && window.__audioEngine.ctx.state") == "running", f"estado {pg.evaluate('window.__audioEngine && window.__audioEngine.ctx.state')}, resume() bloqueados: {pg.evaluate('window.__blockedResume')}")
    check("el audio queda en modo «reproducción» (no lo calla el interruptor de silencio del iPhone)", pg.evaluate("navigator.audioSession.type") == "playback")
    pg.evaluate(PROBE)
    pg.select_option("[data-testid=ambient-kind]", "rain"); pg.wait_for_timeout(2200); pg.evaluate("window.__audioEngine.probes.max = 0"); pg.wait_for_timeout(1500)
    sig = pg.evaluate("window.__audioEngine.probes.max")
    check("el sonido de ambiente (lluvia) suena en el teléfono", sig > 0.005, f"señal {sig:.4f}")
    # el sistema interrumpe el audio (llamada, pestaña en segundo plano) y el siguiente toque lo recupera
    pg.evaluate("window.__audioEngine.ctx.suspend()"); pg.wait_for_timeout(500)
    check("si el sistema interrumpe el audio, queda suspendido", pg.evaluate("window.__audioEngine.ctx.state") != "running")
    pg.get_by_test_id("music-button").tap(); pg.wait_for_timeout(400); pg.get_by_test_id("music-button").tap(); pg.wait_for_timeout(1200)
    check("y el siguiente toque lo recupera", pg.evaluate("window.__audioEngine.ctx.state") == "running")
    pg.evaluate("window.__audioEngine.probes.max = 0"); pg.wait_for_timeout(1500)
    check("y el ambiente vuelve a sonar", pg.evaluate("window.__audioEngine.probes.max") > 0.005)
    # 2) el control de volumen de la música no se ofrece en el móvil
    open_music(pg) if pg.locator("[data-testid=lofi-player]:visible").count() == 0 else None
    check("en el móvil no hay deslizador de volumen de la música (el móvil lo ignora) y se explica", pg.get_by_test_id("lofi-volume").count() == 0 and "botones del móvil" in pg.inner_text("[data-testid=lofi-volume-hint]"))
    check("los tres volúmenes del mezclador local siguen disponibles", all(pg.get_by_test_id(t).count() == 1 for t in ("volume-ambient", "volume-effects", "volume-alerts")))
    errs_a = list(errs); ctx.close(); b.close()

    # ---- 3) tamaño de la ventana mientras suena el vídeo, en varios teléfonos
    for label, engine, dev, vp in (("Pixel 7", p.chromium, "Pixel 7", None), ("iPhone 13", p.webkit, "iPhone 13", None), ("iPhone SE (pequeño)", p.webkit, "iPhone SE", None), ("móvil pequeño 360×640", p.chromium, "Pixel 7", {"width": 360, "height": 640})):
        try:
            br = engine.launch(args=["--use-gl=swiftshader", "--enable-unsafe-swiftshader"] if engine is p.chromium else [])
        except Exception as e:
            print(f"INFO {label} no disponible: {str(e)[:60]}"); continue
        kw = dict(p.devices[dev]);
        if vp: kw["viewport"] = vp
        cx = br.new_context(**kw); cx.route("**/iframe_api", lambda r: r.fulfill(status=200, content_type="application/javascript", body=MOCK_API))
        q = cx.new_page(); q.goto(BASE); q.wait_for_selector(".chip"); q.wait_for_timeout(1000); open_music(q)
        q.get_by_test_id("lofi-play").tap(); q.wait_for_function("document.querySelector('[data-testid=lofi-status]').dataset.status === 'playing'", timeout=8000); q.wait_for_timeout(500)
        size = q.viewport_size; fits(q, label, size["width"], size["height"])
        q.screenshot(path=f"{OUT}/mobile_music_{label.split()[0]}_{size['width']}x{size['height']}.png")
        br.close()

    # ---- 4) en el ordenador no cambia: sigue el deslizador de volumen y el vídeo es pequeño
    b = p.chromium.launch(args=["--use-gl=swiftshader", "--enable-unsafe-swiftshader"])
    cx = b.new_context(viewport={"width": 1100, "height": 760}); cx.route("**/iframe_api", lambda r: r.fulfill(status=200, content_type="application/javascript", body=MOCK_API))
    d = cx.new_page(); d.goto(BASE); d.wait_for_selector(".chip"); d.get_by_test_id("music-button").click(); d.wait_for_selector("[data-testid=lofi-player]")
    d.get_by_test_id("lofi-play").click(); d.wait_for_function("document.querySelector('[data-testid=lofi-status]').dataset.status === 'playing'", timeout=8000)
    check("en el ordenador sigue estando el deslizador de volumen de la música", d.get_by_test_id("lofi-volume").count() == 1 and d.get_by_test_id("lofi-volume-hint").count() == 0)
    box = d.get_by_test_id("lofi-player").bounding_box(); v = d.get_by_test_id("lofi-video").bounding_box()
    check("en el ordenador la ventana cabe y el vídeo es pequeño", box["y"] + box["height"] <= 760 and v["width"] <= 190, f"{box['height']:.0f} px de alto, vídeo {v['width']:.0f} px")
    d.screenshot(path=f"{OUT}/mobile_music_desktop.png")
    check("sin errores de consola", not errs_a, "; ".join(errs_a[:3]))
    b.close()
print(f"\n{len(fails)} fallos" if fails else "\nTodo OK"); sys.exit(1 if fails else 0)
