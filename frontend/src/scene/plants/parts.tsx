import { useFrame } from '@react-three/fiber'
import { useRef } from 'react'
import type { Group } from 'three'
import { palette as P } from '../palette'

/** Flat-shaded material: the low-poly look comes from faceted lighting. */
function M({ color }: { color: string }) {
  return <meshStandardMaterial color={color} flatShading />
}

export interface PartProps {
  /** Fully grown: flowers, fruit and blooms appear only now. */
  mature: boolean
  withered: boolean
}

const tint = (withered: boolean, color: string) => (withered ? P.withered : color)

export function Daisy({ mature, withered }: PartProps) {
  const petals = Array.from({ length: 8 }, (_, i) => (i / 8) * Math.PI * 2)
  return (
    <>
      <mesh castShadow position={[0, 0.2, 0]}>
        <cylinderGeometry args={[0.025, 0.03, 0.4, 5]} />
        <M color={tint(withered, P.stem)} />
      </mesh>
      <mesh castShadow position={[0.1, 0.12, 0]} rotation={[0, 0, -0.9]}>
        <sphereGeometry args={[0.08, 5, 4]} />
        <M color={tint(withered, P.leaf)} />
      </mesh>
      <mesh castShadow position={[-0.08, 0.2, 0.02]} rotation={[0, 0, 0.9]}>
        <sphereGeometry args={[0.06, 5, 4]} />
        <M color={tint(withered, P.leaf)} />
      </mesh>
      {mature ? (
        <>
          {petals.map((a, i) => (
            <mesh key={i} castShadow position={[Math.cos(a) * 0.11, 0.42, Math.sin(a) * 0.11]}>
              <sphereGeometry args={[0.065, 5, 4]} />
              <M color={tint(withered, P.petal)} />
            </mesh>
          ))}
          <mesh castShadow position={[0, 0.43, 0]}>
            <sphereGeometry args={[0.07, 6, 5]} />
            <M color={tint(withered, P.yellow)} />
          </mesh>
        </>
      ) : (
        <mesh castShadow position={[0, 0.42, 0]}>
          <sphereGeometry args={[0.06, 5, 4]} />
          <M color={P.leaf} />
        </mesh>
      )}
    </>
  )
}

const TOMATOES: [number, number, number][] = [[0.15, 0.2, 0.11], [-0.13, 0.28, 0.09], [0.02, 0.17, -0.16], [-0.05, 0.38, -0.05]]

export function Tomato({ mature, withered }: PartProps) {
  return (
    <>
      <mesh castShadow position={[0, 0.2, 0]}>
        <cylinderGeometry args={[0.02, 0.03, 0.4, 5]} />
        <M color={tint(withered, P.stem)} />
      </mesh>
      <mesh castShadow position={[0, 0.3, 0]}>
        <icosahedronGeometry args={[0.24, 0]} />
        <M color={tint(withered, P.leaf)} />
      </mesh>
      <mesh castShadow position={[0, 0.5, 0]}>
        <icosahedronGeometry args={[0.16, 0]} />
        <M color={tint(withered, P.stem)} />
      </mesh>
      {TOMATOES.map((p, i) => (
        <mesh key={i} castShadow position={p} scale={mature ? 1 : 0.001}>
          <sphereGeometry args={[0.075, 6, 5]} />
          <M color={tint(withered, P.red)} />
        </mesh>
      ))}
    </>
  )
}

