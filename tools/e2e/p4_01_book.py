"""E2E P4-01: Libro de Cosechas (estadísticas del mes, fardos/silos/cestas, tabla, navegación de meses, CSV, móvil).

Se ejecuta con tools/e2e/run_isolated.sh.
"""
import csv, io, json, os, sqlite3, subprocess, sys, urllib.request
from datetime import datetime, timedelta, timezone
from playwright.sync_api import sync_playwright

if "POMOFARM_URL" not in os.environ:
    sys.exit("Estos tests resetean la BD y no deben tocar tu partida. Ejecútalos con: tools/e2e/run_isolated.sh tools/e2e/p4_01_book.py")
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
sql("DELETE FROM structures"); sql("DELETE FROM unlocks WHERE kind='animal'"); sql("DELETE FROM plots WHERE NOT (x=1 AND y=1)")
sql("UPDATE plots SET state='empty', plant_type=NULL, planted_at=NULL, grow_s=NULL, matured_at=NULL, harvested=0, life_s=NULL, wilts_at=NULL, collected_to=NULL")
sql("UPDATE players SET silo_micro=0, silo_peak_micro_h=0, silo_level=0, season=1")
sql("DELETE FROM pomodoro_events"); sql("DELETE FROM pomodoros"); sql("DELETE FROM tags")
sql("INSERT INTO settings (player_id,key,value) VALUES (1,'tutorial_done','1') ON CONFLICT(player_id,key) DO UPDATE SET value='1'")
now = datetime.now(timezone.utc)
this = now.strftime("%Y-%m")
y, m = map(int, this.split("-")); prev = f"{y - (m == 1)}-{(m - 2) % 12 + 1:02d}"
def pom(day_utc, plannedS, tag, status="completed", pausedS=0):
    end = datetime.fromisoformat(day_utc).replace(tzinfo=timezone.utc); start = end - timedelta(seconds=plannedS + pausedS)
    tid = None
    if tag:
        sql("INSERT OR IGNORE INTO tags (player_id,name) VALUES (1,?)", (tag,)); c = sqlite3.connect(DB); tid = c.execute("SELECT id FROM tags WHERE name=?", (tag,)).fetchone()[0]; c.close()
    sql("INSERT INTO pomodoros (player_id,plot_id,plant_type,tag_id,planned_s,started_at,paused_total_s,ended_at,reward_focus,status) VALUES (1,NULL,'daisy',?,?,?,?,?,1,?)",
        (tid, plannedS, iso(start), pausedS, iso(end), status))
# mes actual: lunes 5 (3 × 25 min "tesis"), miércoles 7 (2 × 45 min "inglés"), 1 sin etiqueta de 10 min, y uno cancelado que no cuenta
d5 = f"{this}-05T10:00:00"; d7 = f"{this}-07T09:00:00"
for h in (10, 11, 12): pom(f"{this}-05T{h}:00:00", 1500, "tesis")
for h in (9, 10): pom(f"{this}-07T{h:02d}:00:00", 2700, "inglés", pausedS=300)
pom(f"{this}-07T13:00:00", 600, "")
pom(f"{this}-07T14:00:00", 1500, "tesis", status="cancelled")
# mes anterior: 2 × 25 min
for h in (10, 11): pom(f"{prev}-15T{h}:00:00", 1500, "tesis")

