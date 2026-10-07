import { useEffect, useState } from 'react'
import { useGame } from './game'
import { restRemainingMs } from './rest'
import { remainingMs } from './time'

const SYNC_EVERY_MS = 30_000

/** Keeps the store in sync: on load, when the tab becomes visible, and every 30 s (other devices). */
export function useGameSync() {
  const refresh = useGame((s) => s.refresh)
  useEffect(() => {
    void refresh()
    const onVisible = () => {
      if (document.visibilityState === 'visible') void refresh()
    }
    document.addEventListener('visibilitychange', onVisible)
    window.addEventListener('focus', onVisible)
    const id = window.setInterval(onVisible, SYNC_EVERY_MS)
    return () => {
      document.removeEventListener('visibilitychange', onVisible)
      window.removeEventListener('focus', onVisible)
      window.clearInterval(id)
    }
  }, [refresh])
}

/**
 * Milliseconds left on the active Pomodoro (null if none). setInterval only repaints;
 * the value is derived from the server state. When it reaches zero it asks the server
 * to settle, so the plant becomes mature without waiting for the next poll.
 */
export function useRemainingMs(): number | null {
  const pomodoro = useGame((s) => s.state?.pomodoro ?? null)
  const fetchedAt = useGame((s) => s.fetchedAt)
  const refresh = useGame((s) => s.refresh)
  const [now, setNow] = useState(() => performance.now())

  useEffect(() => {
    if (!pomodoro || pomodoro.status !== 'running') return
    const id = window.setInterval(() => setNow(performance.now()), 250)
    return () => window.clearInterval(id)
  }, [pomodoro])

  const ms = pomodoro ? remainingMs(pomodoro, fetchedAt, Math.max(now, fetchedAt)) : null

  useEffect(() => {
    if (pomodoro?.status === 'running' && ms === 0) void refresh()
  }, [pomodoro?.status, ms, refresh])

  return ms
}

/** Milliseconds of rest left (null if none). Repaints every second; asks the server to settle when it reaches zero. */
export function useRestRemainingMs(): number | null {
  const rest = useGame((s) => s.state?.rest ?? null)
  const fetchedAt = useGame((s) => s.fetchedAt)
  const refresh = useGame((s) => s.refresh)
  const [now, setNow] = useState(() => performance.now())

  useEffect(() => {
    if (!rest) return
    const id = window.setInterval(() => setNow(performance.now()), 1000)
    return () => window.clearInterval(id)
  }, [rest])

  const ms = rest ? restRemainingMs(rest, fetchedAt, Math.max(now, fetchedAt)) : null
  useEffect(() => {
    if (ms === 0) void refresh()
  }, [ms, refresh])
  return ms
}
