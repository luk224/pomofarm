# Soluciones propuestas para el GDD de PomoFarm 3D

Documento complementario al GDD. Cada punto indica el problema, la solución propuesta y, cuando aplica, valores iniciales que deberán ajustarse con una hoja de balance.

---

## Prioridades

| Prioridad | Tema | Sección |
| :--- | :--- | :--- |
| 1 | Economía con números y ciclo de vida de plantas | 1 |
| 2 | Timers basados en timestamps y pestañas en segundo plano | 4.1 |
| 3 | MVP y fases de desarrollo | 6 |
| 4 | Backups y modelo de datos | 4.4 |
| 5 | Resto de mejoras | 2, 3, 5 |

---

## 1. Economía y bucle de juego

### 1.1 El Pomodoro debe seguir siendo necesario tras automatizar

**Problema:** una planta madura produce 🪙 indefinidamente. Una vez llena la granja, no hay motivo para volver a plantar ni a concentrarse.

**Solución: ciclo de vida de las plantas.** Cada planta madura produce durante una *vida útil* y luego se marchita. Para volver a producir hay que replantar (nuevo Pomodoro).

| Planta | Vida útil (h) | 🪙 totales por ciclo |
| :--- | :--- | :--- |
| Margarita | 12 | 24 |
| Tomates | 24 | 240 |
| Girasol | 36 | 720 |
| Manzano | 72 | 1.440 (con rebrote, ver abajo) |
| Roble Mágico | 120 | 12.000 |

Valores iniciales, a validar en la hoja de balance. Reglas adicionales:

- **Marchita ≠ pérdida.** La planta marchita se retira gratis y la parcela queda lista para replantar.
- **Rebrote para perennes (Manzano, Roble):** al marchitarse, un mini-Pomodoro de 10 min las revive con vida útil reducida (50%), sin pagar semilla. Da un uso a las sesiones cortas.
- **Notificación amable** de "tu granja necesita atención" cuando varias plantas estén por marchitarse, sin urgencia ni castigo.

### 1.2 Cuellos de botella de 💧 por fase

Los 💧 solo se obtienen concentrándose, así que deben ser el recurso que desbloquea cada hito:

| Fase | Qué se compra con 💧 |
| :--- | :--- |
| Inicio | Semillas Tomate/Girasol, primeras parcelas |
| Medio | Expansión de tierra, primeros animales |
| Avanzado | Mejoras del Silo (capacidad), slots de sinergia |
| Prestigio | Requisito: gastar X 💧 acumulados + mapa automatizado |

Las 🪙 compran solo comodidad y estética más automatizadores. Así, el tiempo real concentrado siempre limita el avance.

### 1.3 Modo Flow (60+ min)

- Rango: 60 a 120 min, elegido con un control deslizante antes de plantar.
- Recompensa base: 0,42 💧/min (equivale a 25 💧 por 60 min).
- Bonificación Flow: +10% de 💧 por cada 15 min por encima de 60 (máximo +40% a 120 min).
- Sin penalización por descansos propios: el Flow sigue siendo una sesión única.

### 1.4 Sinergias con más peso

- Mantener la adyacencia simple (+10% por vecino compatible), pero con tope por parcela (+40%).
- Añadir **patrones** con bonificación fija:
  - Línea de 3 girasoles: +25% a las tres.
  - Cuadrado 2x2 de Tomates: +30% a las cuatro.
  - Manzano con Girasol y Abejas en 3x3: "Huerto completo", +50% 🪙.
- **Penalización suave por monocultivo:** más de 6 plantas iguales conectadas reducen un 10% su generación. Fomenta variedad.
- Mostrar sinergias activas como pequeños iconos flotantes para que el puzle sea legible.

### 1.5 Pausas e interrupciones

El estado se conserva al cerrar el navegador (sin castigo), pero con límites para evitar abusos:

- Pausa manual: máximo 2 por sesión y 10 minutos en total.
- Cierre del navegador: cuenta como pausa y congela el timer.
- Si una sesión permanece congelada más de 24 h, se archiva: la semilla se devuelve y no se otorga recompensa.
- No hay pérdida de progreso mientras se respeten los límites; es una norma de transparencia, no un castigo.

### 1.6 Descansos proporcionales

