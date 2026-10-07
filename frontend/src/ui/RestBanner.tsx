import { useGame } from '../store/game'
import { useRestRemainingMs } from '../store/hooks'
import { restProgress } from '../store/rest'
import { formatClock } from '../store/time'

function CupIcon() {
  return (
    <svg width="22" height="22" viewBox="0 0 22 22" aria-hidden="true" focusable="false">
      <path d="M4 8h11v5a5 5 0 0 1-5 5H9a5 5 0 0 1-5-5z" fill="#f5efe4" stroke="#8a5a3b" strokeWidth="1.4" />
      <path d="M15 9.5h1.5a2.5 2.5 0 0 1 0 5H14.6" fill="none" stroke="#8a5a3b" strokeWidth="1.4" />
      <path d="M7 3.5c-.8 1 .8 1.6 0 2.7M10.5 3.5c-.8 1 .8 1.6 0 2.7" fill="none" stroke="#6b5a4c" strokeWidth="1.1" strokeLinecap="round" />
    </svg>
  )
}

/**
 * The optional break (GDD §3.2). It never blocks anything: the dock stays usable, so planting the next
 * Pomodoro simply ends the rest. The countdown is for information; "Saltar descanso" ends it early.
 */
export function RestBanner() {
  const rest = useGame((s) => s.state?.rest ?? null)
  const skip = useGame((s) => s.skipRest)
  const ms = useRestRemainingMs()
  if (!rest || ms === null || ms <= 0) return null
  const done = restProgress(rest, ms)
  return (
    <div className="rest" data-testid="rest">
      <CupIcon />
      <div className="rest__text">
        <span className="rest__label">Descanso</span>
        <span className="rest__time" role="timer" data-testid="rest-time">{formatClock(ms)}</span>
      </div>
      <div className="rest__bar" aria-hidden="true"><span style={{ width: `${done * 100}%` }} /></div>
      <button type="button" className="btn btn--quiet" data-testid="rest-skip" onClick={() => void skip()}>Saltar descanso</button>
    </div>
  )
}
