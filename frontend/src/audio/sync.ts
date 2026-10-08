import { usePrefs } from '../store/prefs'
import { setAmbient } from './ambient'
import { LAYERS, onUnlock, setLayerVolume } from './engine'

/** Applies one volume set to the engine. */
export function applyVolumes(v: Record<(typeof LAYERS)[number], number>): void {
  for (const l of LAYERS) setLayerVolume(l, v[l])
}

/**
 * Keeps the audio engine in step with the saved preferences: volumes follow the sliders at once, and the chosen
 * ambient sound starts as soon as the browser lets audio play (the first click or key press) and on every change.
 * Returns a function that stops listening.
 */
export function startAudioSync(): () => void {
  const apply = () => {
    const p = usePrefs.getState()
    applyVolumes(p.volumes)
    setAmbient(p.ambient)
  }
  apply()
  const offStore = usePrefs.subscribe(apply)
  const offUnlock = onUnlock(apply)
  return () => {
    offStore()
    offUnlock()
  }
}
