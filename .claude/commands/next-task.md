---
description: Ejecuta la siguiente tarea pendiente de PLANNING.md (o la indicada por ID)
argument-hint: [ID de tarea, p. ej. P1-03]
---

Trabaja en **una sola tarea** de `PLANNING.md`.

1. Lee `CLAUDE.md` y `PLANNING.md`. Si se indicó un ID (`$ARGUMENTS`), usa esa tarea; si no, la primera `[ ]` cuyas dependencias estén todas `[x]`. Si hay bloqueo, dilo y para.
2. Resume en 2–3 líneas qué vas a hacer y qué agente la lleva. Si hay ambigüedad real en el GDD, pregunta antes de empezar.
3. Márcala `[~]`. Impleméntala subtarea a subtarea (delegando en el agente adecuado de `.claude/agents/`), marcando cada subtarea al terminarla.
4. Verifica ejecutando de verdad (tests/build/arranque) y cumple el criterio de "hecho" de la tarea.
5. Márcala `[x]` solo si está verificada; si no, déjala `[~]` o `[!]` con el motivo.
6. Haz un commit con el cambio y la actualización de `PLANNING.md`. No hagas push.
7. Informa: qué se hizo, qué se verificó y cómo, qué queda. **Para aquí**; no empieces la siguiente tarea.
