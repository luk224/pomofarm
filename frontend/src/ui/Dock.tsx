import { useEffect, useRef, useState } from 'react'
import { useGame } from '../store/game'
import { useRemainingMs } from '../store/hooks'
import { formatClock } from '../store/time'
import { formatLife, lifeLeftMs } from '../store/life'
import { useUi } from '../store/ui'
import { harvestPlot, plantSelected, plantTargetId, togglePause } from './actions'
import { SEED_NAMES } from './names'
import type { PlotState } from '../api/types'
import { canClear, effectivePlot, needsHarvest } from './selection'
import { SeedPacket } from './SeedPacket'
import { FlowSlider } from './FlowSlider'
import { Tutorial } from './Tutorial'

const TILTS = [-2.2, 1.6, -1.2, 2.2, -1.8]

/** Two-step button: the first press asks "¿…?", the second (within 3 s) does it. */
function useConfirm(action: () => void) {
  const [armed, setArmed] = useState(false)
  const timer = useRef<number>(0)
  useEffect(() => () => window.clearTimeout(timer.current), [])
  const press = () => {
    if (armed) {
      window.clearTimeout(timer.current)
      setArmed(false)
      action()
    } else {
      setArmed(true)
      timer.current = window.setTimeout(() => setArmed(false), 3000)
    }
  }
  return { armed, press }
}

function RunningDock() {
  const pomodoro = useGame((s) => s.state?.pomodoro)
  const cancel = useGame((s) => s.cancel)
  const ms = useRemainingMs()
  const confirm = useConfirm(() => void cancel())
  if (!pomodoro || ms === null) return null
  const paused = pomodoro.status === 'paused'
  return (
    <div className="dock dock--running">
      <div className="readout">
        <div className="readout__time" role="timer" data-testid="timer" data-status={pomodoro.status}>
          {formatClock(ms)}
        </div>
        <div className="readout__what">
          {SEED_NAMES[pomodoro.plant_type] ?? pomodoro.plant_type}
          {pomodoro.tag ? ` · ${pomodoro.tag}` : ''}
          {paused ? ' · en pausa' : ''}
        </div>
      </div>
      <button type="button" className="btn btn--primary" onClick={() => void togglePause()}>
        {paused ? 'Reanudar' : 'Pausar'}
      </button>
      <button type="button" className={`btn ${confirm.armed ? 'btn--danger' : 'btn--quiet'}`} onClick={confirm.press}>
        {confirm.armed ? '¿Cancelar? Se pierde la planta' : 'Cancelar'}
      </button>
    </div>
  )
}

function IdleDock() {
  const state = useGame((s) => s.state)
  const unlockSeed = useGame((s) => s.unlockSeed)
  const selected = useUi((s) => s.selectedSeed)
  const selectSeed = useUi((s) => s.selectSeed)
  const tag = useUi((s) => s.tag)
  const setTag = useUi((s) => s.setTag)
  if (!state) return null
  const free = plantTargetId() !== null
  return (
    <div className="dock dock--idle">
      <div className="packets" role="radiogroup" aria-label="Semillas">
        {state.seeds.map((s, i) => (
          <SeedPacket key={s.key} seed={s} tilt={TILTS[i % TILTS.length]} selected={selected === s.key}
            canAfford={state.player.focus_points >= s.unlock_cost}
            onSelect={() => selectSeed(selected === s.key ? null : s.key)} onUnlock={() => void unlockSeed(s.key)} />
        ))}
      </div>
      <div className="plant">
        {selected === 'oak' && <FlowSlider />}
        <input className="field" list="recent-tags" maxLength={60} value={tag} onChange={(e) => setTag(e.target.value)}
          placeholder="Etiqueta opcional" aria-label="¿En qué vas a trabajar? (opcional)" />
        <datalist id="recent-tags">
          {state.recent_tags.map((t) => <option key={t} value={t} />)}
        </datalist>
        <button type="button" className="btn btn--primary btn--big" disabled={!selected || !free} onClick={() => void plantSelected()}>
          Plantar
        </button>
        {!free && <p className="hint">Retira una planta para poder sembrar.</p>}
      </div>
    </div>
  )
}

/** When the plot in view has nothing to plant, a way to jump to a free one. */
function GoToFree() {
  const select = useUi((s) => s.selectPlot)
  const free = useGame((s) => s.state?.plots.find((p) => p.state === 'empty'))
  if (!free) return null
  return <button type="button" className="btn btn--quiet" onClick={() => select(free.id)}>Sembrar en otra parcela</button>
}

function ReadyDock({ plot }: { plot: PlotState }) {
  const id = plot.id
  const withered = plot.state === 'withered'
  return (
    <div className="dock dock--ready">
      <p className="dock__msg">{withered ? 'Tu planta se marchitó, pero aún puedes cosechar.' : 'Tu planta está lista.'}</p>
      <button type="button" className="btn btn--sun btn--big" data-testid="harvest" onClick={() => void harvestPlot(id)}>
        Cosechar
      </button>
      <GoToFree />
    </div>
  )
}

function ClearDock({ plot }: { plot: PlotState }) {
  const id = plot.id
  const serverTime = useGame((s) => s.state?.server_time)
  const fetchedAt = useGame((s) => s.fetchedAt)
  const clearPlot = useGame((s) => s.clearPlot)
  const confirm = useConfirm(() => void clearPlot(id, true))
  const withered = plot.state === 'withered'
  // Life left as of the last sync (the state refreshes every 30 s).
  const left = serverTime ? lifeLeftMs(plot, serverTime, fetchedAt, fetchedAt) : null
  return (
    <div className="dock dock--clear">
      <p className="dock__msg">
        {withered
          ? 'Se marchitó. Retírala para sembrar de nuevo.'
          : `Cosechada y produciendo 🪙${left !== null ? ` · se marchita en ${formatLife(left)}` : ''}.`}
      </p>
      {withered ? (
        <button type="button" className="btn btn--primary" data-testid="clear" onClick={() => void clearPlot(id, false)}>
          Retirar planta
        </button>
      ) : (
        <button type="button" className={`btn ${confirm.armed ? 'btn--danger' : 'btn--primary'}`} data-testid="clear" onClick={confirm.press}>
          {confirm.armed ? '¿Seguro? Deja de producir' : 'Retirar planta'}
        </button>
      )}
      <GoToFree />
    </div>
  )
}

export function Dock() {
  const state = useGame((s) => s.state)
  const selectedPlotId = useUi((s) => s.selectedPlotId)
  if (!state) return null
  const plot = effectivePlot(state.plots, selectedPlotId)
  const mode = state.pomodoro ? 'running' : plot && needsHarvest(plot) ? 'ready' : plot && canClear(plot) ? 'clear' : 'idle'
  return (
    <div className="dockwrap">
      <Tutorial />
      {mode === 'running' && <RunningDock />}
      {mode === 'ready' && plot && <ReadyDock plot={plot} />}
      {mode === 'clear' && plot && <ClearDock plot={plot} />}
      {mode === 'idle' && <IdleDock />}
    </div>
  )
}
