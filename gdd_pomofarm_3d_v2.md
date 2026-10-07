# 📝 Documento de Diseño de Juego (GDD): PomoFarm 3D — v2

> **Estado:** v2 integrada. Sustituye al GDD original y a `soluciones_gdd_pomofarm.md`.
> **Cifras:** todas las cifras económicas salen de `pomofarm_balance.xlsx` (fórmulas vivas) y de `sim.py` (simulación). Si cambias un número aquí, cámbialo allí.
> **Lo marcado como `[Decisión]`** es una elección de diseño que he tomado por ti y es reversible; la lista completa está en el apartado 10.

---

## 1. Visión General del Proyecto

PomoFarm 3D es una aplicación web full-stack que fusiona un temporizador Pomodoro con un juego de simulación de granjas y mecánicas de automatización (idle/clicker). Alojado en un servidor local (Docker + Linux + Tailscale), el jugador interactúa en una vista isométrica 3D para plantar, gestionar recursos y construir un ecosistema automatizado financiado por su tiempo de concentración en la vida real.

### 1.1 Principios de diseño

1. **El tiempo real de foco es la única fuente de progreso.** Sin Pomodoros no hay 💧, y sin replantar no hay 🪙 sostenidos. La renta pasiva es proporcional al foco (ver 4.2).
2. **Indulgente, nunca punitivo.** Ningún fallo (cerrar el navegador, pausar, olvidarse días) quita nada que el jugador ya tenía. Lo máximo que pasa es que la granja deja de producir.
3. **Honestidad del timer = confianza.** Es una herramienta personal: el juego no intenta adivinar si el jugador se concentró.
4. **Poca fricción durante el trabajo.** Interfaz discreta, 3D barato, sin notificaciones insistentes.

---

## 2. Dirección de Arte y UI/UX

*   **Cámara:** Isométrica fija (ortográfica) con zoom in/out y paneo.
*   **Estilo Visual:** Low-Poly y Cartoon. Superficies lisas, colores vibrantes, bordes redondeados. Iluminación suave con sombras proyectadas. Presupuesto orientativo: < 2.000 triángulos por planta.
*   **Animaciones:** Fluidas y exageradas (squash & stretch), con interruptor de **reducir animaciones** (ver 2.4).
*   **Visibilidad del Temporizador:** No intrusivo. En lugar de un HUD estático, el tiempo restante se muestra como una barra circular flotante sobre la planta activa. El título de la pestaña también muestra el tiempo restante.
*   **Audio y Experiencia Sensorial (ASMR):**
    *   *Alertas Suaves:* Fin de temporizador con campanillas tibetanas o trinos de aves.
    *   *Interacciones:* Sonidos "crujientes" y satisfactorios al cavar la tierra, regar o recolectar monedas.
    *   *Música (Minireproductor Lofi Girl):* Widget en una esquina integrado con la API de YouTube Iframe. Permite cambiar entre emisiones en directo del canal Lofi Girl, con volumen independiente. Los IDs de los streams son **configurables** y hay audio ambiental local de reserva (lluvia, bosque, fuego) si el iframe falla.
    *   Volumen independiente por capa: ambiente, efectos y alertas.

### 2.1 Primer minuto (FTUE)

1. Al abrir por primera vez: una parcela, una Margarita disponible, tutorial de tres pasos (elegir semilla → plantar → esperar).
2. Primer Pomodoro de 10 min, gratis. Al cosechar: 1 💧 y un cartel que enseña qué se puede comprar con él.
3. Tooltips contextuales para Silo, sinergias y descansos, solo la primera vez que aparece cada concepto.

### 2.2 Etiquetas de tarea

Campo opcional "¿En qué vas a trabajar?" al plantar, con autocompletado de etiquetas recientes. Alimenta el Libro de Cosechas (4.8).

### 2.3 Notificaciones y pestaña en segundo plano

