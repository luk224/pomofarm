import type { PomodoroState } from '../api/types'

/**
 * Remaining time of the Pomodoro, in ms.
 *
 * The server sends `remaining_ms` as of the moment it answered. Between answers we
 * subtract the time elapsed on a MONOTONIC clock (performance.now), so changing the
 * computer's date or time can never alter the result (GDD §6.1, §8). A paused
 * Pomodoro is frozen.
 */
export function remainingMs(p: PomodoroState, fetchedAtMono: number, nowMono: number): number {
  if (p.status !== 'running') return p.remaining_ms
  return Math.max(0, p.remaining_ms - Math.max(0, nowMono - fetchedAtMono))
}

/** mm:ss (h:mm:ss from one hour). Rounds up so it only shows 0:00 when time is really up. */
export function formatClock(ms: number): string {
  const total = Math.max(0, Math.ceil(ms / 1000))
  const h = Math.floor(total / 3600)
  const m = Math.floor((total % 3600) / 60)
  const s = total % 60
  const ss = String(s).padStart(2, '0')
  return h > 0 ? `${h}:${String(m).padStart(2, '0')}:${ss}` : `${m}:${ss}`
}

/** Tab title: the time left while a Pomodoro is active (GDD §2). */
export function tabTitle(p: PomodoroState | null, ms: number, ready = false, restMs: number | null = null): string {
  if (!p) {
    if (restMs !== null && restMs > 0) return `☕ ${formatClock(restMs)} · PomoFarm`
    return ready ? '✔ Lista para cosechar · PomoFarm' : 'PomoFarm'
  }
  const icon = p.status === 'paused' ? '⏸' : '⏱'
  return `${icon} ${formatClock(ms)} · PomoFarm`
}

/** How much of the Pomodoro has elapsed, 0..1 (drives the plant's growth). */
export function growthFraction(plannedS: number, remaining: number): number {
  if (plannedS <= 0) return 1
  return Math.min(1, Math.max(0, 1 - remaining / (plannedS * 1000)))
}
