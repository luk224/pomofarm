import { useEffect, useRef, useState } from 'react'
import { usePrefs } from '../store/prefs'

const supported = typeof Notification !== 'undefined'

/** Per-device alert preferences: chime and browser notifications. */
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
        </div>
      )}
    </div>
  )
}
