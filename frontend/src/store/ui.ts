import { create } from 'zustand'

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
  tag: string
  toasts: Toast[]
  selectSeed: (key: string | null) => void
  selectPlot: (id: number | null) => void
  setFlowMinutes: (m: number) => void
  setTag: (tag: string) => void
  toast: (text: string, kind?: Toast['kind']) => void
  dismiss: (id: number) => void
}

let nextId = 1

export const useUi = create<UiStore>((set, get) => ({
  selectedSeed: null,
  selectedPlotId: null,
  flowMinutes: 60,
  tag: '',
  toasts: [],
  selectSeed: (key) => set({ selectedSeed: key }),
  selectPlot: (id) => set({ selectedPlotId: id }),
  setFlowMinutes: (m) => set({ flowMinutes: m }),
  setTag: (tag) => set({ tag }),
  toast: (text, kind = 'info') => {
    if (get().toasts.some((t) => t.text === text)) return // never stack identical messages
    const id = nextId++
    set({ toasts: [...get().toasts.slice(-2), { id, text, kind }] })
    setTimeout(() => get().dismiss(id), kind === 'error' ? 6000 : 4000)
  },
  dismiss: (id) => set({ toasts: get().toasts.filter((t) => t.id !== id) }),
}))
