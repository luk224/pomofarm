"""E2E P1-07: aviso al terminar (sonido, notificación, toast, título), permiso en el primer Pomodoro, ajustes.

Se ejecuta con tools/e2e/run_isolated.sh (backend y Vite propios, BD temporal). Resetea la BD de desarrollo. Para no esperar 10 min se retrasa
`started_at` en la BD y se dispara el evento `focus`, que es lo que hace el navegador al volver a la pestaña.
Uso: python3 tools/e2e/p1_07_alerts.py
"""
import json, os, subprocess, sys, urllib.request
from playwright.sync_api import sync_playwright

if "POMOFARM_URL" not in os.environ:
    sys.exit("Estos tests resetean la BD y no deben tocar tu partida. Ejecútalos con: tools/e2e/run_isolated.sh tools/e2e/<script>.py")

BASE = os.environ["POMOFARM_URL"]
DB = os.environ.get("POMOFARM_DB", "")
fails = []
def check(name, ok, detail=""):
    print(("PASS " if ok else "FAIL ") + name + (f"  [{detail}]" if detail else ""))
    if not ok: fails.append(name)
def call(m, path, body=None):
    r = urllib.request.Request(BASE + path, method=m, data=json.dumps(body).encode() if body is not None else None, headers={"Content-Type": "application/json"})
    try: return json.load(urllib.request.urlopen(r))
    except urllib.error.HTTPError as e: return json.load(e)
def sql(q): subprocess.run(["sqlite3", DB, q], check=True)
def reset():
    if call("GET", "/api/state").get("pomodoro"): call("POST", "/api/pomodoros/active/cancel")
    sql("UPDATE plots SET state='empty',plant_type=NULL,harvested=0,matured_at=NULL,wilts_at=NULL,planted_at=NULL,grow_s=NULL,life_s=NULL;"
        "UPDATE players SET focus_points=0,lifetime_focus=0; DELETE FROM settings; DELETE FROM unlocks WHERE key<>'daisy';"
        "DELETE FROM pomodoro_events; DELETE FROM pomodoros; DELETE FROM tags;")

INSTRUMENT = """
window.__audio = { ctxs: 0, osc: 0 };
const AC = window.AudioContext;
window.AudioContext = class extends AC {
  constructor() { super(); window.__audio.ctxs++; window.__audio.last = this; }
  createOscillator() { window.__audio.osc++; return super.createOscillator(); }
};
window.__notif = { created: [], asked: 0 };
window.__away = false;
Document.prototype.hasFocus = function () { return !window.__away; };
class N {
  constructor(title, opts) { window.__notif.created.push(Object.assign({ title }, opts)); }
  close() {}
  static requestPermission() { window.__notif.asked++; N.permission = 'granted'; sessionStorage.setItem('np', 'granted'); return Promise.resolve('granted'); }
}
N.permission = sessionStorage.getItem('np') || 'default'; // el navegador real recuerda el permiso
window.Notification = N;
"""
btn = lambda pg, name: pg.get_by_role("button", name=name, exact=True)
def plant(pg):
    pg.locator(".packet__body:not([disabled])").first.click(); btn(pg, "Plantar").click()
    pg.wait_for_selector("[data-testid=timer][data-status=running]")
def finish_soon(pg):
    """Deja ~3 s al Pomodoro en curso y avisa a la página como haría el navegador al recuperar el foco."""
    sql("UPDATE pomodoros SET started_at=strftime('%Y-%m-%dT%H:%M:%fZ','now','-597 seconds') WHERE status='running'")
    pg.evaluate("window.dispatchEvent(new Event('focus'))")
def counts(pg): return pg.evaluate("({osc: window.__audio.osc, notif: window.__notif.created.length, asked: window.__notif.asked, toasts: [...document.querySelectorAll('.toast')].map(t => t.innerText)})")
def next_cycle(pg):
    call("POST", "/api/plots/1/harvest"); call("POST", "/api/plots/1/clear", {"confirm": True}); pg.reload(); pg.wait_for_selector(".packet")

