"""E2E P3-03: audio por capas (ambiente, efectos, alertas): tres volúmenes independientes que persisten, ambiente local
(lluvia, bosque, fuego) y efectos de interacción. Mide la señal real con un AnalyserNode sobre cada capa.

Se ejecuta con tools/e2e/run_isolated.sh.
"""
import json, os, subprocess, sys, urllib.request
from playwright.sync_api import sync_playwright

if "POMOFARM_URL" not in os.environ:
    sys.exit("Estos tests resetean la BD y no deben tocar tu partida. Ejecútalos con: tools/e2e/run_isolated.sh tools/e2e/p3_03_audio.py")
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

# partida limpia y sin Pomodoro activo
call("POST", "/api/pomodoros/active/cancel")
sql("DELETE FROM structures; DELETE FROM plots WHERE NOT (x=1 AND y=1); UPDATE plots SET state='empty', plant_type=NULL, planted_at=NULL, grow_s=NULL, matured_at=NULL, harvested=0, life_s=NULL, wilts_at=NULL, collected_to=NULL")
sql("INSERT INTO settings (player_id,key,value) VALUES (1,'tutorial_done','1') ON CONFLICT(player_id,key) DO UPDATE SET value='1'")

# un analizador por capa: mide la señal que de verdad pasa por cada bus
PROBE = """() => { const a = window.__audioEngine; if (!a) return false; if (a.probes) return true;
  a.probes = {}; for (const l of ['ambient','effects','alerts']) { const an = a.ctx.createAnalyser(); an.fftSize = 1024; a.buses[l].connect(an);
    const buf = new Float32Array(1024); const st = { max: 0 }; a.probes[l] = st;
    setInterval(() => { an.getFloatTimeDomainData(buf); for (const v of buf) st.max = Math.max(st.max, Math.abs(v)) }, 20) } return true }"""
RESET_MAX = "() => { for (const l of Object.keys(window.__audioEngine.probes)) window.__audioEngine.probes[l].max = 0 }"
MAXES = "() => Object.fromEntries(Object.entries(window.__audioEngine.probes).map(([k, v]) => [k, v.max]))"
GAINS = "() => Object.fromEntries(Object.entries(window.__audioEngine.buses).map(([k, v]) => [k, v.gain.value]))"
def set_slider(pg, layer, value):
    pg.evaluate("""([l, v]) => { const el = document.querySelector(`[data-testid=volume-${l}]`);
      Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, 'value').set.call(el, String(v)); el.dispatchEvent(new Event('input', { bubbles: true })) }""", [layer, value])
def open_settings(pg):
    """El mezclador de sonido vive en la ventana de música (chip «Música»)."""
    if pg.locator("[data-testid=sound-settings]").count() == 0: pg.get_by_test_id("music-button").click()
    pg.wait_for_selector("[data-testid=sound-settings]")

