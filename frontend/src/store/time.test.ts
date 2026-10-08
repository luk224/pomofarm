import { describe, expect, it } from 'vitest'
import type { PomodoroState } from '../api/types'
import { formatClock, growthFraction, pausedMs, remainingMs, strictProgress, tabTitle } from './time'

const base: PomodoroState = {
  id: 1, plot_id: 1, plant_type: 'daisy', status: 'running', planned_s: 600,
  remaining_ms: 600_000, started_at: 't', paused_at: null, tag: null, strict: false, pauses: 0, paused_ms: 0,
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
  it('shows the rest countdown when there is no Pomodoro', () => {
    expect(tabTitle(null, 0, false, 272_000)).toBe('☕ 4:32 · PomoFarm')
    expect(tabTitle(null, 0, true, 272_000)).toBe('☕ 4:32 · PomoFarm')
    expect(tabTitle(null, 0, true, 0)).toBe('✔ Lista para cosechar · PomoFarm')
    expect(tabTitle(base, 125_000, false, 272_000)).toBe('⏱ 2:05 · PomoFarm')
  })
  it('says when a plant is ready, as a visual alert for muted players', () => expect(tabTitle(null, 0, true)).toBe('✔ Lista para cosechar · PomoFarm'))
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

describe('strict mode progress', () => {
  const strictBase = { id: 1, plot_id: 1, plant_type: 'daisy', status: 'running' as const, planned_s: 600, remaining_ms: 1000, started_at: 't', paused_at: null, tag: null, strict: true, pauses: 1, paused_ms: 125_000 }
  it('keeps the pause time frozen while running', () => expect(pausedMs(strictBase, 1000, 99_000)).toBe(125_000))
  it('keeps counting while paused, on the monotonic clock', () => expect(pausedMs({ ...strictBase, status: 'paused' }, 1000, 4000)).toBe(128_000))
  it('never goes backwards if the monotonic clock is odd', () => expect(pausedMs({ ...strictBase, status: 'paused' }, 5000, 1000)).toBe(125_000))
  it('reads pauses and whole minutes', () => expect(strictProgress(strictBase, 125_000)).toEqual({ text: 'Modo estricto: 1 de 2 pausas · 2 de 10 min', clean: true }))
  it('the limits are inclusive: 2 pauses and exactly 10 min are still clean', () => expect(strictProgress({ ...strictBase, pauses: 2 }, 600_000).clean).toBe(true))
  it('a third pause or 10 min and a bit more ends the clean run, kindly', () => {
    expect(strictProgress({ ...strictBase, pauses: 3 }, 0).clean).toBe(false)
    expect(strictProgress(strictBase, 600_001).clean).toBe(false)
    expect(strictProgress({ ...strictBase, pauses: 3 }, 0).text).toContain('ya no contará como limpio')
  })
})
