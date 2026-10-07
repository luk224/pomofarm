import { useEffect, useRef } from 'react'
import { useGame } from '../store/game'
import { remainingMs } from '../store/time'
import { announceCompletion } from './announce'
import { detectCompletion } from './completion'

/** Slack after the planned end so the server's own clock has certainly passed it. */
const MARGIN_MS = 400

/**
 * Wires the end-of-Pomodoro alert:
 *  1. a Worker timer (not throttled in hidden tabs) asks the server to settle right when time is up;
 *  2. any state change that turns an active Pomodoro into a mature plant triggers the announcement.
 */
export function useCompletionAlerts() {
  const worker = useRef<Worker | null>(null)

  useEffect(() => {
    const w = new Worker(new URL('./timerWorker.ts', import.meta.url), { type: 'module' })
    worker.current = w
    w.onmessage = () => void useGame.getState().refresh()

    const unsubscribe = useGame.subscribe((s, prev) => {
      if (s.state && prev.state && s.state !== prev.state) {
        const done = detectCompletion(prev.state, s.state)
        if (done) announceCompletion(done)
      }
      // (Re)schedule the wake-up whenever the server state arrives.
      if (s.state !== prev.state || s.fetchedAt !== prev.fetchedAt) {
        const p = s.state?.pomodoro
        if (p && p.status === 'running') {
          w.postMessage({ type: 'set', ms: remainingMs(p, s.fetchedAt, performance.now()) + MARGIN_MS })
        } else {
          w.postMessage({ type: 'clear' })
        }
      }
    })
    return () => {
      unsubscribe()
      w.terminate()
      worker.current = null
    }
  }, [])
}