| Duración del Pomodoro | Descanso |
| :--- | :--- |
| ≤ 25 min | 5 min |
| 26-45 min | 10 min |
| 46 min o más | 15 min |

El descanso es opcional (botón "Saltar"). El jugador puede configurar duraciones en ajustes.

### 1.7 Honestidad del timer

Decisión de diseño recomendada para una app personal: **filosofía de confianza**. Opcionalmente, al terminar, un diálogo breve "¿Te concentraste?" (Sí / A medias / No):

- Sí: 100% de la recompensa.
- A medias: 50%.
- No: 💧 no se otorgan, pero la planta madura igualmente.

Es desactivable en ajustes.

---

## 2. Onboarding y experiencia de usuario

### 2.1 Primer minuto (FTUE)

1. Pantalla inicial con una parcela y una Margarita ya disponible, regalada.
2. Tutorial mínimo en tres pasos: elegir semilla → plantar → esperar.
3. Primer Pomodoro de 10 min como introducción (gratis), con recompensa de 1 💧 y 1 Tomate de regalo.
4. Tooltips contextuales para silo, sinergias y descansos, que aparecen solo la primera vez.

### 2.2 Etiquetas de tarea

- Campo opcional "¿En qué vas a trabajar?" al plantar, con autocompletado de etiquetas recientes.
- Se guarda por Pomodoro y alimenta las estadísticas.

### 2.3 Accesibilidad

- Modo sin audio y subtítulos visuales para las alertas.
- Paletas aptas para daltonismo.
- Opción de **reducir animaciones** (desactiva squash & stretch continuo).
- Vista 2D ligera como alternativa en equipos modestos.
- Atajos de teclado para plantar, pausar y recolectar.

---

## 3. Retención y estadísticas

### 3.1 Libro de Cosechas ampliado

- Mantener la visualización camuflada (fardos, silos, cestas).
- Añadir un modo "tabla" opcional con exportación a CSV.
- Desglose por etiquetas de tarea (tiempo por proyecto) y por día de la semana.

### 3.2 Métricas de éxito (personales)

- Pomodoros completados por semana.
- Racha de días con al menos un Pomodoro (sin castigo al romperla; solo se muestra la mejor racha histórica).
- Retención a 30 días de uso.

### 3.3 Prestigio

- Requisitos explícitos: mapa 100% automatizado + X 💧 acumulados históricos.
- Cada estación otorga un multiplicador permanente (+10% 🪙) y un bioma nuevo.
- Los objetos estéticos comprados se conservan como colección.

---

## 4. Soluciones técnicas

### 4.1 Timers y pestañas en segundo plano

- Guardar en el servidor `started_at` y `duration`. La hora restante se calcula siempre como `now - started_at`, nunca con contadores incrementales.
- Usar `setInterval` solo para refrescar la UI.
- Notificaciones web (`Notification API`) y sonido al terminar, con permiso solicitado en el primer Pomodoro.
- Actualizar el título de la pestaña con el tiempo restante.

### 4.2 Rendimiento del 3D

- **Modo Foco:** mientras corre el timer, bajar a 15-30 FPS (`frameloop="demand"` en R3F) y reducir sombras.
- Pausar el render por completo cuando la pestaña no es visible (`document.visibilityState`).
- Instanciar mallas repetidas (`InstancedMesh`) para plantas y decoración.

### 4.3 Música

- IDs de los streams configurables desde ajustes, no fijos en código.
- Fallback: detectar error de carga del iframe y ofrecer un audio ambiental local (lluvia, bosque, fuego).
- Permitir pegar una URL de YouTube propia.

### 4.4 Modelo de datos, migraciones y backups

Esquema inicial propuesto:

