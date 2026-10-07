import type { GameState } from '../api/types'

export interface Completion {
  pomodoroId: number
  plotId: number
  plantType: string
}

/**
 * A Pomodoro "finished" when the previous state had an active one and the new state has none
 * while that plot now holds a mature, unharvested plant. A cancel (plot emptied) is not a finish,
 * and a page that loads already finished has no previous state, so it stays quiet.
 */
export function detectCompletion(prev: GameState | null, next: GameState): Completion | null {
  const before = prev?.pomodoro
  if (!before || before.plot_id === null) return null
  if (next.pomodoro && next.pomodoro.id === before.id) return null
  const plot = next.plots.find((p) => p.id === before.plot_id)
  if (!plot || plot.state !== 'mature' || plot.harvested) return null
  return { pomodoroId: before.id, plotId: plot.id, plantType: before.plant_type }
}
