import { useGame } from '../store/game'
import { harvestPlot } from '../ui/actions'
import { useRemainingMs } from '../store/hooks'
import { growthFraction } from '../store/time'
import type { PlotState } from '../api/types'
import { Ground } from './Ground'
import { palette as P } from './palette'
import { isPlantKind } from './plants/kinds'
import { PlantView } from './plants/PlantView'
import { RingAnchor } from './RingAnchor'

/** Tilled soil: rim, bed and three furrows. */
function Pad() {
  return (
    <group>
      <mesh receiveShadow position={[0, 0.02, 0]}>
        <boxGeometry args={[0.92, 0.06, 0.92]} />
        <meshStandardMaterial color={P.dirtRim} flatShading />
      </mesh>
      <mesh receiveShadow position={[0, 0.045, 0]}>
        <boxGeometry args={[0.8, 0.06, 0.8]} />
        <meshStandardMaterial color={P.dirt} flatShading />
      </mesh>
      {[-0.22, 0, 0.22].map((z) => (
        <mesh key={z} position={[0, 0.08, z]}>
          <boxGeometry args={[0.7, 0.02, 0.06]} />
          <meshStandardMaterial color={P.dirtDark} flatShading />
        </mesh>
      ))}
    </group>
  )
}

function PlotView({ plot, growth }: { plot: PlotState; growth: number }) {
  const kind = isPlantKind(plot.plant_type) ? plot.plant_type : null
  const mature = plot.state === 'mature' || plot.state === 'withered'
  const ready = plot.state === 'mature' && !plot.harvested
  return (
    <group position={[plot.x, 0, plot.y]}
      onClick={ready ? (e) => { e.stopPropagation(); void harvestPlot(plot.id) } : undefined}
      onPointerOver={ready ? () => { document.body.style.cursor = 'pointer' } : undefined}
      onPointerOut={ready ? () => { document.body.style.cursor = '' } : undefined}>
      <Pad />
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
export function Farm() {
  const plots = useGame((s) => s.state?.plots)
  const pomodoro = useGame((s) => s.state?.pomodoro ?? null)
  const ms = useRemainingMs()
  const growth = pomodoro && ms !== null ? growthFraction(pomodoro.planned_s, ms) : 1

  return (
    <>
      <Ground />
      {plots?.map((p) => (
        <PlotView key={p.id} plot={p} growth={pomodoro && p.id === pomodoro.plot_id ? growth : 1} />
      ))}
      <RingAnchor />
    </>
  )
}
