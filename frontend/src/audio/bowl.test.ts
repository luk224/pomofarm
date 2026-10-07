import { describe, expect, it } from 'vitest'
import { BASE_HZ, BOWL_PARTIALS, MASTER_GAIN, STRIKES } from './bowl'

describe('bowl chime design', () => {
  it('is inharmonic (no partial is an integer multiple), which is what makes it sound like a bowl', () => {
    for (const p of BOWL_PARTIALS.slice(1)) expect(Number.isInteger(p.ratio)).toBe(false)
  })
  it('higher partials are quieter and fade sooner', () => {
    for (let i = 1; i < BOWL_PARTIALS.length; i++) {
      expect(BOWL_PARTIALS[i].gain).toBeLessThan(BOWL_PARTIALS[i - 1].gain)
      expect(BOWL_PARTIALS[i].decay).toBeLessThan(BOWL_PARTIALS[i - 1].decay)
    }
  })
  it('cannot clip: worst-case sum of all partials and strikes stays under full scale', () => {
    const sum = BOWL_PARTIALS.reduce((a, p) => a + p.gain, 0) * Math.max(...STRIKES.map((s) => s.level))
    expect(sum * MASTER_GAIN).toBeLessThan(0.5)
  })
  it('stays in a gentle range: highest partial below 4 kHz', () => {
    expect(BASE_HZ * Math.max(...BOWL_PARTIALS.map((p) => p.ratio))).toBeLessThan(4000)
  })
  it('ends within seconds, not minutes', () => {
    const end = Math.max(...STRIKES.map((s) => s.at)) + Math.max(...BOWL_PARTIALS.map((p) => p.decay))
    expect(end).toBeLessThan(8)
  })
})
