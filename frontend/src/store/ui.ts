import { create } from 'zustand'

export interface Toast {
  id: number
  text: string
  kind: 'info' | 'reward' | 'error'
}

interface UiStore {
  selectedSeed: string | null
  tag: string
  toasts: Toast[]
  selectSeed: (key: string | null) => void
  setTag: (tag: string) => void
  toast: (text: string, kind?: Toast['kind']) => void
  dismiss: (id: number) => void
}

let nextId = 1

export const useUi = create<UiStore>((set, get) => ({
  selectedSeed: null,
  tag: '',
  toasts: [],
  selectSeed: (key) => set({ selectedSeed: key }),
  setTag: (tag) => set({ tag }),
  toast: (text, kind = 'info') => {
    const id = nextId++
    set({ toasts: [...get().toasts.slice(-2), { id, text, kind }] })
    window.setTimeout(() => get().dismiss(id), kind === 'error' ? 6000 : 4000)
  },
  dismiss: (id) => set({ toasts: get().toasts.filter((t) => t.id !== id) }),
}))
