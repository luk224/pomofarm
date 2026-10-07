"""E2E P2-01: producción offline, Silo, recogida manual, Silo lleno, móvil.

Se ejecuta con tools/e2e/run_isolated.sh. Fuerza plantas maduras en el pasado dentro de la BD temporal;
es el SERVIDOR quien calcula la producción al abrir la página (con su reloj real).
"""
import os, re, subprocess, sys
from datetime import datetime, timedelta, timezone
from playwright.sync_api import sync_playwright

if "POMOFARM_URL" not in os.environ:
    sys.exit("Estos tests resetean la BD y no deben tocar tu partida. Ejecútalos con: tools/e2e/run_isolated.sh tools/e2e/p2_01_silo.py")
BASE = os.environ["POMOFARM_URL"]; DB = os.environ["POMOFARM_DB"]
OUT = "/tmp/pomofarm-e2e"; os.makedirs(OUT, exist_ok=True)
fails = []
def check(name, ok, detail=""):
    print(("PASS " if ok else "FAIL ") + name + (f"  [{detail}]" if detail else ""))
    if not ok: fails.append(name)
def sql(q): subprocess.run(["sqlite3", DB, q], check=True)
def iso(dt): return dt.strftime("%Y-%m-%dT%H:%M:%S.%fZ")
def force_mature(hours_ago, life_h=24):
    """Planta madura (cosechada) en la parcela 1 que maduró hace `hours_ago` h y aún no se ha contado."""
    t = datetime.now(timezone.utc) - timedelta(hours=hours_ago)
    sql(f"UPDATE plots SET state='mature', plant_type='daisy', grow_s=600, life_s={life_h*3600}, harvested=1, matured_at='{iso(t)}', "
        f"wilts_at='{iso(t + timedelta(hours=life_h))}', collected_to='{iso(t)}' WHERE id=1;"
        "UPDATE players SET silo_micro=0, silo_peak_micro_h=0, coins_milli=0;")
def num(text):  # "2,5" -> 2.5 ; "1.234" -> 1234
    t = text.strip().replace(".", "").replace(",", "."); return float(re.search(r"[\d.]+", t).group())

