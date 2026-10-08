# Informe de QA — Fase 3 (automatización, estética, audio y música)

Fecha: 2026-10-08 · Commit base: `0142945` · Entorno: Chromium, Firefox 155 y WebKit 26.6 (Playwright), Go 1.26, Node 24.
Todos los E2E corren con `tools/e2e/run_isolated.sh` (backend supervisado y Vite propios, BD temporal), nunca contra la partida de desarrollo.

## Resultado

| Suite | Resultado |
| :--- | :--- |
| Tests Go (`go test ./... -race`) | 159 funciones de test (+ subtests), 0 fallan |
| Tests frontend (vitest) | 219 pasan |
| E2E Playwright, 19 scripts (Fases 1 a 3) | 442 comprobaciones pasan, 0 fallan |
| `go vet`, `gofmt`, `tsc -b`, oxlint, build de producción | limpios |
| `npm audit --omit=dev` | 0 vulnerabilidades |
| Despliegue en `wyse` (esquema 5, partida real intacta) | hecho y verificado |

## Qué se ha comprobado, por tarea

| Tarea | Pruebas |
| :--- | :--- |
| P3-01 Abejas y Perro | 13 tests de API + 2 de juego; E2E de compra, colocación con vista previa, mover gratis y móvil. Cobertura 3×3, sin acumular, cuatro colmenas en el centro cubren las 16 parcelas, el +25% empieza y termina exactamente al comprar/mover (liquidar-antes) |
| P3-02 Decoración | 10 tests de API; E2E de colocar/mover/quitar, sombrero, móvil |
| P3-03 Audio por capas | Tests de curva y preferencias; E2E con un analizador de señal por capa: cada volumen mueve solo su capa, persisten, el ambiente suena de verdad (lluvia, bosque, fuego) y los efectos respetan su volumen |
| P3-04 Lofi | Tests del analizador de enlaces (14 válidos, 14 rechazados, incluidos dominios falsos tipo `youtube.com.evil.com`); E2E con la API simulada (éxito, error del vídeo, sin respuesta a los 10 s, sin red) y contra **YouTube real** (suena) |

## Pruebas nuevas de esta verificación

- **Jugadas aleatorias con decoración:** 20 partidas × 160 jugadas (≈ 3.160 movimientos válidos e inválidos, con el reloj avanzando y retrocediendo), ahora con compra, movimiento y retirada de decoración y compra del sombrero. Invariantes añadidos: cada pieza en una celda libre del terreno, una pieza por celda, ninguna dentro del campo, un solo sombrero y solo con Perro, coherencia de `Hat.Available/Owned`, y la conservación de 🪙 incluye lo gastado en decoración. **Tres defectos introducidos a propósito se detectan**: decorar sobre el campo, sombrero sin Perro y decoración gratis.
- **Concurrencia:** 8 peticiones simultáneas por la misma celda, por las últimas monedas y por el sombrero: exactamente una triunfa y se cobra una sola vez (lo garantizan el índice único de la BD y la transacción).
- **Actualización desde el esquema 4:** una BD con colmenas y Perro se abre con la app actual, conserva todas las filas y los índices nuevos hacen cumplir "una pieza por celda" y "un solo sombrero" a nivel de base de datos.
- **Entrada basura:** las rutas nuevas (`/api/structures`, `/api/decor` y sus variantes, ids enormes o negativos, tipos erróneos, coordenadas de 1e300, cuerpos de 3.000 caracteres) se suman a la matriz hostil: ningún 5xx, todo error es JSON, nada filtra SQL ni trazas.
- **Teclado:** comprar una colmena (flechas + Intro) y decorar (flechas mueven un cursor por las celdas libres, saltándose el campo, Intro coloca, Esc sale) funcionan **solo con teclado**; el lector de pantalla recibe la posición del cursor.
- **Rendimiento:** granja completa (16 parcelas, 4 colmenas, Perro con sombrero, 46 piezas de decoración y el reproductor abierto): **75 draw calls, 18.092 triángulos**.
- **Dos dispositivos:** un segundo navegador ve exactamente las mismas piezas y lo que coloca llega al servidor y al primero.
- **Servidor caído** mientras se decora: aviso claro de conexión, el modo decorar sigue activo y se puede seguir al volver.
- **Accesibilidad:** todos los controles de la tienda, del reproductor y de los ajustes de sonido tienen nombre accesible; el estado del reproductor es un `status`.
- **Otros motores:** WebKit (el de Safari) y Firefox cargan el juego, tienen contexto WebGL, abren el reproductor y la tienda y no dan errores de JavaScript.

