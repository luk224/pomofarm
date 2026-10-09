# Revisión de backups y restauración (P5-04)

Fecha: 2026-10-09 · Revisado: `store/backup.go`, `cmd/pomofarm/main.go`, `docker-compose.yml`, `deploy/*.sh`, `tools/e2e/qa_restore.sh`.

## Hallazgos y qué se hizo

| # | Hallazgo | Riesgo | Corrección |
| :-- | :--- | :--- | :--- |
| 1 | Una copia nueva nunca se comprobaba, y al guardarla se **borraban las más antiguas** | Si las copias salían estropeadas (disco dañado, corte a mitad), en 14 días no quedaría ninguna buena | Cada copia se abre y se verifica (`PRAGMA integrity_check` + tablas del juego) antes de guardarla; si falla se descarta y no se borra nada. Probado con una verificación forzada a fallar |
| 2 | `restore.sh` **borraba la partida actual antes de comprobar la copia** | Restaurar una copia dañada (o la equivocada) destruía la partida sin remedio | Se verifica primero con la propia app; la partida actual se guarda entera en `/backups/pre-restore/<fecha>/` (5 últimas); `restore.sh --undo` la recupera; al final se comprueba que la app responde |
| 3 | Las actualizaciones migraban la BD **sin copia previa** (yo la hacía a mano en cada despliegue) | Una migración defectuosa o una versión con un fallo podía dañar la partida real | `OpenWithSafety`: si hay migraciones pendientes y ya hay partida, se guarda una copia verificada en `/backups/pre-migration/` (5 últimas) **antes** de migrar; si no se puede, la actualización no arranca |
| 4 | Las copias viven solo en un volumen de la misma máquina que la partida | Un fallo de disco de `wyse` o un `docker volume rm` se llevaría partida y copias | `deploy/fetch-backups.sh` las trae a este equipo y las verifica. Documentado en `deploy/README.md` |
| 5 | No había forma rápida de ver el estado de las copias | Se podría estar sin copias recientes sin enterarse | `deploy/status.sh`: salud, esquema, número y antigüedad de las copias, y si la última es válida (avisa si pasa de 36 h) |
| 6 | No había comando para comprobar una copia a mano | — | `pomofarm verify <archivo>` |

Observado en `wyse` (solo lectura): 3 copias (7 y 8 de octubre), la última de ~18 h, todas válidas.
La copia diaria se hace al arrancar y cuando la última supera 24 h, así que la hora del día se desplaza; da igual mientras haya al menos una cada 24 h.

## Pruebas

- **Go (`internal/store`, 9 pruebas nuevas):** `Verify` acepta una copia sana y no la toca (mismo hash); rechaza ocho tipos de archivo (inexistente, vacío, bytes aleatorios, truncado, páginas dañadas, otra base SQLite, sin migraciones, un directorio); una copia que no verifica se descarta y las buenas sobreviven (incluida la más antigua); la copia previa a la migración se hace con el esquema anterior y datos intactos, no se hace sin actualización ni en instalación nueva, la actualización no arranca si no se puede hacer, se conservan solo las 5 últimas y no se confunde con las copias diarias. **Tres defectos introducidos a propósito se detectan** (no verificar, no hacer la copia previa, comprobación de integridad débil).
- **Docker limpio (`tools/e2e/qa_restore.sh`, ahora 21 comprobaciones):** instalación vacía → restaurar → jugable → reinicio; una copia estropeada que además es «la última» se rechaza y la partida sigue igual; una inexistente se rechaza; la partida anterior queda guardada y `--undo` la devuelve exactamente; una copia **real de esquema 5** (tu partida de ayer) se restaura, se actualiza a 6, deja su copia previa verificada y conserva los datos.
- **Fuera de wyse:** `fetch-backups.sh` contra `wyse` (solo lectura): 3 copias traídas, 0 inválidas.

## Pendiente

- **Desplegar esta versión en `wyse`** (necesita tu confirmación): hasta entonces `wyse` hace copias sin verificar y `status.sh` no puede comprobar validez desde dentro (el binario antiguo no tiene `verify`).
- **Automatizar `fetch-backups.sh`** en este equipo (p. ej. semanal) si quieres copias fuera de `wyse` sin acordarte. Es decisión tuya: lo deja instalado un cron o un temporizador de systemd en tu PC.
- **Arranque tras reiniciar `wyse`** y copias en `wyse`: se verifican en P5-05.
