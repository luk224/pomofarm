# Diseño de la interfaz (P1-06)

Sujeto: alguien que se concentra durante horas con la granja de fondo. **La UI susurra mientras corre el Pomodoro** y solo habla cuando hay una decisión (elegir semilla, cosechar).

## Tokens
| Rol | Valor |
| :--- | :--- |
| Tinta (texto) | `#3b2a1f`; suave `#5a4a3d` (5,6:1 sobre kraft) |
| Hoja (acción principal, anillo) | `#2f7d32` / hover `#276b2a` |
| Agua (💧) | `#1f84c9` para gráficos y texto grande; `#17689f` para texto pequeño y fondos de botón (6:1) |
| Sol (listo para cosechar) | `#ffc400` |
| Kraft (sobres de semillas) | `#e6cf9f` / borde `#c9ad74` |
| Panel | blanco 88 % con desenfoque, texto tinta |
| Aviso | `#b3261e` |

Tipografía: **Fredoka** variable (OFL, empaquetada en local), una sola familia. Escala: 13 / 15 / 18 / 28 (temporizador). Líneas de texto < 60 caracteres.

## Elemento memorable
Las semillas son **sobres de papel kraft** (esquina doblada, ilustración, precio como sello), ligeramente inclinados; el elegido se levanta. Todo lo demás es discreto.

## Estados del dock (abajo, centrado)
```
reposo      [sobre][sobre][sobre][sobre][sobre]  [etiqueta……]  [Plantar]
en curso    ( 9:41 · Margarita · "emails" )  [Pausar] [Cancelar]
lista       ( Margarita lista )  [Cosechar +1 💧]      (también clic en la planta)
cosechada   ( Ya cosechada )  [Retirar planta]
```
Arriba a la izquierda: chip de 💧. Móvil: el dock ocupa el ancho y los sobres se desplazan en horizontal.

## Principios
- Un verbo, un nombre: Plantar / Pausar / Reanudar / Cancelar / Cosechar / Retirar / Desbloquear, igual en botón, aviso y atajo.
- Movimiento solo como respuesta a una acción (sobre que se levanta, +💧 que sube). Respeta `prefers-reduced-motion`.
- Errores en lenguaje llano, con qué hacer. Nunca se disculpan.
- Atajos: Espacio pausa/reanuda, Intro planta, H cosecha.
- Accesibilidad: foco visible, objetivos ≥ 44 px, no depender solo del color, `aria-live` para avisos.