Notificación web y sonido al terminar un Pomodoro (permiso solicitado en el primer Pomodoro). El temporizador no depende de que la pestaña esté activa (ver 6.1).

### 2.4 Accesibilidad y rendimiento

*   Modo sin audio con avisos visuales; paletas aptas para daltonismo; atajos de teclado (plantar, pausar, recolectar).
*   **Reducir animaciones** (desactiva el squash & stretch continuo).
*   **Modo Foco:** mientras corre un Pomodoro, el render baja a 15–30 FPS y reduce sombras; el render se pausa si la pestaña no está visible.
*   Vista 2D ligera como alternativa en equipos modestos (Fase 4).

---

## 3. Bucle de Jugabilidad (Gameplay Loop)

### 3.1 Capa Activa: el Pomodoro

El jugador selecciona una semilla desbloqueada y hace clic en una **parcela vacía**. Comienza la animación y el temporizador. Solo puede haber **un Pomodoro activo** por jugador.

*   Si no hay parcelas vacías, puede **arrancar** una planta (con confirmación). Arrancar una planta viva pierde su producción restante; arrancar una marchita es gratis y sin confirmación.

### 3.2 Cosecha y descanso

Al terminar el tiempo, la planta madura pero **no** se recolecta sola. El jugador hace clic en ella (ancla psicológica de fin de tarea) y obtiene sus **Puntos de Enfoque (💧)**. La planta **sigue en pie** y empieza a producir 🪙 hasta el fin de su vida útil.

Esto activa un temporizador de descanso **proporcional y opcional** `[Decisión]`:

| Duración del Pomodoro | Descanso |
| :--- | :--- |
| ≤ 25 min | 5 min |
| 26–45 min | 10 min |
| 46 min o más | 15 min |

Hay botón "Saltar descanso" y las duraciones son configurables en ajustes.

### 3.3 Pausas e interrupciones: modelo indulgente `[Decisión]`

*   Pausar a mano, cerrar el navegador o apagar el equipo **congela** el temporizador exactamente donde estaba. Al volver continúa.
*   **Sin límite de pausas, sin caducidad, sin penalización**. Una sesión congelada nunca se archiva sola ni pierde recompensa.
*   El jugador puede **cancelar** a mano un Pomodoro: la planta desaparece y no se otorga 💧; no hay otro coste. (Es una acción suya, no un castigo del juego.)
*   *Modo estricto (opcional, desactivado por defecto):* máximo 2 pausas y 10 minutos en total por sesión. Es un reto personal sin efecto económico; solo suma al contador de "Pomodoros limpios" del Libro de Cosechas.
*   No existe diálogo de confirmación del tipo "¿te concentraste?".

### 3.4 Capa Pasiva: vida útil e "Idle Cap"

*   Cada planta madura produce 🪙 durante su **vida útil** (4.3) y después **se marchita**: deja de producir y queda en la parcela hasta que se retire (gratis) o se replante.
*   **Silo de Monedas:** acumula las 🪙. Su capacidad se mide en **horas de producción** (4.5). Si se llena, la granja deja de producir hasta que el jugador entre a vaciarlo. Esto evita que la economía se rompa si no se entra en meses.
*   Una notificación suave ("tu granja necesita atención") avisa cuando varias plantas están por marchitarse. Nunca urgente.

---

## 4. Sistema de Progresión y Economía

### 4.1 Divisas

*   💧 **Puntos de Enfoque:** exclusivamente por completar Pomodoros. Se gastan en **desbloqueos únicos** de semillas, parcelas, mejoras del Silo y desbloqueo de animales.
*   🪙 **Monedas de Oro:** generadas pasivamente. Se gastan en automatizadores (colmenas, Perro Pastor), y en decoración.

> **Regla `[Decisión]`:** el coste de una semilla es un **desbloqueo permanente**, no un gasto por cada plantado. Con el reparto del GDD original (coste por plantado), plantar un Girasol, un Manzano o un Roble costaba más 💧 de los que daba (p. ej. Roble: 50 💧 de coste frente a 25 💧 de recompensa), así que el jugador solo habría plantado Margaritas y Tomates.

