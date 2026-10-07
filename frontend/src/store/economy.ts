import type { SiloState } from '../api/types'

const nf1 = new Intl.NumberFormat('es-ES', { maximumFractionDigits: 1 })
const nf0 = new Intl.NumberFormat('es-ES', { maximumFractionDigits: 0 })

/**
 * Thousandths of a coin -> text. One decimal below 100 🪙 (a daisy makes under 1/h), whole numbers above.
 * Rounds down, but forgives one thousandth: repeating decimals (12 h × 0.8333 = 9.999) should read "10".
 */
export function formatCoins(milli: number): string {
  const m = Math.max(0, milli) + 1
  return m < 100_000 ? nf1.format(Math.floor(m / 100) / 10) : nf0.format(Math.floor(m / 1000))
}

/**
 * What to DRAW in the Silo between server answers: the stored amount plus production since it arrived,
 * clamped to capacity. Display only; the server is the one that counts. Uses the monotonic clock.
 */
export function siloNow(s: SiloState, fetchedAtMono: number, nowMono: number): number {
  if (s.full || s.rate_milli_per_h <= 0) return s.content_milli
  const hours = Math.max(0, nowMono - fetchedAtMono) / 3_600_000
  const projected = s.content_milli + s.rate_milli_per_h * hours
  return Math.max(s.content_milli, Math.min(projected, s.capacity_milli))
}

/** 0..1 fill of the Silo bar. */
export function siloFill(content: number, capacity: number): number {
  return capacity <= 0 ? 0 : Math.min(1, Math.max(0, content / capacity))
}

/**
 * The smallest amount worth collecting: 0.1 🪙, the finest step the interface shows. Collecting a crumb of a
 * thousandth would only produce a "+0 🪙" message. The server still accepts any amount.
 */
export const MIN_COLLECT_MILLI = 100

export function canCollect(amountMilli: number): boolean {
  return amountMilli >= MIN_COLLECT_MILLI
}
