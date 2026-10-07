# Informe de QA — Fase 2 (economía y granja)

Fecha: 2026-10-07 · Commit base: `5d45f94` · Entorno: Chromium, Firefox 155 y WebKit 26.6 (Playwright), Go 1.26, Node 24, Docker 29.
Todos los E2E corren con `tools/e2e/run_isolated.sh` (backend supervisado y Vite propios, BD temporal), nunca contra la partida de desarrollo.

## Resultado

| Suite | Resultado |
| :--- | :--- |
| Tests Go (`go test ./... -race`) | 131 pasan, 0 fallan |
| Tests frontend (vitest) | 136 pasan |
| E2E Playwright, 14 scripts (Fases 1 y 2) | 305 comprobaciones pasan, 0 fallan, 0 omitidas |
| Restauración en instalación limpia con Docker (`qa_restore.sh`) | 8 de 8 |
| `go vet`, `gofmt`, `tsc -b`, oxlint, build de producción | limpios |
| `npm audit --omit=dev` | 0 vulnerabilidades |

## Criterios del GDD §8 (los siete)

| Criterio | Estado | Medición |
| :--- | :--- | :--- |
| Cerrar el navegador y volver (±1 s) | **Cumple** | Fase 1, repetido en la regresión de esta fase |
| Dos dispositivos | **Cumple** | idem; además dos dispositivos ven el mismo descanso |
| Cambiar la hora del cliente | **Cumple** | idem |
| Ejemplo de §6.2 con los dos topes de Silo | **Cumple** | Con las tasas redondeadas del GDD: 212,7 y 127,0. Con las exactas de §4.3: 212,89 y 127,0. Por la API completa con tres plantas: 212,9 con Silo de 24 h y 127,0 con el de 12 h |
| Ausencia de 90 días | **Cumple** | Granja de 16 plantas, Silo de 36 h: guarda 2.295,9 🪙 (= su capacidad y por debajo de lo que las plantas podían dar); todas marchitas; otros 90 días fuera no producen nada más |
| Restaurar una copia en instalación limpia | **Cumple** | Stack Docker vacío + `restore.sh`: datos íntegros, jugable, y una copia antigua **se pone al día** (el Silo se llena con lo producido desde que se hizo la copia) |
| `sim.py` coincide con el xlsx (Normal: 1.798,7 🪙/día) | **Cumple** | 1.798,7 en ambos |

Y uno más que el GDD sugiere en §4.2 y ahora se comprueba en el juego real, no solo en la hoja:

| Fórmula | Esperado | Medido |
| :--- | :--- | :--- |
| 🪙/día = foco × e(d) × M, con M = 1 | 180 × e(45) = **1.199,1** | **1.199,1** (0,00 % de diferencia) tras 30 días simulados con el ciclo completo: sembrar 4 manzanos al día, madurar, vivir 72 h, marchitarse, retirar, resembrar y recoger |

## Pruebas nuevas de esta verificación

- **Actualización desde el esquema 1:** una BD creada por la Fase 1, con datos reales de jugador, parcelas, historial, etiquetas y ajustes, se abre con la app actual y conserva **todas** las filas; el índice de "un solo Pomodoro activo" sigue funcionando. Es el camino que seguirá `wyse`.
- **Jugadas aleatorias con invariantes:** 14 partidas × 160 jugadas (2.240 movimientos válidos e inválidos, con el reloj avanzando de 1 s a 80 h y retrocediendo de vez en cuando). Tras cada jugada se comprueba: saldos no negativos, 🪙 que solo suben, lifetime ≥ saldo, Silo sin superar su capacidad, un único Pomodoro activo dueño de la única parcela que crece, plantas maduras con vida pendiente y marchitas sin bono, `collected_to` entre madurez y muerte, y que lo cobrado más lo almacenado nunca supera lo que pueden dar las plantas (×2,0). Tres defectos introducidos a propósito (Silo sin tope, plantas que no se marchitan, cosecha que paga dos veces) se detectan.
- **Entrada basura:** 27 rutas × 21 cuerpos hostiles (JSON roto, tipos erróneos, cadenas de 5.000 caracteres, anidamiento de 200 niveles, ids enormes o negativos, métodos erróneos): ninguna da un 5xx, todo error es JSON y ninguno filtra SQL ni trazas. Un cuerpo de 40 KB se rechaza con **413** (comprobado también contra el servidor real).
- **Teclado y lectores de pantalla (navegador):** elegir parcela con las flechas, anuncio de la parcela en vista, plantar con el teclado en la parcela anunciada, que las flechas no actúen dentro de un campo de texto, foco visible de 3 px y acceso con Tab a *Mejoras*, *Recoger*, *Saltar descanso* y *Retirar planta*, la tienda con el teclado, y contraste AA de los 11 pares de color nuevos (mínimo 5,1:1).
- **Movimiento reducido:** la chispa del Silo no flota y las barras no animan.
- **Reinicio del servidor con todo el estado de la fase:** jugador, parcelas, tienda, ajustes y hora de fin del descanso quedan idénticos; el Silo conserva lo producido. **Recoger sin conexión** da un aviso claro, no cambia el saldo y funciona al volver el servidor.
- **Tres navegadores** (Chromium, Firefox, WebKit) con una granja del 2×2 central: las 4 insignias doradas del Huerto completo, la tienda y recoger el Silo, sin errores de consola.

