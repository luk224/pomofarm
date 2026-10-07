import { create } from 'zustand'

/** Screen position (CSS px, relative to the canvas) of the point the timer ring floats over. */
interface RingAnchor {
  on: boolean
  x: number
  y: number
  set: (on: boolean, x: number, y: number) => void
}

export const useRingAnchor = create<RingAnchor>((set, get) => ({
  on: false,
  x: 0,
  y: 0,
  set: (on, x, y) => {
    const s = get()
    // Ignore sub-pixel jitter so idle frames cause no work.
    if (s.on === on && Math.abs(s.x - x) < 0.5 && Math.abs(s.y - y) < 0.5) return
    set({ on, x, y })
  },
}))
