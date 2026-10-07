import { Canvas } from '@react-three/fiber'
import { useEffect, useState } from 'react'

type Health = { status: string; schema_version: number }

function StatusBadge() {
  const [text, setText] = useState('conectando…')
  useEffect(() => {
    fetch('/api/health')
      .then((r) => r.json() as Promise<Health>)
      .then((h) => setText(`backend ${h.status} · esquema BD v${h.schema_version}`))
      .catch(() => setText('backend no disponible'))
  }, [])
  return (
    <div style={{ position: 'fixed', top: 8, left: 8, padding: '4px 10px', borderRadius: 8,
      background: '#fffc', font: '14px system-ui' }}>
      {text}
    </div>
  )
}

export default function App() {
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
      <StatusBadge />
    </>
  )
}
