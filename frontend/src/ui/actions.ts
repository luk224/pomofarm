import { prepareAlerts } from '../alerts/announce'
import { useGame } from '../store/game'
import { formatCoins } from '../store/economy'
import { useUi } from '../store/ui'
import { SEED_NAMES } from './names'

/** Player actions as plain functions so buttons, keyboard shortcuts and 3D clicks behave identically. */

export function firstFreePlotId(): number | null {
  const plot = useGame.getState().state?.plots.find((p) => p.state === 'empty' || p.state === 'withered')
  return plot ? plot.id : null
}

export async function plantSelected() {
  const { selectedSeed, tag, selectSeed } = useUi.getState()
  const plotId = firstFreePlotId()
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

export function harvestableId(): number | null {
  const plot = useGame.getState().state?.plots.find((p) => p.state === 'mature' && !p.harvested)
  return plot ? plot.id : null
}

export function clearableId(): number | null {
  const plot = useGame.getState().state?.plots.find((p) => (p.state === 'mature' && p.harvested) || p.state === 'withered')
  return plot ? plot.id : null
}

/** Empties the Silo into the balance and says how much arrived. */
export async function collectSilo() {
  const got = await useGame.getState().collectSilo()
  if (got > 0) useUi.getState().toast(`+${formatCoins(got)} 🪙`, 'coin')
}
