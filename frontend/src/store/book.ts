/** Pure helpers for the Harvest Book (GDD §4.8): hours, harvest icons, month navigation and names. */

export type HarvestUnit = 'bale' | 'silo' | 'basket'
export const HARVEST_UNITS: readonly HarvestUnit[] = ['bale', 'silo', 'basket']
export const UNIT_NAMES: Record<HarvestUnit, { one: string; many: string; label: string }> = {
  bale: { one: 'fardo de heno', many: 'fardos de heno', label: 'Fardos' },
  silo: { one: 'silo', many: 'silos', label: 'Silos' },
  basket: { one: 'cesta de manzanas', many: 'cestas de manzanas', label: 'Cestas' },
}

export function isHarvestUnit(v: unknown): v is HarvestUnit {
  return typeof v === 'string' && (HARVEST_UNITS as readonly string[]).includes(v)
}

/** "2 h 05 min", "45 min", "0 min". */
export function formatFocus(seconds: number): string {
  const total = Math.max(0, Math.round(seconds / 60))
  const h = Math.floor(total / 60)
  const m = total % 60
  if (h === 0) return `${m} min`
  return `${h} h ${String(m).padStart(2, '0')} min`
}

export interface Harvest {
  /** Whole icons to draw. */
  full: number
  /** Fraction (0..1) of one more icon, 0 for none. */
  part: number
  /** Hours that one icon stands for: 1, or 5 / 10 / … when the month is long enough to need fewer icons. */
  hoursPerIcon: number
}

const MAX_ICONS = 40

/** One icon per hour of focus, grouped (×5, ×10, …) so a month never needs more than 40 icons. */
export function harvestOf(seconds: number): Harvest {
  const hours = Math.max(0, seconds) / 3600
  let hoursPerIcon = 1
  for (const step of [5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000]) {
    if (hours / hoursPerIcon <= MAX_ICONS) break
    hoursPerIcon = step
  }
  const icons = hours / hoursPerIcon
  const full = Math.floor(icons + 1e-9)
  const part = icons - full
  return { full, part: part < 0.05 ? 0 : Math.round(part * 100) / 100, hoursPerIcon }
}

/** "octubre de 2026". */
export function monthLabel(month: string): string {
  const [y, m] = month.split('-').map(Number)
  return new Intl.DateTimeFormat('es-ES', { month: 'long', year: 'numeric', timeZone: 'UTC' }).format(new Date(Date.UTC(y, m - 1, 15)))
}

export const WEEKDAY_NAMES = ['Lunes', 'Martes', 'Miércoles', 'Jueves', 'Viernes', 'Sábado', 'Domingo'] as const

/** "mié 7 oct". */
export function dayLabel(date: string): string {
  const [y, m, d] = date.split('-').map(Number)
  return new Intl.DateTimeFormat('es-ES', { weekday: 'short', day: 'numeric', month: 'short', timeZone: 'UTC' }).format(new Date(Date.UTC(y, m - 1, d)))
}
