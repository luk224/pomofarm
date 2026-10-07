import type { PlotState } from '../api/types'

/** A plant whose 💧 reward is waiting (mature, or withered while the player was away). */
export const needsHarvest = (p: PlotState) => (p.state === 'mature' || p.state === 'withered') && !p.harvested

/** A harvested plant that can be removed to make room. */
export const canClear = (p: PlotState) => (p.state === 'mature' || p.state === 'withered') && p.harvested

/**
 * The plot the dock talks about. The player's choice wins; with no choice (or a stale one) the game points
 * at what most needs attention: a plant waiting to be harvested, then a free plot, then the first plot.
 */
export function effectivePlot(plots: PlotState[], selectedId: number | null): PlotState | undefined {
  return (
    plots.find((p) => p.id === selectedId) ??
    plots.find(needsHarvest) ??
    plots.find((p) => p.state === 'empty') ??
    plots[0]
  )
}

/** The id of the plot `step` places away from the one in view, wrapping around (for keyboard navigation). */
export function stepPlot(plots: PlotState[], selectedId: number | null, step: number): number | null {
  if (plots.length === 0) return null
  const ordered = [...plots].sort((a, b) => a.id - b.id)
  const current = effectivePlot(plots, selectedId)
  const at = Math.max(0, ordered.findIndex((p) => p.id === current?.id))
  return ordered[(at + step + ordered.length * 4) % ordered.length].id
}

/** What a screen reader hears when the plot in view changes. */
export function describePlot(plots: PlotState[], plot: PlotState | undefined): string {
  if (!plot) return ''
  const ordered = [...plots].sort((a, b) => a.id - b.id)
  const n = ordered.findIndex((p) => p.id === plot.id) + 1
  const what =
    plot.state === 'empty' ? 'libre'
    : plot.state === 'growing' ? 'creciendo'
    : needsHarvest(plot) ? (plot.state === 'withered' ? 'marchita, aún se puede cosechar' : 'lista para cosechar')
    : plot.state === 'withered' ? 'marchita'
    : 'cosechada y produciendo'
  const bonus = plot.bonus && plot.bonus.multiplier > 1 ? `, bono de ${Math.round((plot.bonus.multiplier - 1) * 100)} por ciento` : ''
  return `Parcela ${n} de ${ordered.length}: ${what}${bonus}`
}
