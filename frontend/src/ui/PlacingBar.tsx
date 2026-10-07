import { useEffect } from 'react'
import { formatPrice, hiveArea, newlyCovered } from '../store/automation'
import { useGame } from '../store/game'
import { useUi } from '../store/ui'
import { cancelPlacing, confirmPlacing } from './actions'
import { effectivePlot } from './selection'

/**
 * Choosing where a hive goes. Tapping a plot (or the arrow keys) moves the preview; the button confirms, so a
 * 4.000+ 🪙 purchase is never one stray click away. Escape cancels.
 */
export function PlacingBar() {
  const placing = useUi((s) => s.placing)
  const selectedPlotId = useUi((s) => s.selectedPlotId)
  const plots = useGame((s) => s.state?.plots)
  const bees = useGame((s) => s.state?.automation.bees)

  useEffect(() => {
    if (!placing) return
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') cancelPlacing()
      else if (e.key === 'Enter') {
        e.preventDefault()
        void confirmPlacing()
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [placing])

  if (!placing || !plots || !bees) return null
  const target = effectivePlot(plots, selectedPlotId)
  if (!target) return null
  const buying = placing.hiveId === null
  const covered = hiveArea(plots, target.x, target.y).length
  const gain = newlyCovered(plots, target.x, target.y)
  return (
    <section className="placing" role="group" aria-label="Colocar colmena" data-testid="placing-bar">
      <p className="placing__text" data-testid="placing-text">
        {buying ? 'Elige dónde va la colmena.' : 'Elige a dónde mover la colmena.'}{' '}
        Cubre {covered} {covered === 1 ? 'parcela' : 'parcelas'}
        {gain > 0 ? `, ${gain} produciendo ahora sin abejas` : ''}.
      </p>
      <div className="placing__buttons">
        <button type="button" className="btn btn--primary" data-testid="placing-confirm" onClick={() => void confirmPlacing()}>
          {buying ? `Colocar aquí${bees.next_cost !== null ? ` · ${formatPrice(bees.next_cost)} 🪙` : ''}` : 'Mover aquí (gratis)'}
        </button>
        <button type="button" className="btn" data-testid="placing-cancel" onClick={cancelPlacing}>Cancelar</button>
      </div>
    </section>
  )
}
