"""E2E P1-04: el temporizador sobrevive a recargas, ignora el reloj del navegador y se congela en pausa.

Requiere el backend (:8080) y Vite (:5173) en marcha. Uso: python3 tools/e2e/p1_04_timer.py
Deja capturas en /tmp/pomofarm-e2e/.
"""
import json, os, re, sys, time, urllib.request
from playwright.sync_api import sync_playwright

BASE = os.environ.get("POMOFARM_URL", "http://localhost:5173")
OUT = "/tmp/pomofarm-e2e"; os.makedirs(OUT, exist_ok=True)

def call(method, path, body=None):
    req = urllib.request.Request(BASE + path, method=method, data=json.dumps(body).encode() if body else None,
                                 headers={"Content-Type": "application/json"})
    try:
        with urllib.request.urlopen(req) as r: return json.load(r)
    except urllib.error.HTTPError as e: return {"error": json.load(e).get("error"), "status": e.code}

def secs(text):  # "⏱ 9:58 · daisy" -> 598
    m = re.search(r"(\d+):(\d{2})(?::(\d{2}))?", text)
    parts = [int(x) for x in m.groups() if x is not None]
    return parts[0] * 60 + parts[1] if len(parts) == 2 else parts[0] * 3600 + parts[1] * 60 + parts[2]

fails = []
def check(name, ok, detail=""):
    print(("PASS " if ok else "FAIL ") + name + (f"  [{detail}]" if detail else ""))
    if not ok: fails.append(name)

st = call("GET", "/api/state")
if st.get("pomodoro"): call("POST", "/api/pomodoros/active/cancel")
st = call("POST", "/api/pomodoros", {"plot_id": st["plots"][0]["id"], "plant_type": "daisy"})
check("plantar por API", st.get("pomodoro", {}).get("status") == "running", str(st.get("error")))

with sync_playwright() as p:
    b = p.chromium.launch()
    # El "navegador del usuario" tiene el reloj adelantado una hora.
    ctx = b.new_context(viewport={"width": 900, "height": 600})
    ctx.add_init_script("const _n = Date.now; Date.now = () => _n.call(Date) + 3600000;")
    page = ctx.new_page()
    errors = []; page.on("console", lambda m: errors.append(m.text) if m.type == "error" else None)
    page.on("pageerror", lambda e: errors.append(str(e)))

    t0 = time.monotonic(); page.goto(BASE); page.wait_for_selector('[data-testid=timer][data-status=running]')
    a = secs(page.inner_text('[data-testid=timer]')); ta = time.monotonic()
    check("primera lectura ~10:00", 590 <= a <= 600, f"{a}s")
    check("título de pestaña muestra el tiempo", re.search(r"⏱ \d+:\d{2} · PomoFarm", page.title()) is not None, page.title())

    time.sleep(4)
    page.reload(); page.wait_for_selector('[data-testid=timer][data-status=running]')
    b2 = secs(page.inner_text('[data-testid=timer]')); tb = time.monotonic()
    expected = a - (tb - ta)
    check("tras recargar (reloj +1 h) el tiempo es exacto ±1 s", abs(b2 - expected) <= 1.5, f"leído {b2}s, esperado {expected:.1f}s")

    # el contador baja solo en la pantalla
    c1 = secs(page.inner_text('[data-testid=timer]')); time.sleep(2.2); c2 = secs(page.inner_text('[data-testid=timer]'))
    check("la cuenta atrás avanza sola", 1 <= c1 - c2 <= 3, f"{c1}->{c2}")
    page.screenshot(path=f"{OUT}/running.png")

    # pausa desde la UI
    page.click("text=Pausar"); page.wait_for_selector('[data-testid=timer][data-status=paused]')
    p1 = secs(page.inner_text('[data-testid=timer]')); time.sleep(3.5)
    p2 = secs(page.inner_text('[data-testid=timer]'))
    check("en pausa el tiempo está congelado", p1 == p2, f"{p1} == {p2}")
    check("título en pausa", "⏸" in page.title(), page.title())
    page.reload(); page.wait_for_selector('[data-testid=timer][data-status=paused]')
    p3 = secs(page.inner_text('[data-testid=timer]'))
    check("recargar en pausa conserva el tiempo", p3 == p1, f"{p3} == {p1}")
    page.screenshot(path=f"{OUT}/paused.png")

    # reanudar y cancelar
    page.click("text=Reanudar"); page.wait_for_selector('[data-testid=timer][data-status=running]')
    page.click("text=Cancelar"); page.wait_for_selector('[data-testid=timer][data-status=idle]')
    check("cancelar deja la UI sin Pomodoro", "sin Pomodoro" in page.inner_text('[data-testid=timer]'))
    check("título vuelve a PomoFarm", page.title() == "PomoFarm", page.title())

    # segunda pestaña/dispositivo: ve el Pomodoro existente
    call("POST", "/api/pomodoros", {"plot_id": st["plots"][0]["id"], "plant_type": "daisy"})
    page2 = ctx.new_page(); page2.goto(BASE); page2.wait_for_selector('[data-testid=timer][data-status=running]')
    check("otro dispositivo ve el Pomodoro existente", True)
    check("sin errores en la consola", not errors, "; ".join(errors)[:200])
    call("POST", "/api/pomodoros/active/cancel")
    b.close()

print("\nFALLOS:" if fails else "\nTODO OK", fails or "")
sys.exit(1 if fails else 0)
