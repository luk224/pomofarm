import { describe, expect, it } from 'vitest'
import type { PlotState } from '../api/types'
import { canClear, describePlot, effectivePlot, needsHarvest, stepPlot } from './selection'

const plot = (id: number, state: PlotState['state'], harvested = false): PlotState => ({
  id, x: id, y: 0, state, plant_type: state === 'empty' ? null : 'daisy', matured_at: null, wilts_at: null, harvested, bonus: null,
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

describe('stepPlot (keyboard navigation)', () => {
  const plots = [plot(3, 'empty'), plot(1, 'mature', true), plot(2, 'empty')]
  it('moves through the plots in purchase order and wraps around', () => {
    expect(stepPlot(plots, 1, 1)).toBe(2)
    expect(stepPlot(plots, 2, 1)).toBe(3)
    expect(stepPlot(plots, 3, 1)).toBe(1)
    expect(stepPlot(plots, 1, -1)).toBe(3)
    expect(stepPlot(plots, 3, -2)).toBe(1)
  })
  it('starts from the plot the game is pointing at when nothing is chosen', () => {
    // nothing chosen: the game points at the first free plot in the order the server lists them (plot 3 here),
    // so one step forward wraps around to plot 1
    expect(stepPlot(plots, null, 1)).toBe(1)
  })
  it('is null with no plots and stays put with one', () => {
    expect(stepPlot([], null, 1)).toBeNull()
    expect(stepPlot([plot(5, 'empty')], 5, 1)).toBe(5)
  })
})

describe('describePlot (screen reader text)', () => {
  const all = [plot(1, 'empty'), plot(2, 'growing'), plot(3, 'mature', false), plot(4, 'mature', true), plot(5, 'withered', false), plot(6, 'withered', true)]
  it.each([
    [1, 'Parcela 1 de 6: libre'], [2, 'Parcela 2 de 6: creciendo'], [3, 'Parcela 3 de 6: lista para cosechar'],
    [4, 'Parcela 4 de 6: cosechada y produciendo'], [5, 'Parcela 5 de 6: marchita, aún se puede cosechar'], [6, 'Parcela 6 de 6: marchita'],
  ])('plot %i', (id, text) => expect(describePlot(all, all.find((p) => p.id === id))).toBe(text))
  it('mentions a synergy bonus', () => {
    const b = { ...plot(4, 'mature', true), bonus: { multiplier: 1.35, neighbours: 2, garden: true } }
    expect(describePlot([b], b)).toBe('Parcela 1 de 1: cosechada y produciendo, bono de 35 por ciento')
  })
  it('is empty without a plot', () => expect(describePlot([], undefined)).toBe(''))
})
