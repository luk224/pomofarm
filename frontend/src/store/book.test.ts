import { describe, expect, it } from 'vitest'
import { dayLabel, formatFocus, harvestOf, isHarvestUnit, monthLabel } from './book'

describe('formatFocus', () => {
  it.each([[0, '0 min'], [60, '1 min'], [2700, '45 min'], [3600, '1 h 00 min'], [7500, '2 h 05 min'], [86400, '24 h 00 min'], [-5, '0 min']])('%s s -> %s', (s, out) => expect(formatFocus(s)).toBe(out))
  it('rounds to the nearest minute', () => {
    expect(formatFocus(89)).toBe('1 min')
    expect(formatFocus(91)).toBe('2 min')
  })
})

describe('harvestOf', () => {
  it('draws one icon per hour with the remainder as a partial icon', () => {
    expect(harvestOf(0)).toEqual({ full: 0, part: 0, hoursPerIcon: 1 })
    expect(harvestOf(3600)).toEqual({ full: 1, part: 0, hoursPerIcon: 1 })
    expect(harvestOf(3600 * 2.5)).toEqual({ full: 2, part: 0.5, hoursPerIcon: 1 })
  })
  it('ignores crumbs under 3 minutes of an hour', () => expect(harvestOf(3600 + 120).part).toBe(0))
  it('never needs more than 40 icons: groups hours when the month is long', () => {
    for (const hours of [40, 41, 100, 200, 250, 1000, 5000, 100000]) {
      const h = harvestOf(hours * 3600)
      expect(h.full + (h.part > 0 ? 1 : 0)).toBeLessThanOrEqual(40)
      expect(h.full * h.hoursPerIcon).toBeLessThanOrEqual(hours)
      expect(h.full * h.hoursPerIcon + h.part * h.hoursPerIcon).toBeCloseTo(hours, 1)
    }
  })
  it('uses ×1 up to 40 h and ×5 beyond', () => {
    expect(harvestOf(40 * 3600).hoursPerIcon).toBe(1)
    expect(harvestOf(41 * 3600).hoursPerIcon).toBe(5)
  })
})

describe('labels', () => {
  it('names the month in Spanish', () => expect(monthLabel('2026-10')).toBe('octubre de 2026'))
  it('names a day with its weekday', () => expect(dayLabel('2026-10-07')).toMatch(/mi/i))
  it('validates the saved unit', () => {
    expect(isHarvestUnit('silo')).toBe(true)
    expect(isHarvestUnit('tractor')).toBe(false)
    expect(isHarvestUnit(null)).toBe(false)
  })
})
