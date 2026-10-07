"""E2E P2-02: vida útil y marchitamiento (plantas que se marchitan, recompensa conservada, retirar gratis).

Se ejecuta con tools/e2e/run_isolated.sh. Fuerza plantas maduras en el pasado en la BD temporal.
"""
import json, os, re, subprocess, sys, urllib.request
from datetime import datetime, timedelta, timezone
from playwright.sync_api import sync_playwright

if "POMOFARM_URL" not in os.environ:
    sys.exit("Estos tests resetean la BD y no deben tocar tu partida. Ejecútalos con: tools/e2e/run_isolated.sh tools/e2e/p2_02_wither.py")
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
def sql(q): subprocess.run(["sqlite3", DB, q], check=True)
def iso(dt): return dt.strftime("%Y-%m-%dT%H:%M:%S.%fZ")
def force(matured_hours_ago, harvested, life_h=24, plant="daisy"):
    t = datetime.now(timezone.utc) - timedelta(hours=matured_hours_ago)
    sql(f"UPDATE plots SET state='mature', plant_type='{plant}', grow_s=600, life_s={life_h*3600}, harvested={harvested}, matured_at='{iso(t)}', "
        f"wilts_at='{iso(t + timedelta(hours=life_h))}', collected_to='{iso(t)}' WHERE id=1;"
        "UPDATE players SET silo_micro=0, silo_peak_micro_h=0, coins_milli=0, focus_points=0, lifetime_focus=1;"
        "INSERT INTO settings (player_id,key,value) VALUES (1,'tutorial_done','1') ON CONFLICT(player_id,key) DO UPDATE SET value='1'")
btn = lambda pg, n: pg.get_by_role("button", name=n, exact=True)
SCENE = "window.__three.scene.getObjectByName('plant-daisy')"
# Plant colours live in the geometry (vertex colours): compare the first vertex with the withered tint.
FIRST_COLOR_IS_WITHERED = "(() => { const C = window.__three.scene.background.constructor; const w = new C('#a89a78'); const a = window.__three.scene.getObjectByName('plant-daisy').children[0].children[0].geometry.attributes.color; return Math.abs(a.getX(0)-w.r) < 1e-3 && Math.abs(a.getY(0)-w.g) < 1e-3 && Math.abs(a.getZ(0)-w.b) < 1e-3 })()"