with sync_playwright() as p:
    b = p.chromium.launch(args=["--use-gl=swiftshader", "--enable-unsafe-swiftshader"])
    ctx = b.new_context(viewport={"width": 1100, "height": 760}, timezone_id="Europe/Madrid", accept_downloads=True)
    pg = ctx.new_page(); errs = []
    pg.on("console", lambda m: errs.append(m.text[:200]) if m.type == "error" and "409" not in m.text else None); pg.on("pageerror", lambda e: errs.append(str(e)[:200]))
    pg.goto(BASE); pg.wait_for_selector(".packet")

    # 1) abrir el libro: total del mes
    pg.get_by_test_id("book-button").click(); pg.wait_for_selector("[data-testid=book-total]")
    check("el chip abre el libro y anuncia su estado", pg.get_attribute("[data-testid=book-button]", "aria-expanded") == "true" and pg.get_attribute("[data-testid=harvest-book]", "role") == "dialog")
    total = pg.inner_text("[data-testid=book-total]")
    check("el mes actual suma 2 h 55 min y 6 Pomodoros (los cancelados no cuentan)", "2 h 55 min" in total and "6 Pomodoros" in total, total.replace("\n", " "))
    mes = pg.inner_text("[data-testid=book-month]")
    check("el título es el mes en español", mes.lower().startswith(("enero", "febrero", "marzo", "abril", "mayo", "junio", "julio", "agosto", "septiembre", "setiembre", "octubre", "noviembre", "diciembre")), mes)

    # 2) fardos / silos / cestas
    check("2 h 55 min = 2 fardos enteros y uno a medias (3 iconos)", pg.locator(".harvest__icon").count() == 3, str(pg.locator(".harvest__icon").count()))
    check("el gráfico tiene texto alternativo con las horas y la unidad", "fardos de heno" in pg.get_attribute(".harvest__icons", "aria-label"), pg.get_attribute(".harvest__icons", "aria-label"))
    pg.get_by_test_id("book-unit-silo").click()
    check("cambiar a silos cambia la leyenda y se marca como elegido", "silo" in pg.inner_text("[data-testid=harvest-legend]") and pg.get_attribute("[data-testid=book-unit-silo]", "aria-checked") == "true")
    pg.get_by_test_id("book-unit-basket").click(); check("cestas de manzanas", "cesta de manzanas" in pg.inner_text("[data-testid=harvest-legend]"))
    pg.screenshot(path=f"{OUT}/book_chart.png")
    pg.reload(); pg.wait_for_selector(".packet"); pg.get_by_test_id("book-button").click(); pg.wait_for_selector("[data-testid=book-total]")
    check("la unidad elegida se recuerda tras recargar", pg.get_attribute("[data-testid=book-unit-basket]", "aria-checked") == "true")
    bars = pg.locator(".bars__col").count()
    check("hay una barra por día de la semana (lunes primero)", bars == 7 and pg.locator(".bars__label").first.inner_text() == "L")

    # 3) tabla
    pg.get_by_test_id("book-table-toggle").check(); pg.wait_for_selector("[data-testid=book-tables]")
    tags = pg.locator("table:has(caption:text('Por etiqueta')) tbody tr").all_inner_texts()
    check("tabla por etiqueta: inglés (1 h 30), tesis (1 h 15), sin etiqueta (10 min), con más tiempo primero",
          len(tags) == 3 and "inglés" in tags[0] and "1 h 30 min" in tags[0] and "tesis" in tags[1] and "1 h 15 min" in tags[1] and "Sin etiqueta" in tags[2], str(tags))
    wd = pg.locator("table:has(caption:text('Por día de la semana')) tbody tr").all_inner_texts()
    check("tabla por día de la semana: lunes 3 Pomodoros, miércoles 3", "Lunes" in wd[0] and "3" in wd[0] and "Miércoles" in wd[2] and "3" in wd[2] and "0 min" in wd[1], str(wd))
    check("tabla por día con 2 fechas", pg.locator("table:has(caption:text('Por día')) tbody tr").count() >= 2)
    check("las tablas tienen cabeceras de columna y fila para lectores de pantalla", pg.locator("th[scope=col]").count() >= 9 and pg.locator("th[scope=row]").count() >= 11)
    pg.screenshot(path=f"{OUT}/book_table.png")

    # 4) navegar entre meses
    check("hay un mes anterior con datos y el siguiente está desactivado", pg.get_by_test_id("book-prev").is_enabled() and pg.get_by_test_id("book-next").is_disabled())
    pg.get_by_test_id("book-prev").click(); pg.wait_for_function("document.querySelector('[data-testid=book-total]').innerText.includes('50 min')", timeout=5000)
    check("el mes anterior suma 50 min y 2 Pomodoros", "2 Pomodoros" in pg.inner_text("[data-testid=book-total]"), pg.inner_text("[data-testid=book-total]").replace("\n", " "))
    check("y ahí ya no hay mes anterior, pero sí siguiente", pg.get_by_test_id("book-prev").is_disabled() and pg.get_by_test_id("book-next").is_enabled())
    pg.get_by_test_id("book-next").click(); pg.wait_for_function("document.querySelector('[data-testid=book-total]').innerText.includes('2 h 55')", timeout=5000)

    # 5) exportación CSV: coincide con lo guardado
    with pg.expect_download() as dl: pg.get_by_test_id("book-export").click()
    d = dl.value; path = d.path(); text = open(path, encoding="utf-8-sig").read()
    check("se descarga con un nombre claro", d.suggested_filename == "libro-de-cosechas.csv", d.suggested_filename)
    rows = list(csv.reader(io.StringIO(text)))
    c = sqlite3.connect(DB); stored = c.execute("SELECT p.id, p.planned_s, COALESCE(t.name,''), p.status FROM pomodoros p LEFT JOIN tags t ON t.id=p.tag_id WHERE p.status IN ('completed','cancelled') ORDER BY p.ended_at, p.id").fetchall(); c.close()
    check("el CSV tiene una fila por Pomodoro guardado (9: 6 completados y 1 cancelado este mes, 2 el anterior) más la cabecera", len(rows) == len(stored) + 1 == 10, f"{len(rows) - 1} filas, {len(stored)} guardados")
    ok = all(int(r[0]) == s[0] and abs(float(r[5]) * 60 - s[1]) < 1 and r[3] == s[2] and r[8] == s[3] for r, s in zip(rows[1:], stored))
    check("id, minutos, etiqueta y estado coinciden fila a fila con la base de datos", ok)
    first = next(r for r in rows[1:] if r[3] == "tesis" and r[8] == "completed")
    check("las horas del CSV están en la zona horaria del navegador (Madrid, UTC+2)", first[2].endswith(":00:00") and int(first[2][11:13]) in (12, 13, 14), first[2])
    check("las tildes se conservan", any(r[3] == "inglés" for r in rows))
    mins = sum(float(r[5]) for r in rows[1:] if r[8] == "completed")
    check("la suma de minutos completados del CSV = total de los dos meses (175 + 50 = 225 min)", abs(mins - 225) < 0.2, f"{mins}")

    # 6) Escape y móvil
    pg.keyboard.press("Escape"); check("Escape cierra el libro", pg.locator("[data-testid=harvest-book]").count() == 0)
    m = ctx.new_page(); m.set_viewport_size({"width": 390, "height": 700}); m.goto(BASE); m.wait_for_selector(".chip")
    m.get_by_test_id("book-button").click(); m.wait_for_selector("[data-testid=book-total]")
    box = m.get_by_test_id("harvest-book").bounding_box()
    check("móvil: el libro cabe en pantalla", box and box["x"] >= 0 and box["x"] + box["width"] <= 390 and box["y"] + box["height"] <= 700, str(box))
    check("móvil: sin desbordamiento horizontal", m.evaluate("document.documentElement.scrollWidth <= window.innerWidth"))
    m.get_by_test_id("book-table-toggle").check(); m.wait_for_selector("[data-testid=book-tables]")
    check("móvil: las tablas no desbordan", m.evaluate("document.documentElement.scrollWidth <= window.innerWidth"))
    m.screenshot(path=f"{OUT}/book_mobile.png"); m.close()

    # 7) estado vacío
    sql("DELETE FROM pomodoros")
    e = ctx.new_page(); e.goto(BASE); e.wait_for_selector(".chip"); e.get_by_test_id("book-button").click(); e.wait_for_selector("[data-testid=book-total]")
    check("sin Pomodoros: 0 min, mensaje amable y sin iconos", "0 min" in e.inner_text("[data-testid=book-total]") and e.locator(".harvest__icon").count() == 0 and "Aquí aparecerá" in e.inner_text("[data-testid=harvest-legend]"))
    check("sin Pomodoros no hay mes anterior ni siguiente", e.get_by_test_id("book-prev").is_disabled() and e.get_by_test_id("book-next").is_disabled())

    check("sin errores de consola", not errs, "; ".join(errs[:3]))
    b.close()
print(f"\n{len(fails)} fallos" if fails else "\nTodo OK"); sys.exit(1 if fails else 0)
