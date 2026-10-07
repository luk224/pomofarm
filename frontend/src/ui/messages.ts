/** Server error code -> what the player should read. Says what happened and what to do; never apologises. */
const MESSAGES: Record<string, string> = {
  network: 'No hay conexión con el servidor. Comprueba que estás en Tailscale.',
  no_player: 'La granja aún no está creada en el servidor.',
  pomodoro_active: 'Ya hay un Pomodoro en marcha en otro dispositivo.',
  no_active_pomodoro: 'No hay ningún Pomodoro en marcha.',
  plot_busy: 'Esa parcela está ocupada.',
  not_found: 'No se encontró la parcela.',
  seed_locked: 'Esa semilla aún no está desbloqueada.',
  already_unlocked: 'Ya tienes esa semilla.',
  insufficient_focus: 'Te faltan 💧 para desbloquearla. Se ganan completando Pomodoros.',
  not_mature: 'La planta aún no ha terminado de crecer.',
  already_harvested: 'Esa planta ya está cosechada.',
  harvest_first: 'Cosecha la planta antes de retirarla.',
  needs_confirmation: 'Confirma para retirar la planta.',
  silo_empty: 'El Silo está vacío: aún no ha producido nada que recoger.',
  wrong_state: 'Eso ya está hecho.',
  version_conflict: 'La granja cambió en otro dispositivo. Ya la he recargado.',
  invalid_request: 'Esa petición no es válida.',
}

export function messageFor(code: string): string {
  return MESSAGES[code] ?? 'Algo ha fallado. Inténtalo de nuevo.'
}
