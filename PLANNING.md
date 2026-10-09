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

- [x] **P1-07** Notificación y sonido de fin — agente: `frontend-3d` — dep: P1-06
  - [x] P1-07a Permiso de Notification en el primer Pomodoro; alerta sonora suave
  - **Hecho:** al acabar con la pestaña en segundo plano llega aviso.

- [x] **P1-08** Verificación de Fase 1 — agente: `qa-tester` — dep: P1-01..P1-07
  - [x] P1-08a Criterios GDD §8: cierre/vuelta, dos dispositivos, hora del cliente alterada
  - [x] P1-08b Bugs hallados y corregidos: doble clic en Plantar, aviso de conexión con 502, contrastes AA, dock en móvil (ver `docs/qa/fase-1.md`)
  - [x] P1-08c WebKit 26.6 y Firefox 155 pasan el humo (carga, plantar, cuenta atrás, recarga, lienzo 3D)
  - [ ] P1-08d Probar en el iPhone real por Tailscale (táctil, Safari, web instalada como app)
  - **Hecho:** informe de QA con resultados reales en `docs/qa/fase-1.md`; bugs registrados como subtareas.

---

## Fase 2 — Economía y granja

- [x] **P2-01** Resolución offline de monedas — agente: `backend-go` — dep: P1-03
  - [x] P2-01a Algoritmo por tramos con eventos madura/marchita y tope de Silo (GDD §6.2), 🪙 en milésimas
  - [x] P2-01b Tests: ejemplo 6.2 (212,7 y 127,0), Silo lleno, ausencia 90 días, madura+marchita en la misma ausencia, reloj retrocedido
  - [x] P2-01c Integrar al conectar y heartbeat de `last_seen_at`
  - [x] P2-01d (decisión) Cobro manual con el Silo: las 🪙 van al Silo y se recogen con un toque; el Perro lo hará solo. Migración 0002 (`silo_micro`, `silo_peak_micro_h`), `POST /api/silo/collect`, `silo` en el estado
  - [x] P2-01e UI mínima: saldo de 🪙, barra del Silo y botón Recoger (el resto de la granja 3D llega en P2-07)
  - **Hecho:** todos los casos del GDD §6.2 pasan.

- [x] **P2-02** Vida útil y marchitamiento — agente: `backend-go` — dep: P2-01
  - [x] P2-02a Estados growing → mature → withered; retirar marchita gratis, arrancar viva con confirmación
  - [x] P2-02b (añadida) La marchita conserva su 💧 (se cosecha igual); protegida de sembrar/retirar antes de cosecharla; el dock muestra cuánta vida le queda
  - [x] P2-02c (resuelta en P2-03) Con varias parcelas el dock único no basta: cosechar/retirar debe poder hacerse sobre cada planta (clic en la parcela), no solo en el dock
  - **Hecho:** tests de transición de estado.

- [x] **P2-03** Parcelas y Silo — agente: `backend-go` — dep: P2-01
  - [x] P2-03a Compra de parcelas (2–16) con `ceil(3·1,4^(n−2))`
  - [x] P2-03b Niveles de Silo en horas (GDD §4.5)
  - [x] P2-03c (añadida) Juego multi-parcela: la parcela seleccionada (clic en el 3D) decide qué ofrece el dock; orden de desbloqueo con el 2×2 central primero; tienda "Mejoras" con precios del servidor
  - **Hecho:** compras cuadran con las tablas del GDD.

- [x] **P2-04** Manzano, Roble y Modo Flow — agente: `backend-go` — dep: P2-03
  - [x] P2-04a Semillas 100/250 💧; Flow 60–120 min con 💧 = 25·(d/60)^1,5
  - [x] P2-04b (añadida) Deslizante de Flow en la UI con vista previa servida por el backend (`GET /api/flow`); vida útil continua (sin truncar a horas); duración en horas en el temporizador, el anillo y el título
  - **Hecho:** tabla Flow del GDD §4.3 reproducida en test.

- [x] **P2-05** Sinergias — agente: `backend-go` — dep: P2-03
  - [x] P2-05a Anillo de compatibilidad, adyacencia +10%/vecino (tope +40%), Huerto completo +15%, tope ×2,0
  - [x] P2-05b (añadida) La liquidación corta la línea de tiempo en los eventos de los vecinos (el multiplicador cambia al madurar/marchitarse uno); insignias flotantes (+N%, doradas en Huerto completo), multiplicador en el dock y combinaciones en cada sobre
  - [ ] P2-05c (pendiente de Fase 3) Abejas: `beeCells()` devuelve vacío hasta entonces; el cálculo ya admite +25% sin acumular
  - **Hecho:** tests con mapas de ejemplo.

