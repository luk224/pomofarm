"""E2E P4-02: métricas personales (Pomodoros por semana, mejor racha, Pomodoros limpios) y modo estricto.

Se ejecuta con tools/e2e/run_isolated.sh.
"""
import json, os, sqlite3, sys, urllib.request
from datetime import datetime, timedelta, timezone
from playwright.sync_api import sync_playwright

if "POMOFARM_URL" not in os.environ:
    sys.exit("Estos tests resetean la BD y no deben tocar tu partida. Ejecútalos con: tools/e2e/run_isolated.sh tools/e2e/p4_02_stats.py")
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
          "DELETE FROM settings WHERE key IN ('strict_mode')",
          "UPDATE plots SET state='empty', plant_type=NULL, planted_at=NULL, grow_s=NULL, matured_at=NULL, harvested=0, life_s=NULL, wilts_at=NULL, collected_to=NULL",
          "UPDATE players SET silo_micro=0, silo_peak_micro_h=0, silo_level=0, season=1"):
    sql(q)
sql("INSERT INTO settings (player_id,key,value) VALUES (1,'tutorial_done','1') ON CONFLICT(player_id,key) DO UPDATE SET value='1'")

# historial: 4 días seguidos hace ~3 semanas (mejor racha = 4) y los dos últimos días (racha actual = 2, hoy y ayer)
today = datetime.now(timezone.utc).replace(hour=10, minute=0, second=0, microsecond=0)
def pom(day, strict=0, paused_s=0, pauses=0, status="completed"):
    end = day; start = end - timedelta(seconds=1500 + paused_s)
    c = sqlite3.connect(DB); cur = c.execute("INSERT INTO pomodoros (player_id,plot_id,plant_type,planned_s,started_at,paused_total_s,ended_at,reward_focus,status,strict) VALUES (1,NULL,'daisy',1500,?,?,?,1,?,?)",
        (iso(start), paused_s, iso(end), status, strict)); pid = cur.lastrowid
    for i in range(pauses): c.execute("INSERT INTO pomodoro_events (pomodoro_id,kind,at) VALUES (?, 'pause', ?)", (pid, iso(start + timedelta(minutes=i + 1))))
    c.commit(); c.close()
for back in (24, 23, 22, 21): pom(today - timedelta(days=back))
pom(today - timedelta(days=1)); pom(today, strict=1); pom(today - timedelta(hours=3), strict=1, paused_s=420, pauses=2)       # limpios: 2 de 3 estrictos
pom(today - timedelta(hours=2), strict=1, paused_s=900, pauses=1)                                                           # 15 min de pausa: no es limpio
pom(today - timedelta(days=2), status="cancelled")                                                                          # cancelado: no cuenta para la racha

