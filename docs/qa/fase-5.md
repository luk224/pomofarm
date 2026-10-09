# Informe de la Fase 5 (robustez)

Fecha: 2026-10-09 · Commit base: `4db90ad` + los de esta tarea.

## Qué se hizo

| Tarea | Resultado |
| :--- | :--- |
| P5-01 Modo Foco | Mientras corre un Pomodoro la escena se dibuja a **20 FPS** (medido: 20,0 FPS), con resolución 1× y sombras a 2 Hz; con la pestaña oculta, **0 FPS**. En el equipo de pruebas (render por software) la CPU de la página baja de 101 % a 67 %. Nunca dibuja más que el bucle normal. Ajuste desactivable |
| P5-02 «Tu granja necesita atención» | Aviso suave (≥ 3 plantas en 2 h, una vez por grupo, 6 h de calma, nunca en un Pomodoro). Se calcula en el navegador; con el navegador cerrado depende de P5-06 |
| P5-03 Suite de tiempo | `docs/qa/tiempo.md` + `make test-time`: 44 pruebas de Go y 4 scripts de navegador enlazados con cada criterio del GDD §8. Nuevas: reinicio del servidor a mitad, pausa de 30 días, saltos y extremos del reloj, zona horaria del servidor, "consultar a menudo = consultar una vez" en partidas aleatorias completas. 2 mutaciones detectadas |
| P5-04 Backups y restauración | Copias verificadas antes de guardarse, copia verificada automática antes de migrar, restauración que verifica primero y se puede deshacer, `pomofarm verify`, `fetch-backups.sh`, `status.sh`. 9 tests, 3 mutaciones detectadas. Informe: `docs/qa/backups.md` |
| P5-05 Despliegue final en `wyse` | Ver abajo |
| P5-06 Web Push | **Pendiente** (decisión tuya: se deja sin hacer; ver "Pendiente") |

## P5-05: despliegue final y pruebas en wyse (2026-10-09)

- **Desplegado** (dos veces: la versión completa y la corrección de nginx), con copia previa de la BD de `wyse` en `~/pomofarm-backups/wyse-before-p5-*`. Esquema 6, partida real intacta, puerto solo en la IP de Tailscale (no responde por la LAN).
- **Arranque con el sistema (comprobado, solo lectura):** `docker` y `containerd` están `enabled` y `active`; los dos contenedores tienen `restart: unless-stopped`; el disco tiene 425 GB libres. **No se reinició `wyse`** (aloja AdGuard Home y otros servicios).
- **El servidor se reinicia solo (comprobado en wyse):** al morir el proceso del servidor (`kill -TERM 1` dentro del contenedor, que no cuenta como parada manual) Docker lo reinició en 2 s y el juego volvió a responder. Nota: `docker kill` y `docker stop` **sí** cuentan como paradas manuales y no se reinician solos; eso es lo esperado de `unless-stopped`.
- **Defecto encontrado y corregido:** nginx resolvía `app` una sola vez al arrancar, así que **no arrancaba si arrancaba antes que el servidor** (`host not found in upstream`), que es justo lo que puede pasar al encender `wyse`, y habría guardado una dirección vieja si el servidor cambiaba de IP. Ahora resuelve en cada petición. Probado en Docker limpio: la web arranca sin el servidor, contesta 502 mientras no está, lo encuentra sola cuando vuelve y no necesita reiniciarse.
- **Copias en wyse:** 4 copias, la última verificada como válida; `deploy/status.sh` lo comprueba y avisa si pasa de 36 h.
- **Restauración probada en wyse** (`tools/e2e/qa_restore_remote.sh`, 10 comprobaciones): en un stack aparte (`pfqa`, puerto local de wyse) se restaura una copia fresca de producción con `restore.sh` y los datos coinciden con producción (nombre, 💧 de por vida, semillas, plantas); una copia estropeada se rechaza; producción sigue respondiendo; al terminar no queda ningún contenedor, volumen ni imagen de `pfqa`.
- **Desde otro dispositivo de la tailnet:** el móvil lo usas ya a diario desde Tailscale. **iPhone real sigue sin probarse por mí** (ver Pendiente).

## Pendiente (tuyo)

1. **iPhone real** (P1-08d): audio, silenciador físico, el iframe de YouTube y el rendimiento. WebKit de escritorio pasa todo.
2. **Escuchar y ver**: efectos, ambiente, música, paletas (verano, otoño, invierno, daltonismo) y los iconos del Libro por una persona.
3. **Decisiones de balance** (informe de la Fase 4): ritmo de la 2.ª estación (41 días en perfil Normal) y sumidero de 🪙.
4. **P5-06 Web Push**, pendiente a propósito.
5. **Copias fuera de `wyse`**: `deploy/fetch-backups.sh` cuando quieras (o automatizarlo en tu PC).
6. Confirmar las decisiones del GDD (§10) y jugar una semana para recalibrar precios.
