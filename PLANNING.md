# PomoFarm 3D — Planning

Estados: `[ ]` pendiente · `[~]` en curso · `[x]` hecha · `[!]` bloqueada.
Se ejecuta **una tarea cada vez** (`/next-task` o "haz P1-03"). Una tarea solo se marca `[x]` si su criterio de **Hecho** se ha verificado.
Fases = GDD §7. Referencias `GDD §n` apuntan a `gdd_pomofarm_3d_v2.md`.

---

## Fase 0 — Preparación

- [x] **P0-01** Toolchain y acceso a `wyse` — agente: `devops` — dep: —
  - [x] P0-01a Go instalado (1.26.0)
  - [x] P0-01b `docker compose` v5.1.4 en local
  - [x] P0-01c Comprobado en `wyse`: amd64, Docker 29.0.1 + Compose v2.40.3, 429 GB libres, SSH como `luk`
  - [x] P0-01d Documentar versiones y datos de `wyse` en `README.md`
  - **Hecho:** `go version`, `docker compose version`, `node -v` responden y `wyse` es alcanzable con Docker funcionando.

- [x] **P0-02** Estructura del proyecto y esqueletos — agente: `backend-go`, `frontend-3d` — dep: P0-01
  - [x] P0-02a `backend/`: `go mod init`, Fiber, endpoint `GET /api/health`
  - [x] P0-02b `frontend/`: Vite + React + TS + Zustand + R3F, página en blanco con canvas
  - [x] P0-02c `Makefile` (`make dev`, `make test`, `make build`) y `.gitignore`
  - **Hecho:** `make test` y `make build` pasan; `/api/health` devuelve 200; el frontend arranca.

- [x] **P0-03** Esquema de BD y migraciones — agente: `backend-go` — dep: P0-02
  - [x] P0-03a `migrations/0001_init.sql` con el esquema de GDD §5 (incluido `one_active_pomodoro`)
  - [x] P0-03b Runner de migraciones + tabla `schema_migrations`; SQLite en WAL
  - [x] P0-03c Test: aplica migraciones sobre BD vacía y rechaza un segundo Pomodoro activo
  - **Hecho:** `go test ./internal/store/...` pasa.

- [x] **P0-04** Módulo de configuración de balance — agente: `game-designer` + `backend-go` — dep: P0-02
  - [x] P0-04a Extraer del xlsx: cultivos, parcelas, Silo, sinergias, estructuras (script en `tools/`)
  - [x] P0-04b `internal/game/config.go` con esos valores como única fuente
  - [x] P0-04c Test que compara e(d), costes de parcela y totales (1.167 💧 / 605 💧 / 62.500 🪙) con el GDD
  - **Hecho:** el test de coherencia pasa.

- [x] **P0-05** Docker Compose, backups y despliegue en `wyse` — agente: `devops` — dep: P0-01, P0-02, P0-03
  - [x] P0-05a Dockerfile multi-stage backend y frontend; `docker-compose.yml` (Nginx + app) con volúmenes `/data`, `/backups`; imagen para la arquitectura de `wyse`
  - [x] P0-05d Script `deploy/deploy.sh` (build → copiar/pull → `docker compose up -d` en `wyse`), con confirmación previa
  - [x] P0-05e Primer despliegue "hola mundo" en `wyse` accesible desde la tailnet (`/api/health`)
  - [x] P0-05b Backup diario con `VACUUM INTO`, retención de 14
  - [x] P0-05c Script de restauración y prueba en instalación limpia
  - **Hecho:** `docker compose up -d --build` sirve la app en local y en `wyse`; backup y restore verificados.

- [x] **P0-06** Validar el balance con foco real del usuario — agente: `game-designer` — dep: —
  - [x] P0-06a Pedir al usuario sus horas de foco reales y rellenar hoja `Rendimiento`
  - [x] P0-06b `sim.py` ya está en la raíz del repo (v2)
  - [x] P0-06c Ejecutar `python3 sim.py` y comprobar que coincide con el GDD §4.10 y el xlsx
  - **Hecho:** el usuario confirma los perfiles; `sim.py` reproduce los hitos del GDD.

---

## Fase 1 — MVP jugable

*1 parcela, Margarita/Tomates/Girasol, timer con timestamps, cosecha manual, 💧, persistencia, escena 3D mínima.*

- [x] **P1-01** Dominio del temporizador — agente: `backend-go` — dep: P0-03
  - [x] P1-01a Interfaz `Clock` inyectable
  - [x] P1-01b Funciones puras: restante, pausa, reanudación, cancelación, finalización (GDD §6.1)
  - [x] P1-01c Tests: pausa/reanuda, cierre y vuelta ±1 s, reloj retrocedido
  - **Hecho:** tests de tiempo pasan.

