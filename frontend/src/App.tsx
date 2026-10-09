import { Canvas } from '@react-three/fiber'
import { useEffect, useState } from 'react'
import { unlockAudioOnFirstGesture } from './audio/bowl'
import { startAudioSync } from './audio/sync'
import { startA11ySync } from './ui/a11y'
import { Farm } from './scene/Farm'
import { Gallery } from './scene/Gallery'
import { IsoCamera } from './scene/IsoCamera'
import { biomePalette } from './scene/biomes'
import { FocusLoop } from './scene/FocusLoop'
import { renderMode, usePageHidden } from './scene/renderMode'
import { usePrefs } from './store/prefs'
import { useCompletionAlerts } from './alerts/hooks'
import { useFarmAttention } from './alerts/attention'
import { useGame } from './store/game'
import { useGameSync } from './store/hooks'
import { Dock } from './ui/Dock'
import { DecorBar } from './ui/DecorBar'
import { LofiPlayer } from './ui/LofiPlayer'
import { PlacingBar } from './ui/PlacingBar'
import { PlotAnnouncer } from './ui/PlotAnnouncer'
import { SiloPanel } from './ui/SiloPanel'
import { AlertFlash } from './ui/AlertFlash'
import { Toasts } from './ui/Toasts'
import { TopBar } from './ui/TopBar'
import { TimerRing } from './ui/TimerRing'
import { useShortcuts } from './ui/useShortcuts'

export default function App() {
  useGameSync()
  useShortcuts()
  useCompletionAlerts()
  useFarmAttention()
  useEffect(() => unlockAudioOnFirstGesture(), [])
  useEffect(() => startAudioSync(), [])
  useEffect(() => startA11ySync(), [])
  // Fit the 4×4 farm and the Silo on narrow screens: roughly 8.4 world units across (the 4×4 field plus the Silo).
  const [fitZoom] = useState(() => Math.min(95, Math.max(34, window.innerWidth / 8.4)))
  const sky = biomePalette(useGame((s) => s.state?.player.biome)).sky
  const running = useGame((s) => s.state?.pomodoro?.status === 'running')
  const focusEnabled = usePrefs((s) => s.focusMode)
  const mode = renderMode(usePageHidden(), running, focusEnabled)
  const gallery = import.meta.env.DEV && new URLSearchParams(location.search).get('lab') === 'plants'
  return (
    <>
      <Canvas
        orthographic
        shadows="percentage"
        frameloop={mode}
        dpr={mode === 'demand' ? 1 : [1, 2]}
        onCreated={(state) => {
          if (import.meta.env.DEV) (window as unknown as { __three?: unknown }).__three = state
        }}
      >
        <FocusLoop mode={mode} />
        <color attach="background" args={[sky]} />
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
      {!gallery && <PlacingBar />}
      {!gallery && <DecorBar />}
      {!gallery && <LofiPlayer />}
      <Toasts />
      <AlertFlash />
      <PlotAnnouncer />
    </>
  )
}
