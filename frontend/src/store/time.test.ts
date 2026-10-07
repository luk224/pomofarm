import { describe, expect, it } from 'vitest'
import type { PomodoroState } from '../api/types'
import { formatClock, growthFraction, remainingMs, tabTitle } from './time'

const base: PomodoroState = {
  id: 1, plot_id: 1, plant_type: 'daisy', status: 'running', planned_s: 600,
  remaining_ms: 600_000, started_at: 't', paused_at: null,
}

describe('remainingMs', () => {
  it('counts down from the server value using the monotonic clock', () => {
    expect(remainingMs(base, 1000, 1000)).toBe(600_000)
    expect(remainingMs(base, 1000, 61_000)).toBe(540_000)
  })
  it('never goes below zero', () => {
    expect(remainingMs(base, 0, 10_000_000)).toBe(0)
  })
  it('ignores a monotonic clock that appears to go backwards', () => {
    expect(remainingMs(base, 5000, 1000)).toBe(600_000)
  })
  it('is frozen while paused, however much time passes', () => {
    const paused = { ...base, status: 'paused' as const, remaining_ms: 480_000 }
    expect(remainingMs(paused, 0, 99_999_999)).toBe(480_000)
  })
  it('does not depend on the wall clock', () => {
    const real = Date.now
    Date.now = () => real() + 3_600_000 // user moves the system clock one hour ahead
    try {
      expect(remainingMs(base, 0, 60_000)).toBe(540_000)
    } finally {
      Date.now = real
    }
  })
})

describe('formatClock', () => {
  it.each([
    [600_000, '10:00'],
    [599_001, '10:00'], // rounds up: 0:00 only when really finished
    [59_000, '0:59'],
    [1, '0:01'],
    [0, '0:00'],
    [-5, '0:00'],
    [3_600_000, '1:00:00'],
    [5_430_000, '1:30:30'],
  ])('%i ms -> %s', (ms, out) => expect(formatClock(ms)).toBe(out))
})

describe('tabTitle', () => {
  it('shows the time left while running or paused', () => {
    expect(tabTitle(base, 125_000)).toBe('⏱ 2:05 · PomoFarm')
    expect(tabTitle({ ...base, status: 'paused' }, 125_000)).toBe('⏸ 2:05 · PomoFarm')
  })
  it('is plain when idle', () => expect(tabTitle(null, 0)).toBe('PomoFarm'))
})

describe('growthFraction', () => {
  it('goes from 0 at the start to 1 at the end', () => {
    expect(growthFraction(600, 600_000)).toBe(0)
    expect(growthFraction(600, 300_000)).toBeCloseTo(0.5)
    expect(growthFraction(600, 0)).toBe(1)
  })
  it('is clamped and safe', () => {
    expect(growthFraction(600, 700_000)).toBe(0)
    expect(growthFraction(600, -5)).toBe(1)
    expect(growthFraction(0, 0)).toBe(1)
  })
})