### 4.2 Invariante económico

Con todo el foco del día invertido en un cultivo de duración *d*, la renta pasiva en estado estable es:

> **🪙/día = (minutos de foco al día) × e(d) × M**

donde *e(d)* es el rendimiento por minuto de foco (4.3) y *M* el multiplicador de sinergias (4.6). Es independiente del número de parcelas mientras haya suficientes: **la renta es proporcional al tiempo real concentrado**. Las parcelas solo limitan cuánto de ese foco se puede mantener "vivo" a la vez (columna *parcelas sostenibles* de la hoja).

### 4.3 Tipos de Cultivos

Fórmula: **e(d) = 2 · (d/10)^0,8** (🪙 por minuto de foco). Ciclo: **Y = e · d** (🪙 totales por planta). Ritmo: **g = Y / vida útil** (🪙/h mientras vive).

| Planta | Pomodoro | Desbloqueo (💧, único) | Recompensa (💧) | Vida útil | 🪙 por min de foco | 🪙 por ciclo | 🪙/h en vida | Tarea ideal |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Margarita** | 10 min | Gratis | 1 | 24 h | 2,0 | 20 | 0,8 | Emails rápidos, organización |
| **Tomates** | 25 min | 8 | 5 | 36 h | 4,2 | 104 | 2,9 | Tareas estándar (técnica clásica) |
| **Girasol** | 35 min | 35 | 8 | 54 h | 5,5 | 191 | 3,5 | Desarrollo, redacción media |
| **Manzano** | 45 min | 100 | 15 | 72 h | 6,7 | 300 | 4,2 | Deep work, estudio intenso |
| **Roble Mágico** | 60 min | 250 | 25 | 108 h | 8,4 | 503 | 4,7 | Sesiones extremas |

El rendimiento por minuto de foco del Roble es 4,2 veces el de la Margarita (no 83 veces como en el reparto del GDD original), de modo que las sesiones cortas siguen mereciendo la pena.

#### Modo Flow (Roble Mágico, 60–120 min)

Control deslizante antes de plantar. Sin escalones ni bonos aparte:

*   💧 = 25 · (d/60)^1,5, redondeado. Vida útil = 108 h · d/60. Mismo *e(d)* que el resto.

| Duración | 💧 | Vida útil | 🪙 por ciclo |
| :--- | :--- | :--- | :--- |
| 60 min | 25 | 108 h | 503 |
| 75 min | 35 | 135 h | 752 |
| 90 min | 46 | 162 h | 1.044 |
| 105 min | 58 | 189 h | 1.378 |
| 120 min | 71 | 216 h | 1.752 |

### 4.4 Parcelas y mejoras de 💧

El jugador empieza con 1 parcela. La parcela *n* (de la 2 a la 16) cuesta `ceil(3 · 1,4^(n−2))` 💧:

| Parcela | 2 | 3 | 4 | 5 | 6 | 7 | 8 | 9 | 10 | 11 | 12 | 13 | 14 | 15 | 16 |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **💧** | 3 | 5 | 6 | 9 | 12 | 17 | 23 | 32 | 45 | 62 | 87 | 122 | 171 | 239 | 334 |

Total de las 15 parcelas: **1.167 💧**. Las primeras son baratas a propósito: un jugador necesita varias parcelas para no tener que arrancar plantas vivas a cada Pomodoro.

### 4.5 Silo

La capacidad se mide en **horas de producción** (al ritmo del momento), de modo que escala sola con la granja:

| Nivel | Capacidad | Coste (💧) |
| :--- | :--- | :--- |
| 0 | 12 h | Inicial |
| 1 | 24 h | 25 |
| 2 | 36 h | 70 |
| 3 | 48 h | 160 |
| 4 | 72 h | 350 |

