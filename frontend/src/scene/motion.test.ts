import { describe, expect, it } from 'vitest'
import { popScale } from './motion'

describe('popScale (squash & stretch when a plant matures)', () => {
  it('is neutral before and after the animation', () => {
    expect(popScale(-1)).toEqual({ y: 1, xz: 1 })
    expect(popScale(0.9)).toEqual({ y: 1, xz: 1 })
  })
  it('starts by stretching tall and thin', () => {
    const p = popScale(0)
    expect(p.y).toBeGreaterThan(1)
    expect(p.xz).toBeLessThan(1)
  })
  it('conserves roughly the same volume (y·xz²) while moving', () => {
    for (const t of [0, 0.1, 0.2, 0.3]) {
      const p = popScale(t)
      expect(p.y * p.xz * p.xz).toBeGreaterThan(0.8)
      expect(p.y * p.xz * p.xz).toBeLessThan(1.2)
    }
  })
  it('settles: the swing shrinks over time', () => {
    expect(Math.abs(popScale(0.7).y - 1)).toBeLessThan(Math.abs(popScale(0.1).y - 1))
  })
})
