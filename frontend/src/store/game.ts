import { create } from 'zustand'
import { api, ApiError } from '../api/client'
import type { GameState, PlantRequest } from '../api/types'

interface GameStore {
  state: GameState | null
  /** performance.now() when `state` arrived: base for the monotonic countdown. */
  fetchedAt: number
  error: string | null
  refresh: () => Promise<void>
  clearError: () => void
  plant: (req: PlantRequest) => Promise<void>
  pause: () => Promise<void>
  resume: () => Promise<void>
  cancel: () => Promise<void>
  harvest: (plotId: number) => Promise<number>
  unlockSeed: (key: string) => Promise<void>
  clearPlot: (plotId: number, confirm: boolean) => Promise<void>
  setSetting: (key: string, value: string) => Promise<void>
}

export const useGame = create<GameStore>((set) => {
  const apply = (state: GameState) => set({ state, fetchedAt: performance.now(), error: null })
  const fail = (e: unknown) => {
    set({ error: e instanceof ApiError ? e.code : 'unknown' })
  }
  /** Runs an action; on failure records the error code and re-syncs with the server. */
  const act = async (fn: () => Promise<GameState>) => {
    try {
      apply(await fn())
    } catch (e) {
      fail(e)
      // The usual reason is that another device changed things: reload the truth.
      try {
        apply(await api.state())
        set({ error: e instanceof ApiError ? e.code : 'unknown' })
      } catch {
        /* still offline: keep the error */
      }
    }
  }
  return {
    state: null,
    fetchedAt: 0,
    error: null,
    clearError: () => set({ error: null }),
    refresh: async () => {
      try {
        apply(await api.state())
      } catch (e) {
        fail(e)
      }
    },
    plant: (req) => act(() => api.plant(req)),
    pause: () => act(() => api.pause()),
    resume: () => act(() => api.resume()),
    cancel: () => act(() => api.cancel()),
    unlockSeed: (key) => act(() => api.unlockSeed(key)),
    clearPlot: (plotId, confirm) => act(() => api.clearPlot(plotId, confirm)),
    setSetting: (key, value) => act(() => api.setSetting(key, value)),
    harvest: async (plotId) => {
      try {
        const r = await api.harvest(plotId)
        apply(r.state)
        return r.reward_focus
      } catch (e) {
        fail(e)
        return 0
      }
    },
  }
})