Total: **605 💧**. El Perro Pastor añade **+12 h** permanentes. La hoja `Silo` muestra qué porcentaje de la producción se conserva según las horas de ausencia (p. ej. tras 72 h fuera, con Silo nivel 2 conservas el 50% de lo producible por un Roble; con Silo nivel 4, el 100%).

### 4.6 Sinergias y Adyacencia Espacial

Los cultivos forman un **anillo de compatibilidad**: Margarita – Tomates – Girasol – Manzano – Roble – (vuelta a Margarita). Un cultivo es compatible con sus dos vecinos del anillo. El Girasol junto al Manzano (ejemplo del GDD original) cumple la regla.

Los bonos se **suman** y luego se aplica un tope:

| Fuente | Bono |
| :--- | :--- |
| Adyacencia | +10% por cada vecino ortogonal compatible, **tope +40%** por parcela |
| Patrón **Huerto completo** | +15% a las cuatro parcelas de un 2×2 con 4 cultivos distintos |
| Abejas (área 3×3) | +25% a las parcelas del área; **no se acumulan** entre colmenas |
| **Tope total** por parcela | **×2,0** |
| Prestigio (4.9) | Multiplicador global **por encima** del tope: +10% por estación completada |

No hay penalización por monocultivo. Los bonos solo afectan a parcelas que están produciendo (maduras y no marchitas). Las sinergias activas se muestran como pequeños iconos flotantes.

Para la simulación se supone un bono medio de adyacencia de +15%, +10% de Huerto completo (desde 8 parcelas y 4 cultivos) y las Abejas en proporción a las colmenas colocadas: **M ≈ 1,5** con la granja completa.

### 4.7 Automatización y Decoración (🪙)

*   **Abejas:** se desbloquean con 30 💧; cada colmena cuesta 4.000 / 6.000 / 9.000 / 13.500 🪙. Con 4 se cubre la cuadrícula de 4×4.
*   **Perro Pastor:** se desbloquea con 120 💧; cuesta 30.000 🪙. Recoge automáticamente las 🪙 y añade +12 h de capacidad al Silo.
*   **Total de automatización: 62.500 🪙.**
*   **Economía estética:** caminos de piedra (15 🪙), vallas (25), farolillos (80), accesorios para animales (sombrero de paja del perro, 600). La decoración absorbe el exceso de 🪙 y **no** es requisito de prestigio.

### 4.8 Retención

*   **El Libro de Cosechas:** las estadísticas mensuales se muestran como fardos de heno, silos llenos o cestas de manzanas que representan horas de concentración. Incluye una vista de **tabla** opcional, con desglose por etiqueta y por día de la semana, y **exportación a CSV**.
*   **Métricas personales** (sin racha punitiva): Pomodoros completados por semana y mejor racha histórica (romperla no tiene coste).

### 4.9 Prestigio (Las Estaciones)

*   **Requisitos:** 16 parcelas, Silo en nivel 4, 4 colmenas y Perro Pastor.
*   **Se reinicia:** plantas, estructuras (colmenas, Perro, decoración) y 🪙.
*   **Se conserva:** saldo de 💧, desbloqueos de semillas y animales, parcelas compradas, nivel del Silo, cosméticos desbloqueados, y todo el Libro de Cosechas.
*   **Recompensa:** nuevo bioma (paleta de colores) y **+10% permanente de 🪙** por estación (aditivo: +10%, +20%…).
*   La segunda estación gira en torno a reconstruir los 62.500 🪙 de automatización: ≈ 30–35 días para un jugador "Normal" (estimación aritmética, no simulada).

### 4.10 Ritmo de progresión simulado

Cinco perfiles (los dos últimos son el rango real de uso del autor, 2–4 h al día; su 3 h equivale a "Normal"); los fines de semana cuentan como 30% del foco. Resultado de `sim.py` (juega siempre la opción más barata asequible):

