// Throwaway art-direction comparison (P1-05c). Not part of the game.
import { Canvas } from '@react-three/fiber'
import { useGLTF } from '@react-three/drei'
import { Suspense, useMemo } from 'react'
import * as THREE from 'three'

type Style = 'procedural' | 'kenney' | 'hybrid'
type Kind = 'daisy' | 'tomato' | 'sunflower' | 'apple' | 'oak'
type Stage = 'empty' | 'sprout' | 'growing' | 'mature'

const KINDS: Kind[] = ['daisy', 'tomato', 'sunflower', 'apple', 'oak']
const STAGES: Stage[] = ['mature', 'mature', 'growing', 'mature', 'sprout', 'mature', 'mature', 'growing',
  'empty', 'mature', 'mature', 'mature', 'growing', 'mature', 'empty', 'mature']

const C = { grass: '#8fc65a', grass2: '#82bb50', dirt: '#8a5a3b', stem: '#4f9a3c', leaf: '#5fb046', petal: '#ffffff',
  yellow: '#ffd43b', red: '#e8473f', brown: '#7a4a2a', trunk: '#8b5a3c', canopy: '#4d9f3f', canopy2: '#3f8a35' }

function Mat({ color }: { color: string }) {
  return <meshStandardMaterial color={color} flatShading />
}

function Daisy({ s }: { s: number }) {
  return (
    <group scale={s}>
      <mesh position={[0, 0.2, 0]}><cylinderGeometry args={[0.025, 0.03, 0.4, 5]} /><Mat color={C.stem} /></mesh>
      <mesh position={[0.1, 0.12, 0]} rotation={[0, 0, -0.9]}><sphereGeometry args={[0.08, 5, 4]} /><Mat color={C.leaf} /></mesh>
      {Array.from({ length: 8 }, (_, i) => (
        <mesh key={i} position={[Math.cos((i / 8) * 6.283) * 0.11, 0.42, Math.sin((i / 8) * 6.283) * 0.11]}>
          <sphereGeometry args={[0.065, 5, 4]} /><Mat color={C.petal} />
        </mesh>
      ))}
      <mesh position={[0, 0.43, 0]}><sphereGeometry args={[0.07, 6, 5]} /><Mat color={C.yellow} /></mesh>
    </group>
  )
}

function Tomato({ s, ripe }: { s: number; ripe: boolean }) {
  const fruit: [number, number, number][] = [[0.14, 0.2, 0.1], [-0.12, 0.28, 0.08], [0.02, 0.18, -0.15], [-0.05, 0.36, -0.05]]
  return (
    <group scale={s}>
      <mesh position={[0, 0.2, 0]}><cylinderGeometry args={[0.02, 0.03, 0.4, 5]} /><Mat color={C.stem} /></mesh>
      <mesh position={[0, 0.3, 0]}><icosahedronGeometry args={[0.24, 0]} /><Mat color={C.leaf} /></mesh>
      <mesh position={[0, 0.5, 0]}><icosahedronGeometry args={[0.16, 0]} /><Mat color={C.stem} /></mesh>
      {ripe && fruit.map((p, i) => <mesh key={i} position={p}><sphereGeometry args={[0.07, 6, 5]} /><Mat color={C.red} /></mesh>)}
    </group>
  )
}

function Sunflower({ s, bloom }: { s: number; bloom: boolean }) {
  return (
    <group scale={s}>
      <mesh position={[0, 0.4, 0]}><cylinderGeometry args={[0.03, 0.04, 0.8, 5]} /><Mat color={C.stem} /></mesh>
      <mesh position={[0.13, 0.3, 0]} rotation={[0, 0, -0.8]}><sphereGeometry args={[0.12, 5, 4]} /><Mat color={C.leaf} /></mesh>
      <mesh position={[-0.13, 0.5, 0]} rotation={[0, 0, 0.8]}><sphereGeometry args={[0.12, 5, 4]} /><Mat color={C.leaf} /></mesh>
      {bloom && (
        <group position={[0, 0.85, 0]} rotation={[0.35, 0, 0]}>
          <mesh rotation={[Math.PI / 2, 0, 0]}><cylinderGeometry args={[0.26, 0.26, 0.05, 10]} /><Mat color={C.yellow} /></mesh>
          <mesh position={[0, 0, 0.04]} rotation={[Math.PI / 2, 0, 0]}><cylinderGeometry args={[0.13, 0.13, 0.06, 8]} /><Mat color={C.brown} /></mesh>
        </group>
      )}
    </group>
  )
}

function Tree({ s, big, fruit }: { s: number; big: boolean; fruit: boolean }) {
  const r = big ? 0.5 : 0.38
  return (
    <group scale={s}>
      <mesh position={[0, 0.3, 0]}><cylinderGeometry args={[big ? 0.1 : 0.07, big ? 0.14 : 0.1, 0.6, 6]} /><Mat color={C.trunk} /></mesh>
      <mesh position={[0, 0.8, 0]}><icosahedronGeometry args={[r, 1]} /><Mat color={C.canopy} /></mesh>
      <mesh position={[r * 0.55, 0.65, r * 0.2]}><icosahedronGeometry args={[r * 0.6, 0]} /><Mat color={C.canopy2} /></mesh>
      <mesh position={[-r * 0.5, 0.95, -r * 0.2]}><icosahedronGeometry args={[r * 0.55, 0]} /><Mat color={C.canopy2} /></mesh>
      {fruit && [[0.3, 0.85, 0.25], [-0.25, 0.95, 0.3], [0.05, 1.1, -0.3]].map((p, i) => (
        <mesh key={i} position={p as [number, number, number]}><sphereGeometry args={[0.06, 6, 5]} /><Mat color={C.red} /></mesh>
      ))}
    </group>
  )
}