- [x] **P2-06** Descansos — agente: `backend-go`, `frontend-3d` — dep: P1-06
  - [x] P2-06a Descanso proporcional, saltable y configurable (GDD §3.2)
  - [x] P2-06b (añadida) Descanso en el servidor (migración 0003), ventana de 15 min, aviso al terminar con cuenco y notificación, ajustes de activación y de las tres duraciones; el audio se desbloquea con el primer gesto de cualquier tipo
  - **Hecho:** descanso aparece tras cosechar y se puede saltar.

- [x] **P2-07** Granja 3D completa — agente: `frontend-3d` — dep: P2-03
  - [x] P2-07a Cuadrícula de hasta 4×4 con `InstancedMesh` (suelo y parcelas), Silo 3D con indicador de llenado y clic para recoger, contador de 🪙. 16 plantas maduras: de 409 a 60 llamadas de dibujo
  - [x] P2-07b Iconos de sinergia (etiquetas en la esquina de cada parcela) y panel de compras con 💧
  - [x] P2-07c Comprar con 🪙: colmenas y Perro (P3-01), decoración (P3-02)
  - **Hecho:** granja de 16 parcelas fluida en el navegador.

- [x] **P2-08** Verificación de Fase 2 — agente: `qa-tester` — dep: P2-01..P2-07
  - [x] P2-08a Reproducir criterios económicos y comparar con xlsx: ingreso estable 1.199,1 🪙/día = fórmula del GDD; sim.py = xlsx = 1.798,7
  - [x] P2-08b Tests de actualización desde el esquema 1, jugadas aleatorias con invariantes, entrada basura, teclado/lectores de pantalla, tres navegadores (ver `docs/qa/fase-2.md`)
  - [x] P2-08c Hallazgos corregidos: elegir parcela con el teclado (flechas + anuncio), mínimo de 0,1 🪙 para recoger
  - **Hecho:** informe de QA en `docs/qa/fase-2.md`.

- [x] **P2-09** Despliegue de la Fase 2 en `wyse` (2026-10-07) — agente: `devops`
  - [x] Copia previa de la BD de `wyse` (`~/pomofarm-backups/`), verificada (integridad, jugador, parcela, historial)
  - [x] `deploy/deploy.sh`: esquema 1 → 3 sin pérdida; la partida (margarita madura sin cosechar) intacta; web sin errores

---

## Fase 3 — Automatización y estética

- [x] **P3-01** Abejas y Perro Pastor — agente: `backend-go` — dep: P2-05
  - [x] P3-01a Estructuras en `structures`; Abejas (área 3×3, +25%, sin acumular) 4.000/6.000/9.000/13.500 🪙
  - [x] P3-01b Perro (30.000 🪙, recoge solo, +12 h de Silo)
  - [x] P3-01c Frontend: tienda (desbloqueo con 💧, compra con 🪙), colocación con vista previa de cobertura y confirmación, mover gratis tocando la colmena, colmena y Perro en 3D
  - **Hecho:** tests de bonos y recogida automática; E2E `p3_01_automation.py`; GDD decisión 13.

- [x] **P3-02** Decoración — agente: `backend-go`, `frontend-3d` — dep: P3-01
  - [x] P3-02a Caminos, vallas, farolillos, sombrero del Perro; colocación en el terreno de alrededor (cuadrícula 8×8), mover/quitar gratis
  - **Hecho:** se compran y colocan; no afectan a requisitos de prestigio. Tests `decor_test.go`, E2E `p3_02_decor.py`, GDD decisión 14.

- [x] **P3-03** Audio ASMR por capas — agente: `frontend-3d` — dep: P1-06
  - [x] P3-03a Efectos de interacción (plantar, cosechar, monedas, comprar, colocar), ambiente local (lluvia/bosque/fuego), volumen independiente por capa
  - **Hecho:** tres controles de volumen funcionan (medido con un analizador por capa). E2E `p3_03_audio.py`, GDD decisión 15.

- [x] **P3-04** Reproductor Lofi — agente: `frontend-3d` — dep: P3-03
  - [x] P3-04a Iframe YouTube con IDs configurables, URL propia, fallback local de lluvia/bosque/fuego
  - **Hecho:** si el iframe falla, suena el audio local. E2E `p3_04_lofi.py` (API simulada + prueba contra YouTube real), GDD decisión 16.

