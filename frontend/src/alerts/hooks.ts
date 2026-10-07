import { useEffect, useRef } from 'react'
import { useGame } from '../store/game'
import { parseServerTime } from '../store/life'
import { restRemainingMs } from '../store/rest'
import { remainingMs } from '../store/time'
import { announceCompletion, announceRestEnd } from './announce'
import { detectCompletion } from './completion'
import { detectRestEnd } from '../store/rest'

/** Slack after the planned end so the server's own clock has certainly passed it. */
const MARGIN_MS = 400

/**
 * Wires the end-of-Pomodoro and end-of-rest alerts:
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
        if (detectRestEnd(prev.state, s.state, parseServerTime)) announceRestEnd()
      }
      // (Re)schedule the wake-up for whichever comes first: the end of the Pomodoro or the end of the rest.
      if (s.state !== prev.state || s.fetchedAt !== prev.fetchedAt) {
        const p = s.state?.pomodoro
        const r = s.state?.rest
        const waits: number[] = []
        if (p && p.status === 'running') waits.push(remainingMs(p, s.fetchedAt, performance.now()))
        if (r) waits.push(restRemainingMs(r, s.fetchedAt, performance.now()))
        if (waits.length) w.postMessage({ type: 'set', ms: Math.min(...waits) + MARGIN_MS })
        else w.postMessage({ type: 'clear' })
      }
    })
    return () => {
      unsubscribe()
      w.terminate()
      worker.current = null
    }
  }, [])
}
