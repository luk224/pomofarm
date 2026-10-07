import { describe, expect, it } from 'vitest'
import type { GameState, PlotState } from '../api/types'
import { detectCompletion } from './completion'

const plot = (over: Partial<PlotState> = {}): PlotState => ({
  id: 1, x: 1, y: 1, state: 'growing', plant_type: 'daisy', matured_at: null, wilts_at: null, harvested: false, bonus: null, ...over,
})
const state = (plots: PlotState[], active: boolean): GameState => ({
  server_time: 't',
  player: { name: 'x', focus_points: 0, lifetime_focus: 0, coins_milli: 0, silo_level: 0, season: 1 },
  plots, seeds: [], recent_tags: [], settings: {},
  silo: { content_milli: 0, capacity_milli: 0, capacity_hours: 12, rate_milli_per_h: 0, full: false },
  shop: { plots_owned: 1, plots_max: 16, next_plot: null, silo_upgrade: null },
  rest: null,
  pomodoro: active ? { id: 7, plot_id: 1, plant_type: 'daisy', status: 'running', planned_s: 600, remaining_ms: 1000, started_at: 't', paused_at: null, tag: null } : null,
})

describe('detectCompletion', () => {
  it('fires when the active Pomodoro ends and the plant is mature', () => {
    const c = detectCompletion(state([plot()], true), state([plot({ state: 'mature' })], false))
    expect(c).toEqual({ pomodoroId: 7, plotId: 1, plantType: 'daisy' })
  })
  it('does not fire on cancel (plot is emptied)', () => {
    expect(detectCompletion(state([plot()], true), state([plot({ state: 'empty', plant_type: null })], false))).toBeNull()
  })
  it('does not fire while the Pomodoro is still the same one', () => {
    expect(detectCompletion(state([plot()], true), state([plot()], true))).toBeNull()
  })
  it('does not fire when the page loads already finished (no previous state)', () => {
    expect(detectCompletion(null, state([plot({ state: 'mature' })], false))).toBeNull()
  })
  it('does not fire if there was no active Pomodoro before', () => {
    expect(detectCompletion(state([plot({ state: 'mature' })], false), state([plot({ state: 'mature' })], false))).toBeNull()
  })
  it('does not fire for a plant that is already harvested', () => {
    expect(detectCompletion(state([plot()], true), state([plot({ state: 'mature', harvested: true })], false))).toBeNull()
  })
})
