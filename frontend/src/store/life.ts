import type { PlotState } from '../api/types'

/**
 * Parses the server's RFC 3339 timestamps. The server sends nanoseconds ("…633175758Z"); some browsers only
 * accept milliseconds, so the fraction is trimmed to three digits first.
 */
export function parseServerTime(s: string): number {
  return Date.parse(s.replace(/(\.\d{3})\d+/, '$1'))
}

/**
 * Milliseconds of life a plant has left. Both ends come from the SERVER (wilts_at and server_time), and the time
 * since the answer arrived is measured on the monotonic clock, so the player's system clock never matters.
 */
export function lifeLeftMs(plot: PlotState, serverTime: string, fetchedAtMono: number, nowMono: number): number | null {
  if (!plot.wilts_at) return null
  const left = parseServerTime(plot.wilts_at) - parseServerTime(serverTime) - Math.max(0, nowMono - fetchedAtMono)
  return Math.max(0, left)
}

/** "35 min", "5 h 12 min", "3 h", "4 días". */
export function formatLife(ms: number): string {
  const min = Math.max(0, Math.ceil(ms / 60_000))
  if (min < 60) return `${Math.max(1, min)} min`
  const h = Math.floor(min / 60)
  if (h < 48) return min % 60 === 0 ? `${h} h` : `${h} h ${min % 60} min`
  return `${Math.round(h / 24)} días`
}
