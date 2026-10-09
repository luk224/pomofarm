"""E2E P5-02: «tu granja necesita atención» (aviso suave cuando varias plantas están por marchitarse; nunca urgente).

Se ejecuta con tools/e2e/run_isolated.sh.
"""
import json, os, sqlite3, sys, urllib.request
from datetime import datetime, timedelta, timezone
from playwright.sync_api import sync_playwright

if "POMOFARM_URL" not in os.environ:
    sys.exit("Estos tests resetean la BD y no deben tocar tu partida. Ejecútalos con: tools/e2e/run_isolated.sh tools/e2e/p5_02_attention.py")
BASE = os.environ["POMOFARM_URL"]; DB = os.environ["POMOFARM_DB"]
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

INSTRUMENT = """
window.__notif = { created: [] };
window.__away = false;
Document.prototype.hasFocus = function () { return !window.__away; };
class N { constructor(title, opts) { window.__notif.created.push(Object.assign({ title }, opts)); } close() {} static requestPermission() { N.permission = 'granted'; return Promise.resolve('granted'); } }
N.permission = 'granted';
window.Notification = N;
"""

def setup(plants):
    """plants: lista de minutos hasta que se marchita cada una (parcelas 1..n); el resto, vacías."""
    call("POST", "/api/pomodoros/active/cancel")
    for q in ("DELETE FROM structures", "DELETE FROM plots WHERE NOT (x=1 AND y=1)", "DELETE FROM pomodoro_events", "DELETE FROM pomodoros",
              "UPDATE plots SET state='empty', plant_type=NULL, planted_at=NULL, grow_s=NULL, matured_at=NULL, harvested=0, life_s=NULL, wilts_at=NULL, collected_to=NULL",
              "UPDATE players SET silo_micro=0, silo_peak_micro_h=0, silo_level=0, season=1, biome='spring', coins_milli=0, focus_points=0, rest_started_at=NULL, rest_until=NULL"):
        sql(q)
    for x in range(4):
        for y in range(4): sql("INSERT OR IGNORE INTO plots (player_id,x,y) VALUES (1,?,?)", (x, y))
    t = datetime.now(timezone.utc) - timedelta(hours=10)
    for i, mins in enumerate(plants, start=1):
        wilts = datetime.now(timezone.utc) + timedelta(minutes=mins)
        sql("UPDATE plots SET state='mature', plant_type='daisy', grow_s=600, life_s=86400, harvested=1, matured_at=?, wilts_at=?, collected_to=? WHERE id=?", (iso(t), iso(wilts), iso(t), i))
sql("INSERT INTO settings (player_id,key,value) VALUES (1,'tutorial_done','1') ON CONFLICT(player_id,key) DO UPDATE SET value='1'")
sql("INSERT INTO settings (player_id,key,value) VALUES (1,'rest_enabled','0') ON CONFLICT(player_id,key) DO UPDATE SET value='0'")

def fresh_page(b, **kw):
    ctx = b.new_context(viewport={"width": 1100, "height": 760}, **kw); ctx.add_init_script(INSTRUMENT); pg = ctx.new_page(); return ctx, pg
def toast_text(pg): return " | ".join(pg.locator(".toast").all_inner_texts())
def notifs(pg): return pg.evaluate("window.__notif.created")

