"""QA de la Fase 1 (P1-08): criterios del GDD §8 aplicables + casos límite de uso real.

Se ejecuta con tools/e2e/run_isolated.sh. Imprime PASS/FAIL con valores medidos.
GDD §8 cubiertos aquí: cerrar y volver (±1 s), dos dispositivos, hora del cliente alterada.
(Restaurar copia en instalación limpia: tools/e2e/qa_restore.sh. Los de resolución offline son de la Fase 2.)
"""
import json, os, re, signal, subprocess, sys, time, urllib.request
from datetime import datetime, timezone
from playwright.sync_api import sync_playwright

if "POMOFARM_URL" not in os.environ:
    sys.exit("Estos tests resetean la BD y no deben tocar tu partida. Ejecútalos con: tools/e2e/run_isolated.sh tools/e2e/p1_08_qa.py")
BASE = os.environ["POMOFARM_URL"]; DB = os.environ["POMOFARM_DB"]
OUT = "/tmp/pomofarm-e2e"; os.makedirs(OUT, exist_ok=True)
fails = []
def check(name, ok, detail=""):
    print(("PASS " if ok else "FAIL ") + name + (f"  [{detail}]" if detail else ""))
    if not ok: fails.append(name)
def call(m, path, body=None, base=None):
    r = urllib.request.Request((base or BASE) + path, method=m, data=json.dumps(body).encode() if body is not None else None, headers={"Content-Type": "application/json"})
    try: return json.load(urllib.request.urlopen(r))
    except urllib.error.HTTPError as e: return {"_status": e.code, **json.load(e)}
def sql(q): subprocess.run(["sqlite3", DB, q], check=True)
def reset():
    if call("GET", "/api/state").get("pomodoro"): call("POST", "/api/pomodoros/active/cancel")
    sql("UPDATE plots SET state='empty',plant_type=NULL,harvested=0,matured_at=NULL,wilts_at=NULL,planted_at=NULL,grow_s=NULL,life_s=NULL;"
        "UPDATE players SET focus_points=0,lifetime_focus=0; DELETE FROM settings; DELETE FROM unlocks WHERE key<>'daisy';"
        "DELETE FROM pomodoro_events; DELETE FROM pomodoros; DELETE FROM tags;")
def secs(text):
    m = re.search(r"(\d+):(\d{2})(?::(\d{2}))?", text); g = [int(x) for x in m.groups() if x is not None]
    return g[0] * 60 + g[1] if len(g) == 2 else g[0] * 3600 + g[1] * 60 + g[2]
btn = lambda pg, n: pg.get_by_role("button", name=n, exact=True)
def plant_ui(pg, tag=""):
    pg.locator(".packet__body:not([disabled])").first.click()
    if tag: pg.fill(".field", tag)
    btn(pg, "Plantar").click(); pg.wait_for_selector("[data-testid=timer][data-status=running]")
def epoch(iso): return datetime.fromisoformat(re.sub(r"(\.\d{6})\d+Z$", r"\1+00:00", iso.replace("Z", "+00:00") if "." not in iso else iso)).timestamp()
def new_ctx(b, **kw):
    ctx = b.new_context(viewport={"width": 1100, "height": 760}, **kw); pg = ctx.new_page(); errs = []
    pg.on("console", lambda m: errs.append(m.text[:200]) if m.type == "error" and "409" not in m.text and "502" not in m.text else None); pg.on("pageerror", lambda e: errs.append(str(e)[:200]))
    return ctx, pg, errs

