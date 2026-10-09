# Operación de PomoFarm en wyse

Todo se maneja desde este repositorio con scripts que hablan con `wyse` por SSH/Docker. Si estás en casa, usa la IP de la red local para SSH
(`WYSE_HOST=luk@192.168.1.11`); el puerto del juego se publica **solo** en la IP de Tailscale (`100.102.106.119:8080`), vengas por donde vengas.

| Quiero… | Comando |
| :--- | :--- |
| Desplegar la versión de este equipo | `WYSE_HOST=luk@192.168.1.11 deploy/deploy.sh` (pide confirmación; `-y` para no preguntar) |
| Ver si todo va bien (salud, última copia, espacio) | `WYSE_HOST=luk@192.168.1.11 deploy/status.sh` |
| Traer una copia de los backups a este equipo | `WYSE_HOST=luk@192.168.1.11 deploy/fetch-backups.sh` (a `~/pomofarm-backups/wyse`) |
| Restaurar la copia más reciente | `DOCKER_HOST=ssh://luk@192.168.1.11 deploy/restore.sh` |
| Restaurar una concreta | `DOCKER_HOST=ssh://luk@192.168.1.11 deploy/restore.sh pomofarm-20261009-101046.db` |
| Deshacer la última restauración | `DOCKER_HOST=ssh://luk@192.168.1.11 deploy/restore.sh --undo` |

## Dónde están los datos

Dos volúmenes de Docker en `wyse`: `pomofarm_data` (la base de datos SQLite, `/data`) y `pomofarm_backups` (`/backups`). Nunca dentro de la imagen.
`deploy/deploy.sh` reconstruye las imágenes sin tocar los volúmenes. **No uses `docker compose down -v`** ni `docker volume rm`: borran la partida
**y** sus copias a la vez, porque están en la misma máquina (por eso existe `fetch-backups.sh`).

## Copias de seguridad

- **Diaria:** el servidor hace una copia consistente (`VACUUM INTO`) al arrancar y cada vez que la última tiene más de 24 h. Se conservan las **14** últimas
  (`/backups/pomofarm-AAAAMMDD-HHMMSS.db`).
- **Verificada:** cada copia nueva se abre y se comprueba (integridad de SQLite y tablas del juego) **antes** de guardarla. Si no vale, se descarta, se avisa en el
  log y **no se borra ninguna copia buena** por culpa de una mala.
- **Antes de actualizar:** si una versión nueva trae migraciones y ya hay una partida, se guarda una copia verificada de la base de datos tal como estaba en
  `/backups/pre-migration/pomofarm-v<esquema>-….db` (las 5 últimas) **antes** de migrar. Si no se puede hacer esa copia, la actualización no arranca.
- **Fuera de wyse:** `deploy/fetch-backups.sh` trae todo a este equipo y verifica cada archivo. Conviene ejecutarlo de vez en cuando (p. ej. cada semana).
- **Comprobar una copia a mano:** `pomofarm verify <archivo>` (en el contenedor: `docker compose exec app pomofarm verify /backups/…`).

## Restaurar

1. `deploy/restore.sh [copia]` **verifica la copia primero**. Si no es válida, no cambia nada.
2. Para la app, **guarda la partida actual entera** en `/backups/pre-restore/<fecha>/` (se conservan las 5 últimas), coloca la copia y arranca.
3. Comprueba que la app responde. Si algo no es lo esperado, `deploy/restore.sh --undo` devuelve la partida anterior.

Una copia **antigua** (de un esquema anterior) se actualiza sola al arrancar, con su copia de seguridad previa en `/backups/pre-migration/`.

## Qué comprueban las pruebas

`tools/e2e/qa_restore.sh` levanta un stack Docker limpio y comprueba: restaurar en una instalación vacía, una copia estropeada (también si es «la última») y una inexistente
se rechazan sin tocar nada, `--undo`, que una copia de un esquema antiguo se actualiza con su copia previa verificada, y que todo sobrevive a reiniciar el contenedor.
Los tests de Go (`internal/store`) cubren la verificación, la copia previa a la migración y la retención.