| Perfil | Foco/día | Sesión máx. | Prestigio disponible |
| :--- | :--- | :--- | :--- |
| Ligero | 1,5 h | 25 min | día 163 |
| Normal | 3 h | 45 min | día 57 |
| Intenso | 5 h | 60 min | día 36 |
| **Tú mín** `[Decisión]` | 2 h | 45 min | día 83 |
| **Tú máx** `[Decisión]` | 4 h | 60 min | día 40 |

Hitos del perfil Normal: 8 parcelas el día 4, Girasol el 9, Manzano el 19, 16 parcelas el 49, Perro Pastor el 54, Roble Mágico el 42. La hoja `Hitos` tiene la tabla completa.

---

## 5. Modelo de datos

```sql
-- Importes de 🪙 en milésimas (INTEGER) para no perder fracciones.
CREATE TABLE players (
  id              INTEGER PRIMARY KEY,
  name            TEXT    NOT NULL,
  focus_points    INTEGER NOT NULL DEFAULT 0,
  lifetime_focus  INTEGER NOT NULL DEFAULT 0,
  coins_milli     INTEGER NOT NULL DEFAULT 0,
  silo_level      INTEGER NOT NULL DEFAULT 0,
  season          INTEGER NOT NULL DEFAULT 1,
  biome           TEXT    NOT NULL DEFAULT 'spring',
  created_at      TEXT    NOT NULL,
  last_seen_at    TEXT    NOT NULL,
  version         INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE unlocks (
  player_id INTEGER NOT NULL REFERENCES players(id),
  kind      TEXT    NOT NULL,       -- seed | animal | cosmetic
  key       TEXT    NOT NULL,       -- tomato | sunflower | bees | dog ...
  at        TEXT    NOT NULL,
  PRIMARY KEY (player_id, kind, key)
);

CREATE TABLE plots (
  id            INTEGER PRIMARY KEY,
  player_id     INTEGER NOT NULL REFERENCES players(id),
  x INTEGER NOT NULL, y INTEGER NOT NULL,
  plant_type    TEXT,
  state         TEXT NOT NULL DEFAULT 'empty', -- empty | growing | mature | withered
  planted_at    TEXT,
  grow_s        INTEGER,                       -- duración del Pomodoro
  matured_at    TEXT,
  harvested     INTEGER NOT NULL DEFAULT 0,
  life_s        INTEGER,                       -- vida útil
  wilts_at      TEXT,                          -- matured_at + life_s
  collected_to  TEXT,                          -- hasta cuándo se han contado sus 🪙
  version       INTEGER NOT NULL DEFAULT 0,
  UNIQUE (player_id, x, y)
);

CREATE TABLE tags (
  id INTEGER PRIMARY KEY,
  player_id INTEGER NOT NULL REFERENCES players(id),
  name TEXT NOT NULL,
  UNIQUE (player_id, name)
);

CREATE TABLE pomodoros (
  id            INTEGER PRIMARY KEY,
  player_id     INTEGER NOT NULL REFERENCES players(id),
  plot_id       INTEGER REFERENCES plots(id),
  plant_type    TEXT    NOT NULL,
  tag_id        INTEGER REFERENCES tags(id),
  planned_s     INTEGER NOT NULL,
  started_at    TEXT    NOT NULL,
  paused_at     TEXT,                    -- no nulo mientras está pausado
  paused_total_s INTEGER NOT NULL DEFAULT 0,
  ended_at      TEXT,
  reward_focus  INTEGER NOT NULL DEFAULT 0,
  status        TEXT    NOT NULL         -- running | paused | completed | cancelled
);
-- Un solo Pomodoro activo por jugador:
CREATE UNIQUE INDEX one_active_pomodoro
  ON pomodoros(player_id) WHERE status IN ('running','paused');

CREATE TABLE pomodoro_events (           -- auditoría de pausas
  id INTEGER PRIMARY KEY,
  pomodoro_id INTEGER NOT NULL REFERENCES pomodoros(id),
  kind TEXT NOT NULL,                    -- start | pause | resume | complete | cancel
  at TEXT NOT NULL
);

CREATE TABLE structures (
  id INTEGER PRIMARY KEY,
  player_id INTEGER NOT NULL REFERENCES players(id),
  kind TEXT NOT NULL,                    -- hive | dog | path | fence | lantern ...
  x INTEGER NOT NULL, y INTEGER NOT NULL
);

CREATE TABLE settings (
  player_id INTEGER NOT NULL REFERENCES players(id),
  key TEXT NOT NULL, value TEXT NOT NULL,
  PRIMARY KEY (player_id, key)
);

CREATE TABLE schema_migrations (version INTEGER PRIMARY KEY, applied_at TEXT NOT NULL);
```

