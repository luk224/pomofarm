import { Canvas } from '@react-three/fiber'
import { Farm } from './scene/Farm'
import { Gallery } from './scene/Gallery'
import { IsoCamera } from './scene/IsoCamera'
import { palette } from './scene/palette'
import { useGameSync } from './store/hooks'
import { Dock } from './ui/Dock'
import { Toasts } from './ui/Toasts'
import { TopBar } from './ui/TopBar'
import { TimerRing } from './ui/TimerRing'
import { useShortcuts } from './ui/useShortcuts'

export default function App() {
  useGameSync()
  useShortcuts()
  const gallery = import.meta.env.DEV && new URLSearchParams(location.search).get('lab') === 'plants'
  return (
    <>
      <Canvas
        orthographic
        shadows="percentage"
        onCreated={(state) => {
          if (import.meta.env.DEV) (window as unknown as { __three?: unknown }).__three = state
        }}
      >
        <color attach="background" args={[palette.sky]} />
        <IsoCamera center={gallery ? [3.4, 2.55] : [1.5, 1.5]} initialZoom={gallery ? 62 : 95} />
        <hemisphereLight args={['#ffffff', '#9ac27a', 0.9]} />
        <directionalLight
          position={[6, 10, 4]}
          intensity={1.6}
          castShadow
          shadow-mapSize={[1024, 1024]}
          shadow-camera-left={-8}
          shadow-camera-right={8}
          shadow-camera-top={8}
          shadow-camera-bottom={-8}
          shadow-bias={-0.0004}
          shadow-normalBias={0.03}
        />
        {gallery ? <Gallery /> : <Farm />}
      </Canvas>
      <TopBar />
      {!gallery && <TimerRing />}
      {!gallery && <Dock />}
      <Toasts />
    </>
  )
}