## Defectos encontrados y corregidos durante la fase

| Defecto | Dónde | Corrección |
| :--- | :--- | :--- |
| Una pieza movida (o una parcela comprada después) podía dejar de ser clicable: el cálculo de colisiones de las mallas instanciadas guardaba una esfera de contorno antigua | `Decor3D`, `Pads` | Se recalcula al reubicar las instancias |
| El panel de Mejoras, más largo, quedaba tapado por el dock | CSS (capas) | La barra superior pasa por encima y los paneles tienen scroll |
| Los precios de 4 cifras salían sin separador ("4000") | `formatPrice`, `economy.ts` | Separador de millares siempre |
| La colmena detrás de la planta no se podía tocar | `Automation3D` | Va en la esquina izquierda de la parcela |
| Decorar solo se podía con ratón | `DecorBar`, atajos | Cursor de teclado (ver arriba) |
| Con el terreno lleno el modo decorar no decía nada | `DecorBar` | Avisa: "No quedan sitios libres: quita alguna pieza" |
| La radio principal de Lofi Girl no se puede incrustar (error 150) | `stations.ts` | Emisiones por defecto sustituidas por tres que sí suenan, comprobadas con el reproductor real |
| Al implementarlo: tras recargar, el ambiente elegido no arrancaba porque `resume()` es asíncrono | `engine.ts` | Los oyentes de desbloqueo se avisan al estar realmente en marcha |

## Observaciones de diseño (no son defectos; decisión tuya)

1. **La decoración casi no absorbe 🪙.** El GDD dice que la decoración "absorbe el exceso de 🪙", pero el área de 8×8 tiene 46 celdas libres: con todas de farolillo (80 🪙) son **3.680 🪙** más el sombrero (600), es decir **unos 3,5 días** de ingresos de una granja completa (1.199 🪙/día). Después, las monedas no tienen en qué gastarse hasta el prestigio (Fase 4). Si quieres un sumidero real: piezas más caras y vistosas (fuentes, estanques, arcos), zonas de terreno ampliables, o decoración que se renueve por estaciones. Se puede decidir al llegar a P4-03.
2. **El audio solo se ha medido, no escuchado.** Las señales están en el rango esperado (sin saturar), pero el timbre de los efectos y del ambiente sintetizado es una apreciación humana; todos los parámetros son constantes fáciles de afinar.
3. **YouTube puede retirar o restringir emisiones** en cualquier momento (ya pasó con la radio principal). Es configuración, hay fallback y se pueden pegar enlaces propios; conviene revisar las tres por defecto de vez en cuando.

## Sin verificar

- **Safari en un iPhone real** (P1-08d): WebKit de escritorio pasa, pero el audio en iOS (silenciador físico, desbloqueo por gesto) y el iframe de YouTube en iOS solo se confirman en el dispositivo. Pendiente de que lo pruebes en el móvil: `wyse` ya tiene esta versión.
- **GPU real:** el rendimiento se midió con swiftshader (software). 75 draw calls y 18 k triángulos son cifras bajas, pero los FPS reales en un móvil de gama baja no se han medido.
- **Escucha humana** de efectos, ambiente y música (ver arriba).
