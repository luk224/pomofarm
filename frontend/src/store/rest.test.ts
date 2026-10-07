import { describe, expect, it } from 'vitest'
import type { GameState } from '../api/types'
import { parseServerTime } from './life'
import { detectRestEnd, restProgress, restRemainingMs } from './rest'

const rest = { total_s: 300, remaining_ms: 300_000, ends_at: '2026-10-07T12:05:00Z' }
const state = (r: GameState['rest'], at: string) => ({ server_time: at, rest: r } as GameState)

describe('restRemainingMs', () => {
  it('counts down on the monotonic clock and stops at zero', () => {
    expect(restRemainingMs(rest, 1000, 61_000)).toBe(240_000)
    expect(restRemainingMs(rest, 0, 9_999_999)).toBe(0)
    expect(restRemainingMs(rest, 5000, 1000)).toBe(300_000) // a clock going backwards changes nothing
  })
})

describe('restProgress', () => {
  it.each([[300_000, 0], [150_000, 0.5], [0, 1]])('%i ms left -> %s', (left, p) => expect(restProgress(rest, left)).toBeCloseTo(p))
  it('is safe with a zero-length rest', () => expect(restProgress({ ...rest, total_s: 0 }, 0)).toBe(1))
})

describe('detectRestEnd', () => {
  const before = state(rest, '2026-10-07T12:04:50Z')
  it('fires when the rest is gone and its end time has passed', () => {
    expect(detectRestEnd(before, state(null, '2026-10-07T12:05:01Z'), parseServerTime)).toBe(true)
  })
  it('does not fire when the player skipped it early', () => {
    expect(detectRestEnd(before, state(null, '2026-10-07T12:02:00Z'), parseServerTime)).toBe(false)
  })
  it('does not fire while it still runs, or when there was none', () => {
    expect(detectRestEnd(before, state(rest, '2026-10-07T12:05:01Z'), parseServerTime)).toBe(false)
    expect(detectRestEnd(state(null, '2026-10-07T12:00:00Z'), state(null, '2026-10-07T12:10:00Z'), parseServerTime)).toBe(false)
    expect(detectRestEnd(null, state(null, '2026-10-07T12:10:00Z'), parseServerTime)).toBe(false)
  })
})
