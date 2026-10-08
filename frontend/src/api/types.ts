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
  /** Inside the 3×3 area of a hive. */
  bees: boolean
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
  /** Strict mode (chosen when it started): the player's own challenge, never a block. */
  strict: boolean
  pauses: number
  /** Time spent paused so far, including a pause in progress, as of the answer. */
  paused_ms: number
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

export interface HiveState {
  id: number
  plot_id: number
  x: number
  y: number
}

/** Bees: unlocked with 💧, then up to `max` hives bought with 🪙 (GDD §4.7). Costs are whole 🪙. */
export interface BeesState {
  unlocked: boolean
  unlock_cost: number
  hives: HiveState[]
  max: number
  next_cost: number | null
}

export interface DogState {
  unlocked: boolean
  unlock_cost: number
  owned: boolean
  cost: number
  silo_bonus_hours: number
}

export interface AutomationState {
  bees: BeesState
  dog: DogState
}

export type DecorKind = 'path' | 'fence' | 'lantern'

export interface DecorItem {
  id: number
  kind: DecorKind
  x: number
  y: number
}

export interface DecorState {
  items: DecorItem[]
  /** Pieces for sale, in shop order, with their price in whole 🪙. */
  catalog: { kind: DecorKind; cost: number }[]
  /** Background cells run from min to max on both axes (the 4×4 field itself is never decorated). */
  min: number
  max: number
  hat: { owned: boolean; available: boolean; cost: number }
  blocked: [number, number][]
}

/** Personal numbers (GDD §4.8): weekly Pomodoros, best streak and clean Pomodoros. */
export interface Stats {
  time_zone: string
  weeks: { start: string; pomodoros: number; seconds: number }[]
  this_week: number
  total: number
  best_streak_days: number
  best_streak_end: string
  current_streak_days: number
  strict_total: number
  clean: number
  strict_on: boolean
}

/** One month of the Harvest Book (GDD §4.8). Days and weekdays are in the player's time zone. */
export interface BookMonth {
  month: string
  time_zone: string
  pomodoros: number
  seconds: number
  days: { date: string; pomodoros: number; seconds: number }[]
  /** Most time first; an empty name means "no tag". */
  tags: { name: string; pomodoros: number; seconds: number }[]
  /** Monday first (index 0) to Sunday (6). */
  weekdays: { weekday: number; pomodoros: number; seconds: number }[]
  months: string[]
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
  automation: AutomationState
  decor: DecorState
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
