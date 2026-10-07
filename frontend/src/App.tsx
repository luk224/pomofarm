import { Canvas } from '@react-three/fiber'
import { useEffect, useState } from 'react'
import { unlockAudioOnFirstGesture } from './audio/bowl'
import { Farm } from './scene/Farm'
import { Gallery } from './scene/Gallery'
import { IsoCamera } from './scene/IsoCamera'
import { palette } from './scene/palette'
import { useCompletionAlerts } from './alerts/hooks'
import { useGameSync } from './store/hooks'
import { Dock } from './ui/Dock'
import { PlotAnnouncer } from './ui/PlotAnnouncer'
import { SiloPanel } from './ui/SiloPanel'
import { Toasts } from './ui/Toasts'
import { TopBar } from './ui/TopBar'
import { TimerRing } from './ui/TimerRing'
import { useShortcuts } from './ui/useShortcuts'

export default function App() {
  useGameSync()
  useShortcuts()
  useCompletionAlerts()
  useEffect(() => unlockAudioOnFirstGesture(), [])
  // Fit the 4×4 farm and the Silo on narrow screens: roughly 8.4 world units across (the 4×4 field plus the Silo).
  const [fitZoom] = useState(() => Math.min(95, Math.max(34, window.innerWidth / 8.4)))
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
        <IsoCamera center={gallery ? [3.4, 2.55] : [1.5, 1.5]} initialZoom={gallery ? 62 : fitZoom} />
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
      {!gallery && <SiloPanel />}
      {!gallery && <TimerRing />}
      {!gallery && <Dock />}
      <Toasts />
      <PlotAnnouncer />
    </>
  )
}
