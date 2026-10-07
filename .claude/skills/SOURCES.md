# Origen de las skills (vendorizadas, fijadas a un commit)

Revisadas a mano antes de copiarlas (SKILL.md y scripts). No se actualizan solas: para actualizar, revisar el diff del upstream.

| Skills | Origen | Commit | Licencia | Scripts |
| :--- | :--- | :--- | :--- | :--- |
| `frontend-design`, `webapp-testing` | https://github.com/anthropics/skills | `683bc88` | Apache-2.0 | `webapp-testing/scripts/with_server.py` (lanza el servidor que le indiques con `subprocess`; sin red) |
| `threejs-impl-react-three-fiber`, `-drei`, `-lighting`, `-shadows`, `-animation`, `-audio`, `threejs-errors-performance`, `-rendering`, `threejs-agents-scene-builder` | https://github.com/Impertio-Studio/Three.js-Claude-Skill-Package (antes OpenAEC-Foundation) | `6c190f0` | MIT | ninguno |
| `golang-testing`, `-database`, `-error-handling`, `-concurrency`, `-security`, `-safety`, `-context` | https://github.com/samber/cc-skills-golang | `8e899e2` | MIT | ninguno (declaran `allowed-tools` con `Bash(go:*)`, `git`, etc.) |

## Valoradas y NO instaladas

| Candidata | Motivo |
| :--- | :--- |
| `threejs-agents-model-optimizer` (Impertio) | Solo útil si importamos modelos .glb; de momento el 3D es procedural. Añadir si se usan packs de Kenney/Quaternius. |
| `vercel-labs/agent-skills` → `react-best-practices` | 70 reglas pensadas para Next.js/RSC; poco aplicable a Vite + R3F. |
| `vercel-labs/agent-skills` → `web-design-guidelines` | Descarga sus instrucciones de una URL remota en cada uso (el contenido puede cambiar sin revisión). Mejor una checklist propia de accesibilidad. |
| Skills de Docker de agregadores (claudskills, vibeindex…) | Reputación y mantenimiento poco claros; el despliegue ya está hecho y probado. |
| `golang-lint`, `golang-continuous-integration` (samber) | Requieren `golangci-lint` y CI, que aún no tenemos. Valorar en Fase 5. |
| Blender MCP (`ahujasid/blender-mcp`) | Requiere Blender + addon. Valorar si el 3D procedural se queda corto. |
