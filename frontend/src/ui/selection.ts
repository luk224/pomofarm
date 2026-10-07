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
