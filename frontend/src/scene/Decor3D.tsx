import { useLayoutEffect, useMemo, useRef } from 'react'
import { Matrix4, Quaternion, Vector3, type BufferGeometry, type InstancedMesh } from 'three'
import type { DecorItem, DecorKind } from '../api/types'
import { freeCells } from '../store/decorCursor'
import { useGame } from '../store/game'
import { useUi } from '../store/ui'
import { decorateCell, startMovingDecor } from '../ui/actions'
import { BODY_MAT } from './materials'
import { mergeParts, type Part } from './plants/geometry'

const MAX = 64 // the 8×8 decoration area

const PARTS: Record<DecorKind, Part[]> = {
  path: [
    { shape: 'box', args: [0.78, 0.05, 0.78], pos: [0, 0.025, 0], color: '#a8a39a' },
    { shape: 'box', args: [0.34, 0.07, 0.34], pos: [-0.2, 0.035, -0.18], color: '#c4bfb5' },
    { shape: 'box', args: [0.3, 0.07, 0.36], pos: [0.2, 0.035, 0.16], color: '#b8b3a9' },
    { shape: 'box', args: [0.26, 0.06, 0.24], pos: [0.22, 0.03, -0.24], color: '#c9c4ba' },
  ],
  fence: [
    { shape: 'box', args: [0.94, 0.05, 0.06], pos: [0, 0.2, 0], color: '#a8743f' },
    { shape: 'box', args: [0.94, 0.05, 0.06], pos: [0, 0.34, 0], color: '#a8743f' },
    { shape: 'box', args: [0.09, 0.46, 0.09], pos: [-0.4, 0.23, 0], color: '#8a5a3b' },
    { shape: 'box', args: [0.09, 0.46, 0.09], pos: [0, 0.23, 0], color: '#8a5a3b' },
    { shape: 'box', args: [0.09, 0.46, 0.09], pos: [0.4, 0.23, 0], color: '#8a5a3b' },
  ],
  lantern: [
    { shape: 'cyl', args: [0.04, 0.05, 0.62, 6], pos: [0, 0.31, 0], color: '#4a3a30' },
    { shape: 'box', args: [0.2, 0.2, 0.2], pos: [0, 0.72, 0], color: '#ffd45c' },
    { shape: 'cone', args: [0.17, 0.12, 4], pos: [0, 0.88, 0], rot: [0, Math.PI / 4, 0], color: '#4a3a30' },
    { shape: 'cyl', args: [0.12, 0.14, 0.05, 6], pos: [0, 0.03, 0], color: '#6b5b4e' },
  ],
}

const geoCache = new Map<DecorKind, BufferGeometry>()
function geometryOf(kind: DecorKind): BufferGeometry {
  let g = geoCache.get(kind)
  if (!g) {
    g = mergeParts(PARTS[kind])
    geoCache.set(kind, g)
  }
  return g
}

const ID = new Quaternion()
const ONE = new Vector3(1, 1, 1)

function place(mesh: InstancedMesh | null, spots: [number, number, number][]) {
  if (!mesh) return
  const m = new Matrix4()
  spots.forEach(([x, y, z], i) => mesh.setMatrixAt(i, m.compose(new Vector3(x, y, z), ID, ONE)))
  mesh.count = spots.length
  mesh.instanceMatrix.needsUpdate = true
  // Raycasting caches a bounding sphere on first use: refresh it, or pieces that moved stop being clickable.
  mesh.computeBoundingSphere()
}

/** One draw call per kind of piece; a tap maps back to the piece through the instance id. */
function Pieces({ kind, items }: { kind: DecorKind; items: DecorItem[] }) {
  const ref = useRef<InstancedMesh>(null)
  const geometry = useMemo(() => geometryOf(kind), [kind])
  useLayoutEffect(() => {
    place(ref.current, items.map((i) => [i.x, 0.0, i.y]))
  }, [items])
  if (items.length === 0) return null
  return (
    <instancedMesh ref={ref} name={`decor-${kind}`} args={[geometry, BODY_MAT, MAX]} castShadow receiveShadow
      onClick={(e) => {
        if (e.instanceId === undefined) return
        const item = items[e.instanceId]
        if (!item || useUi.getState().placing) return
        e.stopPropagation()
        startMovingDecor(item.id, item.kind)
      }}
      onPointerOver={() => { document.body.style.cursor = 'pointer' }}
      onPointerOut={() => { document.body.style.cursor = '' }} />
  )
}

/** While decorating: every free background cell as a faint square; tapping one places (or moves) the piece there. */
function FreeCells() {
  const ref = useRef<InstancedMesh>(null)
  const decor = useGame((s) => s.state?.decor)
  const cursor = useUi((s) => s.decorCursor)
  const movingId = useUi((s) => s.decorMode?.itemId ?? null)
  const cells = useMemo(() => (decor ? freeCells(decor, movingId) : []), [decor, movingId])
  useLayoutEffect(() => {
    place(ref.current, cells.map(([x, y]) => [x, 0.012, y]))
  }, [cells])
  return (
    <>
    {cursor && (
      <mesh name="decor-cursor" position={[cursor[0], 0.03, cursor[1]]} raycast={() => null}>
        <boxGeometry args={[0.92, 0.04, 0.92]} />
        <meshBasicMaterial color="#ffd45c" transparent opacity={0.85} depthWrite={false} />
      </mesh>
    )}
    <instancedMesh ref={ref} name="decor-free-cells" args={[undefined, undefined, MAX]}
      onClick={(e) => {
        e.stopPropagation()
        const cell = e.instanceId === undefined ? undefined : cells[e.instanceId]
        if (cell) void decorateCell(cell[0], cell[1])
      }}
      onPointerOver={() => { document.body.style.cursor = 'copy' }}
      onPointerOut={() => { document.body.style.cursor = '' }}>
      <boxGeometry args={[0.86, 0.02, 0.86]} />
      <meshBasicMaterial color="#ffffff" transparent opacity={0.28} depthWrite={false} />
    </instancedMesh>
    </>
  )
}

/** Paths, fences and lanterns around the farm. Pure decoration: nothing here touches the economy. */
export function Decor3D() {
  const items = useGame((s) => s.state?.decor.items)
  const decorating = useUi((s) => s.decorMode !== null)
  const byKind = useMemo(() => {
    const m: Record<DecorKind, DecorItem[]> = { path: [], fence: [], lantern: [] }
    for (const i of items ?? []) m[i.kind]?.push(i)
    return m
  }, [items])
  return (
    <>
      <Pieces kind="path" items={byKind.path} />
      <Pieces kind="fence" items={byKind.fence} />
      <Pieces kind="lantern" items={byKind.lantern} />
      {decorating && <FreeCells />}
    </>
  )
}