with sync_playwright() as p:
    b = p.chromium.launch(args=["--use-gl=swiftshader", "--enable-unsafe-swiftshader", "--autoplay-policy=no-user-gesture-required"])
    ctx = b.new_context(viewport={"width": 1100, "height": 760})
    pg = ctx.new_page(); errs = []
    pg.on("console", lambda m: errs.append(m.text[:200]) if m.type == "error" and "409" not in m.text else None); pg.on("pageerror", lambda e: errs.append(str(e)[:200]))
    pg.goto(BASE); pg.wait_for_selector(".packet")

    # 1) controles y valores por defecto
    open_settings(pg)
    vals = {l: int(pg.input_value(f"[data-testid=volume-{l}]")) for l in ("ambient", "effects", "alerts")}
    check("hay tres controles de volumen con valores por defecto 50/70/80", vals == {"ambient": 50, "effects": 70, "alerts": 80}, str(vals))
    check("y un selector de ambiente que empieza sin ambiente", pg.input_value("[data-testid=ambient-kind]") == "off")
    check("los deslizadores tienen nombre accesible", all(pg.get_by_role("slider", name=f"Volumen de {n}").count() == 1 for n in ("ambiente", "efectos", "alertas")))
    pg.wait_for_function("window.__audioEngine && window.__audioEngine.ctx.state === 'running'", timeout=8000)
    pg.evaluate(PROBE)

    # 2) cada volumen mueve SOLO su capa
    g0 = pg.evaluate(GAINS)
    check("ganancias iniciales = (volumen)²", all(abs(g0[l] - (vals[l] / 100) ** 2) < 0.01 for l in vals), str(g0))
    set_slider(pg, "ambient", 20); pg.wait_for_timeout(500); g = pg.evaluate(GAINS)
    check("ambiente a 20 → ganancia 0,04 y las otras no se mueven", abs(g["ambient"] - 0.04) < 0.01 and abs(g["effects"] - g0["effects"]) < 0.01 and abs(g["alerts"] - g0["alerts"]) < 0.01, str(g))
    set_slider(pg, "effects", 100); pg.wait_for_timeout(500); g = pg.evaluate(GAINS)
    check("efectos a 100 → ganancia 1 y ambiente sigue en 0,04", abs(g["effects"] - 1) < 0.01 and abs(g["ambient"] - 0.04) < 0.01, str(g))
    set_slider(pg, "alerts", 0); pg.wait_for_timeout(500); g = pg.evaluate(GAINS)
    check("alertas a 0 → silencio y efectos sigue en 1", g["alerts"] < 0.001 and abs(g["effects"] - 1) < 0.01, str(g))
    check("el valor se muestra junto al deslizador", pg.inner_text(".slider:has([data-testid=volume-alerts]) output") == "0")

    # 3) persistencia
    pg.reload(); pg.wait_for_selector(".packet"); open_settings(pg)
    vals2 = {l: int(pg.input_value(f"[data-testid=volume-{l}]")) for l in ("ambient", "effects", "alerts")}
    check("los volúmenes persisten tras recargar", vals2 == {"ambient": 20, "effects": 100, "alerts": 0}, str(vals2))
    pg.get_by_test_id("volume-effects").focus(); pg.keyboard.press("ArrowLeft")  # gesto + cambio de teclado
    pg.wait_for_function("window.__audioEngine && window.__audioEngine.ctx.state === 'running'", timeout=8000); pg.wait_for_timeout(500)
    g = pg.evaluate(GAINS)
    check("tras recargar, el motor arranca con los volúmenes guardados", abs(g["ambient"] - 0.04) < 0.01 and g["alerts"] < 0.001 and abs(g["effects"] - 0.99 ** 2) < 0.02, str(g))
    set_slider(pg, "effects", 70); set_slider(pg, "alerts", 80); set_slider(pg, "ambient", 50); pg.wait_for_timeout(400)
    pg.evaluate(PROBE)

    # 4) ambiente local: señal real en la capa ambiente
    check("sin ambiente elegido no hay señal en la capa", pg.evaluate("() => { window.__audioEngine.probes.ambient.max = 0; return true }") and (pg.wait_for_timeout(300) or True) and pg.evaluate(MAXES)["ambient"] < 0.0005, str(pg.evaluate(MAXES)))
    for kind in ("rain", "forest", "fire"):
        pg.evaluate(RESET_MAX); pg.select_option("[data-testid=ambient-kind]", kind); pg.wait_for_timeout(1800)
        m = pg.evaluate(MAXES)
        check(f"ambiente {kind}: hay señal en la capa ambiente y no en las otras", m["ambient"] > 0.005 and m["alerts"] < 0.0005, str(m))
    # el volumen del ambiente manda sobre la señal
    set_slider(pg, "ambient", 100); pg.wait_for_timeout(600); pg.evaluate(RESET_MAX); pg.wait_for_timeout(800); loud = pg.evaluate(MAXES)["ambient"]
    set_slider(pg, "ambient", 10); pg.wait_for_timeout(600); pg.evaluate(RESET_MAX); pg.wait_for_timeout(800); quiet = pg.evaluate(MAXES)["ambient"]
    check("subir/bajar el volumen cambia de verdad la señal (100 vs 10)", loud > quiet * 5, f"{loud:.4f} vs {quiet:.4f}")
    set_slider(pg, "ambient", 0); pg.wait_for_timeout(600); pg.evaluate(RESET_MAX); pg.wait_for_timeout(600)
    check("ambiente a 0: la capa queda en silencio", pg.evaluate(MAXES)["ambient"] < 0.0005, str(pg.evaluate(MAXES)))
    set_slider(pg, "ambient", 50)
    pg.select_option("[data-testid=ambient-kind]", "off"); pg.wait_for_timeout(1800); pg.evaluate(RESET_MAX); pg.wait_for_timeout(500)
    check("elegir 'Sin ambiente' lo apaga con un fundido", pg.evaluate(MAXES)["ambient"] < 0.0005, str(pg.evaluate(MAXES)))

    # 5) el ambiente elegido vuelve tras recargar (al primer gesto)
    pg.select_option("[data-testid=ambient-kind]", "fire"); pg.wait_for_timeout(300)
    pg.reload(); pg.wait_for_selector(".packet"); check("la elección de ambiente persiste", (open_settings(pg) or True) and pg.input_value("[data-testid=ambient-kind]") == "fire")
    pg.wait_for_function("window.__audioEngine && window.__audioEngine.ctx.state === 'running'", timeout=8000) if pg.evaluate("!!window.__audioEngine") else pg.mouse.click(5, 5)
    pg.wait_for_function("window.__audioEngine && window.__audioEngine.ctx.state === 'running'", timeout=8000); pg.evaluate(PROBE); pg.evaluate(RESET_MAX); pg.wait_for_timeout(2000)
    check("tras recargar y el primer gesto, vuelve a sonar el fuego", pg.evaluate(MAXES)["ambient"] > 0.005, str(pg.evaluate(MAXES)))
    pg.select_option("[data-testid=ambient-kind]", "off"); pg.get_by_test_id("music-button").click() if pg.locator("[data-testid=lofi-player]").count() else None

    # 6) efectos: plantar suena en la capa de efectos y respeta su volumen
    pg.evaluate(RESET_MAX); pg.locator(".packet__body:not([disabled])").first.click(); pg.get_by_role("button", name="Plantar", exact=True).click()
    pg.wait_for_selector("[data-testid=timer][data-status=running]"); pg.wait_for_timeout(700); m = pg.evaluate(MAXES)
    check("plantar suena en la capa de efectos", m["effects"] > 0.01, str(m))
    call("POST", "/api/pomodoros/active/cancel"); pg.reload(); pg.wait_for_selector(".packet"); open_settings(pg)
    set_slider(pg, "effects", 0); pg.wait_for_timeout(800); pg.get_by_test_id("music-button").click() if pg.locator("[data-testid=lofi-player]").count() else None
    pg.wait_for_function("window.__audioEngine && window.__audioEngine.ctx.state === 'running'", timeout=8000); pg.evaluate(PROBE); pg.evaluate(RESET_MAX)
    pg.locator(".packet__body:not([disabled])").first.click(); pg.get_by_role("button", name="Plantar", exact=True).click()
    pg.wait_for_selector("[data-testid=timer][data-status=running]"); pg.wait_for_timeout(700); m = pg.evaluate(MAXES)
    check("con efectos a 0, plantar no se oye", m["effects"] < 0.001, str(m))

    # 7) monedas: recoger el Silo suena (capa de efectos)
    call("POST", "/api/pomodoros/active/cancel"); pg.reload(); pg.wait_for_selector(".packet"); open_settings(pg); set_slider(pg, "effects", 80); pg.get_by_test_id("music-button").click() if pg.locator("[data-testid=lofi-player]").count() else None
    pg.wait_for_function("window.__audioEngine && window.__audioEngine.ctx.state === 'running'", timeout=8000); pg.wait_for_timeout(500); pg.evaluate(PROBE)
    from datetime import datetime, timedelta, timezone
    t = datetime.now(timezone.utc) - timedelta(hours=3); f = lambda d: d.strftime("%Y-%m-%dT%H:%M:%S.%fZ")
    sql(f"UPDATE plots SET state='mature', plant_type='daisy', grow_s=600, life_s=86400, harvested=1, matured_at='{f(t)}', wilts_at='{f(t+timedelta(hours=24))}', collected_to='{f(t)}'")
    pg.reload(); pg.wait_for_selector("[data-testid=silo]"); pg.mouse.click(5, 5); pg.wait_for_function("window.__audioEngine && window.__audioEngine.ctx.state === 'running'", timeout=8000); pg.evaluate(PROBE); pg.evaluate(RESET_MAX)
    pg.get_by_test_id("silo-collect").click(); pg.wait_for_selector(".toast--coin", timeout=5000); pg.wait_for_timeout(500)
    check("recoger monedas suena en efectos, no en alertas", pg.evaluate(MAXES)["effects"] > 0.01 and pg.evaluate(MAXES)["alerts"] < 0.0005, str(pg.evaluate(MAXES)))

    # 8) alertas: la campana (vista previa del deslizador) suena en SU capa
    open_settings(pg)
    set_slider(pg, "alerts", 80); pg.wait_for_timeout(500); pg.evaluate(RESET_MAX)
    pg.get_by_test_id("volume-alerts").focus(); pg.keyboard.press("ArrowRight"); pg.wait_for_timeout(700); m = pg.evaluate(MAXES)
    check("la campana suena en la capa de alertas", m["alerts"] > 0.005, str(m))

    check("sin errores de consola", not errs, "; ".join(errs[:3]))
    b.close()
print(f"\n{len(fails)} fallos" if fails else "\nTodo OK"); sys.exit(1 if fails else 0)