with sync_playwright() as p:
    b = p.chromium.launch(args=["--use-gl=swiftshader", "--enable-unsafe-swiftshader"])
    ctx = b.new_context(viewport={"width": 1100, "height": 760}, timezone_id="UTC")
    pg = ctx.new_page(); errs = []
    pg.on("console", lambda m: errs.append(m.text[:200]) if m.type == "error" and "409" not in m.text else None); pg.on("pageerror", lambda e: errs.append(str(e)[:200]))
    pg.goto(BASE); pg.wait_for_selector(".packet")

    # 1) números en el Libro
    pg.get_by_test_id("book-button").click(); pg.wait_for_selector("[data-testid=personal-numbers]")
    check("el Libro muestra 'Tus números'", pg.locator("[data-testid=personal-numbers]").count() == 1)
    check("mejor racha: 4 días seguidos", "4 días seguidos" in pg.inner_text("[data-testid=num-best]"), pg.inner_text("[data-testid=num-best]"))
    check("racha actual: 2 días (hoy y ayer)", pg.inner_text("[data-testid=num-current]") == "2 días", pg.inner_text("[data-testid=num-current]"))
    week = int(pg.inner_text("[data-testid=num-week]").split()[0])
    monday = (today - timedelta(days=today.weekday())).date()
    done = [today - timedelta(days=d) for d in (24, 23, 22, 21, 1)] + [today, today - timedelta(hours=3), today - timedelta(hours=2)]  # los completados sembrados
    expected_week = sum(1 for d in done if d.date() >= monday)
    check("'esta semana' cuenta los completados desde el lunes", week == expected_week, f"{week} (esperado {expected_week})")
    check("limpios: 2 de 3 en modo estricto (15 min de pausa no es limpio)", pg.inner_text("[data-testid=num-clean]") == "2 de 3 en modo estricto", pg.inner_text("[data-testid=num-clean]"))
    check("el gráfico semanal tiene 12 barras y texto alternativo", pg.locator(".weeks__col").count() == 12 and "últimas 12" in pg.get_attribute("[data-testid=weeks-chart]", "aria-label"))
    check("el texto no usa lenguaje de castigo", not any(w in pg.inner_text("[data-testid=personal-numbers]").lower() for w in ("perdiste", "rota", "fallaste", "pierdes")))
    pg.screenshot(path=f"{OUT}/stats_book.png")
    pg.keyboard.press("Escape")

    # 2) modo estricto: ajuste, sesión en vivo, nunca bloquea
    pg.get_by_role("button", name="Ajustes de avisos").click(); pg.wait_for_selector("[data-testid=strict-setting]")
    check("el ajuste explica que no cambia nada del juego", "No cambia nada del juego" in pg.inner_text("[data-testid=strict-setting]") and not pg.get_by_test_id("strict-mode").is_checked())
    pg.get_by_test_id("strict-mode").check(); pg.wait_for_timeout(500)
    check("se guarda en el servidor (se comparte entre dispositivos)", call("GET", "/api/state")["settings"].get("strict_mode") == "1")
    pg.keyboard.press("Escape")
    pg.locator(".packet__body:not([disabled])").first.click(); pg.get_by_role("button", name="Plantar", exact=True).click(); pg.wait_for_selector("[data-testid=timer][data-status=running]")
    check("durante una sesión estricta se ve el progreso (0 de 2 pausas)", "0 de 2 pausas · 0 de 10 min" in pg.inner_text("[data-testid=strict-progress]"), pg.inner_text("[data-testid=strict-progress]"))
    for i in range(1, 4):  # tres pausas: la tercera pasa el límite pero el juego deja pausar
        pg.get_by_role("button", name="Pausar", exact=True).click(); pg.wait_for_selector("[data-testid=timer][data-status=paused]")
        pg.get_by_role("button", name="Reanudar", exact=True).click(); pg.wait_for_selector("[data-testid=timer][data-status=running]")
    t = pg.inner_text("[data-testid=strict-progress]")
    check("la tercera pausa se permite y el aviso es amable", "3 de 2 pausas" in t and "ya no contará como limpio" in t and pg.get_attribute("[data-testid=strict-progress]", "data-clean") == "false", t)
    check("la sesión sigue corriendo con normalidad", call("GET", "/api/state")["pomodoro"]["status"] == "running")
    call("POST", "/api/pomodoros/active/cancel")

    # 3) sin modo estricto no hay línea de progreso
    pg.reload(); pg.wait_for_selector(".packet"); pg.get_by_role("button", name="Ajustes de avisos").click(); pg.get_by_test_id("strict-mode").uncheck(); pg.wait_for_timeout(400); pg.keyboard.press("Escape")
    pg.locator(".packet__body:not([disabled])").first.click(); pg.get_by_role("button", name="Plantar", exact=True).click(); pg.wait_for_selector("[data-testid=timer][data-status=running]")
    check("sin modo estricto no aparece el progreso", pg.locator("[data-testid=strict-progress]").count() == 0)
    call("POST", "/api/pomodoros/active/cancel")

    # 4) el CSV lleva las columnas estricto y limpio
    csv_text = urllib.request.urlopen(BASE + "/api/book/export.csv?tz=UTC").read().decode("utf-8-sig")
    lines = csv_text.strip().splitlines()
    check("el CSV incluye 'estricto' y 'limpio'", lines[0].endswith("estado,estricto,limpio"), lines[0])
    check("el CSV marca exactamente 2 Pomodoros limpios", sum(1 for l in lines[1:] if l.endswith(",1,1")) == 2, str([l[-12:] for l in lines[1:]]))

    # 5) móvil
    m = ctx.new_page(); m.set_viewport_size({"width": 390, "height": 700}); m.goto(BASE); m.wait_for_selector(".chip"); m.get_by_test_id("book-button").click(); m.wait_for_selector("[data-testid=personal-numbers]")
    box = m.get_by_test_id("harvest-book").bounding_box()
    check("móvil: el Libro con los números cabe en pantalla", box and box["x"] >= 0 and box["x"] + box["width"] <= 390 and box["y"] + box["height"] <= 700 and m.evaluate("document.documentElement.scrollWidth <= window.innerWidth"), str(box))
    m.screenshot(path=f"{OUT}/stats_mobile.png"); m.close()
    check("sin errores de consola", not errs, "; ".join(errs[:3]))
    b.close()
print(f"\n{len(fails)} fallos" if fails else "\nTodo OK"); sys.exit(1 if fails else 0)
