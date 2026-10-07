import { describe, expect, it } from 'vitest'
import type { PlotState } from '../api/types'
import { formatLife, lifeLeftMs, parseServerTime } from './life'

const plot = (wilts: string | null): PlotState => ({
  id: 1, x: 1, y: 1, state: 'mature', plant_type: 'daisy', matured_at: null, wilts_at: wilts, harvested: true, bonus: null,
})

describe('parseServerTime', () => {
  it('accepts nanosecond timestamps like the server sends', () => {
    expect(parseServerTime('2026-10-07T13:27:28.633175758Z')).toBe(Date.parse('2026-10-07T13:27:28.633Z'))
  })
  it('accepts timestamps without fraction or with fewer digits', () => {
    expect(parseServerTime('2026-10-07T13:27:28Z')).toBe(Date.parse('2026-10-07T13:27:28Z'))
    expect(parseServerTime('2026-10-07T13:27:28.5Z')).toBe(Date.parse('2026-10-07T13:27:28.5Z'))
  })
})

describe('lifeLeftMs', () => {
  const now = '2026-10-07T12:00:00.000000000Z'
  it('is wilts_at minus the SERVER time, minus the monotonic time since the answer', () => {
    expect(lifeLeftMs(plot('2026-10-07T14:00:00Z'), now, 1000, 1000)).toBe(2 * 3_600_000)
    expect(lifeLeftMs(plot('2026-10-07T14:00:00Z'), now, 1000, 61_000)).toBe(2 * 3_600_000 - 60_000)
  })
  it('never goes negative', () => expect(lifeLeftMs(plot('2026-10-07T11:00:00Z'), now, 0, 0)).toBe(0))
  it('is null when the plant has no lifespan yet', () => expect(lifeLeftMs(plot(null), now, 0, 0)).toBeNull())
  it('does not depend on the browser clock', () => {
    const real = Date.now
    Date.now = () => real() + 10 * 24 * 3_600_000
    try {
      expect(lifeLeftMs(plot('2026-10-07T14:00:00Z'), now, 0, 0)).toBe(2 * 3_600_000)
    } finally {
      Date.now = real
    }
  })
})

describe('formatLife', () => {
  it.each([
    [0, '1 min'], [30_000, '1 min'], [35 * 60_000, '35 min'], [59 * 60_000, '59 min'], [60 * 60_000, '1 h'],
    [5 * 3_600_000 + 12 * 60_000, '5 h 12 min'], [47 * 3_600_000, '47 h'], [72 * 3_600_000, '3 días'],
  ])('%i ms -> %s', (ms, out) => expect(formatLife(ms)).toBe(out))
})