with sync_playwright() as p:
    b = p.chromium.launch(args=["--use-gl=swiftshader", "--enable-unsafe-swiftshader"])

    # 1) menos de tres plantas por marchitarse: nada
    setup([30, 45]); ctx, pg = fresh_page(b); errs = []
    pg.on("console", lambda m: errs.append(m.text[:200]) if m.type == "error" and "409" not in m.text else None); pg.on("pageerror", lambda e: errs.append(str(e)[:200]))
    pg.goto(BASE); pg.wait_for_selector(".chip"); pg.wait_for_timeout(2500)
    check("con solo 2 plantas por marchitarse no se avisa", pg.locator(".toast").count() == 0 and not notifs(pg))
    ctx.close()

    # 2) varias plantas pero con mucha vida por delante: nada
    setup([300, 400, 500, 900]); ctx, pg = fresh_page(b); pg.goto(BASE); pg.wait_for_selector(".chip"); pg.wait_for_timeout(2500)
    check("4 plantas con más de 2 h de vida no avisan", pg.locator(".toast").count() == 0)
    ctx.close()

    # 3) varias por marchitarse, con la pestaña a la vista: mensaje en la página, sin notificación del sistema
    setup([20, 40, 60, 90]); ctx, pg = fresh_page(b); pg.goto(BASE); pg.wait_for_selector(".chip"); pg.wait_for_selector(".toast", timeout=8000)
    txt = toast_text(pg)
    check("con 4 plantas a punto de marchitarse sale un mensaje en la página", "4 plantas" in txt, txt)
    check("el mensaje es tranquilo: sin exclamaciones, urgencias ni pérdidas", not any(w in txt.lower() for w in ("!", "¡", "urgente", "perder", "pierdes", "cuidado", "rápido")), txt)
    check("con la pestaña a la vista no hay notificación del sistema", not notifs(pg))
    pg.screenshot(path="/tmp/pomofarm-e2e/attention_toast.png")
    # no se repite para el mismo grupo
    pg.reload(); pg.wait_for_selector(".chip"); pg.wait_for_timeout(2500)
    check("tras recargar no se repite el aviso para las mismas plantas", pg.locator(".toast").count() == 0)
    # un grupo distinto pero dentro de las 6 h: tampoco
    sql("UPDATE plots SET state='mature', plant_type='daisy', grow_s=600, life_s=86400, harvested=1, matured_at=?, wilts_at=?, collected_to=? WHERE id=5", (iso(datetime.now(timezone.utc) - timedelta(hours=5)), iso(datetime.now(timezone.utc) + timedelta(minutes=30)), iso(datetime.now(timezone.utc) - timedelta(hours=5))))
    pg.reload(); pg.wait_for_selector(".chip"); pg.wait_for_timeout(2500)
    check("un grupo distinto dentro del periodo de calma de 6 h tampoco avisa", pg.locator(".toast").count() == 0)
    # pasadas 6 h (simulado en la memoria del navegador) sí
    pg.evaluate("localStorage.setItem('pomofarm.attention', JSON.stringify({ key: JSON.parse(localStorage.getItem('pomofarm.attention')).key, at: Date.now() - 7 * 3600 * 1000 }))")
    pg.reload(); pg.wait_for_selector(".chip"); pg.wait_for_selector(".toast", timeout=8000)
    check("pasadas 6 h, un grupo nuevo vuelve a avisar", "5 plantas" in toast_text(pg), toast_text(pg))
    ctx.close()

    # 4) con la pestaña en segundo plano y permiso concedido: notificación silenciosa
    setup([20, 40, 60]); ctx, pg = fresh_page(b); pg.evaluate("window.__away = true") if False else None
    pg.add_init_script("window.__away = true")
    pg.goto(BASE); pg.wait_for_selector(".chip"); pg.wait_for_function("window.__notif.created.length > 0", timeout=8000)
    n = notifs(pg)[0]
    check("en segundo plano es una notificación del sistema silenciosa y única (tag)", n["silent"] is True and n["tag"] == "pomofarm-attention" and len(notifs(pg)) == 1, str(n))
    check("el texto no es urgente", not any(w in (n["title"] + n["body"]).lower() for w in ("!", "urgente", "perder", "cuidado")), n["title"] + " / " + n["body"])
    check("y no se duplica con un mensaje en la página", pg.locator(".toast").count() == 0)
    ctx.close()

    # 5) durante un Pomodoro: silencio total
    setup([20, 40, 60, 80]); sql("INSERT OR IGNORE INTO unlocks (player_id,kind,key,at) VALUES (1,'seed','daisy','t')")
    sql("UPDATE plots SET state='empty', plant_type=NULL, harvested=0, matured_at=NULL, wilts_at=NULL, collected_to=NULL, grow_s=NULL, life_s=NULL WHERE id=9")
    ctx, pg = fresh_page(b); pg.goto(BASE); pg.wait_for_selector(".chip"); pg.wait_for_selector(".toast", timeout=8000)   # aviso inicial
    sql("DELETE FROM settings WHERE key='x'"); pg.evaluate("localStorage.removeItem('pomofarm.attention')")
    pg.locator(".toast").first.click(); pg.wait_for_timeout(300)
    pg.locator(".packet__body:not([disabled])").first.click(); pg.get_by_role("button", name="Plantar", exact=True).click(); pg.wait_for_selector("[data-testid=timer][data-status=running]")
    pg.wait_for_timeout(2000)
    check("al empezar el Pomodoro no sale ningún aviso", pg.locator(".toast").count() == 0 and not notifs(pg))
    pg.evaluate("localStorage.removeItem('pomofarm.attention')")   # que no haya memoria: solo el Pomodoro en marcha puede impedir el aviso
    pg.reload(); pg.wait_for_selector("[data-testid=timer]"); pg.wait_for_timeout(3000)
    check("mientras corre un Pomodoro no se avisa (estás concentrado)", pg.locator(".toast").count() == 0 and not notifs(pg))
    call("POST", "/api/pomodoros/active/cancel"); ctx.close()

    # 6) ajuste: se puede desactivar
    setup([20, 40, 60, 80]); ctx, pg = fresh_page(b); pg.goto(BASE); pg.wait_for_selector(".chip"); pg.wait_for_selector(".toast", timeout=8000); pg.locator(".toast").first.click()
    pg.get_by_role("button", name="Ajustes de avisos").click(); pg.wait_for_selector("[data-testid=farm-attention]")
    check("el ajuste está activado por defecto", pg.get_by_test_id("farm-attention").is_checked())
    pg.get_by_test_id("farm-attention").uncheck(); pg.keyboard.press("Escape")
    pg.evaluate("localStorage.removeItem('pomofarm.attention')"); pg.reload(); pg.wait_for_selector(".chip"); pg.wait_for_timeout(3000)
    check("desactivado, no avisa aunque haya plantas por marchitarse", pg.locator(".toast").count() == 0 and not notifs(pg))
    pg.get_by_role("button", name="Ajustes de avisos").click(); pg.get_by_test_id("farm-attention").check(); pg.keyboard.press("Escape")
    ctx.close()

    # 7) nunca pisa otro mensaje: con un aviso en pantalla, espera
    setup([20, 40, 60, 80]); ctx, pg = fresh_page(b); pg.goto(BASE); pg.wait_for_selector(".chip")
    pg.evaluate("window.__pf = 0"); pg.wait_for_selector(".toast", timeout=8000)
    check("el aviso aparece una sola vez (no se apila)", pg.locator(".toast").count() == 1)
    check("sin errores de consola", not errs, "; ".join(errs[:3]))
    ctx.close(); b.close()
print(f"\n{len(fails)} fallos" if fails else "\nTodo OK"); sys.exit(1 if fails else 0)
