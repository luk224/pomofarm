import { describe, expect, it } from 'vitest'
import { DEFAULT_VOLUMES, parsePrefs } from './prefs'

describe('parsePrefs', () => {
  it('gives sensible defaults for an empty or missing save', () => {
    for (const raw of [{}, null, undefined, 'x', 5]) {
      expect(parsePrefs(raw)).toEqual({ sound: true, notify: true, volumes: DEFAULT_VOLUMES, ambient: 'off' })
    }
  })
  it('keeps old saves (before volumes existed) working', () => {
    expect(parsePrefs({ sound: false, notify: false })).toEqual({ sound: false, notify: false, volumes: DEFAULT_VOLUMES, ambient: 'off' })
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
})