```sql
CREATE TABLE players (
  id INTEGER PRIMARY KEY,
  name TEXT NOT NULL,
  focus_points INTEGER NOT NULL DEFAULT 0,
  coins INTEGER NOT NULL DEFAULT 0,
  season INTEGER NOT NULL DEFAULT 1,
  prestige_multiplier REAL NOT NULL DEFAULT 1.0,
  last_seen_at TEXT NOT NULL
);

CREATE TABLE plots (
  id INTEGER PRIMARY KEY,
  player_id INTEGER NOT NULL REFERENCES players(id),
  x INTEGER NOT NULL,
  y INTEGER NOT NULL,
  plant_type TEXT,
  state TEXT NOT NULL DEFAULT 'empty', -- empty | growing | mature | withered
  started_at TEXT,
  duration_s INTEGER,
  paused_total_s INTEGER NOT NULL DEFAULT 0,
  matured_at TEXT,
  UNIQUE (player_id, x, y)
);

CREATE TABLE pomodoros (
  id INTEGER PRIMARY KEY,
  player_id INTEGER NOT NULL REFERENCES players(id),
  plant_type TEXT NOT NULL,
  tag TEXT,
  started_at TEXT NOT NULL,
  completed_at TEXT,
  duration_s INTEGER NOT NULL,
  reward_focus INTEGER NOT NULL DEFAULT 0,
  status TEXT NOT NULL -- running | completed | archived
);

CREATE TABLE structures (
  id INTEGER PRIMARY KEY,
  player_id INTEGER NOT NULL REFERENCES players(id),
  kind TEXT NOT NULL, -- bee | dog | fence | path | lantern ...
  x INTEGER NOT NULL,
  y INTEGER NOT NULL
);
```

- **Migraciones versionadas** (por ejemplo, `golang-migrate` o ficheros SQL numerados).
- **Backups:** copia diaria del archivo SQLite con `VACUUM INTO` o la API de backup, guardando las últimas 14. Añadir un volumen separado en Docker Compose para `/backups`.
- Activar modo WAL para mejor concurrencia.

### 4.5 Resolución offline

- Usar siempre la hora del servidor; ignorar relojes del cliente.
- Calcular `delta = max(0, now - last_seen_at)` para evitar valores negativos.
- Aplicar el tope del Silo al calcular las 🪙 acumuladas y la vida útil de cada planta (una planta que se marchita offline deja de producir en ese momento).
- Guardar `last_seen_at` periódicamente (heartbeat cada 30-60 s) y al cerrar (`beforeunload`/`sendBeacon`).

### 4.6 Concurrencia y multiusuario

- Fuente de verdad única: el servidor.
- Un solo Pomodoro activo por jugador; si se abre en otro dispositivo, mostrar el estado existente.
- Control de versión optimista (campo `version` o `updated_at`) en las parcelas para detectar conflictos.
- Multiusuario: cuentas simples por nombre (con Tailscale como capa de acceso) o identificación por cabecera de Tailscale. Decidir si es un solo jugador o varios al principio.

---

## 5. Contenido y arte

- Priorizar un conjunto pequeño de assets low-poly reutilizables (paleta de colores común, mismo grosor de bordes).
- Definir un presupuesto de polígonos por planta (por ejemplo, < 2.000 triángulos) para mantener rendimiento.
- Plantear el sonido como capas: ambiente base, efectos de interacción y alertas, con control de volumen independiente por capa.

---

## 6. Plan de desarrollo por fases

### Fase 0 - Preparación
- Hoja de cálculo de balance económico.
- Esquema de base de datos y estructura del proyecto (Go + Fiber, React + Vite).

### Fase 1 - MVP jugable
- Una parcela, Margarita/Tomate/Girasol.
- Timer basado en timestamps, cosecha manual, 💧.
- Persistencia en SQLite, estado al cerrar el navegador.
- Escena 3D mínima (cámara isométrica, una planta animada).

### Fase 2 - Economía y granja
- Múltiples parcelas, 🪙 pasivas, Silo y resolución offline.
- Ciclo de vida de plantas, Manzano y Roble Mágico.
- Sinergias de adyacencia.

### Fase 3 - Automatización y estética
- Abejas, Perro Pastor, decoración.
- Audio ASMR y minireproductor Lofi.

### Fase 4 - Retención
- Libro de Cosechas con etiquetas y exportación.
- Prestigio por estaciones.
- Accesibilidad y modo 2D.

### Fase 5 - Robustez
- Backups automáticos, migraciones, pruebas de carga y de tiempos.
- Modo Foco de rendimiento y notificaciones.

---

## 7. Próximos pasos inmediatos

1. Crear la hoja de balance con los datos de la sección 1 y simular 30 días de juego.
2. Decidir las preguntas abiertas: ¿un jugador o varios?, ¿filosofía de confianza o confirmación al final?
3. Implementar el timer basado en timestamps antes que cualquier otra mecánica.
4. Configurar backups desde el primer despliegue.
