# PomoFarm 3D — Instrucciones para Claude

App web full-stack: temporizador Pomodoro + granja idle en 3D isométrico. El tiempo real de foco es la única fuente de progreso. Un solo jugador por instalación, desplegado en local (Docker + Tailscale).

## Fuentes de verdad (en orden)

1. `gdd_pomofarm_3d_v2.md` — diseño vigente. **Sustituye** a `gdd_pomofarm.md` (histórico, no seguir).
2. `pomofarm_balance.xlsx` — cifras económicas (fórmulas vivas). Si cambia un número en el GDD, cambia aquí y viceversa.
3. `PLANNING.md` — tareas y subtareas. Es el estado del proyecto.

Si el código contradice al GDD, gana el GDD, salvo que el usuario diga lo contrario. Si el GDD es ambiguo o se contradice, **pregunta** antes de inventar.

## Stack

- Backend: Go + Fiber, SQLite en modo WAL (driver `modernc.org/sqlite`, sin CGO), migraciones SQL numeradas.
- Frontend: React + Vite + Zustand (TypeScript).
- 3D: React Three Fiber + Three.js (cámara ortográfica isométrica, low-poly, `InstancedMesh`).
- Despliegue: **todo corre en Docker** (también en desarrollo cuando sea práctico). Destino: el minipc **`wyse`** (Linux, Tailscale `100.102.106.119`), accesible solo por la tailnet. Docker Compose (Nginx + app Go), volúmenes `/data` y `/backups`.

## Estructura prevista

```
backend/    Go: cmd/, internal/{game,timer,store,api}, migrations/
frontend/   React: src/{scene,ui,store,audio}
deploy/     docker-compose.yml, nginx.conf
tools/      scripts (validación del xlsx; sim.py vive en la raíz)
.claude/    agents/ y commands/
```

## Reglas inamovibles (del GDD)

- **El servidor es la única fuente de verdad.** Nunca se usa el reloj del cliente. El tiempo restante se calcula con timestamps (`planned_s − (ahora − started_at − paused_total_s)`), jamás con contadores incrementales.
- **Sin cron jobs.** La producción offline se resuelve al conectar (GDD 6.2), por tramos entre eventos, con tope de Silo.
- **Modelo indulgente:** nada que el jugador ya tenía se le quita. Sin penalizaciones, sin "¿te concentraste?".
- 🪙 se guarda en **milésimas (INTEGER)**. Nada de floats para dinero.
- Un solo Pomodoro activo (índice parcial `one_active_pomodoro`). Control optimista con `version`.
- Coste de semilla = desbloqueo único, no gasto por plantado.
- Los valores de balance (precios, vidas útiles, fórmulas) viven en **un único módulo de configuración** que refleja el xlsx; no se hardcodean dispersos.

## Cómo trabajar

- **Una tarea de `PLANNING.md` cada vez.** Cuando el usuario diga "siguiente tarea" o un ID (p. ej. `P1-03`): léela, márcala `[~]`, impleméntala por subtareas, verifica, márcala `[x]` y para. No encadenes tareas sin que se te pida.
- Antes de dar algo por hecho, **ejecútalo**: tests, build, `docker compose up`. Di claramente lo que no pudiste verificar.
- Si descubres trabajo nuevo, añádelo a `PLANNING.md` como subtarea; no lo hagas a escondidas.
- Delega en los agentes de `.claude/agents/` cuando la tarea encaje en su especialidad (ver abajo).
- Commits pequeños y por tarea, formato `tipo(scope): mensaje` (`feat(timer): ...`). Un commit por tarea terminada. **No hagas push sin que se pida.**
- Idioma: respuestas y documentación en español; código, identificadores y commits en inglés.
- Nada de dependencias nuevas sin justificarlo en una línea.

## Pruebas obligatorias (del GDD §8)

El ejemplo de GDD 6.2 (Silo 24 h → 212,7 🪙; Silo 12 h → 127,0 🪙), reloj del sistema retrocedido, planta que madura y se marchita en la misma ausencia, Silo lleno, ausencia de 90 días, dos dispositivos con un Pomodoro activo.

## Agentes (`.claude/agents/`)

| Agente | Úsalo para |
| :--- | :--- |
| `game-designer` | Interpretar el GDD, balance, validar cifras contra el xlsx, resolver ambigüedades |
| `backend-go` | API Fiber, SQLite, migraciones, temporizador, resolución offline |
| `frontend-3d` | React, Zustand, escena R3F, UI, audio |
| `qa-tester` | Tests de tiempo/economía, criterios de aceptación, verificación en navegador |
| `devops` | Docker Compose, Nginx, backups, despliegue en `wyse` por Tailscale |
| `planner` | Mantener `PLANNING.md`, desglosar tareas, detectar dependencias |

Comando: `/next-task [ID]` ejecuta la siguiente tarea pendiente (o la indicada).

## Despliegue (wyse)

- Las imágenes se construyen para la arquitectura de `wyse` (comprobar con `uname -m` allí; probablemente `amd64`).
- Nunca expongas puertos a internet; solo la tailnet. Datos persistentes en volúmenes (`/data`, `/backups`), nunca dentro de la imagen.
- **Conectarse o desplegar en `wyse` (ssh, `docker context`, `docker compose` remoto) es una acción sobre otra máquina: pide confirmación la primera vez de cada sesión** y dime qué vas a ejecutar.

## `sim.py`

Está en la raíz. Es la simulación de referencia del GDD (`python3 sim.py` / `python3 sim.py json`). Si cambia un parámetro de balance, cambia a la vez `sim.py`, el xlsx y `internal/game/config.go`.
