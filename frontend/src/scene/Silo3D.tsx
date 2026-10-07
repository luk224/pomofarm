import { useFrame } from '@react-three/fiber'
import { useMemo, useRef } from 'react'
import { BufferGeometry, Group, MeshStandardMaterial } from 'three'
import { canCollect, siloFill } from '../store/economy'
import { useGame } from '../store/game'
import { useSiloAmount } from '../store/hooks'
import { useUi } from '../store/ui'
import { collectSilo } from '../ui/actions'
import { BonusBadge } from './BonusBadge'
import { palette as P } from './palette'
import { mergeParts, type Part } from './plants/geometry'
import { useReducedMotion } from './motion'

/**
 * Where the Silo stands: the left corner of the field (seen from the camera), outside the 4×4 grid. Nothing
 * grows in front of it and it never covers a plot.
 */
const SILO_POSITION: [number, number, number] = [-1.15, 0, 4.15]

const BODY_MAT = new MeshStandardMaterial({ vertexColors: true, flatShading: true })
const GOLD_MAT = new MeshStandardMaterial({ color: P.sparkle, emissive: P.sparkle, emissiveIntensity: 0.9, flatShading: true })
const SLOT_H = 0.5
const FACING = Math.PI / 4 // towards the isometric camera

const cache = new Map<number, BufferGeometry>()

/** The Silo gets taller and gains a band for each upgrade level, so growth is visible in the scene. */
function siloGeometry(level: number): BufferGeometry {
  let g = cache.get(level)
  if (!g) {
    const h = 0.7 + 0.16 * level
    const parts: Part[] = [
      { shape: 'cyl', args: [0.3, 0.34, h, 10], pos: [0, h / 2, 0], color: '#d9bd85' },
      { shape: 'cone', args: [0.42, 0.36, 10], pos: [0, h + 0.18, 0], color: '#a8553b' },
      { shape: 'box', args: [0.18, 0.28, 0.05], pos: [0.2, 0.15, 0.2], rot: [0, FACING, 0], color: '#7a4a2a' },
      { shape: 'box', args: [0.13, SLOT_H + 0.06, 0.04], pos: [0.21, h * 0.58, 0.21], rot: [0, FACING, 0], color: '#3b2a1f' },
    ]
    for (let i = 0; i <= level; i++) parts.push({ shape: 'cyl', args: [0.345, 0.345, 0.05, 10], pos: [0, 0.2 + i * ((h - 0.35) / Math.max(level, 1)) * 0.9, 0], color: '#a8874a' })
    g = mergeParts(parts)
    cache.set(level, g)
  }
  return g
}

/**
 * The Silo as a building of the farm. Its gauge shows how full it is; a gold spark floats above it when there is
 * something to collect, and tapping it empties it (the panel's button and the C key do the same).
 */
export function Silo3D() {
  const level = useGame((s) => s.state?.player.silo_level ?? 0)
  const silo = useGame((s) => s.state?.silo)
  const amount = useSiloAmount()
  const reduce = useReducedMotion()
  const spark = useRef<Group>(null)
  const geometry = useMemo(() => siloGeometry(level), [level])
  const h = 0.7 + 0.16 * level
  const fill = silo ? siloFill(amount, silo.capacity_milli) : 0
  const ready = canCollect(amount)
  const full = !!silo?.full

  useFrame((state) => {
    const s = spark.current
    if (s) {
      s.position.y = h + 0.75 + (reduce ? 0 : Math.sin(state.clock.elapsedTime * 3) * 0.06)
      s.rotation.y = state.clock.elapsedTime * 1.5
    }
  })

  return (
    <group
      name="silo-3d"
      position={SILO_POSITION}
      onClick={(e) => {
        e.stopPropagation()
        if (ready) void collectSilo()
        else useUi.getState().toast('El Silo aún no tiene monedas que recoger.')
      }}
      onPointerOver={() => { document.body.style.cursor = 'pointer' }}
      onPointerOut={() => { document.body.style.cursor = '' }}
    >
      <mesh geometry={geometry} material={BODY_MAT} castShadow receiveShadow dispose={null} />
      {/* gauge: grows from the bottom of the slot */}
      <group position={[0.225, h * 0.58 - SLOT_H / 2, 0.225]} rotation-y={FACING}>
        <group name="silo-fill" scale-y={Math.max(fill, 0.0001)} userData={{ fill }}>
          <mesh position={[0, SLOT_H / 2, 0.02]} material={GOLD_MAT} dispose={null}>
            <boxGeometry args={[0.07, SLOT_H, 0.03]} />
          </mesh>
        </group>
      </group>
      {ready && (
        <group ref={spark} position={[0, h + 0.75, 0]}>
          <mesh scale={full ? 1.5 : 1}>
            <octahedronGeometry args={[0.11, 0]} />
            <meshStandardMaterial color={full ? '#ff9f1c' : P.sparkle} emissive={full ? '#ff9f1c' : P.sparkle} emissiveIntensity={2} flatShading />
          </mesh>
        </group>
      )}
      {full && silo && <BonusBadge text="¡Lleno!" gold y={h + 1.15} size={0.7} />}
    </group>
  )
}
