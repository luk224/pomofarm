import { useFrame } from '@react-three/fiber'
import { useEffect, useRef } from 'react'
import { MathUtils, MeshStandardMaterial, type Group } from 'three'
import { popScale, useReducedMotion } from '../motion'
import { palette as P } from '../palette'
import { PLANT_HEIGHT, type PlantKind } from './kinds'
import { orbsGeometry, plantGeometry } from './geometry'

// One material for every plant: colours live in the geometry (vertex colours), so all plants batch cheaply.
const PLANT_MATERIAL = new MeshStandardMaterial({ vertexColors: true, flatShading: true })
const ORB_MATERIAL = new MeshStandardMaterial({ color: P.magic, emissive: P.magic, emissiveIntensity: 1.4, flatShading: true, vertexColors: true })

/** Slowly orbiting blue orbs: what makes the Oak "Mágico". */
function MagicOrbs() {
  const ref = useRef<Group>(null)
  useFrame((_, dt) => {
    if (ref.current) ref.current.rotation.y += dt * 0.8
  })
  return (
    <group ref={ref} position={[0, 0.9, 0]}>
      <mesh geometry={orbsGeometry()} material={ORB_MATERIAL} dispose={null} />
    </group>
  )
}

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
    // A withered plant is not only a different colour: it is also shorter and leans over, so it reads without colour vision
    const droop = withered ? 0.82 : 1
    g.scale.set(size.current * sxz, size.current * sy * droop, size.current * sxz)
    g.rotation.z = withered ? 0.22 : reduce ? 0 : Math.sin(state.clock.elapsedTime * 1.3 + phase.current) * (kind === 'apple' || kind === 'oak' ? 0.012 : 0.04)
    const s = sparkle.current
    if (s) {
      s.position.y = PLANT_HEIGHT[kind] + 0.25 + (reduce ? 0 : Math.sin(state.clock.elapsedTime * 3) * 0.06)
      s.rotation.y += dt * 1.5
    }
  })

  return (
    <group name={`plant-${kind}`}>
      <group ref={body} scale={MIN_SCALE}>
        <mesh geometry={plantGeometry(kind, mature, withered)} material={PLANT_MATERIAL} castShadow dispose={null} />
        {kind === 'oak' && mature && !withered && <MagicOrbs />}
      </group>
      {ready && (
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
