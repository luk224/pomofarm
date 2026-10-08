# Informe de QA — Fase 4 (retención, prestigio y accesibilidad)

Fecha: 2026-10-09 · Commit base: `509b290` · Entorno: Chromium, Firefox 155 y WebKit 26.6 (Playwright), Go 1.26, Node 24.
Todos los E2E corren con `tools/e2e/run_isolated.sh` (backend supervisado y Vite propios, BD temporal), nunca contra la partida de desarrollo.

## Resultado

| Suite | Resultado |
| :--- | :--- |
| Tests Go (`go test ./...`) | 198 funciones de test (+ subtests), 0 fallan |
| Tests frontend (vitest) | 255 pasan |
| E2E Playwright, 24 scripts (Fases 1 a 4) | 586 comprobaciones pasan, 0 fallan |
| `go vet`, `gofmt`, `tsc -b`, oxlint, build de producción | limpios |
| `npm audit --omit=dev` | 0 vulnerabilidades |

## Qué se ha comprobado, por tarea

| Tarea | Pruebas |
| :--- | :--- |
| P4-01 Libro de Cosechas | 7 tests de API (totales, por etiqueta, lunes primero, zona horaria, entrada mala, selector de meses, CSV fila a fila); 4 mutaciones detectadas (semana mal contada, CSV sin neutralizar fórmulas, cancelados contados, fechas en UTC); E2E con descarga real del CSV comparada con la BD |
| P4-02 Métricas y modo estricto | Rachas, "limpio" y semanas (tests de juego y de API); el modo estricto **nunca bloquea una pausa** (6 pausas de 20 min y la sesión paga igual); 3 mutaciones detectadas; migración 6 |
| P4-03 Prestigio | Reglas exactas de reinicio/conservación (E2E y tests), +10% medido (×1,10 y ×1,20), doble confirmación, carrera de 6 peticiones (solo una triunfa); 6 mutaciones detectadas |
| P4-04 Accesibilidad | Contraste AA y simulación de protanopia/deuteranopia/tritanopia con la hoja de estilos real; silencio medido con un analizador de señal; aviso visual al terminar sin audio |

## Pruebas nuevas de esta verificación

- **Jugadas aleatorias con prestigio:** 20 partidas × 160 jugadas (≈ 3.160 movimientos válidos e inválidos, con el reloj avanzando y retrocediendo), ahora con "granja completa + nueva estación" como jugada posible. Tras cada prestigio se comprueba temporada +1, 🪙 a cero, 16 parcelas, sin decoración ni colmenas; y el resto de invariantes (saldos, Silo, un solo Pomodoro activo, decoración, estadísticas: limpios ⊆ estrictos ⊆ total, mejor racha ≥ actual) siguen valiendo. Una mutación (la temporada sube de 2 en 2) se detecta, lo que demuestra que la jugada se ejecuta.
- **Horario de verano (Madrid):** el domingo de 25 horas (la hora 02:30 ocurre dos veces) cuenta un solo día con los dos Pomodoros; sábado-domingo-lunes forman una racha de 3 y la semana empieza el lunes 26. El cambio de primavera (29 de marzo) no duplica ni pierde ninguno.
- **Fronteras de mes en zonas lejanas:** el mismo Pomodoro cae en octubre en Auckland (UTC+13), en septiembre en UTC y en septiembre en Honolulu (UTC−10).
- **Historial grande:** 730 días × 12 Pomodoros (8.760): Libro **50 ms**, estadísticas **50 ms**, CSV **62 ms**, y la racha de 730 días y los 4.380 estrictos salen exactos.
- **Etiquetas raras:** japonés, emoji, comas, comillas, saltos de línea, `<script>`, 60 caracteres, `=SUM(A1)`, `@cmd`, `-1` y tabulaciones: el Libro cuenta cada una, el CSV tiene siempre 11 columnas y ninguna se ejecutaría como fórmula en una hoja de cálculo.
- **Actualización desde el esquema 5:** una BD con historial, eventos, decoración, sombrero y ajustes se abre con la app actual, conserva todas las filas y los Pomodoros anteriores quedan como "no estrictos".
- **Entrada hostil:** las rutas nuevas (`/api/prestige`, `/api/stats`, `/api/book` y `/api/book/export.csv`) con meses y zonas horarias mal formados (`%00`, `../../etc/passwd`, 500 caracteres, `0000-00`, `9999-12`, zonas repetidas) y cuerpos con `confirm` de tipos raros: ningún 5xx, todo error es JSON y nada filtra SQL ni trazas. Una URL de 6.000 caracteres contra el servidor real responde **431** y el servidor sigue vivo.
- **Recorrido completo (E2E, Chromium):** 40 días de historial → Libro abierto solo con teclado (racha de 40 días, 20 de 20 limpios, totales = BD) → zona horaria de otro navegador (Auckland) → un segundo dispositivo hace el prestigio → el primero lo recibe al recuperar el foco sin recargar → Libro y métricas intactos → una compra con el estado viejo recibe un 409 claro, no un 500 → servidor caído al abrir el Libro (aviso de conexión, se recupera) → CSV con una fila por Pomodoro (79 de 79) tras el prestigio.
- **Accesibilidad de lo nuevo:** todos los controles del Libro y de la sección de estaciones tienen nombre accesible; los gráficos son imágenes con texto alternativo; los requisitos dicen "cumplido / falta" para lectores de pantalla.
- **Rendimiento:** estación 2 con la granja reconstruida y el Libro abierto: 16 draw calls.
- **Otros motores:** WebKit (Safari) y Firefox abren el Libro con la tabla y la descarga del CSV, la sección de estaciones, los ajustes de accesibilidad (paleta para daltonismo) y silencian con M, sin errores de JavaScript.

## Observaciones (no son defectos; decisión tuya)

1. **El ritmo de la 2.ª estación.** Simulado con `sim.py`: Normal 41 días, tu rango real (2–4 h/día) entre 25 y 61. Si 61 días te parecen mucho para quien juega 2 h, se ajusta con el bono (+10%) o con los precios. El GDD estimaba 30–35 y estaba optimista.
2. **Sumidero de 🪙 (de la Fase 3):** la decoración sigue absorbiendo solo ~3.680 🪙. Con el prestigio, las 🪙 vuelven a tener un destino (reconstruir la automatización), así que el problema es menor ahora, pero entre estaciones sigue habiendo un tramo final sin nada en lo que gastar.
3. **El Libro recuerda el último mes mirado** mientras no recargues (comodidad). Si prefieres que siempre abra en el mes actual, es un cambio de una línea.
4. **Racha actual en el Libro:** se muestra como dato neutro. Si prefieres no verla (solo la mejor), se quita sin tocar nada más.

## Sin verificar

- **Safari en un iPhone real** (P1-08d): WebKit de escritorio pasa, pero el audio en iOS, el silenciador físico y el iframe de YouTube solo se confirman en el dispositivo.
- **GPU real y móviles de gama baja:** el rendimiento se midió con swiftshader. Las cifras son bajas (16–75 draw calls), pero los FPS reales no se han medido.
- **Escucha humana** de efectos, ambiente y música, y **vista humana** de las paletas (verano, otoño, invierno y la de daltonismo): se comprobaron por valores y capturas, no por una persona con daltonismo.
- **Despliegue:** `wyse` sigue en la versión de la Fase 3 (esquema 5). Esta fase añade las migraciones 6 (modo estricto) y requiere desplegar con copia previa de la BD.
