import type { PlotBonus } from '../api/types'

/** "+35%" for a multiplier of 1.35, or null when there is nothing to show. Rounded, never inventing a bonus. */
export function bonusLabel(b: PlotBonus | null | undefined): string | null {
  if (!b) return null
  const pct = Math.round((b.multiplier - 1) * 100 + 1e-9) // epsilon: 1.325 must read +33%, not +32% from float error
  return pct > 0 ? `+${pct}%` : null
}

/** "×1,35" for the dock text. */
export function formatMultiplier(m: number): string {
  return `×${new Intl.NumberFormat('es-ES', { maximumFractionDigits: 2 }).format(m)}`
}
