import { beforeEach, describe, expect, it } from 'vitest'
import { DEFAULT_STATIONS } from '../audio/stations'
import { parseMusic, stationsOf, useMusic } from './music'

const A = 'abcdefghijk'
const B = 'ABCDEFGHIJK'

describe('parseMusic', () => {
  it('has sensible defaults', () => {
    for (const raw of [{}, null, undefined, 'x', 7]) {
      expect(parseMusic(raw)).toEqual({ custom: [], selected: DEFAULT_STATIONS[0].id, volume: 0.5 })
    }
  })
  it('keeps valid custom links and drops junk, duplicates and built-ins', () => {
    const m = parseMusic({ custom: [{ id: A, name: 'Mía' }, { id: A, name: 'Otra vez' }, { id: 'nope' }, { id: DEFAULT_STATIONS[0].id }, 5, null] })
    expect(m.custom).toEqual([{ id: A, name: 'Mía', custom: true }])
  })
  it('limits the saved list to 8 custom links', () => {
    const many = Array.from({ length: 12 }, (_, i) => ({ id: `abcdefghi${String(i).padStart(2, '0')}`, name: `n${i}` }))
    expect(parseMusic({ custom: many }).custom).toHaveLength(8)
  })
  it('falls back when the selected station is unknown, and clamps the volume', () => {
    expect(parseMusic({ selected: 'zzzzzzzzzzz' }).selected).toBe(DEFAULT_STATIONS[0].id)
    expect(parseMusic({ volume: 9 }).volume).toBe(1)
    expect(parseMusic({ volume: -1 }).volume).toBe(0)
    expect(parseMusic({ volume: 'loud' }).volume).toBe(0.5)
  })
})

describe('music store', () => {
  beforeEach(() => {
    useMusic.setState({ custom: [], selected: DEFAULT_STATIONS[0].id, volume: 0.5, status: 'idle', failure: null })
  })
  it('adds a pasted link, selects it and does not duplicate it', () => {
    expect(useMusic.getState().addCustom(`https://youtu.be/${A}`, 'Mi radio')).toBe(A)
    expect(useMusic.getState().selected).toBe(A)
    expect(stationsOf(useMusic.getState().custom).map((s) => s.name)).toContain('Mi radio')
    expect(useMusic.getState().addCustom(`https://www.youtube.com/watch?v=${A}`)).toBe(A)
    expect(useMusic.getState().custom).toHaveLength(1)
  })
  it('refuses things that are not YouTube videos', () => {
    expect(useMusic.getState().addCustom('https://example.com/x')).toBeNull()
    expect(useMusic.getState().custom).toHaveLength(0)
  })
  it('selecting a built-in link just selects it', () => {
    expect(useMusic.getState().addCustom(DEFAULT_STATIONS[1].id)).toBe(DEFAULT_STATIONS[1].id)
    expect(useMusic.getState().custom).toHaveLength(0)
    expect(useMusic.getState().selected).toBe(DEFAULT_STATIONS[1].id)
  })
  it('removing the selected link goes back to the first built-in', () => {
    useMusic.getState().addCustom(A)
    useMusic.getState().addCustom(B)
    useMusic.getState().removeCustom(B)
    expect(useMusic.getState().selected).toBe(DEFAULT_STATIONS[0].id)
    expect(useMusic.getState().custom.map((s) => s.id)).toEqual([A])
  })
  it('ignores unknown stations and clamps volume', () => {
    useMusic.getState().select('zzzzzzzzzzz')
    expect(useMusic.getState().selected).toBe(DEFAULT_STATIONS[0].id)
    useMusic.getState().setVolume(5)
    expect(useMusic.getState().volume).toBe(1)
    useMusic.getState().setVolume(NaN)
    expect(useMusic.getState().volume).toBe(0)
  })
  it('stops accepting links at 8', () => {
    for (let i = 0; i < 8; i++) useMusic.getState().addCustom(`abcdefghi${String(i).padStart(2, '0')}`)
    expect(useMusic.getState().addCustom(B)).toBeNull()
    expect(useMusic.getState().custom).toHaveLength(8)
  })
})
