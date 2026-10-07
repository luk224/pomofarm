import { create } from 'zustand'

/** Per-device preferences (sound and notifications belong to the browser, not to the game save). */
interface Prefs {
  sound: boolean
  notify: boolean
  setSound: (on: boolean) => void
  setNotify: (on: boolean) => void
}

const KEY = 'pomofarm.prefs'

function load(): { sound: boolean; notify: boolean } {
  try {
    const raw = JSON.parse(localStorage.getItem(KEY) ?? '{}')
    return { sound: raw.sound !== false, notify: raw.notify !== false }
  } catch {
    return { sound: true, notify: true } // storage blocked: use defaults
  }
}

function save(p: { sound: boolean; notify: boolean }) {
  try {
    localStorage.setItem(KEY, JSON.stringify(p))
  } catch {
    /* private mode: the choice lasts until reload */
  }
}

export const usePrefs = create<Prefs>((set, get) => ({
  ...load(),
  setSound: (sound) => {
    set({ sound })
    save({ sound, notify: get().notify })
  },
  setNotify: (notify) => {
    set({ notify })
    save({ sound: get().sound, notify })
  },
}))
