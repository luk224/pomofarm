# Informe de QA — Fase 1 (MVP jugable)

Fecha: 2026-10-07 · Commit base: `7490677` · Entorno: Chromium 1243 y Firefox 155 (Playwright), Go 1.26, Node 24, Docker 29.
Todos los E2E se ejecutan con `tools/e2e/run_isolated.sh` (backend y Vite propios, BD temporal), nunca contra la partida de desarrollo.

## Resultado

| Suite | Resultado |
| :--- | :--- |
| Tests Go (`go test ./... -race`) | 42 pasan, 0 fallan |
| Tests frontend (vitest) | 44 pasan |
| E2E (Playwright), 6 scripts | 111 comprobaciones pasan, 0 fallan, 1 omitida |
| Restauración en instalación limpia (`qa_restore.sh`) | 7 de 7 |
| Lint (oxlint), `tsc -b`, build de producción | limpios |
| `npm audit --omit=dev` | 0 vulnerabilidades |

## Criterios del GDD §8

| Criterio | Estado | Medición |
| :--- | :--- | :--- |
| Cerrar el navegador a mitad de un Pomodoro y volver: tiempo exacto ±1 s | **Cumple** | Navegador cerrado del todo 7 s; al volver la UI marca 593 s frente a 592,0 s esperados (0,96 s: la pantalla redondea hacia arriba) |
| Dos dispositivos: el segundo muestra el Pomodoro y no deja iniciar otro | **Cumple** | B ve el mismo Pomodoro (593 s / 591 s), no ofrece sobres ni Plantar y el servidor responde 409 `pomodoro_active`; pausar en B se refleja en A |
| Cambiar la hora del cliente no altera ningún cálculo | **Cumple** | Reloj del cliente +3 días y otra zona horaria: UI 589 s, servidor 588,4 s |
| Restaurar una copia en instalación limpia deja el juego jugable | **Cumple** | Stack Docker vacío → `restore.sh` → datos íntegros, se puede retirar, plantar y sobrevive a un reinicio |
| `sim.py` coincide con el xlsx (Normal: 1.798,7 🪙/día) | **Cumple** | sim.py 1798,7 · xlsx 1798,7 |
| Ejemplo de §6.2 con los dos topes de Silo | Pendiente | Es de la Fase 2 (P2-01) |
| Ausencia de 90 días | Pendiente | Es de la Fase 2 (P2-01) |

## Bugs encontrados por el QA (todos corregidos, con test)

| # | Bug | Causa | Corrección |
| :--- | :--- | :--- | :--- |
| 1 | Doble clic en Plantar mostraba dos avisos "Ya hay un Pomodoro en marcha en otro dispositivo" al propio jugador | La segunda petición salía antes de que llegara la respuesta de la primera | Una sola mutación a la vez en el store; los avisos idénticos no se apilan |
| 2 | Con el servidor caído el aviso decía "Algo ha fallado" | El proxy devuelve 502/503/504 y el cliente no lo reconocía como fallo de conexión | 502/503/504 se tratan como `network` → "No hay conexión con el servidor…" |
| 3 | Colores por debajo de WCAG AA (4,0–4,3:1) en texto pequeño: sello +💧, botón Desbloquear, duración del sobre, placeholder | Azul agua y gris suave demasiado claros | Nuevos tokens `--water-deep` (6,0:1), `--ink-soft` (5,6:1 sobre kraft) y placeholder (5,5:1) |
| 4 | En móvil el dock medía 604 px en una pantalla de 390 | Los sobres ensanchaban la rejilla | `minmax(0,1fr)` y `min-width:0` (visto en P1-06) |
| 5 | Una respuesta de error sin mapear (`harvest_first`) salía como 500 | Faltaba registrarla en el mapa de estados | Registrada; test que lo atrapó |

Además, de proceso: los E2E reseteaban la BD de la partida de desarrollo y un test dejaba un backend huérfano. Ahora los scripts se niegan a ejecutarse fuera de `run_isolated.sh`, que usa BD temporal, puertos propios y comprueba que no queden procesos.

## Otras comprobaciones que pasan
Teclado solo (Tab → elegir con Espacio → Plantar con Intro; foco visible de 3 px; Espacio en un botón no dispara el atajo global) · etiqueta con HTML mostrada como texto (sin XSS) · etiqueta con SQL guardada como texto · límite de 60 caracteres en cliente y servidor (400) · reinicio del backend a mitad de Pomodoro: el tiempo continúa (600 s → 597 s) y la UI se recupera sola · 360×640 sin scroll horizontal · Firefox: carga, planta, cuenta atrás, recarga y lienzo 3D sin errores · contraste AA en los 15 pares de color medidos.

## Limitaciones conocidas (no son fallos de esta fase)
- **WebKit/Safari sin probar**: el sistema no tiene `libmanette-0.2-0` (`sudo apt-get install libmanette-0.2-0`).
- **Sonido sin oír**: se verifica que se programa (8 osciladores) y que no puede saturar; la calidad al oído la juzga una persona.
- **Segundo plano real**: se simula con `hasFocus`, no con una pestaña oculta durante minutos.
- **Navegador cerrado**: no hay notificación (requeriría Web Push con servicio en el servidor).
- **Táctil real y lectores de pantalla**: sin probar.
- **Cobertura de accesibilidad**: contraste y teclado verificados; sin auditoría con axe.
- **Rendimiento (FPS)**: no medido; el render por software de las pruebas no es representativo (Fase 5).
