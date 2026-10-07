import type { GameState } from '../api/types'

/** Milliseconds of rest left, counted on the monotonic clock from the moment the server answered. */
export function restRemainingMs(rest: NonNullable<GameState['rest']>, fetchedAtMono: number, nowMono: number): number {
  return Math.max(0, rest.remaining_ms - Math.max(0, nowMono - fetchedAtMono))
}

/** Fraction of the rest already elapsed, 0..1. */
export function restProgress(rest: NonNullable<GameState['rest']>, remainingMs: number): number {
  if (rest.total_s <= 0) return 1
  return Math.min(1, Math.max(0, 1 - remainingMs / (rest.total_s * 1000)))
}

/**
 * The rest finished BY ITSELF (not skipped, not ended by planting a new Pomodoro): it was running before, it is gone
 * now, and the server's clock has reached its end. Used to announce "Descanso terminado".
 */
export function detectRestEnd(prev: GameState | null, next: GameState, parse: (s: string) => number): boolean {
  if (!prev?.rest || next.rest) return false
  return parse(next.server_time) >= parse(prev.rest.ends_at) - 1000
}
