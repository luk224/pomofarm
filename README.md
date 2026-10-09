# PomoFarm 3D

Temporizador Pomodoro + granja idle en 3D isométrico. Diseño en `gdd_pomofarm_3d_v2.md`, tareas en `PLANNING.md`.

## Entorno

| Dónde | Versiones |
| :--- | :--- |
| Desarrollo (thinkpad) | Go 1.26.0, Node 24, Docker 29 + Compose v5.1.4 |
| Servidor `wyse` | Debian 13 (kernel 6.12, **amd64**), Docker 29.0.1 + Compose v2.40.3, 7,6 GB RAM, 429 GB libres |

`wyse`: Tailscale `100.102.106.119`, usuario `luk` (grupo `docker`), acceso SSH por clave. Ya hay otros servicios (opengym, latex-to-pdf-api, AdGuard Home en 80/443/53/3000). PomoFarm usa el **8080** ligado solo a la IP de Tailscale.

## Cómo se usa y se opera

| Quiero… | Dónde |
| :--- | :--- |
| Jugar en el móvil o el portátil | `http://100.102.106.119:8080` (con Tailscale activo) |
| Desplegar, ver el estado, copias de seguridad y restauraciones | [`deploy/README.md`](deploy/README.md) |
| Ejecutar todas las pruebas | `make test` (Go y frontend) · `tools/e2e/run_isolated.sh tools/e2e/*.py` (navegador, contra un backend y una BD temporales) |
| Ejecutar la suite de tiempo (GDD §8) | `make test-time` (con `E2E=1` también en navegador) — ver [`docs/qa/tiempo.md`](docs/qa/tiempo.md) |
| Probar la restauración en Docker limpio / en wyse (stack aparte) | `tools/e2e/qa_restore.sh` · `WYSE_HOST=… tools/e2e/qa_restore_remote.sh` |
| Ver el estado de cada fase y sus informes de calidad | [`PLANNING.md`](PLANNING.md) · `docs/qa/fase-1.md` … `fase-5.md` |

El juego se reinicia solo si el servidor falla o si se reinicia `wyse` (Docker arranca con el sistema y los contenedores tienen `restart: unless-stopped`).