function ProceduralPlant({ kind, stage }: { kind: Kind; stage: Stage }) {
  if (stage === 'empty') return null
  const s = stage === 'sprout' ? 0.35 : stage === 'growing' ? 0.7 : 1
  const full = stage === 'mature'
  switch (kind) {
    case 'daisy': return <Daisy s={s} />
    case 'tomato': return <Tomato s={s} ripe={full} />
    case 'sunflower': return <Sunflower s={s} bloom={full} />
    case 'apple': return <Tree s={s} big={false} fruit={full} />
    case 'oak': return <Tree s={s} big fruit={false} />
  }
}

const K = (n: string) => `/lab/kenney/${n}.glb`
// Kenney's native palette is mint + orange; we recolour by material name to the warm game palette.
const RECOLOR: Record<string, string> = {
  grass: C.leaf, leafsGreen: C.canopy, woodBark: C.trunk, dirt: C.dirt, dirtDark: '#6f4429',
  colorYellow: C.yellow, colorWhite: '#ffffff', colorRed: C.red, wood: '#a8734a', woodDark: '#85583a',
}
function Model({ name, scale = 1, y = 0, ground }: { name: string; scale?: number; y?: number; ground?: string }) {
  const { scene } = useGLTF(K(name))
  const obj = useMemo(() => {
    const c = scene.clone(true)
    c.traverse((o) => {
      const m = o as THREE.Mesh
      if (!m.isMesh) return
      m.castShadow = true
      m.receiveShadow = true
      const src = m.material as THREE.MeshStandardMaterial
      const color = (ground && src.name === 'grass' ? ground : RECOLOR[src.name]) ?? '#cccccc'
      m.material = new THREE.MeshStandardMaterial({ color, flatShading: true })
    })
    return c
  }, [scene, ground])
  return <primitive object={obj} scale={scale} position={[0, y, 0]} />
}

function KenneyPlant({ kind, stage }: { kind: Kind; stage: Stage }) {
  if (stage === 'empty') return null
  const s = stage === 'sprout' ? 0.35 : stage === 'growing' ? 0.7 : 1
  switch (kind) {
    case 'daisy': return <Model name="flower_yellowA" scale={2.6 * s} />
    case 'tomato': return <Model name="crops_leafsStageB" scale={1.0 * s} />
    case 'sunflower': return <Model name="flower_redA" scale={3.2 * s} />
    case 'apple': return <Model name="tree_default" scale={0.55 * s} />
    case 'oak': return <Model name="tree_fat" scale={0.95 * s} />
  }
}

function HybridPlant({ kind, stage }: { kind: Kind; stage: Stage }) {
  return kind === 'apple' || kind === 'oak' ? <KenneyPlant kind={kind} stage={stage} /> : <ProceduralPlant kind={kind} stage={stage} />
}

function Farm({ style }: { style: Style }) {
  const T = style === 'procedural' ? 1 : 1
  const plant = style === 'procedural' ? ProceduralPlant : style === 'kenney' ? KenneyPlant : HybridPlant
  const Plant = plant
  const ground = Array.from({ length: 36 }, (_, i) => ({ x: (i % 6) - 1, z: Math.floor(i / 6) - 1 }))
  return (
    <>
      {ground.map(({ x, z }) => (
        <group key={`${x},${z}`} position={[x * T, 0, z * T]}>
          {style === 'procedural' ? (
            <mesh receiveShadow position={[0, -0.05, 0]}><boxGeometry args={[T, 0.1, T]} /><Mat color={(x + z) % 2 ? C.grass : C.grass2} /></mesh>
          ) : <Model name="ground_grass" ground={(x + z) % 2 ? C.grass : C.grass2} />}
        </group>
      ))}
      {STAGES.map((stage, i) => {
        const x = i % 4, z = Math.floor(i / 4)
        return (
          <group key={i} position={[x * T, 0, z * T]}>
            {style === 'procedural'
              ? <mesh receiveShadow position={[0, 0.02, 0]}><boxGeometry args={[T * 0.86, 0.08, T * 0.86]} /><Mat color={C.dirt} /></mesh>
              : <Model name="crops_dirtSingle" scale={2.1} />}
            <Plant kind={KINDS[i % 5]} stage={stage} />
          </group>
        )
      })}
    </>
  )
}

export default function ArtLab({ style }: { style: Style }) {
  return (
    <>
      <Canvas orthographic shadows camera={{ position: [11.5, 10, 11.5], zoom: 70, near: 0.1, far: 100 }}
        onCreated={({ camera }) => camera.lookAt(1.5, 0, 1.5)}>
        <color attach="background" args={['#cfe9f5']} />
        <hemisphereLight args={['#ffffff', '#9ac27a', 0.9]} />
        <directionalLight position={[6, 10, 4]} intensity={1.6} castShadow shadow-mapSize={[2048, 2048]}
          shadow-camera-left={-8} shadow-camera-right={8} shadow-camera-top={8} shadow-camera-bottom={-8} />
        <Suspense fallback={null}><Farm style={style} /></Suspense>
      </Canvas>
      <div style={{ position: 'fixed', top: 8, left: 8, padding: '6px 12px', borderRadius: 10, background: '#fffd', font: '600 15px system-ui' }}>
        Opción: {style === 'procedural' ? 'A · 3D procedural' : style === 'kenney' ? 'B · Pack Kenney (CC0)' : 'C · Híbrido'}
      </div>
    </>
  )
}