- [x] **P3-05** Verificación de Fase 3 — agente: `qa-tester` — dep: P3-01..P3-04
  - **Hecho:** informe en `docs/qa/fase-3.md`; E2E `p3_05_qa.py` (teclado, rendimiento, dos dispositivos, servidor caído, WebKit y Firefox); 442 comprobaciones E2E, 3 mutaciones detectadas. Corregido: decorar con teclado, aviso de terreno lleno.
  - [ ] P3-05b (pendiente de ti) Probar en iPhone real: audio, iframe de YouTube y rendimiento (P1-08d)
  - [ ] P3-05c (decisión tuya) Sumidero de 🪙: la decoración solo absorbe ~3.680 🪙 (ver informe, observación 1)

---

## Fase 4 — Retención

- [x] **P4-01** Etiquetas y Libro de Cosechas — agente: `backend-go`, `frontend-3d` — dep: P1-06
  - [x] P4-01a Estadísticas por etiqueta y día de la semana (`GET /api/book`, zona horaria del jugador)
  - [x] P4-01b Visualización (fardos/silos/cestas) + vista tabla + export CSV
  - **Hecho:** CSV exportado coincide con los Pomodoros guardados (E2E `p4_01_book.py`, tests `book_test.go`, GDD decisión 17).

- [x] **P4-02** Métricas personales — agente: `backend-go` — dep: P4-01
  - [x] P4-02a Pomodoros/semana, mejor racha, "Pomodoros limpios" (modo estricto opcional)
  - **Hecho:** `GET /api/stats`, migración 6 (`strict`), tests `stats_test.go` y `stats_test.go` del juego, E2E `p4_02_stats.py`, GDD decisión 18. Tres mutaciones detectadas.

- [x] **P4-03** Prestigio por estaciones — agente: `backend-go` — dep: P3-01
  - [x] P4-03a Requisitos, qué se reinicia/conserva, +10% 🪙 por estación (GDD §4.9)
  - [x] P4-03b Nuevo bioma (elegido: verano dorado; ciclo de estaciones)
  - **Hecho:** test de reinicio conserva exactamente lo que dice el GDD (`prestige_test.go`, 6 mutaciones detectadas), E2E `p4_03_prestige.py`, GDD decisión 19, `sim.py` simula la estación 2 (Normal: 41 días).

- [x] **P4-04** Accesibilidad y vista 2D — agente: `frontend-3d` — dep: P2-07
  - [x] P4-04a Reducir animaciones, modo sin audio (aviso visual), daltonismo, atajo M
  - [-] P4-04b Vista 2D ligera — **descartada por el usuario** (2026-10-08)

  - **Hecho:** E2E `p4_04_a11y.py`, tests de contraste y simulación de daltonismo con la hoja de estilos real, GDD decisión 20.

- [x] **P4-05** Verificación de Fase 4 — agente: `qa-tester` — dep: P4-01..P4-04
  - **Hecho:** informe en `docs/qa/fase-4.md`; tests `qa_phase4_test.go` (horario de verano, fronteras de mes, 8.760 Pomodoros, etiquetas raras), prestigio en las jugadas aleatorias, E2E `p4_05_qa.py`; 586 comprobaciones E2E.

---

## Fase 5 — Robustez

- [x] **P5-01** Modo Foco y rendimiento — agente: `frontend-3d` — dep: P2-07
  - [x] P5-01a 15–30 FPS durante el Pomodoro (20), render pausado en pestaña oculta, sombras a 2 Hz, resolución 1×
  - **Hecho:** E2E `p5_01_focus.py` (FPS medidos, CPU 67 % frente a 101 % con render por software, pestaña oculta = 0 FPS), GDD decisión 22.
- [x] **P5-02** Notificación "tu granja necesita atención" — agente: `backend-go`, `frontend-3d` — dep: P2-02
  - **Hecho:** aviso calculado en el navegador (el servidor no empuja nada; con el navegador cerrado depende de P5-06): ≥ 3 plantas en 2 h, una vez por grupo, 6 h de calma, nunca durante un Pomodoro. Tests `attention.test.ts`, E2E `p5_02_attention.py`, GDD decisión 23.
- [ ] **P5-03** Suite de pruebas de tiempo completa — agente: `qa-tester` — dep: P2-01
- [ ] **P5-04** Revisión de backups y restauración — agente: `devops` — dep: P0-05
- [ ] **P5-06** Notificaciones con el navegador cerrado (Web Push) — agente: `backend-go`, `frontend-3d` — dep: P1-07 — *valorar si hace falta*
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
