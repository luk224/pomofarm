import { useMemo } from 'react'
import type { BufferGeometry } from 'three'
import { hiveArea } from '../store/automation'
import { useGame } from '../store/game'
import { useUi } from '../store/ui'
import { effectivePlot } from '../ui/selection'
import { startPlacing } from '../ui/actions'
import { BODY_MAT } from './materials'
import { mergeParts, type Part } from './plants/geometry'

/** A hive stands on the screen-left corner of its plot's cell, so it never covers the plant. */
const HIVE_OFFSET: [number, number, number] = [-0.34, 0, 0.34]
const DOG_POSITION: [number, number, number] = [-0.3, 0, 4.2] // beside the Silo (-1.15, 0, 4.15)

let hiveGeo: BufferGeometry | null = null
function hiveGeometry(): BufferGeometry {
  if (!hiveGeo) {
    const parts: Part[] = [
      { shape: 'box', args: [0.3, 0.05, 0.3], pos: [0, 0.025, 0], color: '#7a4a2a' },
      { shape: 'cyl', args: [0.12, 0.14, 0.12, 8], pos: [0, 0.11, 0], color: '#e8b84a' },
      { shape: 'cyl', args: [0.11, 0.12, 0.12, 8], pos: [0, 0.23, 0], color: '#d9a233' },
      { shape: 'cyl', args: [0.1, 0.11, 0.12, 8], pos: [0, 0.35, 0], color: '#e8b84a' },
      { shape: 'cone', args: [0.14, 0.12, 8], pos: [0, 0.47, 0], color: '#a8553b' },
      { shape: 'box', args: [0.06, 0.03, 0.02], pos: [0, 0.1, 0.13], color: '#3b2a1f' },
    ]
    hiveGeo = mergeParts(parts)
  }
  return hiveGeo
}

const dogGeos = new Map<boolean, BufferGeometry>()
function dogGeometry(hat: boolean): BufferGeometry {
  let dogGeo = dogGeos.get(hat)
  if (!dogGeo) {
    const parts: Part[] = [
      { shape: 'box', args: [0.34, 0.2, 0.18], pos: [0, 0.22, 0], color: '#c9894a' },
      { shape: 'box', args: [0.18, 0.18, 0.18], pos: [0.2, 0.34, 0], color: '#d99a5a' },
      { shape: 'box', args: [0.08, 0.06, 0.1], pos: [0.3, 0.3, 0], color: '#3b2a1f' },
      { shape: 'box', args: [0.05, 0.12, 0.05], pos: [0.18, 0.46, 0.08], color: '#8a5a3b' },
      { shape: 'box', args: [0.05, 0.12, 0.05], pos: [0.18, 0.46, -0.08], color: '#8a5a3b' },
      { shape: 'box', args: [0.05, 0.14, 0.05], pos: [-0.12, 0.07, 0.06], color: '#c9894a' },
      { shape: 'box', args: [0.05, 0.14, 0.05], pos: [-0.12, 0.07, -0.06], color: '#c9894a' },
      { shape: 'box', args: [0.05, 0.14, 0.05], pos: [0.12, 0.07, 0.06], color: '#c9894a' },
      { shape: 'box', args: [0.05, 0.14, 0.05], pos: [0.12, 0.07, -0.06], color: '#c9894a' },
      { shape: 'box', args: [0.16, 0.05, 0.05], pos: [-0.24, 0.34, 0], rot: [0, 0, 0.6], color: '#c9894a' },
    ]
    if (hat) {
      parts.push({ shape: 'cyl', args: [0.17, 0.17, 0.02, 10], pos: [0.2, 0.45, 0], color: '#e6c36a' })
      parts.push({ shape: 'cyl', args: [0.08, 0.1, 0.09, 10], pos: [0.2, 0.5, 0], color: '#d9ad4a' })
      parts.push({ shape: 'cyl', args: [0.1, 0.1, 0.02, 10], pos: [0.2, 0.52, 0], color: '#a8553b' })
    }
    dogGeo = mergeParts(parts)
    dogGeos.set(hat, dogGeo)
  }
  return dogGeo
}

/** Translucent amber squares over the plots a hive would cover from the plot in view. */
function CoveragePreview() {
  const plots = useGame((s) => s.state?.plots)
  const selectedPlotId = useUi((s) => s.selectedPlotId)
  const placing = useUi((s) => s.placing)
  if (!placing || !plots) return null
  const target = effectivePlot(plots, selectedPlotId)
  if (!target) return null
  return (
    <>
      {hiveArea(plots, target.x, target.y).map((p) => (
        <mesh key={p.id} position={[p.x, 0.16, p.y]} rotation-x={-Math.PI / 2} raycast={() => null}>
          <planeGeometry args={[0.9, 0.9]} />
          <meshBasicMaterial color="#ff9f1c" transparent opacity={0.6} depthWrite={false} />
        </mesh>
      ))}
      <group position={[target.x + HIVE_OFFSET[0], 0.06, target.y + HIVE_OFFSET[2]]} raycast={() => null}>
        <mesh geometry={hiveGeometry()} material={BODY_MAT} dispose={null} raycast={() => null} />
      </group>
    </>
  )
}

/** Hives, the Dog and the placement preview. Hives are tappable to be moved. */
export function Automation3D() {
  const hives = useGame((s) => s.state?.automation.bees.hives)
  const dog = useGame((s) => s.state?.automation.dog.owned)
  const hat = useGame((s) => s.state?.decor.hat.owned ?? false)
  const geometry = useMemo(() => hiveGeometry(), [])
  return (
    <>
      {hives?.map((h) => (
        <group key={h.id} name={`hive-${h.id}`} position={[h.x + HIVE_OFFSET[0], 0.06, h.y + HIVE_OFFSET[2]]}
          onClick={(e) => {
            e.stopPropagation()
            startPlacing(h.id)
          }}
          onPointerOver={() => { document.body.style.cursor = 'pointer' }}
          onPointerOut={() => { document.body.style.cursor = '' }}>
          <mesh geometry={geometry} material={BODY_MAT} castShadow dispose={null} />
        </group>
      ))}
      {dog && (
        <group name="dog-3d" position={DOG_POSITION} rotation-y={-Math.PI / 4}>
          <mesh geometry={dogGeometry(hat)} material={BODY_MAT} castShadow dispose={null} />
        </group>
      )}
      <CoveragePreview />
    </>
  )
}