## Hallazgos y correcciones durante la verificación

| # | Hallazgo | Corrección |
| :--- | :--- | :--- |
| 1 | Con varias parcelas, **elegir parcela solo era posible con el ratón**: el teclado y los lectores de pantalla quedaban fuera (GDD §2.4) | Flechas para cambiar de parcela, anuncio en una región `aria-live` ("Parcela 2 de 5: lista para cosechar, bono de 35 por ciento") y lista de atajos en Ajustes |
| 2 | Recoger un resto minúsculo del Silo (una milésima) daba un aviso "+0 🪙" | Mínimo para recoger de 0,1 🪙 (la resolución con la que se muestra) en el botón, la tecla C y el Silo 3D |
| 3 | La restauración de un script de Fase 1 fallaba al correr en Fase 2 | No era un defecto: una planta pasada de fecha se marchita al abrir, como debe. Fixture actualizado y se añade la comprobación de que la copia se pone al día |

### Hallazgos durante el desarrollo de la fase (todos corregidos, con test)

| Hallazgo | Cómo apareció |
| :--- | :--- |
| El ejemplo de §6.2 daba 216,7 en vez de 212,9 | Al implementar las sinergias: el ejemplo supone M = 1 y las plantas estaban juntas. Se colocan sin vecinos compatibles |
| La cosecha dependía de la fila del Pomodoro completado y daba 500 si faltaba | E2E de marchitamiento; ahora usa lo que recuerda la parcela |
| La vida útil del Flow se truncaba a horas enteras (97 min → 174 h en vez de 174,6) | Test con duraciones intermedias; ahora se calcula en segundos |
| 16 plantas maduras hacían **409 llamadas de dibujo** | Medición en P2-07; ahora **60** (una malla por planta, tierra instanciada) |
| Insignias de bono amontonadas con 16 plantas y Silo tapado por árboles | Captura de pantalla; insignias en la esquina de cada parcela y Silo en el rincón libre |
| El audio quedaba bloqueado tras recargar la página | E2E de descansos; se desbloquea con el primer gesto de cualquier tipo |
| El interruptor de descansos parpadearía por Tailscale | E2E; ahora cambia al instante |
| Campo y botón *Plantar* se salían 20 px del dock en 360 px | E2E del Flow en móvil |
| Un error sin mapear (`harvest_first`) salía como 500 | Test de la tarea; registrado |

## Análisis de balance (sinergias)

`sim.py` supone un multiplicador medio de ×1,25 sin Abejas y ×1,50 con ellas. Con las reglas reales, una búsqueda por optimización de la mejor distribución de cultivos da:

| Parcelas | Mejor posible (sin Abejas) | Mejor con todas las Abejas | Distribución al azar |
| :--- | :--- | :--- | :--- |
| 4 | ×1,300 | ×1,550 | ×1,113 |
| 8 | ×1,350 | ×1,600 | ×1,141 |
| 12 | ×1,350 | ×1,600 | ×1,147 |
| 16 | ×1,403 | ×1,653 | ×1,173 |

**Conclusión:** el supuesto de `sim.py` es alcanzable pero exige colocar con cabeza; una granja al azar queda ≈ 0,08–0,10 por debajo. Aun así **el ritmo de progresión es robusto**: el día de prestigio del perfil Normal es el 57 con el supuesto de la hoja, el 57 con una granja al azar y el 63 si no hubiera ninguna sinergia, porque el avance lo marcan los 💧 (parcelas, Silo, semillas) y no las monedas.

## Limitaciones conocidas (no son fallos de esta fase)

- **FPS reales sin medir:** el render por software de las pruebas sirve para comparar antes y después (409 → 60 llamadas), no para predecir los FPS en una GPU real ni en un móvil. Fase 5 (Modo Foco).
- **Sonido sin oír:** se verifica que se programa (8 osciladores) y que no puede saturar; el timbre lo juzga una persona.
- **Safari en iPhone** y el uso táctil real sin probar; sin auditoría con axe ni lectores de pantalla reales (se prueba la estructura y los textos que oirían).
- **Segundo plano real** simulado con `hasFocus`; **navegador cerrado**: sin notificación (haría falta Web Push, P5-06).
- **Abejas, Perro Pastor y decoración** no existen hasta la Fase 3: el cálculo ya admite las Abejas (+25 %) y el Perro (recoge solo, +12 h) y están probados en el servidor, pero no se pueden comprar aún.
- **Despliegue:** `wyse` sigue en la Fase 1. La Fase 2 añade las migraciones 2 y 3 (se aplican solas al arrancar; la actualización está probada) y requiere hacer antes una copia de la BD de `wyse`.
