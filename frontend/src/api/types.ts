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

/** What a producing plant earns from its surroundings (GDD §4.6). */
export interface PlotBonus {
  multiplier: number
  neighbours: number
  garden: boolean
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
  /** Present only while the plant is producing. */
  bonus: PlotBonus | null
}

export interface SeedState {
  key: string
  duration_min: number
  unlock_cost: number
  reward: number
  life_h: number
  unlocked: boolean
  /** The two crops this one combines with. */
  compatible: string[]
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

export interface PlotOffer {
  number: number
  cost: number
  x: number
  y: number
}

export interface SiloOffer {
  level: number
  cost: number
  capacity_hours: number
}

/** What can be bought and for how much (prices come from the server, never hardcoded here). */
export interface ShopState {
  plots_owned: number
  plots_max: number
  next_plot: PlotOffer | null
  silo_upgrade: SiloOffer | null
}

/** The optional break after a Pomodoro (GDD §3.2). */
export interface RestState {
  total_s: number
  remaining_ms: number
  ends_at: string
}

export interface GameState {
  server_time: string
  player: PlayerState
  plots: PlotState[]
  seeds: SeedState[]
  pomodoro: PomodoroState | null
  silo: SiloState
  shop: ShopState
  rest: RestState | null
  recent_tags: string[]
  settings: Record<string, string>
}

export interface PlantRequest {
  plot_id: number
  plant_type: string
  tag?: string
  duration_min?: number
}

/** One minute of the Oak's Flow range (GDD §4.3), served by the backend. */
export interface FlowRow {
  duration_min: number
  reward: number
  life_h: number
  yield: number
  rest_min: number
}
