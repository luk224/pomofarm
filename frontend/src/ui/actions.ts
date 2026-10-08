import { prepareAlerts } from '../alerts/announce'
import { useGame } from '../store/game'
import { formatCoins } from '../store/economy'
import { canClear, effectivePlot, needsHarvest } from './selection'
import { useUi } from '../store/ui'
import { SEED_NAMES } from './names'
import { playEffect } from '../audio/effects'
import type { DecorKind } from '../api/types'

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
  const { selectedSeed, tag, selectSeed, flowMinutes } = useUi.getState()
  const plotId = plantTargetId()
  if (!selectedSeed || plotId === null || useGame.getState().state?.pomodoro || useUi.getState().placing || useUi.getState().decorMode) return
  prepareAlerts() // inside the user's click: unlock audio and ask for notification permission
  await useGame.getState().plant({
    plot_id: plotId,
    plant_type: selectedSeed,
    tag: tag.trim() || undefined,
    duration_min: selectedSeed === 'oak' ? flowMinutes : undefined, // only the Oak has Flow mode
  })
  if (useGame.getState().state?.pomodoro) {
    selectSeed(null)
    playEffect('dig')
  }
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
  playEffect('harvest')
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
  if (got > 0) {
    playEffect('coin')
    useUi.getState().toast(`+${formatCoins(got)} 🪙`, 'coin')
  }
}

/** Buys the next plot, then looks at it so the dock offers to plant there. */
export async function buyPlot() {
  const had = new Set(useGame.getState().state?.plots.map((p) => p.id))
  await useGame.getState().buyPlot()
  const added = useGame.getState().state?.plots.find((p) => !had.has(p.id))
  if (added) {
    playEffect('buy')
    useUi.getState().selectPlot(added.id)
    useUi.getState().toast('Nueva parcela. ¡A sembrar!')
  }
}

export async function upgradeSilo() {
  const before = useGame.getState().state?.silo.capacity_hours ?? 0
  await useGame.getState().upgradeSilo()
  const now = useGame.getState().state?.silo.capacity_hours ?? 0
  if (now > before) playEffect('buy')
  if (now > before) useUi.getState().toast(`Silo ampliado: ahora guarda ${now} h de producción.`)
}

/** Unlocks Bees or the Dog with 💧. */
export async function unlockAnimal(key: 'bees' | 'dog') {
  await useGame.getState().unlockAnimal(key)
  if (useGame.getState().state?.automation[key === 'bees' ? 'bees' : 'dog'].unlocked) {
    playEffect('buy')
    useUi.getState().toast(key === 'bees' ? 'Abejas desbloqueadas. Ya puedes comprar colmenas.' : 'Perro desbloqueado. Ya puedes comprarlo.')
  }
}

export async function buyDog() {
  await useGame.getState().buyDog()
  if (useGame.getState().state?.automation.dog.owned) playEffect('buy')
  if (useGame.getState().state?.automation.dog.owned) useUi.getState().toast('El Perro vigila el Silo: lo recoge solo y guarda 12 h más.')
}

/** Starts choosing where a new hive goes (hiveId null) or where an existing one moves to. */
export function startPlacing(hiveId: number | null) {
  useUi.getState().setPlacing({ hiveId })
}

export function cancelPlacing() {
  useUi.getState().setPlacing(null)
}

/** Confirms the placement on the plot in view: buys the hive there, or moves it for free. */
export async function confirmPlacing() {
  const { placing, selectedPlotId } = useUi.getState()
  const plots = useGame.getState().state?.plots ?? []
  const target = effectivePlot(plots, selectedPlotId)
  if (!placing || !target) return
  const before = useGame.getState().state?.automation.bees.hives.length ?? 0
  if (placing.hiveId === null) await useGame.getState().buyHive(target.id)
  else await useGame.getState().moveHive(placing.hiveId, target.id)
  const after = useGame.getState().state?.automation.bees.hives
  const hive = after?.find((h) => (placing.hiveId === null ? true : h.id === placing.hiveId) && h.plot_id === target.id)
  if (hive && (placing.hiveId !== null || (after?.length ?? 0) > before)) {
    useUi.getState().setPlacing(null)
    playEffect('place')
    useUi.getState().toast(placing.hiveId === null ? 'Colmena colocada: +25% a su alrededor.' : 'Colmena movida.')
  }
}

// ---------- decoration ----------

export const DECOR_NAMES: Record<string, string> = { path: 'Camino de piedra', fence: 'Valla', lantern: 'Farolillo' }

export function startDecor(piece: DecorKind) {
  useUi.getState().setDecorMode({ piece, itemId: null })
}

export function startMovingDecor(itemId: number, piece: DecorKind) {
  useUi.getState().setDecorMode({ piece, itemId })
}

export function stopDecor() {
  useUi.getState().setDecorMode(null)
}

/** A tap on a background cell while decorating: places the chosen piece there, or moves the piece being moved. */
export async function decorateCell(x: number, y: number) {
  const mode = useUi.getState().decorMode
  if (!mode) return
  const game = useGame.getState()
  const had = useGame.getState().state?.decor.items.length ?? 0
  if (mode.itemId === null) {
    await game.buyDecor(mode.piece, x, y)
    if ((useGame.getState().state?.decor.items.length ?? 0) > had) playEffect('place')
  } else {
    await game.moveDecor(mode.itemId, x, y)
    if (!useGame.getState().error) {
      playEffect('place')
      useUi.getState().setDecorMode(null)
    }
  }
}

/** Takes the piece being moved off the farm (free, no refund). */
export async function removeDecorPiece() {
  const mode = useUi.getState().decorMode
  if (!mode || mode.itemId === null) return
  await useGame.getState().removeDecor(mode.itemId)
  if (!useGame.getState().error) {
    useUi.getState().setDecorMode(null)
    useUi.getState().toast('Pieza retirada.')
  }
}

export async function buyHat() {
  await useGame.getState().buyHat()
  if (useGame.getState().state?.decor.hat.owned) playEffect('buy')
  if (useGame.getState().state?.decor.hat.owned) useUi.getState().toast('El Perro luce su sombrero de paja.')
}
