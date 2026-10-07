import { Canvas } from '@react-three/fiber'
import { useGameSync } from './store/hooks'
import { DevPanel } from './ui/DevPanel'
import { Hud } from './ui/Hud'

export default function App() {
  useGameSync()
  return (
    <>
      <Canvas orthographic camera={{ position: [10, 10, 10], zoom: 60 }}>
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