---

## 6. Arquitectura Técnica

### Stack

*   **Backend:** Go + Fiber. **BD:** SQLite incrustada (modo WAL). **Frontend:** React (Vite) + Zustand. **3D:** React Three Fiber + Three.js. **Despliegue:** Docker Compose (Nginx + Go app), con volúmenes separados para `/data` y `/backups`.

### 6.1 Temporizador

*   El servidor guarda `started_at`, `planned_s`, `paused_at` y `paused_total_s`. El tiempo restante es siempre:
    `restante = planned_s − (ahora − started_at − paused_total_s)` (con `ahora = paused_at` si está pausado). Nunca se usan contadores incrementales.
*   `setInterval` solo refresca la interfaz; el estado real lo calcula el servidor con su reloj.

### 6.2 Resolución offline

No hay *cron jobs*. Al conectar, el servidor hace lo siguiente:

1. `delta = max(0, ahora_servidor − last_seen_at)`. Se ignora el reloj del cliente.
2. Reúne los **eventos** de la ventana `[last_seen_at, ahora]`: cada planta madura en `matured_at` y se marchita en `wilts_at`.
3. Recorre los tramos entre eventos; en cada uno, `ritmo = Σ g · M` de las plantas vivas.
4. El tope del Silo es `cap_monedas = horas_silo × ritmo_máximo_de_la_ventana`. En cada tramo: `acumulado = min(acumulado + ritmo·Δt, cap_monedas)`.
5. Guarda el acumulado **en el Silo** (no en el saldo): el jugador lo recoge con un toque y el Perro Pastor lo hace solo (decisión 9). Actualiza `last_seen_at`.

El contenido del Silo se guarda en millonésimas de moneda y el tope usa el ritmo máximo visto *desde la última recogida* (no solo en esta ventana), de modo que liquidar una vez o mil veces da el mismo resultado.

**Ejemplo** (M = 1, ausencia de 30 h, 3 plantas vivas): Tomates (2,89 🪙/h, se marchita a las 6 h), Girasol (3,53 🪙/h, a las 20 h) y Manzano (4,16 🪙/h, vive todo el tramo).

| Tramo | Ritmo | Producción |
| :--- | :--- | :--- |
| 0–6 h | 10,58 🪙/h | 63,5 |
| 6–20 h | 7,69 🪙/h | 107,7 |
| 20–30 h | 4,16 🪙/h | 41,6 |

Sin tope: 212,7 🪙 (con las tasas redondeadas a dos decimales; con las exactas de 4.3 sale 212,9). Con Silo de 24 h, `cap = 24 × 10,58 = 253,9` → no trunca, se ingresan **212,7 🪙**. Con Silo de 12 h, `cap = 127,0` → el silo se llena a las ≈ 14,3 h y se ingresan **127,0 🪙**.

Casos límite a cubrir con pruebas: reloj del sistema retrocedido (delta = 0), planta que madura y se marchita dentro de la misma ausencia, Silo ya lleno, ausencia de meses.

### 6.3 Concurrencia

*   El servidor es la única fuente de verdad. El índice parcial `one_active_pomodoro` impide dos Pomodoros activos.
*   Control optimista con `version` en `players` y `plots`; si hay conflicto, el cliente recarga el estado y reintenta.

