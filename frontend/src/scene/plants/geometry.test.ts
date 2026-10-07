import { describe, expect, it } from 'vitest'
import { Color } from 'three'
import { palette as P } from '../palette'
import { orbsGeometry, plantGeometry, triangleCount } from './geometry'
import type { PlantKind } from './kinds'

const KINDS: PlantKind[] = ['daisy', 'tomato', 'sunflower', 'apple', 'oak']

describe('merged plant geometry', () => {
  it('keeps every plant, in every stage, under the 2000-triangle budget (GDD §2)', () => {
    for (const k of KINDS) for (const mature of [false, true]) for (const withered of [false, true]) {
      expect(triangleCount(k, mature, withered), `${k} mature=${mature} withered=${withered}`).toBeLessThan(2000)
    }
  })
  it('is one geometry per plant: position, normal and colour in the same buffers', () => {
    const g = plantGeometry('apple', true, false)
    expect(Object.keys(g.attributes).sort()).toEqual(['color', 'normal', 'position'])
    expect(g.index).toBeNull() // flat shading needs unshared vertices
  })
  it('is shared: the same kind and stage returns the very same object', () => {
    expect(plantGeometry('daisy', true, false)).toBe(plantGeometry('daisy', true, false))
    expect(plantGeometry('daisy', true, false)).not.toBe(plantGeometry('daisy', false, false))
  })
  it('shows flowers, fruit and blooms only when mature', () => {
    for (const k of ['daisy', 'tomato', 'sunflower', 'apple'] as PlantKind[]) {
      expect(triangleCount(k, true, false), k).toBeGreaterThan(triangleCount(k, false, false))
    }
    // the Oak has no fruit: its orbs are a separate object
    expect(triangleCount('oak', true, false)).toBe(triangleCount('oak', false, false))
  })
  it('paints a withered plant in the withered tint only', () => {
    const colors = plantGeometry('sunflower', true, true).getAttribute('color')
    const w = new Color(P.withered)
    for (let i = 0; i < colors.count; i += 7) {
      expect(colors.getX(i)).toBeCloseTo(w.r, 5)
      expect(colors.getY(i)).toBeCloseTo(w.g, 5)
      expect(colors.getZ(i)).toBeCloseTo(w.b, 5)
    }
  })
  it('a healthy plant uses several colours (stem, leaves, petals…)', () => {
    const c = plantGeometry('daisy', true, false).getAttribute('color')
    const seen = new Set<string>()
    for (let i = 0; i < c.count; i++) seen.add(`${c.getX(i).toFixed(3)},${c.getY(i).toFixed(3)},${c.getZ(i).toFixed(3)}`)
    expect(seen.size).toBeGreaterThanOrEqual(4)
  })
  it('stands on the soil and does not grow absurdly tall or wide', () => {
    for (const k of KINDS) {
      const g = plantGeometry(k, true, false)
      g.computeBoundingBox()
      const b = g.boundingBox!
      expect(b.min.y, k).toBeGreaterThanOrEqual(-0.01)
      expect(b.max.y, k).toBeLessThan(1.7)
      expect(b.max.x - b.min.x, k).toBeLessThan(1.05) // fits inside a plot (1 unit), canopies may just touch
    }
  })
  it('the orbs are a single small geometry', () => {
    const g = orbsGeometry()
    expect(g.getAttribute('position').count / 3).toBeLessThan(40)
    expect(orbsGeometry()).toBe(g)
  })
})
