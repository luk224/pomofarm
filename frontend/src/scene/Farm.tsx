import { useGame } from '../store/game'
import { useUi } from '../store/ui'
import { harvestPlot } from '../ui/actions'
import { useRemainingMs } from '../store/hooks'
import { growthFraction } from '../store/time'
import type { PlotState } from '../api/types'
import { bonusLabel } from '../store/bonus'
import { effectivePlot } from '../ui/selection'
import { Automation3D } from './Automation3D'
import { BonusBadge } from './BonusBadge'
import { Ground } from './Ground'
import { Pads } from './Pads'
import { Silo3D } from './Silo3D'
import { isPlantKind } from './plants/kinds'
import { PlantView } from './plants/PlantView'
import { RingAnchor } from './RingAnchor'

/** Square outline around the plot the dock is talking about. */
function SelectionMarker() {
  return (
    <group rotation-x={-Math.PI / 2} position={[0, 0.1, 0]}>
      <mesh rotation-z={Math.PI / 4}>
        <ringGeometry args={[0.67, 0.73, 4]} />
        <meshBasicMaterial color="#ffffff" transparent opacity={0.95} />
      </mesh>
    </group>
  )
}

function PlotView({ plot, growth, selected }: { plot: PlotState; growth: number; selected: boolean }) {
  const kind = isPlantKind(plot.plant_type) ? plot.plant_type : null
  const mature = plot.state === 'mature' || plot.state === 'withered'
  const ready = (plot.state === 'mature' || plot.state === 'withered') && !plot.harvested
  const label = bonusLabel(plot.bonus)
  return (
    <group position={[plot.x, 0, plot.y]}
      onClick={(e) => {
        e.stopPropagation()
        useUi.getState().selectPlot(plot.id)
        if (ready) void harvestPlot(plot.id)
      }}
      onPointerOver={() => { document.body.style.cursor = 'pointer' }}
      onPointerOut={() => { document.body.style.cursor = '' }}>
      {selected && <SelectionMarker />}
      {kind && label && plot.state === 'mature' && (
        // Small tag on the front corner of the soil: always readable and never on top of a neighbour's tree.
        <BonusBadge text={label} gold={!!plot.bonus?.garden} position={[0.3, 0.2, 0.3]} size={0.42} />
      )}
      {kind && plot.state !== 'empty' && (
        <group position={[0, 0.08, 0]}>
          <PlantView kind={kind} growth={mature ? 1 : growth} mature={mature} withered={plot.state === 'withered'}
            ready={ready} />
        </group>
      )}
    </group>
  )
}

/** Draws the farm from the server state. The client never decides game outcomes. */
function pickPlot(plot: PlotState) {
  useUi.getState().selectPlot(plot.id)
  if ((plot.state === 'mature' || plot.state === 'withered') && !plot.harvested) void harvestPlot(plot.id)
}

export function Farm() {
  const plots = useGame((s) => s.state?.plots)
  const pomodoro = useGame((s) => s.state?.pomodoro ?? null)
  const ms = useRemainingMs()
  const selectedPlotId = useUi((s) => s.selectedPlotId)
  const looking = plots ? effectivePlot(plots, selectedPlotId)?.id : undefined
  const growth = pomodoro && ms !== null ? growthFraction(pomodoro.planned_s, ms) : 1

  return (
    <>
      <Ground />
      <Silo3D />
      <Automation3D />
      {plots && <Pads plots={plots} onPick={pickPlot} />}
      {plots?.map((p) => (
        <PlotView key={p.id} plot={p} growth={pomodoro && p.id === pomodoro.plot_id ? growth : 1} selected={p.id === looking && !pomodoro && (plots?.length ?? 0) > 1} />
      ))}
      <RingAnchor />
    </>
  )
}
