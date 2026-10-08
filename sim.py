"""Simulador de economía de PomoFarm 3D (v2).

Modelo simplificado, día a día, para tres perfiles de jugador. Los precios y
parámetros de abajo son los mismos que usa la hoja de balance (pestaña Parametros).
Uso:  python3 sim.py            -> imprime hitos por perfil
      python3 sim.py json       -> vuelca el resultado completo en JSON
"""
import math, json, sys

# ---------- Parámetros del modelo ----------
E0, BETA = 2.0, 0.8            # rendimiento por minuto de foco: e(d) = E0*(d/10)^BETA
CROPS = [  # nombre, duración (min), desbloqueo 💧, recompensa 💧, vida útil (h)
    ("Margarita", 10, 0, 1, 24),
    ("Tomates", 25, 8, 5, 36),
    ("Girasol", 35, 35, 8, 54),
    ("Manzano", 45, 100, 15, 72),
    ("Roble Mágico", 60, 250, 25, 108),
]
PLOT_BASE, PLOT_GROWTH, MAX_PLOTS = 3, 1.4, 16
SILO = [(12, 0), (24, 25), (36, 70), (48, 160), (72, 350)]   # (horas de capacidad, coste 💧)
ANIMAL_UNLOCK = {"Abejas": 30, "Perro": 120}                  # coste 💧 (desbloqueo único)
HIVE_BASE, HIVE_GROWTH, DOG_COST = 4000, 1.5, 30000           # coste 🪙
M_ADJ, M_HUERTO, M_BEES, M_CAP = 0.15, 0.10, 0.25, 2.0        # multiplicadores medios supuestos
PRESTIGE_BONUS = 0.10                                         # +10% de 🪙 por estación completada, por encima del tope
WEEK = (1, 1, 1, 1, 1, 0.3, 0.3)                              # fracción de foco L..D
PROFILES = {"Ligero": (90, 25), "Normal": (180, 45), "Intenso": (300, 60),
            "Tú mín": (120, 45), "Tú máx": (240, 60)}  # (min foco/día, sesión máx min); los dos últimos = rango real del usuario (2–4 h)


def e(d): return E0 * (d / 10) ** BETA
def Y(d): return e(d) * d
def g(c): return Y(c[1]) / c[4]
def plot_cost(n): return math.ceil(PLOT_BASE * PLOT_GROWTH ** (n - 2))
def hive_cost(i): return round(HIVE_BASE * HIVE_GROWTH ** i)


def run(F, session_max, days=200, season=1):
    """season=1: partida nueva. season>1: justo tras un Prestigio (GDD 4.9): se conservan semillas, animales (desbloqueos),
    16 parcelas y Silo máximo; se reinician 🪙, colmenas, Perro y plantas; y cada 🪙 vale +10% por estación completada."""
    prestige_mult = 1 + PRESTIGE_BONUS * (season - 1)
    if season == 1:
        unlocked = [0]
        P, silo, hives, dog = 1, 0, 0, False
        animals = set()
    else:
        unlocked = list(range(len(CROPS)))
        P, silo, hives, dog = MAX_PLOTS, len(SILO) - 1, 0, False
        animals = set(ANIMAL_UNLOCK)
    water = coins = C = 0.0
    hist, milestones, prestige_day = [], {}, None
    for day in range(1, days + 1):
        f = F * WEEK[(day - 1) % 7]
        crop = CROPS[max(i for i in unlocked if CROPS[i][1] <= session_max)]
        d = crop[1]
        sessions = f / d
        water += sessions * crop[3]
        C = min(C + sessions * Y(d), P * Y(d))     # potencial de monedas en las parcelas
        adj = M_ADJ if P >= 2 else 0.0
        huerto = M_HUERTO if (P >= 8 and len(unlocked) >= 4) else 0.0
        bees = M_BEES * min(1.0, hives / 4)
        M = min(M_CAP, 1 + adj + huerto + bees)
        burn = min(C, 24 * P * g(crop))
        C -= burn
        coins += burn * M * prestige_mult
        while True:   # compras: lo más barato asequible
            opts, nxt = [], len(unlocked)
            if nxt < len(CROPS): opts.append(("crop", CROPS[nxt][2], "w"))
            if P < MAX_PLOTS: opts.append(("plot", plot_cost(P + 1), "w"))
            if silo + 1 < len(SILO): opts.append(("silo", SILO[silo + 1][1], "w"))
            for a, c in ANIMAL_UNLOCK.items():
                if a not in animals: opts.append((a, c, "w"))
            if "Abejas" in animals and hives < 4: opts.append(("hive", hive_cost(hives), "c"))
            if "Perro" in animals and not dog: opts.append(("dog", DOG_COST, "c"))
            afford = [o for o in opts if (water if o[2] == "w" else coins) >= o[1]]
            if not afford: break
            o = min(afford, key=lambda x: x[1])
            if o[2] == "w": water -= o[1]
            else: coins -= o[1]
            if o[0] == "crop": unlocked.append(nxt); milestones[f"Desbloqueo {CROPS[nxt][0]}"] = day
            elif o[0] == "plot":
                P += 1
                if P in (4, 8, 12, 16): milestones[f"{P} parcelas"] = day
            elif o[0] == "silo":
                silo += 1
                if silo == len(SILO) - 1: milestones["Silo nivel máximo"] = day
            elif o[0] in ("Abejas", "Perro"): animals.add(o[0]); milestones[f"Desbloqueo {o[0]}"] = day
            elif o[0] == "hive":
                hives += 1
                if hives == 4: milestones["4 colmenas"] = day
            elif o[0] == "dog": dog = True; milestones["Perro Pastor comprado"] = day
        if prestige_day is None and P == MAX_PLOTS and silo == len(SILO) - 1 and hives == 4 and dog:
            prestige_day = day
            milestones["Prestigio disponible"] = day
        hist.append(dict(day=day, focus=round(f), crop=crop[0], water=round(water, 1), coins=round(coins, 1),
                         plots=P, hives=hives, dog=int(dog), silo=silo, M=round(M, 2), income=round(burn * M, 1)))
    return hist, milestones, prestige_day


def all_results(days=200):
    return {k: run(F, sm, days) for k, (F, sm) in PROFILES.items()}


if __name__ == "__main__":
    res = all_results()
    if len(sys.argv) > 1 and sys.argv[1] == "json":
        print(json.dumps({k: dict(hist=h, milestones=m, prestige=p) for k, (h, m, p) in res.items()}, ensure_ascii=False))
    else:
        print("colmenas", [hive_cost(i) for i in range(4)], "perro", DOG_COST)
        print("\nEstación 2 (tras el Prestigio): días hasta volver a tener 4 colmenas y el Perro")
        for k, (F, sm) in PROFILES.items():
            _, _, d2 = run(F, sm, 400, season=2)
            _, _, d3 = run(F, sm, 400, season=3)
            print(f"   {k}: estación 2 -> {d2} días · estación 3 -> {d3} días")
        print()
        for k, (h, m, p) in res.items():
            print(k, "-> prestigio disponible el día", p)
            for a, b in sorted(m.items(), key=lambda x: x[1]): print("   ", b, a)