export function Sunflower({ mature, withered }: PartProps) {
  return (
    <>
      <mesh castShadow position={[0, 0.4, 0]}>
        <cylinderGeometry args={[0.03, 0.04, 0.8, 5]} />
        <M color={tint(withered, P.stem)} />
      </mesh>
      <mesh castShadow position={[0.13, 0.3, 0]} rotation={[0, 0, -0.8]}>
        <sphereGeometry args={[0.12, 5, 4]} />
        <M color={tint(withered, P.leaf)} />
      </mesh>
      <mesh castShadow position={[-0.13, 0.5, 0]} rotation={[0, 0, 0.8]}>
        <sphereGeometry args={[0.12, 5, 4]} />
        <M color={tint(withered, P.leaf)} />
      </mesh>
      {/* Head faces the isometric camera (up-and-right in the view). */}
      <group position={[0, 0.85, 0]} rotation={[0, Math.PI / 4, 0]} scale={mature ? 1 : 0.001}>
        <group rotation={[0.45, 0, 0]}>
          {Array.from({ length: 12 }, (_, i) => (i / 12) * Math.PI * 2).map((a, i) => (
            <mesh key={i} castShadow position={[Math.cos(a) * 0.27, Math.sin(a) * 0.27, 0]} rotation={[0, 0, a]}>
              <coneGeometry args={[0.085, 0.2, 4]} />
              <M color={tint(withered, P.yellow)} />
            </mesh>
          ))}
          <mesh castShadow rotation={[Math.PI / 2, 0, 0]}>
            <cylinderGeometry args={[0.22, 0.22, 0.07, 10]} />
            <M color={tint(withered, P.yellow)} />
          </mesh>
          <mesh castShadow position={[0, 0, 0.05]} rotation={[Math.PI / 2, 0, 0]}>
            <cylinderGeometry args={[0.15, 0.15, 0.07, 8]} />
            <M color={tint(withered, P.brown)} />
          </mesh>
        </group>
      </group>
    </>
  )
}

const APPLES: [number, number, number][] = [[0.3, 0.85, 0.25], [-0.25, 0.95, 0.3], [0.05, 1.1, -0.3], [-0.3, 0.8, -0.2]]

/** Slowly orbiting blue orbs: what makes the Oak "Mágico". */
function MagicOrbs({ y }: { y: number }) {
  const ref = useRef<Group>(null)
  useFrame((_, dt) => {
    if (ref.current) ref.current.rotation.y += dt * 0.8
  })
  return (
    <group ref={ref} position={[0, y, 0]}>
      {[0, 2.1, 4.2].map((a, i) => (
        <mesh key={i} position={[Math.cos(a) * 0.7, Math.sin(a * 2) * 0.12, Math.sin(a) * 0.7]}>
          <octahedronGeometry args={[0.09, 0]} />
          <meshStandardMaterial color={P.magic} emissive={P.magic} emissiveIntensity={1.4} flatShading />
        </mesh>
      ))}
    </group>
  )
}

export function Tree({ mature, withered, oak }: PartProps & { oak: boolean }) {
  const r = oak ? 0.5 : 0.38
  const crown = oak ? P.oakCanopy : P.canopy
  return (
    <>
      <mesh castShadow position={[0, 0.3, 0]}>
        <cylinderGeometry args={[oak ? 0.1 : 0.07, oak ? 0.14 : 0.1, 0.6, 6]} />
        <M color={tint(withered, P.trunk)} />
      </mesh>
      <mesh castShadow position={[0, 0.8, 0]}>
        <icosahedronGeometry args={[r, 1]} />
        <M color={tint(withered, crown)} />
      </mesh>
      <mesh castShadow position={[r * 0.55, 0.65, r * 0.2]}>
        <icosahedronGeometry args={[r * 0.6, 0]} />
        <M color={tint(withered, P.leafDark)} />
      </mesh>
      <mesh castShadow position={[-r * 0.5, 0.95, -r * 0.2]}>
        <icosahedronGeometry args={[r * 0.55, 0]} />
        <M color={tint(withered, P.leafDark)} />
      </mesh>
      {!oak && APPLES.map((p, i) => (
        <mesh key={i} castShadow position={p} scale={mature ? 1 : 0.001}>
          <sphereGeometry args={[0.065, 6, 5]} />
          <M color={tint(withered, P.red)} />
        </mesh>
      ))}
      {oak && mature && !withered && <MagicOrbs y={0.9} />}
    </>
  )
}
