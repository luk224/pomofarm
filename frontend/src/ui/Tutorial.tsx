import { useGame } from '../store/game'
import { useUi } from '../store/ui'

/**
 * Three-step coach mark for the first Pomodoro (GDD §2.1): choose a seed, plant it, wait.
 * It is a real sequence, so the steps are numbered. Ends on its own after the first harvest.
 */
export function Tutorial() {
  const state = useGame((s) => s.state)
  const setSetting = useGame((s) => s.setSetting)
  const selected = useUi((s) => s.selectedSeed)
  if (!state || state.settings.tutorial_done === '1' || state.player.lifetime_focus > 0) return null
  if (state.plots.some((p) => p.state === 'mature')) return null

  const step = state.pomodoro ? 3 : selected ? 2 : 1
  const text = [
    'Elige una semilla. La Margarita es gratis y dura 10 minutos.',
    'Pulsa Plantar para empezar a concentrarte.',
    'Trabaja tranquilo: la planta crece mientras pasa el tiempo. Al terminar, tócala para cosechar.',
  ][step - 1]

  return (
    <div className="tutorial" role="note" data-testid="tutorial" data-step={step}>
      <span className="tutorial__step" aria-hidden="true">{step}/3</span>
      <p>{text}</p>
      <button type="button" className="btn btn--quiet" onClick={() => void setSetting('tutorial_done', '1')}>Saltar</button>
    </div>
  )
}
