import { describe, expect, it } from 'vitest'
import type { PlotState } from '../api/types'
import { ATTENTION_COOLDOWN_MS, ATTENTION_WINDOW_MS, atRiskPlots, attentionKey, attentionMessage, shouldNotify } from './attention'

const SERVER = '2026-10-09T10:00:00.000000000Z'
const at = (mins: number) => new Date(Date.parse('2026-10-09T10:00:00Z') + mins * 60_000).toISOString().replace('.000Z', '.123456789Z')
const plot = (id: number, state: PlotState['state'], wiltsInMin: number | null): PlotState => ({
  id, x: id % 4, y: Math.floor(id / 4), state, plant_type: 'daisy', matured_at: null, wilts_at: wiltsInMin === null ? null : at(wiltsInMin), harvested: true, bonus: null,
})

describe('atRiskPlots', () => {
  it('picks the producing plants that wither within two hours', () => {
    const ps = [plot(1, 'mature', 30), plot(2, 'mature', 119), plot(3, 'mature', 121), plot(4, 'mature', 600)]
    expect(atRiskPlots(ps, SERVER, 0, 0).map((p) => p.id)).toEqual([1, 2])
  })
  it('ignores growing, empty and already-withered plants', () => {
    const ps = [plot(1, 'growing', 30), plot(2, 'empty', null), plot(3, 'withered', -10), plot(4, 'mature', null)]
    expect(atRiskPlots(ps, SERVER, 0, 0)).toEqual([])
  })
  it('keeps counting on the monotonic clock between server answers', () => {
    const ps = [plot(1, 'mature', 130)]
    expect(atRiskPlots(ps, SERVER, 1000, 1000)).toHaveLength(0)
    expect(atRiskPlots(ps, SERVER, 1000, 1000 + 15 * 60_000)).toHaveLength(1) // 15 minutes later it is within the window
  })
  it('does not count a plant whose life already ended while we were not looking', () => {
    expect(atRiskPlots([plot(1, 'mature', 5)], SERVER, 0, 10 * 60_000)).toEqual([])
  })
  it('uses the window the GDD-style rule describes: 2 hours', () => expect(ATTENTION_WINDOW_MS).toBe(7_200_000))
})

describe('shouldNotify', () => {
  const three = [plot(1, 'mature', 30), plot(2, 'mature', 40), plot(3, 'mature', 50)]
  it('needs at least three plants ("varias")', () => {
    expect(shouldNotify(three.slice(0, 2), null, 0)).toBe(false)
    expect(shouldNotify(three, null, 0)).toBe(true)
  })
  it('says nothing twice about the same group', () => {
    const memory = { key: attentionKey(three), at: 0 }
    expect(shouldNotify(three, memory, ATTENTION_COOLDOWN_MS * 10)).toBe(false)
  })
  it('waits six hours before mentioning a different group', () => {
    const other = [...three, plot(4, 'mature', 20)]
    const memory = { key: attentionKey(three), at: 1000 }
    expect(shouldNotify(other, memory, 1000 + ATTENTION_COOLDOWN_MS - 1)).toBe(false)
    expect(shouldNotify(other, memory, 1000 + ATTENTION_COOLDOWN_MS)).toBe(true)
  })
  it('keys do not depend on the order of the plants', () => expect(attentionKey([plot(3, 'mature', 1), plot(1, 'mature', 1)])).toBe(attentionKey([plot(1, 'mature', 1), plot(3, 'mature', 1)])))
})

describe('attentionMessage', () => {
  const text = attentionMessage(4)
  it('says how many and what it means', () => expect(text).toContain('4 plantas'))
  it('is calm: no exclamation, no urgency, nothing about losing anything', () => {
    expect(text).not.toMatch(/[!¡]/)
    expect(text.toLowerCase()).not.toMatch(/urgente|ya|perder|pierdes|perderás|última|cuidado|rápido|ahora mismo/)
  })
})
