// Mirrors backend/internal/service/state.go. The client only displays this.
export type PlotStatus = 'empty' | 'growing' | 'mature' | 'withered'
export type PomodoroStatus = 'running' | 'paused'

export interface PlayerState {
  name: string
  focus_points: number
  lifetime_focus: number
  coins_milli: number
  silo_level: number
  season: number
}

export interface PlotState {
  id: number
  x: number
  y: number
  state: PlotStatus
  plant_type: string | null
  matured_at: string | null
  wilts_at: string | null
  harvested: boolean
}

export interface SeedState {
  key: string
  duration_min: number
  unlock_cost: number
  reward: number
  life_h: number
  unlocked: boolean
}

export interface PomodoroState {
  id: number
  plot_id: number | null
  plant_type: string
  status: PomodoroStatus
  planned_s: number
  remaining_ms: number
  started_at: string
  paused_at: string | null
  tag: string | null
}

export interface SiloState {
  content_milli: number
  capacity_milli: number
  capacity_hours: number
  /** What the farm produces right now, in thousandths of a coin per hour. */
  rate_milli_per_h: number
  full: boolean
}

export interface GameState {
  server_time: string
  player: PlayerState
  plots: PlotState[]
  seeds: SeedState[]
  pomodoro: PomodoroState | null
  silo: SiloState
  recent_tags: string[]
  settings: Record<string, string>
}

export interface PlantRequest {
  plot_id: number
  plant_type: string
  tag?: string
  duration_min?: number
}
