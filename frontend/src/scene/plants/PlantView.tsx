import { useFrame } from '@react-three/fiber'
import { useEffect, useRef } from 'react'
import { MathUtils, type Group } from 'three'
import { popScale, useReducedMotion } from '../motion'
import { palette as P } from '../palette'
import { PLANT_HEIGHT, type PlantKind } from './kinds'
import { Daisy, Sunflower, Tomato, Tree } from './parts'

/** Smallest size (sprout) and how fast the plant eases towards its target size. */
const MIN_SCALE = 0.3
const EASE = 6

interface Props {
  kind: PlantKind
  /** 0..1 elapsed fraction of the Pomodoro; the plant grows continuously with it. */
  growth: number
  mature: boolean
  withered?: boolean
  /** Waiting to be harvested: shows a bobbing sparkle. */
  ready?: boolean
}

export function PlantView({ kind, growth, mature, withered = false, ready = false }: Props) {
  const body = useRef<Group>(null)
  const sparkle = useRef<Group>(null)
  const size = useRef(MIN_SCALE) // smoothed uniform size
  const popT = useRef(-1) // seconds since maturing, -1 = idle
  const wasMature = useRef(mature)
  const reduce = useReducedMotion()
  const phase = useRef(0)

  useEffect(() => {
    phase.current = Math.random() * 6.28 // desynchronise the sway of neighbouring plants
  }, [])

  useEffect(() => {
    if (mature && !wasMature.current && !reduce) popT.current = 0
    wasMature.current = mature
  }, [mature, reduce])

  useFrame((state, dt) => {
    const g = body.current
    if (!g) return
    const target = mature ? 1 : MIN_SCALE + (1 - MIN_SCALE) * growth
    size.current = reduce ? target : MathUtils.damp(size.current, target, EASE, dt)
    let sy = 1
    let sxz = 1
    if (popT.current >= 0) {
      popT.current += dt
      const p = popScale(popT.current)
      sy = p.y
      sxz = p.xz
      if (popT.current > 0.8) popT.current = -1
    }
    g.scale.set(size.current * sxz, size.current * sy, size.current * sxz)
    g.rotation.z = reduce ? 0 : Math.sin(state.clock.elapsedTime * 1.3 + phase.current) * (kind === 'apple' || kind === 'oak' ? 0.012 : 0.04)
    const s = sparkle.current
    if (s) {
      s.position.y = PLANT_HEIGHT[kind] + 0.25 + (reduce ? 0 : Math.sin(state.clock.elapsedTime * 3) * 0.06)
      s.rotation.y += dt * 1.5
    }
  })

  const props = { mature, withered }
  return (
    <group name={`plant-${kind}`}>
      <group ref={body} scale={MIN_SCALE}>
        {kind === 'daisy' && <Daisy {...props} />}
        {kind === 'tomato' && <Tomato {...props} />}
        {kind === 'sunflower' && <Sunflower {...props} />}
        {kind === 'apple' && <Tree {...props} oak={false} />}
        {kind === 'oak' && <Tree {...props} oak />}
      </group>
      {ready && !withered && (
        <group ref={sparkle} position={[0, PLANT_HEIGHT[kind] + 0.25, 0]}>
          <mesh>
            <octahedronGeometry args={[0.13, 0]} />
            <meshStandardMaterial color={P.sparkle} emissive={P.sparkle} emissiveIntensity={2} flatShading />
          </mesh>
        </group>
      )}
    </group>
  )
}
