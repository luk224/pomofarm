import { useEffect } from 'react'
import { useGame } from '../store/game'
import { useRemainingMs } from '../store/hooks'
import { formatClock, tabTitle } from '../store/time'

const box: React.CSSProperties = {
  position: 'fixed', top: 8, left: 8, padding: '8px 12px', borderRadius: 10,
  background: '#fffd', font: '14px system-ui', display: 'grid', gap: 4, minWidth: 150,
}

/** Minimal readout; the real HUD arrives with P1-06. */
export function Hud() {
  const state = useGame((s) => s.state)
  const error = useGame((s) => s.error)
  const ms = useRemainingMs()
  const pomodoro = state?.pomodoro ?? null

  useEffect(() => {
    document.title = tabTitle(pomodoro, ms ?? 0)
  }, [pomodoro, ms])

  if (!state) return <div style={box}>{error ? `sin conexión (${error})` : 'cargando…'}</div>
  return (
    <div style={box}>
      <div data-testid="focus">💧 {state.player.focus_points}</div>
      {pomodoro && ms !== null ? (
        <div data-testid="timer" data-status={pomodoro.status}>
          {pomodoro.status === 'paused' ? '⏸' : '⏱'} {formatClock(ms)} · {pomodoro.plant_type}
        </div>
      ) : (
        <div data-testid="timer" data-status="idle">sin Pomodoro</div>
      )}
      {error && <div data-testid="error" style={{ color: '#b00020' }}>{error}</div>}
    </div>
  )
}
