import { useEffect, useRef, useState } from 'react'
import { useGame } from '../store/game'
import { AMBIENT_KINDS, isAmbientKind } from '../audio/ambient'
import { playBowl } from '../audio/bowl'
import { playEffect } from '../audio/effects'
import { unlockAudio, type Layer } from '../audio/engine'
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

const AMBIENT_LABELS: Record<string, string> = { off: 'Sin ambiente', rain: 'Lluvia', forest: 'Bosque', fire: 'Fuego' }
const LAYER_LABELS: Record<Layer, string> = { ambient: 'Ambiente', effects: 'Efectos', alerts: 'Alertas' }
/** Plays a short sample of the layer, so the player hears what the slider just set. */
const PREVIEWS: Partial<Record<Layer, () => void>> = { effects: () => playEffect('coin'), alerts: () => playBowl(0.5) }

/** Three independent volumes (ambient, effects, alerts) and the local ambient sound (GDD §2, ASMR). Saved on this device. */
function SoundSettings() {
  const volumes = usePrefs((s) => s.volumes)
  const ambient = usePrefs((s) => s.ambient)
  const setVolume = usePrefs((s) => s.setVolume)
  const setAmbient = usePrefs((s) => s.setAmbient)
  return (
    <fieldset className="restset" data-testid="sound-settings">
      <legend>Sonido</legend>
      <label className="restfield">
        <span>Ambiente</span>
        <select value={ambient} data-testid="ambient-kind"
          onChange={(e) => { unlockAudio(); if (isAmbientKind(e.target.value)) setAmbient(e.target.value) }}>
          {AMBIENT_KINDS.map((k) => <option key={k} value={k}>{AMBIENT_LABELS[k]}</option>)}
        </select>
      </label>
      {(Object.keys(LAYER_LABELS) as Layer[]).map((l) => (
        <label key={l} className="slider">
          <span>{LAYER_LABELS[l]}</span>
          <input type="range" min={0} max={100} step={1} value={Math.round(volumes[l] * 100)} data-testid={`volume-${l}`}
            aria-label={`Volumen de ${LAYER_LABELS[l].toLowerCase()}`} aria-valuetext={`${Math.round(volumes[l] * 100)} por ciento`}
            onChange={(e) => { unlockAudio(); setVolume(l, Number(e.target.value) / 100) }}
            onPointerUp={() => PREVIEWS[l]?.()} onKeyUp={() => PREVIEWS[l]?.()} />
          <output>{Math.round(volumes[l] * 100)}</output>
        </label>
      ))}
    </fieldset>
  )
}

/** Accessibility (GDD §2.4): no sound with visual alerts, fewer animations, and a colour-blind-safe palette. Saved on this device. */
function AccessibilitySettings() {
  const muted = usePrefs((s) => s.muted)
  const reduceMotion = usePrefs((s) => s.reduceMotion)
  const palette = usePrefs((s) => s.palette)
  const setMuted = usePrefs((s) => s.setMuted)
  const setReduceMotion = usePrefs((s) => s.setReduceMotion)
  const setPalette = usePrefs((s) => s.setPalette)
  const osReduces = typeof matchMedia === 'function' && matchMedia('(prefers-reduced-motion: reduce)').matches
  return (
    <fieldset className="restset" data-testid="a11y-settings">
      <legend>Accesibilidad</legend>
      <label className="check">
        <input type="checkbox" checked={muted} data-testid="a11y-muted" onChange={(e) => setMuted(e.target.checked)} />
        <span>Sin audio</span>
      </label>
      <p className="hint">Silencia todo (ambiente, efectos, alertas y música). Cuando algo termine verás un marco dorado y un aviso. Atajo: M.</p>
      <label className="check">
        <input type="checkbox" checked={reduceMotion || osReduces} disabled={osReduces} data-testid="a11y-motion" onChange={(e) => setReduceMotion(e.target.checked)} />
        <span>Reducir animaciones</span>
      </label>
      <p className="hint">{osReduces ? 'Tu sistema ya lo pide, así que está activado.' : 'Quita el balanceo y los saltos de las plantas y los movimientos de la interfaz.'}</p>
      <label className="check">
        <input type="checkbox" checked={palette === 'cb'} data-testid="a11y-palette" onChange={(e) => setPalette(e.target.checked ? 'cb' : 'default')} />
        <span>Paleta para daltonismo</span>
      </label>
      <p className="hint">Azul y naranja en lugar de verde y rojo. Los números y las formas ya acompañan siempre al color.</p>
    </fieldset>
  )
}

/** Strict mode (GDD §3.3): a personal challenge. It never blocks a pause; it only decides what the Harvest Book counts as clean. */
function StrictSetting() {
  const on = useGame((s) => s.state?.settings.strict_mode === '1')
  const setSetting = useGame((s) => s.setSetting)
  const [wanted, setWanted] = useState<{ value: boolean; base: boolean } | null>(null)
  const shown = wanted && wanted.base === on ? wanted.value : on
  return (
    <fieldset className="restset" data-testid="strict-setting">
      <legend>Reto personal</legend>
      <label className="check">
        <input type="checkbox" checked={shown} data-testid="strict-mode"
          onChange={(e) => { setWanted({ value: e.target.checked, base: on }); void setSetting('strict_mode', e.target.checked ? '1' : '0') }} />
        <span>Modo estricto</span>
      </label>
      <p className="hint">Máximo 2 pausas y 10 minutos de pausa por Pomodoro. No cambia nada del juego: puedes pausar cuanto quieras. Los que cumplan se cuentan como limpios en el Libro.</p>
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
            <span>Campana al terminar</span>
          </label>
          <label className="check">
            <input type="checkbox" checked={notify && !blocked} disabled={!supported || blocked} onChange={(e) => setNotify(e.target.checked)} />
            <span>Notificación del navegador</span>
          </label>
          {!supported && <p className="hint">Este navegador no admite notificaciones.</p>}
          {blocked && <p className="hint">Las notificaciones están bloqueadas en el navegador. Actívalas en los permisos del sitio.</p>}
          {!sound && <p className="hint">Sin campana, verás el aviso en la pestaña.</p>}
          <AccessibilitySettings />
          <SoundSettings />
          <RestSettings />
          <StrictSetting />
          <fieldset className="restset" data-testid="shortcuts">
            <legend>Atajos</legend>
            <dl className="keys">
              <dt>Espacio</dt><dd>Pausar o reanudar</dd>
              <dt>Intro</dt><dd>Plantar la semilla elegida</dd>
              <dt>H</dt><dd>Cosechar</dd>
              <dt>C</dt><dd>Recoger el Silo</dd>
              <dt>M</dt><dd>Silenciar todo</dd>
              <dt>← →</dt><dd>Cambiar de parcela</dd>
            </dl>
          </fieldset>
        </div>
      )}
    </div>
  )
}
