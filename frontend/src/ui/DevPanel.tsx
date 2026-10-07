import { useGame } from '../store/game'

/** Development-only controls to exercise the API by hand. Not shipped (see App). */
export function DevPanel() {
  const { state, plant, pause, resume, cancel, harvest } = useGame()
  const plot = state?.plots[0]
  const p = state?.pomodoro
  return (
    <div style={{ position: 'fixed', bottom: 8, left: 8, display: 'flex', gap: 6, flexWrap: 'wrap' }}>
      <button disabled={!plot || !!p} onClick={() => plot && plant({ plot_id: plot.id, plant_type: 'daisy' })}>
        Plantar margarita
      </button>
      <button disabled={p?.status !== 'running'} onClick={() => pause()}>Pausar</button>
      <button disabled={p?.status !== 'paused'} onClick={() => resume()}>Reanudar</button>
      <button disabled={!p} onClick={() => cancel()}>Cancelar</button>
      <button
        disabled={!plot || plot.state !== 'mature' || plot.harvested}
        onClick={() => plot && harvest(plot.id)}
      >
        Cosechar
      </button>
    </div>
  )
}
