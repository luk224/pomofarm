import type { PlotState } from '../api/types'
import { lifeLeftMs } from './life'

/**
 * "Tu granja necesita atención" (GDD §4): a soft notice when several plants are about to wither. It is never urgent and never
 * about loss (a withered plant costs nothing; it only stops producing), so it is rare and quiet: at least 3 plants, within the
 * next 2 hours, once per group of plants, and no more than one notice every 6 hours.
 */
export const ATTENTION_WINDOW_MS = 2 * 60 * 60 * 1000
export const ATTENTION_MIN_PLANTS = 3
export const ATTENTION_COOLDOWN_MS = 6 * 60 * 60 * 1000

/** What was last said: which plants, and when (wall clock; only to space the notices out, never for game rules). */
export interface AttentionMemory {
  key: string
  at: number
}

/** The producing plants whose life ends within the window. Time comes from the server, kept live on the monotonic clock. */
export function atRiskPlots(plots: PlotState[], serverTime: string, fetchedAtMono: number, nowMono: number): PlotState[] {
  return plots.filter((p) => {
    if (p.state !== 'mature') return false // growing plants are not at risk yet, withered ones already are
    const left = lifeLeftMs(p, serverTime, fetchedAtMono, nowMono)
    return left !== null && left > 0 && left <= ATTENTION_WINDOW_MS
  })
}

/** Identifies a group of plants, so the same group is only mentioned once. */
export function attentionKey(plots: PlotState[]): string {
  return plots.map((p) => p.id).sort((a, b) => a - b).join(',')
}

/** True when it is worth saying something now. */
export function shouldNotify(risk: PlotState[], memory: AttentionMemory | null, nowWall: number): boolean {
  if (risk.length < ATTENTION_MIN_PLANTS) return false
  if (!memory) return true
  if (memory.key === attentionKey(risk)) return false // already told about exactly this
  return nowWall - memory.at >= ATTENTION_COOLDOWN_MS
}

/** Calm wording: what is happening, and that nothing is lost. */
export function attentionMessage(count: number): string {
  return `Tu granja pide un poco de atención: ${count} plantas dejarán de producir pronto. Cuando quieras, retíralas y vuelve a sembrar.`
}

const KEY = 'pomofarm.attention'

export function loadMemory(): AttentionMemory | null {
  try {
    const raw = JSON.parse(localStorage.getItem(KEY) ?? 'null')
    return raw && typeof raw.key === 'string' && typeof raw.at === 'number' && Number.isFinite(raw.at) ? { key: raw.key, at: raw.at } : null
  } catch {
    return null
  }
}

export function saveMemory(m: AttentionMemory): void {
  try {
    localStorage.setItem(KEY, JSON.stringify(m))
  } catch {
    /* private mode: it will simply be said again after a reload */
  }
}
