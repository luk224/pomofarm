import { describe, expect, it } from 'vitest'
import { isAmbientKind } from './ambient'
import { EFFECT_LEVEL } from './effects'
import { LAYERS, volumeToGain } from './engine'

describe('volumeToGain', () => {
  it('is silent at 0 and full at 1', () => {
    expect(volumeToGain(0)).toBe(0)
    expect(volumeToGain(1)).toBe(1)
  })
  it('follows a squared curve, so half the slider is a quarter of the gain', () => {
    expect(volumeToGain(0.5)).toBeCloseTo(0.25)
  })
  it('only grows with the slider', () => {
    for (let v = 0; v < 1; v += 0.1) expect(volumeToGain(v + 0.1)).toBeGreaterThan(volumeToGain(v))
  })
  it.each([[-3, 0], [7, 1], [NaN, 0], [Infinity, 0]])('clamps garbage %s', (v, out) => expect(volumeToGain(v)).toBe(out))
})

describe('layers', () => {
  it('are exactly ambient, effects and alerts', () => expect([...LAYERS]).toEqual(['ambient', 'effects', 'alerts']))
})

describe('effects', () => {
  it('stay soft: no effect starts above half scale, so the layer slider decides loudness', () => {
    for (const v of Object.values(EFFECT_LEVEL)) expect(v).toBeLessThanOrEqual(0.5)
  })
})

describe('isAmbientKind', () => {
  it.each(['off', 'rain', 'forest', 'fire'])('accepts %s', (k) => expect(isAmbientKind(k)).toBe(true))
  it.each(['ocean', '', null, 3, undefined])('rejects %s', (k) => expect(isAmbientKind(k)).toBe(false))
})
