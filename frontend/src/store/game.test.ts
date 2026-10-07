import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { GameState } from '../api/types'

vi.mock('../api/client', async (orig) => {
  const real = await orig<typeof import('../api/client')>()
  return { ...real, api: { buyPlot: vi.fn(), upgradeSilo: vi.fn(), collectSilo: vi.fn(), state: vi.fn(), plant: vi.fn(), pause: vi.fn(), resume: vi.fn(), cancel: vi.fn(), harvest: vi.fn(), clearPlot: vi.fn(), setSetting: vi.fn(), unlockSeed: vi.fn() } }
})

import { api } from '../api/client'
import { useUi } from './ui'
import { useGame } from './game'

const empty: GameState = {
  server_time: 't', player: { name: 'x', focus_points: 0, lifetime_focus: 0, coins_milli: 0, silo_level: 0, season: 1 },
  plots: [], seeds: [], pomodoro: null, recent_tags: [], settings: {},
  silo: { content_milli: 0, capacity_milli: 0, capacity_hours: 12, rate_milli_per_h: 0, full: false },
  shop: { plots_owned: 1, plots_max: 16, next_plot: null, silo_upgrade: null },
}
const later = <T,>(v: T, ms = 20) => new Promise<T>((r) => setTimeout(() => r(v), ms))

beforeEach(() => { vi.clearAllMocks(); useUi.setState({ toasts: [] }) })

describe('one mutation at a time', () => {
  it('a double click sends the request once', async () => {
    vi.mocked(api.plant).mockImplementation(() => later(empty))
    const plant = { plot_id: 1, plant_type: 'daisy' }
    await Promise.all([useGame.getState().plant(plant), useGame.getState().plant(plant)])
    expect(api.plant).toHaveBeenCalledTimes(1)
  })
  it('allows the next action once the first finished', async () => {
    vi.mocked(api.pause).mockResolvedValue(empty)
    await useGame.getState().pause()
    await useGame.getState().pause()
    expect(api.pause).toHaveBeenCalledTimes(2)
  })
  it('releases the lock after a failure', async () => {
    const { ApiError } = await import('../api/client')
    vi.mocked(api.pause).mockRejectedValueOnce(new ApiError(0, 'network')).mockResolvedValue(empty)
    vi.mocked(api.state).mockRejectedValue(new ApiError(0, 'network'))
    await useGame.getState().pause()
    expect(useGame.getState().error).toBe('network')
    await useGame.getState().pause()
    expect(api.pause).toHaveBeenCalledTimes(2)
  })
})

describe('toasts', () => {
  it('never stack identical messages', () => {
    useUi.getState().toast('Hola', 'error')
    useUi.getState().toast('Hola', 'error')
    expect(useUi.getState().toasts).toHaveLength(1)
  })
})
