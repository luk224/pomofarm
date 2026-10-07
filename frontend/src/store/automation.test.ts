import { describe, expect, it } from 'vitest'
import type { PlotState } from '../api/types'
import { formatPrice, hiveArea, newlyCovered } from './automation'

const plot = (id: number, x: number, y: number, state: PlotState['state'] = 'empty', bees = false): PlotState => ({
  id, x, y, state, plant_type: null, matured_at: null, wilts_at: null, harvested: false,
  bonus: state === 'mature' ? { multiplier: 1, neighbours: 0, garden: false, bees } : null,
})
const grid = (state: PlotState['state'] = 'empty') => Array.from({ length: 16 }, (_, i) => plot(i + 1, i % 4, Math.floor(i / 4), state))

describe('hiveArea', () => {
  it('covers 9 plots in the middle, 6 on an edge, 4 in a corner', () => {
    expect(hiveArea(grid(), 1, 1)).toHaveLength(9)
    expect(hiveArea(grid(), 0, 1)).toHaveLength(6)
    expect(hiveArea(grid(), 0, 0)).toHaveLength(4)
  })
  it('only counts plots that exist', () => {
    expect(hiveArea([plot(1, 0, 0), plot(2, 3, 3)], 0, 0)).toHaveLength(1)
  })
})

describe('newlyCovered', () => {
  it('counts producing plants without bees yet', () => {
    expect(newlyCovered(grid('mature'), 1, 1)).toBe(9)
    expect(newlyCovered(grid('growing'), 1, 1)).toBe(0)
    const g = grid('mature'); g[5].bonus!.bees = true
    expect(newlyCovered(g, 1, 1)).toBe(8)
  })
})

describe('formatPrice', () => {
  it('uses the Spanish thousands separator', () => expect(formatPrice(13500)).toBe('13.500'))
})
