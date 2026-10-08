import { describe, expect, it } from 'vitest'
import { BIOMES, biomePalette, isBiome } from './biomes'

const lum = (hex: string) => {
  const c = [1, 3, 5].map((i) => parseInt(hex.slice(i, i + 2), 16) / 255).map((v) => (v <= 0.03928 ? v / 12.92 : ((v + 0.055) / 1.055) ** 2.4))
  return 0.2126 * c[0] + 0.7152 * c[1] + 0.0722 * c[2]
}
const contrast = (a: string, b: string) => (Math.max(lum(a), lum(b)) + 0.05) / (Math.min(lum(a), lum(b)) + 0.05)

describe('biomes', () => {
  it('has the four seasons, each a valid pair of colours', () => {
    expect(Object.keys(BIOMES)).toEqual(['spring', 'summer', 'autumn', 'winter'])
    for (const b of Object.values(BIOMES)) for (const c of [b.sky, b.grassA, b.grassB]) expect(c).toMatch(/^#[0-9a-f]{6}$/i)
  })
  it('falls back to spring for anything unknown', () => {
    for (const v of ['', 'desert', null, undefined, 3, 'SPRING']) expect(biomePalette(v)).toBe(BIOMES.spring)
    expect(isBiome('winter')).toBe(true)
    expect(isBiome('toString')).toBe(false)
  })
  it('keeps the checkerboard readable: the two greens are close but distinct', () => {
    for (const b of Object.values(BIOMES)) {
      expect(b.grassA).not.toBe(b.grassB)
      expect(contrast(b.grassA, b.grassB)).toBeLessThan(1.3)
    }
  })
  it('keeps the soil of a plot (#8a5a3b) visibly different from the ground in every biome', () => {
    for (const b of Object.values(BIOMES)) expect(contrast('#8a5a3b', b.grassA)).toBeGreaterThan(1.8)
  })
})
