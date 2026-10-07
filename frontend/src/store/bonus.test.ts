import { describe, expect, it } from 'vitest'
import { bonusLabel, formatMultiplier } from './bonus'

const b = (multiplier: number, garden = false) => ({ multiplier, neighbours: 0, garden, bees: false })

describe('bonusLabel', () => {
  it.each([[1.1, '+10%'], [1.35, '+35%'], [1.55, '+55%'], [1.2, '+20%'], [1.325, '+33%']])('×%s -> %s', (m, out) => expect(bonusLabel(b(m))).toBe(out))
  it('shows nothing without a real bonus', () => {
    expect(bonusLabel(b(1))).toBeNull()
    expect(bonusLabel(b(1.004))).toBeNull()
    expect(bonusLabel(null)).toBeNull()
    expect(bonusLabel(undefined)).toBeNull()
  })
})

describe('formatMultiplier', () => {
  it.each([[1, '×1'], [1.1, '×1,1'], [1.35, '×1,35'], [2, '×2']])('%s -> %s', (m, out) => expect(formatMultiplier(m)).toBe(out))
})
