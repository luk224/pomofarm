# PomoFarm 3D

Temporizador Pomodoro + granja idle en 3D isométrico. Diseño en `gdd_pomofarm_3d_v2.md`, tareas en `PLANNING.md`.

## Entorno

| Dónde | Versiones |
| :--- | :--- |
| Desarrollo (thinkpad) | Go 1.26.0, Node 24, Docker 29 + Compose v5.1.4 |
| Servidor `wyse` | Debian 13 (kernel 6.12, **amd64**), Docker 29.0.1 + Compose v2.40.3, 7,6 GB RAM, 429 GB libres |

`wyse`: Tailscale `100.102.106.119`, usuario `luk` (grupo `docker`), acceso SSH por clave. Puertos 80, 443 y 8080 libres (comprobado el 2026-10-07).
