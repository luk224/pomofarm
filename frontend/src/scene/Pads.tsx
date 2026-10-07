import { useLayoutEffect, useMemo, useRef } from 'react'
import { Matrix4, Quaternion, Vector3, type InstancedMesh } from 'three'
import type { PlotState } from '../api/types'
import { palette as P } from './palette'

const MAX = 16 // the 4×4 grid
const ID = new Quaternion()
const ONE = new Vector3(1, 1, 1)

function place(mesh: InstancedMesh | null, slots: [number, number, number][]) {
  if (!mesh) return
  const m = new Matrix4()
  slots.forEach(([x, y, z], i) => mesh.setMatrixAt(i, m.compose(new Vector3(x, y, z), ID, ONE)))
  mesh.count = slots.length
  mesh.instanceMatrix.needsUpdate = true
  // Raycasting caches a bounding sphere on first use: refresh it, or pieces that moved stop being clickable.
  mesh.computeBoundingSphere()
}

interface Props {
  plots: PlotState[]
  /** Called with the plot whose soil was clicked. */
  onPick: (plot: PlotState) => void
}

/**
 * All tilled soil of the farm in three draw calls: one InstancedMesh for the rim, one for the bed and one for the
 * furrows (three per plot). Clicks are mapped back to a plot through the instance id.
 */
export function Pads({ plots, onPick }: Props) {
  const rim = useRef<InstancedMesh>(null)
  const bed = useRef<InstancedMesh>(null)
  const furrows = useRef<InstancedMesh>(null)
  const FURROW_Z = useMemo(() => [-0.22, 0, 0.22], [])

  useLayoutEffect(() => {
    place(rim.current, plots.map((p) => [p.x, 0.02, p.y]))
    place(bed.current, plots.map((p) => [p.x, 0.045, p.y]))
    place(furrows.current, plots.flatMap((p) => FURROW_Z.map((dz): [number, number, number] => [p.x, 0.08, p.y + dz])))
  }, [plots, FURROW_Z])

  const pick = (e: { stopPropagation: () => void; instanceId?: number }) => {
    e.stopPropagation()
    const plot = e.instanceId === undefined ? undefined : plots[e.instanceId]
    if (plot) onPick(plot)
  }
  const hover = (on: boolean) => () => {
    document.body.style.cursor = on ? 'pointer' : ''
  }
  return (
    <>
      <instancedMesh ref={rim} args={[undefined, undefined, MAX]} receiveShadow onClick={pick} onPointerOver={hover(true)} onPointerOut={hover(false)}>
        <boxGeometry args={[0.92, 0.06, 0.92]} />
        <meshStandardMaterial color={P.dirtRim} flatShading />
      </instancedMesh>
      <instancedMesh ref={bed} args={[undefined, undefined, MAX]} receiveShadow onClick={pick} onPointerOver={hover(true)} onPointerOut={hover(false)}>
        <boxGeometry args={[0.8, 0.06, 0.8]} />
        <meshStandardMaterial color={P.dirt} flatShading />
      </instancedMesh>
      <instancedMesh ref={furrows} args={[undefined, undefined, MAX * 3]} raycast={() => null}>
        <boxGeometry args={[0.7, 0.02, 0.06]} />
        <meshStandardMaterial color={P.dirtDark} flatShading />
      </instancedMesh>
    </>
  )
}
