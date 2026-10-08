import { create } from 'zustand'
import { DEFAULT_STATIONS, parseVideoId, type Station } from '../audio/stations'

/**
 * The Lofi player's state. `status`:
 * idle (nothing requested), loading (script/iframe starting), playing, paused,
 * fallback (YouTube failed: the local ambient sound plays instead).
 */
export type MusicStatus = 'idle' | 'loading' | 'playing' | 'paused' | 'fallback'

export interface MusicSaved {
  custom: Station[]
  selected: string
  volume: number
}

const KEY = 'pomofarm.music'
const MAX_CUSTOM = 8

/** Reads the saved player settings, tolerating missing or garbage values. */
export function parseMusic(raw: unknown): MusicSaved {
  const r = (raw && typeof raw === 'object' ? raw : {}) as Record<string, unknown>
  const custom: Station[] = []
  if (Array.isArray(r.custom)) {
    for (const c of r.custom) {
      const id = c && typeof c === 'object' ? parseVideoId(String((c as Record<string, unknown>).id ?? '')) : null
      const name = c && typeof c === 'object' ? String((c as Record<string, unknown>).name ?? '').slice(0, 40) : ''
      if (id && !custom.some((x) => x.id === id) && !DEFAULT_STATIONS.some((d) => d.id === id) && custom.length < MAX_CUSTOM) {
        custom.push({ id, name: name || `Mi enlace (${id.slice(0, 4)}…)`, custom: true })
      }
    }
  }
  const all = [...DEFAULT_STATIONS, ...custom]
  const selected = typeof r.selected === 'string' && all.some((s) => s.id === r.selected) ? r.selected : DEFAULT_STATIONS[0].id
  const volume = typeof r.volume === 'number' && Number.isFinite(r.volume) ? Math.min(1, Math.max(0, r.volume)) : 0.5
  return { custom, selected, volume }
}

function load(): MusicSaved {
  try {
    return parseMusic(JSON.parse(localStorage.getItem(KEY) ?? '{}'))
  } catch {
    return parseMusic({})
  }
}

function save(m: MusicSaved) {
  try {
    localStorage.setItem(KEY, JSON.stringify(m))
  } catch {
    /* private mode: the choice lasts until reload */
  }
}

interface MusicStore extends MusicSaved {
  status: MusicStatus
  /** Whether the widget is shown. */
  open: boolean
  /** Why the last attempt fell back, for the message. */
  failure: 'network' | 'timeout' | 'video' | null
  setOpen: (open: boolean) => void
  setStatus: (s: MusicStatus, failure?: MusicStore['failure']) => void
  select: (id: string) => void
  setVolume: (v: number) => void
  /** Adds a station from a pasted link or ID. Returns its id, or null if the text is not a YouTube video. */
  addCustom: (text: string, name?: string) => string | null
  removeCustom: (id: string) => void
}

export const stationsOf = (custom: Station[]): Station[] => [...DEFAULT_STATIONS, ...custom]

export const useMusic = create<MusicStore>((set, get) => {
  const persist = (patch: Partial<MusicSaved>) => {
    set(patch)
    const s = get()
    save({ custom: s.custom, selected: s.selected, volume: s.volume })
  }
  return {
    ...load(),
    status: 'idle',
    open: false,
    failure: null,
    setOpen: (open) => set({ open }),
    setStatus: (status, failure = null) => set({ status, failure }),
    select: (id) => {
      if (stationsOf(get().custom).some((s) => s.id === id)) persist({ selected: id })
    },
    setVolume: (v) => persist({ volume: Math.min(1, Math.max(0, Number.isFinite(v) ? v : 0)) }),
    addCustom: (text, name) => {
      const id = parseVideoId(text)
      if (!id) return null
      const known = stationsOf(get().custom).find((s) => s.id === id)
      if (known) {
        persist({ selected: id })
        return id
      }
      if (get().custom.length >= MAX_CUSTOM) return null
      persist({ custom: [...get().custom, { id, name: (name ?? '').trim().slice(0, 40) || `Mi enlace (${id.slice(0, 4)}…)`, custom: true }], selected: id })
      return id
    },
    removeCustom: (id) => {
      const custom = get().custom.filter((s) => s.id !== id)
      persist({ custom, selected: get().selected === id ? DEFAULT_STATIONS[0].id : get().selected })
    },
  }
})