- [x] **P1-02** API de Pomodoro — agente: `backend-go` — dep: P1-01
  - [x] P1-02a `POST /pomodoros` (plantar), `/pause`, `/resume`, `/cancel`, `GET /state`
  - [x] P1-02b Un solo Pomodoro activo (409 si ya hay uno); `version` optimista
  - [x] P1-02c `POST /plots/:id/harvest` otorga 💧 (recompensa de GDD §4.3)
  - **Hecho:** tests de API incluyen "dos dispositivos, un Pomodoro".

- [x] **P1-03** Estado inicial y FTUE de datos — agente: `backend-go` — dep: P1-02
  - [x] P1-03a Crear jugador, 1 parcela y Margarita desbloqueada al primer arranque
  - [x] P1-03b Desbloqueo de semillas con 💧 (Tomates 8, Girasol 35): `POST /unlocks`
  - **Hecho:** BD nueva → estado jugable por API.

- [x] **P1-04** Cliente de estado y temporizador — agente: `frontend-3d` — dep: P1-02
  - [x] P1-04a Cliente API tipado + store Zustand
  - [x] P1-04b Hook de tiempo restante (servidor manda, `setInterval` solo refresca) y título de pestaña
  - **Hecho:** recargar la página a mitad de Pomodoro mantiene el tiempo exacto.

- [x] **P1-05** Escena isométrica mínima — agente: `frontend-3d` — dep: P0-02
  - [x] P1-05a Cámara ortográfica isométrica con zoom y paneo
  - [x] P1-05c Dirección de arte: elegida la opción A (3D procedural, paleta cálida en `scene/palette.ts`). Kenney (CC0) queda para la decoración de la Fase 3. Tipografía y "elemento firma" de la UI pasan a P1-06
  - [x] P1-05b Parcela y planta low-poly por etapa; barra circular flotante del timer
  - **Hecho:** escena visible en el navegador con una planta animada.

- [x] **P1-06** UI de juego (plantar, pausar, cosechar) — agente: `frontend-3d` — dep: P1-04, P1-05
  - [x] P1-06a Selector de semilla y de etiqueta opcional (GDD §2.2)
  - [x] P1-06b Botones pausar/reanudar/cancelar; cosecha al hacer clic en planta madura
  - [x] P1-06c Tutorial de 3 pasos y primer Pomodoro gratis (GDD §2.1)
  - [x] P1-06d (añadida) Retirar planta cosechada, para poder replantar con una sola parcela; etiquetas recientes y ajustes en el estado
  - **Hecho:** flujo completo plantar → esperar → cosechar → 💧 jugable.

- [ ] **P1-07** Notificación y sonido de fin — agente: `frontend-3d` — dep: P1-06
  - [ ] P1-07a Permiso de Notification en el primer Pomodoro; alerta sonora suave
  - **Hecho:** al acabar con la pestaña en segundo plano llega aviso.

- [ ] **P1-08** Verificación de Fase 1 — agente: `qa-tester` — dep: P1-01..P1-07
  - [ ] P1-08a Criterios GDD §8: cierre/vuelta, dos dispositivos, hora del cliente alterada
  - **Hecho:** informe de QA con resultados reales; bugs registrados como subtareas.

---

## Fase 2 — Economía y granja

- [ ] **P2-01** Resolución offline de monedas — agente: `backend-go` — dep: P1-03
  - [ ] P2-01a Algoritmo por tramos con eventos madura/marchita y tope de Silo (GDD §6.2), 🪙 en milésimas
  - [ ] P2-01b Tests: ejemplo 6.2 (212,7 y 127,0), Silo lleno, ausencia 90 días, madura+marchita en la misma ausencia, reloj retrocedido
  - [ ] P2-01c Integrar al conectar y heartbeat de `last_seen_at`
  - **Hecho:** todos los casos del GDD §6.2 pasan.

- [ ] **P2-02** Vida útil y marchitamiento — agente: `backend-go` — dep: P2-01
  - [ ] P2-02a Estados growing → mature → withered; retirar marchita gratis, arrancar viva con confirmación
  - **Hecho:** tests de transición de estado.

- [ ] **P2-03** Parcelas y Silo — agente: `backend-go` — dep: P2-01
  - [ ] P2-03a Compra de parcelas (2–16) con `ceil(3·1,4^(n−2))`
  - [ ] P2-03b Niveles de Silo en horas (GDD §4.5)
  - **Hecho:** compras cuadran con las tablas del GDD.

- [ ] **P2-04** Manzano, Roble y Modo Flow — agente: `backend-go` — dep: P2-03
  - [ ] P2-04a Semillas 100/250 💧; Flow 60–120 min con 💧 = 25·(d/60)^1,5
  - **Hecho:** tabla Flow del GDD §4.3 reproducida en test.

- [ ] **P2-05** Sinergias — agente: `backend-go` — dep: P2-03
  - [ ] P2-05a Anillo de compatibilidad, adyacencia +10%/vecino (tope +40%), Huerto completo +15%, tope ×2,0
  - **Hecho:** tests con mapas de ejemplo.

- [ ] **P2-06** Descansos — agente: `backend-go`, `frontend-3d` — dep: P1-06
  - [ ] P2-06a Descanso proporcional, saltable y configurable (GDD §3.2)
  - **Hecho:** descanso aparece tras cosechar y se puede saltar.