### 6.4 Usuarios `[Decisión]`

Un jugador por instalación; Tailscale es la capa de acceso. El esquema ya lleva `player_id`, así que añadir más jugadores no exige migración de estructura.

### 6.5 Copias de seguridad

Copia diaria del archivo SQLite con `VACUUM INTO` (o la API de backup), conservando las últimas 14, en el volumen `/backups`. Migraciones numeradas (`golang-migrate` o ficheros SQL) con la tabla `schema_migrations`.

### 6.6 Música

Componente React aislado con `iframe` de YouTube, IDs configurables y posibilidad de pegar una URL propia. Si el iframe da error, se pasa al audio ambiental local.

---

## 7. Plan de desarrollo por fases

**Fase 0 – Preparación.** Esquema de BD, estructura del proyecto, `docker-compose` con backups. Validar `pomofarm_balance.xlsx` con tus cifras reales de foco.
**Fase 1 – MVP jugable.** 1 parcela, Margarita/Tomates/Girasol, temporizador con timestamps, cosecha manual, 💧, persistencia, cerrar y volver. Escena 3D mínima.
**Fase 2 – Economía y granja.** Parcelas, 🪙 pasivas con vida útil, Silo, resolución offline, Manzano y Roble, sinergias.
**Fase 3 – Automatización y estética.** Abejas, Perro, decoración, audio ASMR y Lofi.
**Fase 4 – Retención.** Libro de Cosechas con etiquetas y CSV, Prestigio, accesibilidad, modo 2D.
**Fase 5 – Robustez.** Pruebas de tiempo, Modo Foco, notificaciones, revisión de copias de seguridad.

---

## 8. Criterios de aceptación y pruebas

*   Cerrar el navegador a mitad de un Pomodoro y volver: el tiempo restante es exacto (±1 s) y no se pierde nada.
*   Abrir en dos dispositivos: el segundo muestra el Pomodoro existente y no permite iniciar otro.
*   Cambiar la hora del sistema del cliente no altera ningún cálculo.
*   Ejemplo de 6.2 reproducido en una prueba automática, con los dos topes de Silo.
*   Ausencia de 90 días: se ingresa como máximo lo que permite el Silo; ninguna planta produce más allá de su vida útil.
*   Restaurar una copia de seguridad en una instalación limpia deja el juego jugable.
*   La suma de 🪙 simulada en `sim.py` coincide con la hoja de cálculo para el perfil Normal (1.798,7 🪙/día en régimen estable).

---

## 9. Métricas de éxito (personales)

Pomodoros completados por semana, horas de foco por mes y si sigues abriendo la app a los 30 y 90 días. No hay métricas de captación ni retención comercial.

---

## 10. Decisiones tomadas y pendientes

**Decisiones tomadas por mí (reversibles):**