with sync_playwright() as p:
    b = p.chromium.launch(args=["--use-gl=swiftshader", "--enable-unsafe-swiftshader"])
    pg = b.new_page(viewport={"width": 1100, "height": 760}); errs = []
    pg.on("console", lambda m: errs.append(m.text[:200]) if m.type == "error" and "409" not in m.text else None); pg.on("pageerror", lambda e: errs.append(str(e)[:200]))
    pg.goto(BASE); pg.wait_for_selector(".packet")

    # 1) murió mientras el jugador estaba fuera y nunca la cosechó
    force(30, 0); pg.reload(); pg.wait_for_selector("[data-testid=harvest]"); pg.wait_for_timeout(1200)
    st = call("GET", "/api/state")
    check("el servidor la marca como marchita al abrir", st["plots"][0]["state"] == "withered" and not st["plots"][0]["harvested"], st["plots"][0]["state"])
    check("el dock explica que se marchitó pero aún se puede cosechar", "se marchitó" in pg.inner_text(".dock") and "cosechar" in pg.inner_text(".dock"), pg.inner_text(".dock"))
    check("la planta se dibuja apagada (color de marchita)", pg.evaluate(FIRST_COLOR_IS_WITHERED))
    check("conserva el destello de 'cosechar'", pg.evaluate(SCENE + ".children.length") == 2)
    check("su producción se detuvo y el Silo guarda como máximo 10 🪙", st["silo"]["rate_milli_per_h"] == 0 and 9900 <= st["silo"]["content_milli"] <= 10000, str(st["silo"]))
    pg.screenshot(path=f"{OUT}/withered.png")
    btn(pg, "Cosechar").click(); pg.wait_for_selector(".toast--reward")
    check("una marchita aún paga su 💧", "+1" in pg.inner_text(".toast--reward"), pg.inner_text(".toast--reward"))
    pg.wait_for_selector("[data-testid=clear]")
    check("tras cosechar: Retirar planta sin confirmación", "se marchitó" in pg.inner_text(".dock").lower() or "marchit" in pg.inner_text(".dock"), pg.inner_text(".dock"))
    btn(pg, "Retirar planta").click(); pg.wait_for_selector(".packet")
    check("un solo clic retira la planta marchita (gratis)", btn(pg, "Plantar").is_visible())
    check("la parcela queda vacía y reutilizable", call("GET", "/api/state")["plots"][0]["state"] == "empty")

    # 2) sigue viva: muestra cuánta vida le queda y retirarla pide confirmación
    force(22, 1); pg.reload(); pg.wait_for_selector("[data-testid=clear]"); pg.wait_for_timeout(800)
    txt = pg.inner_text(".dock__msg")
    check("viva: 'se marchita en' ≈2 h (le quedan 2 h de 24)", re.search(r"se marchita en (1 h 5\d min|2 h)", txt) is not None, txt)
    btn(pg, "Retirar planta").click()
    check("viva: Retirar pide confirmación y avisa de que deja de producir", "Deja de producir" in pg.inner_text(".dock"))
    pg.wait_for_timeout(3300); check("la confirmación caduca sola a los 3 s", "Deja de producir" not in pg.inner_text(".dock"))
    pg.screenshot(path=f"{OUT}/alive_clear.png")

    # 3) marchita ya cosechada: se puede sembrar directamente encima desde la API; sin cosechar, no
    force(30, 0); r = call("POST", "/api/pomodoros", {"plot_id": 1, "plant_type": "daisy"})
    check("no se siembra sobre una marchita sin cosechar (protege el 💧)", r.get("_status") == 409 and r.get("error") == "harvest_first", str(r))
    call("POST", "/api/plots/1/harvest"); r = call("POST", "/api/pomodoros", {"plot_id": 1, "plant_type": "daisy"})
    check("tras cosecharla se puede sembrar encima", "error" not in r and r["pomodoro"] is not None, str(r.get("error")))
    call("POST", "/api/pomodoros/active/cancel")

    # 4) la espera del navegador: el cambio de viva a marchita llega con la sincronización (30 s)
    t = datetime.now(timezone.utc) - timedelta(hours=24) + timedelta(seconds=3)   # se marchita en 3 s
    sql(f"UPDATE plots SET state='mature', plant_type='daisy', grow_s=600, life_s=86400, harvested=1, matured_at='{iso(t)}', wilts_at='{iso(t + timedelta(hours=24))}', collected_to='{iso(t)}' WHERE id=1")
    pg.reload(); pg.wait_for_selector("[data-testid=clear]"); check("recién cargada sigue viva", "produciendo" in pg.inner_text(".dock__msg"), pg.inner_text(".dock__msg"))
    pg.wait_for_timeout(4000); pg.evaluate("window.dispatchEvent(new Event('focus'))"); pg.wait_for_function("document.querySelector('.dock__msg').innerText.includes('marchit')", timeout=8000)
    check("al volver a la pestaña, la planta aparece marchita", "Se marchitó" in pg.inner_text(".dock__msg"), pg.inner_text(".dock__msg"))

    check("sin errores de consola", not errs, "; ".join(errs)[:240])
    b.close()
sql("UPDATE plots SET state='empty',plant_type=NULL,harvested=0,matured_at=NULL,wilts_at=NULL,collected_to=NULL,grow_s=NULL,life_s=NULL; UPDATE players SET silo_micro=0,silo_peak_micro_h=0,coins_milli=0,focus_points=0,lifetime_focus=0; DELETE FROM settings")
print("\nFALLOS:" if fails else "\nTODO OK", fails or "")
sys.exit(1 if fails else 0)
