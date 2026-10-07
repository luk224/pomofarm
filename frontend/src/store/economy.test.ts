import { describe, expect, it } from 'vitest'
import type { SiloState } from '../api/types'
import { canCollect, formatCoins, MIN_COLLECT_MILLI, siloFill, siloNow } from './economy'

const silo = (over: Partial<SiloState> = {}): SiloState => ({
  content_milli: 2000, capacity_milli: 10_000, capacity_hours: 12, rate_milli_per_h: 3_600_000, full: false, ...over,
})

describe('formatCoins', () => {
  it.each([
    [0, '0'], [4166, '4,1'], [833, '0,8'], [9_999, '10'], [99_998, '99,9'], [99_999, '100'], [100_000, '100'], [12_345_678, '12.345'], [-5, '0'],
  ])('%i milli -> %s', (m, out) => expect(formatCoins(m)).toBe(out))
  it('rounds down, forgiving a single thousandth', () => {
    expect(formatCoins(4198)).toBe('4,1')
    expect(formatCoins(4199)).toBe('4,2')
  })
})

describe('siloNow (display between server answers)', () => {
  it('grows with the monotonic clock at the farm rate', () => {
    // 3 600 000 milli/h = 1000 milli/s
    expect(siloNow(silo(), 0, 1000)).toBe(3000)
  })
  it('stops at capacity', () => expect(siloNow(silo(), 0, 60_000)).toBe(10_000))
  it('does not move when the Silo is full or nothing is produced', () => {
    expect(siloNow(silo({ full: true, content_milli: 10_000 }), 0, 99_999)).toBe(10_000)
    expect(siloNow(silo({ rate_milli_per_h: 0 }), 0, 99_999)).toBe(2000)
  })
  it('never shows less than the server said, even if capacity is below the content', () => {
    expect(siloNow(silo({ capacity_milli: 500 }), 0, 5000)).toBe(2000)
  })
  it('ignores a monotonic clock going backwards', () => expect(siloNow(silo(), 5000, 1000)).toBe(2000))
})

describe('siloFill', () => {
  it('is clamped to 0..1', () => {
    expect(siloFill(5, 10)).toBe(0.5)
    expect(siloFill(50, 10)).toBe(1)
    expect(siloFill(5, 0)).toBe(0)
  })
})

describe('canCollect', () => {
  it('needs at least 0.1 🪙, the finest amount the interface shows', () => {
    expect(MIN_COLLECT_MILLI).toBe(100)
    expect(canCollect(99)).toBe(false)
    expect(canCollect(100)).toBe(true)
    expect(canCollect(0)).toBe(false)
  })
  it('never offers a collection that would be announced as "+0"', () => {
    for (let milli = 0; milli < 2000; milli++) if (canCollect(milli)) expect(formatCoins(milli)).not.toBe('0')
  })
})