with sync_playwright() as p:
    b = p.chromium.launch(args=["--use-gl=swiftshader", "--enable-unsafe-swiftshader"])
    pg = b.new_page(viewport={"width": 1100, "height": 760}); errs = []
    pg.on("console", lambda m: errs.append(m.text[:200]) if m.type == "error" and "409" not in m.text else None); pg.on("pageerror", lambda e: errs.append(str(e)[:200]))

    pg.goto(BASE); pg.wait_for_selector(".packet")
    check("sin plantas maduras no hay Silo ni saldo de 🪙", pg.locator("[data-testid=silo]").count() == 0 and pg.locator("[data-testid=coins]").count() == 0)

    # 3 h de producción mientras la página estaba cerrada
    force_mature(3); pg.reload(); pg.wait_for_selector("[data-testid=silo]"); pg.wait_for_timeout(500)
    amt = num(pg.inner_text("[data-testid=silo-amount]").split("/")[0])
    check("el Silo muestra ≈2,5 🪙 (3 h × 0,83 🪙/h calculadas por el servidor)", 2.3 <= amt <= 2.6, str(amt))
    cap = num(pg.inner_text("[data-testid=silo-amount]").split("/")[1])
    check("capacidad = 12 h × 0,83 = 10 🪙", 9.8 <= cap <= 10.1, str(cap))
    check("la nota indica el ritmo (+0,8 por hora)", "0,8" in pg.inner_text("[data-testid=silo-note]"), pg.inner_text("[data-testid=silo-note]"))
    check("el saldo de 🪙 aparece pero sigue en 0 hasta recoger", pg.locator("[data-testid=coins]").count() == 1 and num(pg.inner_text("[data-testid=coins]")) == 0)
    meter = pg.get_attribute("[role=meter]", "aria-valuemax"); check("barra accesible (role=meter con máximo)", meter in ("10", "9"), str(meter))
    pg.screenshot(path=f"{OUT}/silo_filling.png")

    pg.get_by_test_id("silo-collect").click(); pg.wait_for_selector(".toast--coin")
    check("recoger muestra +N 🪙", "+2," in pg.inner_text(".toast--coin"), pg.inner_text(".toast--coin"))
    coins = num(pg.inner_text("[data-testid=coins]")); check("el saldo sube a lo recogido", 2.3 <= coins <= 2.6, str(coins))
    check("el Silo queda vacío y Recoger se desactiva", pg.get_by_test_id("silo-collect").is_disabled() and num(pg.inner_text("[data-testid=silo-amount]").split("/")[0]) < 0.2)
    pg.reload(); pg.wait_for_selector("[data-testid=coins]"); check("el saldo persiste al recargar", abs(num(pg.inner_text("[data-testid=coins]")) - coins) < 0.11)

    # Silo lleno: lo producido más allá se pierde
    force_mature(40, life_h=24)   # vivió 24 h: 20 🪙 en total, pero el Silo de 12 h solo guarda 10
    pg.reload(); pg.wait_for_selector(".silo--full"); full_amt = num(pg.inner_text("[data-testid=silo-amount]").split("/")[0])
    check("Silo lleno: guarda 10 🪙 aunque la planta produjo 20", 9.95 <= full_amt <= 10.1, str(full_amt))
    check("avisa de que está lleno y qué hacer", "Lleno" in pg.inner_text("[data-testid=silo-note]") and "recoge" in pg.inner_text("[data-testid=silo-note]"))
    pg.screenshot(path=f"{OUT}/silo_full.png")
    pg.keyboard.press("c"); pg.wait_for_selector(".toast--coin")
    check("la tecla C recoge", "+10" in pg.inner_text(".toast--coin"), pg.inner_text(".toast--coin"))
    pg.wait_for_timeout(500)
    check("planta marchita: el Silo desaparece y el saldo se queda", pg.locator("[data-testid=silo]").count() == 0 and num(pg.inner_text("[data-testid=coins]")) >= 9.8)
    pg.keyboard.press("c"); pg.wait_for_timeout(600)
    check("C con el Silo vacío no hace nada ni da error", pg.locator(".toast--error").count() == 0)

    # móvil
    force_mature(3); pg.set_viewport_size({"width": 360, "height": 640}); pg.reload(); pg.wait_for_selector("[data-testid=silo]"); pg.wait_for_timeout(600)
    sb, tb, db = pg.locator("[data-testid=silo]").bounding_box(), pg.locator(".topbar").bounding_box(), pg.locator(".dock").bounding_box()
    check("360×640: el Silo no tapa la barra superior ni el dock", sb["y"] >= tb["y"] + tb["height"] - 1 and sb["y"] + sb["height"] <= db["y"], f"silo {sb['y']:.0f}-{sb['y']+sb['height']:.0f}, barra hasta {tb['y']+tb['height']:.0f}, dock desde {db['y']:.0f}")
    check("360×640: sin scroll horizontal", not pg.evaluate("document.documentElement.scrollWidth > document.documentElement.clientWidth"))
    check("el botón Recoger mide ≥ 44 px", pg.get_by_test_id("silo-collect").bounding_box()["height"] >= 44)
    pg.screenshot(path=f"{OUT}/silo_mobile.png")
    check("sin errores de consola", not errs, "; ".join(errs)[:240])
    b.close()
sql("UPDATE plots SET state='empty',plant_type=NULL,harvested=0,matured_at=NULL,wilts_at=NULL,collected_to=NULL,grow_s=NULL,life_s=NULL; UPDATE players SET silo_micro=0,silo_peak_micro_h=0,coins_milli=0")
print("\nFALLOS:" if fails else "\nTODO OK", fails or "")
sys.exit(1 if fails else 0)