reset()
with sync_playwright() as p:
    b = p.chromium.launch(args=["--use-gl=swiftshader", "--enable-unsafe-swiftshader"])
    ctx = b.new_context(viewport={"width": 1100, "height": 760}); ctx.add_init_script(INSTRUMENT)
    pg = ctx.new_page(); errs = []
    pg.on("console", lambda m: errs.append(m.text[:200]) if m.type == "error" else None); pg.on("pageerror", lambda e: errs.append(str(e)[:200]))
    pg.goto(BASE); pg.wait_for_selector(".packet")
    check("al cargar no se pide permiso ni se crea audio", counts(pg)["asked"] == 0 and pg.evaluate("window.__audio.ctxs") == 0)

    # 1) enfocada, con sonido
    plant(pg)
    c = counts(pg); check("el permiso de notificaciones se pide al plantar el primer Pomodoro", c["asked"] == 1, str(c["asked"]))
    check("el audio se crea y se reanuda con el gesto del usuario", pg.evaluate("window.__audio.ctxs") == 1 and pg.evaluate("window.__audio.last.state") == "running", pg.evaluate("window.__audio.last.state"))
    finish_soon(pg); pg.wait_for_selector(".toast", timeout=15000); pg.wait_for_timeout(600); c = counts(pg)
    check("toast al terminar", any("lista" in t for t in c["toasts"]), str(c["toasts"]))
    check("suena el cuenco (2 golpes × 4 parciales) más el efecto de plantar (1)", c["osc"] == 9, str(c["osc"]))
    check("con la pestaña enfocada no hay notificación del sistema", c["notif"] == 0)
    check("el título avisa de que está lista", pg.title().startswith("✔ Lista para cosechar"), pg.title())
    check("el dock ofrece Cosechar", pg.locator("[data-testid=harvest]").count() == 1)
    next_cycle(pg)

    # 2) en segundo plano
    pg.evaluate("window.__away = true"); plant(pg)
    check("tras recargar, el permiso ya concedido no se vuelve a pedir", counts(pg)["asked"] == 0)
    finish_soon(pg); pg.wait_for_function("window.__notif.created.length === 1", timeout=15000); c = counts(pg)
    n = pg.evaluate("window.__notif.created[0]")
    check("en segundo plano: notificación del sistema", n["title"] == "Pomodoro terminado" and "margarita" in n["body"], str(n))
    check("la notificación es silenciosa (suena nuestro cuenco) y única por Pomodoro", n["silent"] is True and n["tag"].startswith("pomofarm-"))
    check("y también suena el cuenco (+1 del efecto de plantar)", c["osc"] == 9, str(c["osc"]))
    next_cycle(pg)

    # 3) sin sonido
    pg.evaluate("window.__away = true")
    pg.get_by_role("button", name="Ajustes de avisos").click()
    check("el menú de ajustes se abre y anuncia su estado", pg.get_attribute("[aria-controls=settings-panel]", "aria-expanded") == "true")
    pg.get_by_label("Campana al terminar").uncheck(); pg.keyboard.press("Escape")
    check("Escape cierra el menú", pg.locator("#settings-panel").count() == 0)
    plant(pg); finish_soon(pg); pg.wait_for_function("window.__notif.created.length === 1", timeout=15000); c = counts(pg)
    check("sin campana: solo suena el efecto de plantar, no el cuenco", c["osc"] == 1, str(c["osc"]))
    check("sin sonido: sigue el toast visible y la notificación", any("lista" in t for t in c["toasts"]) and c["notif"] == 1)
    next_cycle(pg)

    # 4) cancelar no avisa
    before = counts(pg); plant(pg); btn(pg, "Cancelar").click(); pg.get_by_role("button", name="¿Cancelar? Se pierde la planta").click(); pg.wait_for_selector(".packet"); pg.wait_for_timeout(1500)
    after = counts(pg); check("cancelar no genera aviso (solo suena el efecto de plantar)", after["osc"] == before["osc"] + 1 and after["notif"] == before["notif"] and not after["toasts"], str(after))

    # 5) cargar una página con el Pomodoro ya terminado no avisa; 6) la preferencia persiste
    sql("UPDATE plots SET state='mature',plant_type='daisy',harvested=0,matured_at=strftime('%Y-%m-%dT%H:%M:%SZ','now'),wilts_at=strftime('%Y-%m-%dT%H:%M:%SZ','now','+1 day') WHERE id=1")
    pg.reload(); pg.wait_for_selector("[data-testid=harvest]"); pg.wait_for_timeout(1200)
    c = counts(pg); check("cargar con la planta ya lista no hace ruido", c["osc"] == 0 and not c["toasts"] and c["notif"] == 0, str(c))
    pg.get_by_role("button", name="Ajustes de avisos").click()
    check("la preferencia 'sin sonido' persiste tras recargar", not pg.get_by_label("Campana al terminar").is_checked())
    check("el ajuste de notificaciones sigue activo", pg.get_by_label("Notificación del navegador").is_checked())
    pg.screenshot(path="/tmp/pomofarm-e2e/alerts_settings.png")
    check("sin errores de consola", not errs, "; ".join(errs)[:240])
    b.close()
reset()
print("\nFALLOS:" if fails else "\nTODO OK", fails or "")
sys.exit(1 if fails else 0)
