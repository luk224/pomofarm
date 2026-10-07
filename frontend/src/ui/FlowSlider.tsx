import { clampFlow, flowRow, FLOW_MAX, FLOW_MIN, formatHours, useFlowTable } from '../store/flow'
import { useUi } from '../store/ui'
import { formatClock } from '../store/time'
import { DropIcon } from './icons'

/** Flow mode (Oak, 60–120 min): one slider, no steps and no extra bonuses (GDD §4.3). The numbers come from the server. */
export function FlowSlider() {
  const minutes = useUi((s) => s.flowMinutes)
  const setMinutes = useUi((s) => s.setFlowMinutes)
  const table = useFlowTable(true)
  const row = flowRow(table, minutes)
  const summary = row
    ? `+${row.reward} gotas · vive ${formatHours(row.life_h)}`
    : 'Calculando…'
  return (
    <div className="flow" data-testid="flow">
      <label className="flow__label" htmlFor="flow-range">
        Duración <strong data-testid="flow-minutes">{formatClock(minutes * 60_000)}</strong>
      </label>
      <input id="flow-range" className="flow__range" type="range" min={FLOW_MIN} max={FLOW_MAX} step={1} value={minutes}
        aria-valuetext={`${minutes} minutos. ${summary}`} onChange={(e) => setMinutes(clampFlow(Number(e.target.value)))} />
      <div className="flow__summary" data-testid="flow-summary" aria-live="polite">
        {row ? <>+{row.reward} <DropIcon size={13} /> · vive {formatHours(row.life_h)}</> : summary}
      </div>
    </div>
  )
}
