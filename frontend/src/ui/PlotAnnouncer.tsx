import { useGame } from '../store/game'
import { useUi } from '../store/ui'
import { describePlot, effectivePlot } from './selection'

/** Says aloud (to screen readers) which plot is in view and what state it is in; invisible on screen. */
export function PlotAnnouncer() {
  const plots = useGame((s) => s.state?.plots)
  const selected = useUi((s) => s.selectedPlotId)
  const text = plots && plots.length > 1 ? describePlot(plots, effectivePlot(plots, selected)) : ''
  return (
    <div className="sr-only" role="status" aria-live="polite" data-testid="plot-announcer">
      {text}
    </div>
  )
}
