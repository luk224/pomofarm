import { useEffect, useRef } from 'react'
import { useRingAnchor } from '../store/anchor'
import { useGame } from '../store/game'
import { useRemainingMs } from '../store/hooks'
import { formatClock } from '../store/time'
import { palette as P } from '../scene/palette'

const R = 22
const CIRC = 2 * Math.PI * R

/**
 * Floating circular progress over the active plant (GDD §2). Shows what is LEFT, draining
 * clockwise. Decorative for assistive tech: the HUD carries the accessible readout.
 * Position is written straight to the element's style (no React re-render per frame).
 */
export function TimerRing() {
  const pomodoro = useGame((s) => s.state?.pomodoro ?? null)
  const ms = useRemainingMs()
  const el = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const place = (a: { on: boolean; x: number; y: number }) => {
      const n = el.current
      if (!n) return
      n.style.visibility = a.on ? 'visible' : 'hidden'
      n.style.transform = `translate(${a.x}px, ${a.y}px) translate(-50%, -100%)`
    }
    place(useRingAnchor.getState())
    return useRingAnchor.subscribe(place)
  }, [pomodoro])

  if (!pomodoro || ms === null) return null
  const left = Math.min(1, Math.max(0, ms / (pomodoro.planned_s * 1000)))
  const paused = pomodoro.status === 'paused'
  return (
    <div ref={el} aria-hidden="true" data-testid="timer-ring" data-left={left.toFixed(3)}
      style={{ position: 'fixed', left: 0, top: 0, width: 56, height: 56, pointerEvents: 'none', visibility: 'hidden', willChange: 'transform' }}>
      <svg width="56" height="56" viewBox="0 0 56 56" style={{ transform: 'rotate(-90deg)' }}>
        <circle cx="28" cy="28" r={R} fill="#fffc" stroke={P.ringTrack} strokeWidth="5" />
        <circle cx="28" cy="28" r={R} fill="none" stroke={paused ? P.ringPaused : P.ringRunning} strokeWidth="5"
          strokeLinecap="round" strokeDasharray={CIRC} strokeDashoffset={CIRC * (1 - left)} />
      </svg>
      <div style={{ position: 'absolute', inset: 0, display: 'grid', placeItems: 'center', font: '700 12px system-ui',
        color: '#3a2a1a', fontVariantNumeric: 'tabular-nums' }}>
        {paused ? '⏸' : formatClock(ms)}
      </div>
    </div>
  )
}
