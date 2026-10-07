import { useEffect, useRef, useState } from 'react'
import { useGame } from '../store/game'
import { usePrefs } from '../store/prefs'

const supported = typeof Notification !== 'undefined'

const REST_MIN = 1
const REST_MAX = 60

/** One rest length in minutes. Saved when the field loses focus or Enter is pressed; out-of-range values are clamped. */
function RestMinutes({ label, settingKey, fallback }: { label: string; settingKey: string; fallback: number }) {
  const stored = useGame((s) => s.state?.settings[settingKey])
  const setSetting = useGame((s) => s.setSetting)
  const current = Number(stored) >= REST_MIN && Number(stored) <= REST_MAX ? Number(stored) : fallback
  const [draft, setDraft] = useState<string | null>(null)
  const commit = () => {
    if (draft === null) return
    const n = Math.min(REST_MAX, Math.max(REST_MIN, Math.round(Number(draft)) || current))
    setDraft(null)
    if (n !== current) void setSetting(settingKey, String(n))
  }
  return (
    <label className="restfield">
      <span>{label}</span>
      <input type="number" inputMode="numeric" min={REST_MIN} max={REST_MAX} value={draft ?? current} data-testid={`rest-input-${settingKey}`}
        onChange={(e) => setDraft(e.target.value)} onBlur={commit} onKeyDown={(e) => e.key === 'Enter' && commit()} />
      <span>min</span>
    </label>
  )
}

/** Rest after each Pomodoro (GDD §3.2): on/off and the three lengths. Stored in the game, so every device shares them. */
function RestSettings() {
  const enabled = useGame((s) => s.state?.settings.rest_enabled !== '0')
  const setSetting = useGame((s) => s.setSetting)
  // Show the new value at once and let the server's answer confirm it: over a slow link a controlled
  // checkbox that waits for the answer would flicker back for a moment.
  // The pending choice remembers the server value it was made against; once the server's value changes
  // (confirming it), the pending choice is simply ignored: no effect needed.
  const [wanted, setWanted] = useState<{ value: boolean; base: boolean } | null>(null)
  const shown = wanted && wanted.base === enabled ? wanted.value : enabled
  return (
    <fieldset className="restset">
      <legend>Descansos</legend>
      <label className="check">
        <input type="checkbox" checked={shown} data-testid="rest-enabled"
          onChange={(e) => { setWanted({ value: e.target.checked, base: enabled }); void setSetting('rest_enabled', e.target.checked ? '1' : '0') }} />
        <span>Proponer un descanso al cosechar</span>
      </label>
      {shown && (
        <>
          <RestMinutes label="Pomodoros de hasta 25 min" settingKey="rest_short_min" fallback={5} />
          <RestMinutes label="De 26 a 45 min" settingKey="rest_medium_min" fallback={10} />
          <RestMinutes label="De más de 45 min" settingKey="rest_long_min" fallback={15} />
        </>
      )}
    </fieldset>
  )
}

export function Settings() {
  const [open, setOpen] = useState(false)
  const sound = usePrefs((s) => s.sound)
  const notify = usePrefs((s) => s.notify)
  const setSound = usePrefs((s) => s.setSound)
  const setNotify = usePrefs((s) => s.setNotify)
  const root = useRef<HTMLDivElement>(null)
  const blocked = supported && Notification.permission === 'denied'

  useEffect(() => {
    if (!open) return
    const onKey = (e: KeyboardEvent) => e.key === 'Escape' && setOpen(false)
    const onDown = (e: PointerEvent) => !root.current?.contains(e.target as Node) && setOpen(false)
    window.addEventListener('keydown', onKey)
    window.addEventListener('pointerdown', onDown)
    return () => {
      window.removeEventListener('keydown', onKey)
      window.removeEventListener('pointerdown', onDown)
    }
  }, [open])

  return (
    <div className="settings" ref={root}>
      <button type="button" className="chip chip--button" aria-expanded={open} aria-controls="settings-panel"
        aria-label="Ajustes de avisos" onClick={() => setOpen(!open)}>
        <svg width="20" height="20" viewBox="0 0 20 20" aria-hidden="true" focusable="false">
          <path d="M10 2.5a1.2 1.2 0 0 1 1.2 1.2v.6a5 5 0 0 1 3.1 3.1l.1 2.2 1.5 2v1.3H4.1v-1.3l1.5-2 .1-2.2a5 5 0 0 1 3.1-3.1v-.6A1.2 1.2 0 0 1 10 2.5zM8 15.2h4a2 2 0 0 1-4 0z" fill="currentColor" />
        </svg>
      </button>
      {open && (
        <div id="settings-panel" className="panelbox" role="group" aria-label="Avisos al terminar">
          <label className="check">
            <input type="checkbox" checked={sound} onChange={(e) => setSound(e.target.checked)} />
            <span>Sonido al terminar</span>
          </label>
          <label className="check">
            <input type="checkbox" checked={notify && !blocked} disabled={!supported || blocked} onChange={(e) => setNotify(e.target.checked)} />
            <span>Notificación del navegador</span>
          </label>
          {!supported && <p className="hint">Este navegador no admite notificaciones.</p>}
          {blocked && <p className="hint">Las notificaciones están bloqueadas en el navegador. Actívalas en los permisos del sitio.</p>}
          {!sound && <p className="hint">Sin sonido, verás el aviso en la pestaña.</p>}
          <RestSettings />
          <fieldset className="restset" data-testid="shortcuts">
            <legend>Atajos</legend>
            <dl className="keys">
              <dt>Espacio</dt><dd>Pausar o reanudar</dd>
              <dt>Intro</dt><dd>Plantar la semilla elegida</dd>
              <dt>H</dt><dd>Cosechar</dd>
              <dt>C</dt><dd>Recoger el Silo</dd>
              <dt>← →</dt><dd>Cambiar de parcela</dd>
            </dl>
          </fieldset>
        </div>
      )}
    </div>
  )
}