reset()
GL = ["--use-gl=swiftshader", "--enable-unsafe-swiftshader"]
with sync_playwright() as p:
    b = p.chromium.launch(args=GL)

    print("\n## §8.1 Cerrar el navegador a mitad de un Pomodoro y volver")
    ctx1, pg1, errs = new_ctx(b); pg1.goto(BASE); pg1.wait_for_selector(".packet"); plant_ui(pg1, "tesis")
    st = call("GET", "/api/state")["pomodoro"]; started = epoch(st["started_at"])
    ctx1.close()                                  # el navegador se cierra del todo (se pierde todo el estado del cliente)
    time.sleep(7)
    ctx2, pg2, errs2 = new_ctx(b); pg2.goto(BASE); pg2.wait_for_selector("[data-testid=timer][data-status=running]"); ui = secs(pg2.inner_text("[data-testid=timer]")); now = time.time()
    expected = 600 - (now - started)
    check("tiempo restante exacto ±1 s tras cerrar y volver", abs(ui - expected) <= 1.5, f"UI {ui}s, esperado {expected:.1f}s, diferencia {abs(ui-expected):.2f}s")
    check("no se pierde nada: planta y etiqueta siguen", "tesis" in pg2.inner_text(".readout__what") and call("GET", "/api/state")["plots"][0]["state"] == "growing", pg2.inner_text(".readout__what"))

    print("\n## §8.2 Abrir en dos dispositivos")
    ctx3, pg3, _ = new_ctx(b)                       # contexto nuevo = otro dispositivo (sin almacenamiento compartido)
    pg3.goto(BASE); pg3.wait_for_selector("[data-testid=timer][data-status=running]"); t2 = secs(pg3.inner_text("[data-testid=timer]"))
    check("el segundo dispositivo muestra el Pomodoro existente", abs(t2 - ui) <= 12, f"A {ui}s, B {t2}s")
    check("y no ofrece iniciar otro (no hay sobres ni Plantar)", pg3.locator(".packet").count() == 0 and btn(pg3, "Plantar").count() == 0)
    r = call("POST", "/api/pomodoros", {"plot_id": 1, "plant_type": "daisy"})
    check("el servidor rechaza un segundo Pomodoro (409)", r.get("_status") == 409 and r.get("error") == "pomodoro_active", str(r))
    btn(pg3, "Pausar").click(); pg2.evaluate("window.dispatchEvent(new Event('focus'))")
    pg2.wait_for_selector("[data-testid=timer][data-status=paused]", timeout=8000)
    check("pausar en B se refleja en A al volver a ella", True)
    btn(pg2, "Reanudar").click(); ctx3.close()

    print("\n## §8.3 Cambiar la hora del sistema del cliente")
    ctx4, pg4, e4 = new_ctx(b, timezone_id="Pacific/Auckland")
    ctx4.add_init_script("const _n = Date.now; Date.now = () => _n.call(Date) + 3*24*3600*1000; const _D = Date; ")
    pg4.goto(BASE); pg4.wait_for_selector("[data-testid=timer][data-status=running]"); u4 = secs(pg4.inner_text("[data-testid=timer]")); srv = call("GET", "/api/state")["pomodoro"]["remaining_ms"] / 1000
    check("con el reloj del cliente +3 días y otra zona horaria, el tiempo coincide con el servidor ±1.5 s", abs(u4 - srv) <= 1.5, f"UI {u4}s, servidor {srv:.1f}s")
    ctx4.close()

    print("\n## Casos límite de uso real")
    call("POST", "/api/pomodoros/active/cancel"); pg2.reload(); pg2.wait_for_selector(".packet")
    # doble clic en Plantar: un solo Pomodoro y sin avisos de error engañosos
    pg2.locator(".packet__body:not([disabled])").first.click(); btn(pg2, "Plantar").dblclick(); pg2.wait_for_selector("[data-testid=timer]"); pg2.wait_for_timeout(1200)
    n = sql_count = int(subprocess.run(["sqlite3", DB, "SELECT COUNT(*) FROM pomodoros WHERE status IN ('running','paused')"], capture_output=True, text=True).stdout.strip())
    toasts = pg2.locator(".toast--error").all_inner_texts()
    check("doble clic en Plantar crea un solo Pomodoro", n == 1, str(n))
    check("y no muestra un error engañoso ('otro dispositivo') al propio jugador", not toasts, str(toasts))
    pg2.get_by_role("button", name="Cancelar", exact=True).click(); pg2.get_by_role("button", name="¿Cancelar? Se pierde la planta").click(); pg2.wait_for_selector(".packet")

    # solo teclado
    pg2.keyboard.press("Tab")
    for _ in range(12):
        if pg2.evaluate("document.activeElement && document.activeElement.classList.contains('packet__body') && !document.activeElement.disabled"): break
        pg2.keyboard.press("Tab")
    pg2.keyboard.press("Space"); check("teclado: un sobre se elige con Espacio", pg2.evaluate("document.activeElement.getAttribute('aria-checked')") == "true")
    for _ in range(12):
        if pg2.evaluate("document.activeElement && document.activeElement.textContent.trim() === 'Plantar'"): break
        pg2.keyboard.press("Tab")
    outline = pg2.evaluate("getComputedStyle(document.activeElement).outlineStyle + ' ' + getComputedStyle(document.activeElement).outlineWidth")
    check("el foco por teclado es visible", "solid 3px" in outline, outline)
    pg2.keyboard.press("Enter"); pg2.wait_for_selector("[data-testid=timer][data-status=running]", timeout=8000)
    check("teclado: Tab hasta Plantar y Intro inicia el Pomodoro", True)
    pg2.keyboard.press("Tab"); pg2.keyboard.press("Space"); pg2.wait_for_selector("[data-testid=timer][data-status=paused]", timeout=8000)
    check("teclado: un botón enfocado se activa con Espacio y no dispara el atajo global a la vez", pg2.locator("[data-testid=timer][data-status=paused]").count() == 1)
    call("POST", "/api/pomodoros/active/cancel"); pg2.reload(); pg2.wait_for_selector(".packet")

    # etiquetas hostiles y largas
    evil = '<img src=x onerror="window.__xss=1"> & "q"'[:60]
    plant_ui(pg2, evil); pg2.wait_for_timeout(600)
    check("una etiqueta con HTML se muestra como texto (sin XSS)", pg2.evaluate("window.__xss") is None and "<img" in pg2.inner_text(".readout__what"), pg2.inner_text(".readout__what"))
    call("POST", "/api/pomodoros/active/cancel"); pg2.reload(); pg2.wait_for_selector(".packet")
    pg2.locator(".packet__body:not([disabled])").first.click(); pg2.fill(".field", "x" * 80)
    check("el campo de etiqueta limita a 60 caracteres", len(pg2.input_value(".field")) == 60, str(len(pg2.input_value(".field"))))
    r = call("POST", "/api/pomodoros", {"plot_id": 1, "plant_type": "daisy", "tag": "y" * 61})
    check("el servidor rechaza etiquetas de más de 60 caracteres (400)", r.get("_status") == 400, str(r))
    r = call("POST", "/api/pomodoros", {"plot_id": 1, "plant_type": "daisy", "tag": "'; DROP TABLE players;--"}); call("POST", "/api/pomodoros/active/cancel")
    check("una etiqueta con SQL se guarda como texto (consultas parametrizadas)", "error" not in r and call("GET", "/api/state")["player"]["name"] == "Granjero", str(r.get("error")))

    # persistencia ante reinicio del servidor y recuperación sin conexión
    pg2.reload(); pg2.wait_for_selector(".packet"); plant_ui(pg2, "reinicio"); before = secs(pg2.inner_text("[data-testid=timer]"))
    # matar SOLO el servidor de este entorno (hijo del supervisor), nunca por nombre: podría ser la partida de desarrollo
    pid = subprocess.run(["pgrep", "-P", os.environ["POMOFARM_SUPERVISOR_PID"], "-x", "pomofarm"], capture_output=True, text=True).stdout.split()[0]
    hold = os.environ["POMOFARM_HOLD_FILE"]; open(hold, "w").close()   # el supervisor no lo relanza mientras exista
    os.kill(int(pid), signal.SIGTERM); time.sleep(0.5)
    btn(pg2, "Pausar").click(); pg2.wait_for_selector(".toast--error", timeout=8000); msg = pg2.inner_text(".toast--error")
    check("sin servidor: aviso claro de conexión, sin romper la pantalla", "conexión" in msg and pg2.locator("[data-testid=timer]").count() == 1, msg)
    os.remove(hold)   # ahora sí: el supervisor lo relanza (como restart: unless-stopped)
    for _ in range(60):
        try: urllib.request.urlopen(BASE + "/api/health"); break
        except Exception: time.sleep(0.25)
    pg2.evaluate("window.dispatchEvent(new Event('focus'))"); pg2.wait_for_timeout(1500); after = secs(pg2.inner_text("[data-testid=timer]"))
    check("tras reiniciar el servidor, el Pomodoro sigue y el tiempo es coherente", before - 12 <= after <= before, f"antes {before}s, después {after}s")
    check("la UI se recupera sola al volver el servidor", pg2.locator("[data-testid=timer][data-status=running]").count() == 1)
    call("POST", "/api/pomodoros/active/cancel")

    # móvil con etiqueta larga y pantalla pequeña
    pg2.set_viewport_size({"width": 360, "height": 640}); pg2.reload(); pg2.wait_for_selector(".packet"); pg2.locator(".packet__body:not([disabled])").first.click(); pg2.fill(".field", "x" * 60)
    box = pg2.locator(".dock").bounding_box(); over = pg2.evaluate("document.documentElement.scrollWidth > document.documentElement.clientWidth")
    check("360×640: dock dentro de la pantalla y sin scroll horizontal", box["x"] >= 0 and box["x"] + box["width"] <= 360.5 and not over, str(box))
    check("360×640: el botón Plantar es visible y alcanzable", btn(pg2, "Plantar").is_visible() and btn(pg2, "Plantar").bounding_box()["y"] + 44 <= 640)
    pg2.screenshot(path=f"{OUT}/qa_mobile_360.png")
    check("sin errores de consola (Chromium)", not (errs + errs2 + e4), "; ".join(errs + errs2 + e4)[:240])
    ctx2.close(); b.close()

    print("\n## Navegadores: humo en Firefox y WebKit")
    for name in ["firefox", "webkit"]:
        try:
            br = getattr(p, name).launch()
            ctx = br.new_context(viewport={"width": 1000, "height": 700}); pg = ctx.new_page(); er = []
            pg.on("pageerror", lambda e: er.append(str(e)[:160])); pg.on("console", lambda m: er.append(m.text[:160]) if m.type == "error" and "WebGL" not in m.text and "409" not in m.text else None)
            reset(); pg.goto(BASE); pg.wait_for_selector(".packet", timeout=20000); plant_ui(pg, "humo"); time.sleep(2.2); t = secs(pg.inner_text("[data-testid=timer]"))
            check(f"{name}: carga, planta y la cuenta atrás avanza", 595 <= t <= 598, f"{t}s")
            pg.reload(); pg.wait_for_selector("[data-testid=timer][data-status=running]"); t2 = secs(pg.inner_text("[data-testid=timer]")); check(f"{name}: recargar conserva el tiempo", abs(t2 - (t - 2)) <= 3, f"{t2}s")
            has_canvas = pg.evaluate("(() => { const c = document.querySelector('canvas'); return !!c && c.width > 0 })()")
            check(f"{name}: el lienzo 3D se crea", has_canvas); check(f"{name}: sin errores de consola", not er, "; ".join(er)[:200])
            pg.screenshot(path=f"{OUT}/qa_{name}.png"); br.close()
        except Exception as e:
            if "missing dependencies" in str(e):
                print(f"SKIP {name}: faltan dependencias del sistema (sudo apt-get install libmanette-0.2-0)")
            else:
                check(f"{name}: el navegador se puede lanzar y probar", False, str(e).splitlines()[0][:160])
reset()
print("\nFALLOS:" if fails else "\nTODO OK", fails or "")
sys.exit(1 if fails else 0)
