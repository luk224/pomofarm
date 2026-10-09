import { describe, expect, it } from 'vitest'
import { DEFAULT_VOLUMES, parsePrefs } from './prefs'

describe('parsePrefs', () => {
  it('gives sensible defaults for an empty or missing save', () => {
    for (const raw of [{}, null, undefined, 'x', 5]) {
      expect(parsePrefs(raw)).toEqual({ sound: true, notify: true, volumes: DEFAULT_VOLUMES, ambient: 'off', reduceMotion: false, muted: false, palette: 'default', focusMode: true, farmAttention: true })
    }
  })
  it('keeps old saves (before volumes existed) working', () => {
    expect(parsePrefs({ sound: false, notify: false })).toEqual({ sound: false, notify: false, volumes: DEFAULT_VOLUMES, ambient: 'off', reduceMotion: false, muted: false, palette: 'default', focusMode: true, farmAttention: true })
  })
  it('reads each layer on its own', () => {
    expect(parsePrefs({ volumes: { ambient: 0.1, effects: 0.2, alerts: 0.3 }, ambient: 'rain' }).volumes).toEqual({ ambient: 0.1, effects: 0.2, alerts: 0.3 })
    expect(parsePrefs({ volumes: { effects: 0 } }).volumes).toEqual({ ...DEFAULT_VOLUMES, effects: 0 })
  })
  it('clamps and rejects bad values', () => {
    const p = parsePrefs({ volumes: { ambient: 9, effects: -1, alerts: 'loud' }, ambient: 'ocean' })
    expect(p.volumes).toEqual({ ambient: 1, effects: 0, alerts: DEFAULT_VOLUMES.alerts })
    expect(p.ambient).toBe('off')
  })
  it('reads the accessibility choices and ignores anything odd', () => {
    expect(parsePrefs({ reduceMotion: true, muted: true, palette: 'cb' })).toMatchObject({ reduceMotion: true, muted: true, palette: 'cb' })
    expect(parsePrefs({ reduceMotion: 'yes', muted: 1, palette: 'neon' })).toMatchObject({ reduceMotion: false, muted: false, palette: 'default', focusMode: true, farmAttention: true })
  })
  it('keeps Modo Foco on unless it was explicitly turned off', () => {
    expect(parsePrefs({}).focusMode).toBe(true)
    expect(parsePrefs({ focusMode: false }).focusMode).toBe(false)
    expect(parsePrefs({ focusMode: 'no' }).focusMode).toBe(true)
  })
})
