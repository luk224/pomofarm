import { create } from 'zustand'
import type { DecorKind } from '../api/types'

export interface Toast {
  id: number
  text: string
  kind: 'info' | 'reward' | 'coin' | 'error'
}

interface UiStore {
  selectedSeed: string | null
  /** The plot the player is looking at; it decides what the dock offers. */
  selectedPlotId: number | null
  /** Length of the next Oak (Flow) session, in minutes. */
  flowMinutes: number
  /** Hive placement: `hiveId` null buys a new one, otherwise moves that hive. Null when not placing. */
  placing: { hiveId: number | null } | null
  /** Decoration mode: `itemId` null places new pieces of `piece` one tap at a time; otherwise moves that piece. */
  decorMode: { piece: DecorKind; itemId: number | null } | null
  /** Keyboard cursor while decorating: the cell Enter would place on. */
  decorCursor: [number, number] | null
  /** A visual alert for when the sound is off: shown for a few seconds, then gone (works with reduced motion too). */
  flash: { id: number; text: string } | null
  tag: string
  toasts: Toast[]
  selectSeed: (key: string | null) => void
  selectPlot: (id: number | null) => void
  setFlowMinutes: (m: number) => void
  setPlacing: (p: { hiveId: number | null } | null) => void
  setDecorMode: (m: { piece: DecorKind; itemId: number | null } | null) => void
  setDecorCursor: (c: [number, number] | null) => void
  setTag: (tag: string) => void
  showFlash: (text: string) => void
  toast: (text: string, kind?: Toast['kind']) => void
  dismiss: (id: number) => void
}

let nextId = 1

export const useUi = create<UiStore>((set, get) => ({
  selectedSeed: null,
  selectedPlotId: null,
  flowMinutes: 60,
  flash: null,
  placing: null,
  decorMode: null,
  decorCursor: null,
  tag: '',
  toasts: [],
  selectSeed: (key) => set({ selectedSeed: key }),
  selectPlot: (id) => set({ selectedPlotId: id }),
  setFlowMinutes: (m) => set({ flowMinutes: m }),
  setPlacing: (p) => set({ placing: p, decorMode: p ? null : get().decorMode }),
  setDecorMode: (m) => set({ decorMode: m, placing: m ? null : get().placing, decorCursor: null }),
  setDecorCursor: (c) => set({ decorCursor: c }),
  setTag: (tag) => set({ tag }),
  showFlash: (text) => {
    const id = nextId++
    set({ flash: { id, text } })
    setTimeout(() => get().flash?.id === id && set({ flash: null }), 3000)
  },
  toast: (text, kind = 'info') => {
    if (get().toasts.some((t) => t.text === text)) return // never stack identical messages
    const id = nextId++
    set({ toasts: [...get().toasts.slice(-2), { id, text, kind }] })
    setTimeout(() => get().dismiss(id), kind === 'error' ? 6000 : 4000)
  },
  dismiss: (id) => set({ toasts: get().toasts.filter((t) => t.id !== id) }),
}))
