import { stopMusic } from '../audio/lofi'
import { setMasterMuted } from '../audio/engine'
import { useMusic } from '../store/music'
import { usePrefs } from '../store/prefs'

/**
 * Applies the accessibility preferences (GDD §2.4) to the page: the CSS reads `data-reduce-motion` and `data-palette` from
 * <html>, and "Sin audio" silences the audio engine and stops the Lofi player (no sound can come out of it either).
 * Returns a function that stops listening.
 */
export function startA11ySync(): () => void {
  const root = document.documentElement
  const apply = () => {
    const p = usePrefs.getState()
    root.dataset.reduceMotion = String(p.reduceMotion)
    root.dataset.palette = p.palette
    setMasterMuted(p.muted)
    if (p.muted && useMusic.getState().status !== 'idle') stopMusic()
  }
  apply()
  return usePrefs.subscribe(apply)
}
