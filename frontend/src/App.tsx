import { Canvas } from '@react-three/fiber'

export default function App() {
  return (
    <Canvas orthographic camera={{ position: [10, 10, 10], zoom: 60 }}>
      <ambientLight intensity={0.8} />
      <directionalLight position={[5, 10, 5]} />
      <mesh>
        <boxGeometry />
        <meshStandardMaterial color="#6ab04c" />
      </mesh>
    </Canvas>
  )
}
