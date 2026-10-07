import { prepareAlerts } from '../alerts/announce'
import { useGame } from '../store/game'
import { formatCoins } from '../store/economy'
import { canClear, effectivePlot, needsHarvest } from './selection'
import { useUi } from '../store/ui'
import { SEED_NAMES } from './names'

/** Player actions as plain functions so buttons, keyboard shortcuts and 3D clicks behave identically. */

export function firstFreePlotId(): number | null {
  const plot = useGame.getState().state?.plots.find((p) => p.state === 'empty')
  return plot ? plot.id : null
}

/** Where the chosen seed goes: the plot the player is looking at if it is free, otherwise the first free one. */
export function plantTargetId(): number | null {
  const plots = useGame.getState().state?.plots ?? []
  const looking = effectivePlot(plots, useUi.getState().selectedPlotId)
  return looking?.state === 'empty' ? looking.id : firstFreePlotId()
}

export async function plantSelected() {
  const { selectedSeed, tag, selectSeed } = useUi.getState()
  const plotId = plantTargetId()
  if (!selectedSeed || plotId === null || useGame.getState().state?.pomodoro) return
  prepareAlerts() // inside the user's click: unlock audio and ask for notification permission
  await useGame.getState().plant({ plot_id: plotId, plant_type: selectedSeed, tag: tag.trim() || undefined })
  if (useGame.getState().state?.pomodoro) selectSeed(null)
}

export async function togglePause() {
  const p = useGame.getState().state?.pomodoro
  if (!p) return
  await (p.status === 'running' ? useGame.getState().pause() : useGame.getState().resume())
}

/** Harvests and tells the player what they got; the very first time, what 💧 are for. */
export async function harvestPlot(plotId: number) {
  const before = useGame.getState().state?.player.lifetime_focus ?? 0
  const reward = await useGame.getState().harvest(plotId)
  if (reward <= 0) return
  const toast = useUi.getState().toast
  toast(`+${reward} 💧`, 'reward')
  if (before === 0) {
    const next = useGame.getState().state?.seeds.find((s) => !s.unlocked)
    if (next) toast(`Con ${next.unlock_cost} 💧 desbloqueas ${SEED_NAMES[next.key] ?? next.key}.`)
  }
}

/** The plant to harvest: the one in view if it is ready, otherwise the first one that is. */
export function harvestableId(): number | null {
  const plots = useGame.getState().state?.plots ?? []
  const looking = effectivePlot(plots, useUi.getState().selectedPlotId)
  const plot = looking && needsHarvest(looking) ? looking : plots.find(needsHarvest)
  return plot ? plot.id : null
}

export function clearableId(): number | null {
  const plots = useGame.getState().state?.plots ?? []
  const looking = effectivePlot(plots, useUi.getState().selectedPlotId)
  const plot = looking && canClear(looking) ? looking : plots.find(canClear)
  return plot ? plot.id : null
}

/** Empties the Silo into the balance and says how much arrived. */
export async function collectSilo() {
  const got = await useGame.getState().collectSilo()
  if (got > 0) useUi.getState().toast(`+${formatCoins(got)} 🪙`, 'coin')
}

/** Buys the next plot, then looks at it so the dock offers to plant there. */
export async function buyPlot() {
  const had = new Set(useGame.getState().state?.plots.map((p) => p.id))
  await useGame.getState().buyPlot()
  const added = useGame.getState().state?.plots.find((p) => !had.has(p.id))
  if (added) {
    useUi.getState().selectPlot(added.id)
    useUi.getState().toast('Nueva parcela. ¡A sembrar!')
  }
}

export async function upgradeSilo() {
  const before = useGame.getState().state?.silo.capacity_hours ?? 0
  await useGame.getState().upgradeSilo()
  const now = useGame.getState().state?.silo.capacity_hours ?? 0
  if (now > before) useUi.getState().toast(`Silo ampliado: ahora guarda ${now} h de producción.`)
}
