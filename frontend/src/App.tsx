import { Canvas } from '@react-three/fiber'
import { lazy, Suspense } from 'react'
import { useGameSync } from './store/hooks'
import { DevPanel } from './ui/DevPanel'
import { IsoCamera } from './scene/IsoCamera'
import { Hud } from './ui/Hud'

const ArtLab = lazy(() => import('./lab/ArtLab'))

export default function App() {
  useGameSync()
  const lab = new URLSearchParams(location.search).get('lab')
  if (import.meta.env.DEV && (lab === 'procedural' || lab === 'kenney' || lab === 'hybrid')) {
    return <Suspense fallback={null}><ArtLab style={lab} /></Suspense>
  }
  return (
    <>
      <Canvas orthographic>
        <IsoCamera />
        <ambientLight intensity={0.8} />
        <directionalLight position={[5, 10, 5]} />
        <mesh>
          <boxGeometry />
          <meshStandardMaterial color="#6ab04c" />
        </mesh>
      </Canvas>
      <Hud />
      {import.meta.env.DEV && <DevPanel />}
    </>
  )
}