- [ ] **P2-07** Granja 3D completa — agente: `frontend-3d` — dep: P2-03
  - [ ] P2-07a Cuadrícula de hasta 4×4 con `InstancedMesh`, silo y contador de 🪙
  - [ ] P2-07b Iconos de sinergia, panel de compras con 💧 y 🪙
  - **Hecho:** granja de 16 parcelas fluida en el navegador.

- [ ] **P2-08** Verificación de Fase 2 — agente: `qa-tester` — dep: P2-01..P2-07
  - [ ] P2-08a Reproducir criterios económicos y comparar con xlsx (Normal: 1.798,7 🪙/día)
  - **Hecho:** informe de QA.

---

## Fase 3 — Automatización y estética

- [ ] **P3-01** Abejas y Perro Pastor — agente: `backend-go` — dep: P2-05
  - [ ] P3-01a Estructuras en `structures`; Abejas (área 3×3, +25%, sin acumular) 4.000/6.000/9.000/13.500 🪙
  - [ ] P3-01b Perro (30.000 🪙, recoge solo, +12 h de Silo)
  - **Hecho:** tests de bonos y recogida automática.

- [ ] **P3-02** Decoración — agente: `backend-go`, `frontend-3d` — dep: P3-01
  - [ ] P3-02a Caminos, vallas, farolillos, accesorios; colocación en la escena
  - **Hecho:** se compran y colocan; no afectan a requisitos de prestigio.

- [ ] **P3-03** Audio ASMR por capas — agente: `frontend-3d` — dep: P1-06
  - [ ] P3-03a Efectos de interacción, volumen independiente por capa
  - **Hecho:** tres controles de volumen funcionan.

- [ ] **P3-04** Reproductor Lofi — agente: `frontend-3d` — dep: P3-03
  - [ ] P3-04a Iframe YouTube con IDs configurables, URL propia, fallback local de lluvia/bosque/fuego
  - **Hecho:** si el iframe falla, suena el audio local.

- [ ] **P3-05** Verificación de Fase 3 — agente: `qa-tester` — dep: P3-01..P3-04

---

## Fase 4 — Retención

- [ ] **P4-01** Etiquetas y Libro de Cosechas — agente: `backend-go`, `frontend-3d` — dep: P1-06
  - [ ] P4-01a Estadísticas por etiqueta y día de la semana
  - [ ] P4-01b Visualización (fardos/silos/cestas) + vista tabla + export CSV
  - **Hecho:** CSV exportado coincide con los Pomodoros guardados.

- [ ] **P4-02** Métricas personales — agente: `backend-go` — dep: P4-01
  - [ ] P4-02a Pomodoros/semana, mejor racha, "Pomodoros limpios" (modo estricto opcional)

- [ ] **P4-03** Prestigio por estaciones — agente: `backend-go` — dep: P3-01
  - [ ] P4-03a Requisitos, qué se reinicia/conserva, +10% 🪙 por estación (GDD §4.9)
  - [ ] P4-03b Nuevo bioma (pendiente: el usuario elige el de la 2.ª estación)
  - **Hecho:** test de reinicio conserva exactamente lo que dice el GDD.

- [ ] **P4-04** Accesibilidad y vista 2D — agente: `frontend-3d` — dep: P2-07
  - [ ] P4-04a Reducir animaciones, modo sin audio, daltonismo, atajos
  - [ ] P4-04b Vista 2D ligera

- [ ] **P4-05** Verificación de Fase 4 — agente: `qa-tester` — dep: P4-01..P4-04

---

## Fase 5 — Robustez

- [ ] **P5-01** Modo Foco y rendimiento — agente: `frontend-3d` — dep: P2-07
  - [ ] P5-01a 15–30 FPS durante el Pomodoro, render pausado en pestaña oculta, menos sombras
- [ ] **P5-02** Notificación "tu granja necesita atención" — agente: `backend-go`, `frontend-3d` — dep: P2-02
- [ ] **P5-03** Suite de pruebas de tiempo completa — agente: `qa-tester` — dep: P2-01
- [ ] **P5-04** Revisión de backups y restauración — agente: `devops` — dep: P0-05
- [ ] **P5-05** Despliegue final en `wyse` y cierre — agente: `devops` — dep: P5-04
  - [ ] P5-05a Despliegue estable con reinicio automático tras reiniciar `wyse`
  - [ ] P5-05b Backups corriendo en `wyse` y restauración probada allí
  - **Hecho:** la app responde desde otro dispositivo de la tailnet (iPhone incluido); README de despliegue.

---

## Pendientes del usuario (GDD §10)

- [ ] Confirmar las 7 decisiones tomadas en el GDD
- [x] Ajustar perfiles de foco reales en la hoja `Rendimiento` (2–4 h: perfiles "Tú mín" y "Tú máx")
- [ ] Elegir el bioma de la segunda estación
- [ ] Jugar una semana y recalibrar precios antes de darlos por buenos
