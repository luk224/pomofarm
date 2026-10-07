import type { PlotState } from '../api/types'

/** Plots inside the 3×3 area of a hive standing beside the plot at (x, y): that plot and its eight neighbours. */
export function hiveArea(plots: PlotState[], x: number, y: number): PlotState[] {
  return plots.filter((p) => Math.abs(p.x - x) <= 1 && Math.abs(p.y - y) <= 1)
}

/** How many plots of the area are producing now and are not already covered by another hive (the bonus does not stack). */
export function newlyCovered(plots: PlotState[], x: number, y: number): number {
  return hiveArea(plots, x, y).filter((p) => p.state === 'mature' && p.bonus !== null && !p.bonus.bees).length
}

/** "4.000" for whole 🪙 prices. */
export function formatPrice(coins: number): string {
  return new Intl.NumberFormat('es-ES', { useGrouping: 'always' }).format(coins)
}