1. Coste de semilla = desbloqueo único (4.1).
2. Modelo indulgente sin límite de pausas ni confirmación "¿te concentraste?" (3.3).
3. Descanso proporcional y saltable (3.2).
4. Un jugador por instalación (6.4).
5. El Prestigio conserva 💧, parcelas y Silo; reinicia las 🪙 y la automatización (4.9).
6. Animales: desbloqueo con 💧, compra con 🪙 (4.7), para reconciliar dos frases del GDD original que se contradecían.
7. Perro Pastor = +12 h de Silo (el original solo decía "extiende la capacidad").
8. Retirar una planta madura exige haberla cosechado antes (el 💧 nunca se pierde) y una confirmación en dos pasos; una marchita se retira gratis (3.1).
9. Cobro de 🪙 manual: la producción va al Silo y se recoge con un toque; si se llena, la granja deja de producir hasta vaciarlo (3.4). El Perro Pastor lo recoge automáticamente (4.7), lo que le da una función además de las +12 h.
10. Una planta marchita sin cosechar conserva su 💧: se puede cosechar tras marchitarse, y no se puede sembrar encima ni retirar hasta hacerlo. Retirar una marchita ya cosechada es gratis y sin confirmación (3.1, 3.4).
11. Sinergias: solo las plantas que producen (maduras y vivas) dan y reciben bonos. Si un vecino aún crece o se marchita, el bono se apaga; el "Huerto completo" exige que las cuatro plantas del 2×2 estén produciendo y se paga una sola vez aunque la planta pertenezca a varios bloques (4.6).
12. Descansos: los guarda el servidor (todos los dispositivos coinciden y sobreviven a recargar). Se proponen al cosechar solo si se cosecha dentro de los 15 min siguientes a terminar el Pomodoro; una planta que se marchitó estando fuera no da descanso. Empezar otro Pomodoro termina el descanso. Se activan o desactivan y se configuran sus tres duraciones en Ajustes (3.2).
13. Colmenas (4.7): se colocan **junto** a una parcela, en su esquina, sin ocuparla; su área 3×3 se cuenta desde esa parcela. Moverlas es gratis y sin límite (un error de colocación no debe costar miles de 🪙). Comprar y mover liquidan antes la producción, así que el +25% empieza y termina exactamente en ese momento. El Perro recoge el Silo solo cada vez que el servidor liquida y suma 12 h de capacidad.
14. Decoración (4.7): se coloca en el terreno de alrededor de la granja (cuadrícula de 8×8 que rodea el campo de 4×4, sin las celdas del Silo y del Perro), una pieza por celda y sin límite de piezas. Colocar cuesta 🪙 (camino 15, valla 25, farolillo 80); moverla es gratis y quitarla también, sin reembolso. El sombrero del Perro (600 🪙, una vez) exige tener el Perro. Nada de esto cuenta para el prestigio ni cambia la producción.

**Limitaciones de la simulación:** modelo agregado (no coloca plantas en el mapa), bonos de sinergia como medias supuestas, visitas frecuentes al día, sin multiplicador de prestigio, y ritmo de compra "voraz". Sirve para dimensionar precios, no para predecir jugadores reales. **Hace falta una semana de juego propio para recalibrar** antes de dar los precios por buenos.

**Pendientes para ti:** confirmar las decisiones anteriores, ajustar los perfiles de foco a tu uso real en la hoja `Rendimiento`, y elegir el bioma de la segunda estación.

---

## 11. Registro de cambios respecto a las versiones anteriores

| Cambio | Motivo |
| :--- | :--- |
| Vida útil de las plantas (4.3) | Sin ella, una granja llena producía para siempre y el Pomodoro dejaba de importar. |
| Valores de rendimiento derivados de una fórmula | El borrador anterior daba al Roble 83 veces el rendimiento por minuto de la Margarita y tenía una tabla con un error de cálculo (Manzano). |
| Coste de semilla como desbloqueo único | El coste por plantado del GDD original hacía que Girasol, Manzano y Roble dieran pérdida neta de 💧. No lo detecté en mis análisis anteriores. |
| Rebrote de perennes eliminado | Con semilla única no tenía función, y en el borrador anterior daba un Roble casi gratis. |
| Penalizaciones y confirmación eliminadas | Contradecían el principio "sin penalizaciones" del GDD original. |
| Penalización por monocultivo eliminada | Chocaba con los patrones y no definía cómo se combinaban los bonos. Ahora: suma con tope ×2,0. |
| Vida útil ×3 respecto al primer ajuste | Con las vidas iniciales, un jugador "Normal" solo podía mantener ≈ 4 de 16 parcelas vivas; ahora ≈ 12. |
| Silo medido en horas | Escala solo con la granja. |
| Prestigio con requisitos concretos | Se eliminaron los "X 💧" sin valor. |
| Esquema SQL completo, algoritmo offline con ejemplo, pruebas de aceptación | Faltaban en el documento anterior. |
