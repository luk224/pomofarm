import { create } from 'zustand'
import { isAmbientKind, type AmbientKind } from '../audio/ambient'
import type { Layer } from '../audio/engine'

/** Per-device preferences (sound and notifications belong to the browser, not to the game save). */
export interface PrefsData {
  /** Chime when a Pomodoro or rest ends. */
  sound: boolean
  notify: boolean
  /** 0..1 per sound layer. */
  volumes: Record<Layer, number>
  ambient: AmbientKind
}

interface Prefs extends PrefsData {
  setSound: (on: boolean) => void
  setNotify: (on: boolean) => void
  setVolume: (layer: Layer, v: number) => void
  setAmbient: (kind: AmbientKind) => void
}

const KEY = 'pomofarm.prefs'
export const DEFAULT_VOLUMES: Record<Layer, number> = { ambient: 0.5, effects: 0.7, alerts: 0.8 }

const clamp01 = (v: unknown, fallback: number) => (typeof v === 'number' && Number.isFinite(v) ? Math.min(1, Math.max(0, v)) : fallback)

/** Reads saved prefs, tolerating missing, old-format or garbage values. */
export function parsePrefs(raw: unknown): PrefsData {
  const r = (raw && typeof raw === 'object' ? raw : {}) as Record<string, unknown>
  const v = (r.volumes && typeof r.volumes === 'object' ? r.volumes : {}) as Record<string, unknown>
  return {
    sound: r.sound !== false,
    notify: r.notify !== false,
    volumes: {
      ambient: clamp01(v.ambient, DEFAULT_VOLUMES.ambient),
      effects: clamp01(v.effects, DEFAULT_VOLUMES.effects),
      alerts: clamp01(v.alerts, DEFAULT_VOLUMES.alerts),
    },
    ambient: isAmbientKind(r.ambient) ? r.ambient : 'off',
  }
}

function load(): PrefsData {
  try {
    return parsePrefs(JSON.parse(localStorage.getItem(KEY) ?? '{}'))
  } catch {
    return parsePrefs({}) // storage blocked: use defaults
  }
}

function save(p: PrefsData) {
  try {
    localStorage.setItem(KEY, JSON.stringify(p))
  } catch {
    /* private mode: the choice lasts until reload */
  }
}

const data = (s: Prefs): PrefsData => ({ sound: s.sound, notify: s.notify, volumes: s.volumes, ambient: s.ambient })

export const usePrefs = create<Prefs>((set, get) => {
  const update = (patch: Partial<PrefsData>) => {
    set(patch)
    save(data(get()))
  }
  return {
    ...load(),
    setSound: (sound) => update({ sound }),
    setNotify: (notify) => update({ notify }),
    setVolume: (layer, v) => update({ volumes: { ...get().volumes, [layer]: clamp01(v, get().volumes[layer]) } }),
    setAmbient: (ambient) => update({ ambient }),
  }
})
