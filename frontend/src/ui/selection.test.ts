import { describe, expect, it } from 'vitest'
import type { PlotState } from '../api/types'
import { canClear, effectivePlot, needsHarvest } from './selection'

const plot = (id: number, state: PlotState['state'], harvested = false): PlotState => ({
  id, x: id, y: 0, state, plant_type: state === 'empty' ? null : 'daisy', matured_at: null, wilts_at: null, harvested,
})

describe('effectivePlot', () => {
  const plots = [plot(1, 'mature', true), plot(2, 'empty'), plot(3, 'mature', false), plot(4, 'empty')]
  it('respects the player\'s choice, whatever it is', () => {
    expect(effectivePlot(plots, 1)?.id).toBe(1)
    expect(effectivePlot(plots, 4)?.id).toBe(4)
  })
  it('without a choice, points at a plant waiting to be harvested first', () => {
    expect(effectivePlot(plots, null)?.id).toBe(3)
  })
  it('then at a free plot', () => {
    expect(effectivePlot([plot(1, 'mature', true), plot(2, 'empty')], null)?.id).toBe(2)
  })
  it('then at the first plot', () => {
    expect(effectivePlot([plot(1, 'mature', true), plot(2, 'withered', true)], null)?.id).toBe(1)
  })
  it('ignores a stale choice (a plot that no longer exists)', () => {
    expect(effectivePlot(plots, 99)?.id).toBe(3)
  })
  it('is undefined with no plots', () => expect(effectivePlot([], null)).toBeUndefined())
})

describe('plot predicates', () => {
  it('withered unharvested plants still need harvesting', () => {
    expect(needsHarvest(plot(1, 'withered'))).toBe(true)
    expect(needsHarvest(plot(1, 'withered', true))).toBe(false)
    expect(needsHarvest(plot(1, 'growing'))).toBe(false)
  })
  it('only harvested plants can be cleared', () => {
    expect(canClear(plot(1, 'mature', true))).toBe(true)
    expect(canClear(plot(1, 'withered', true))).toBe(true)
    expect(canClear(plot(1, 'mature', false))).toBe(false)
    expect(canClear(plot(1, 'empty'))).toBe(false)
  })
})
