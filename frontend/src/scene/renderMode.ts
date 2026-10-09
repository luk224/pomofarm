import { useEffect, useState } from 'react'

/**
 * Modo Foco (GDD §2.4): while a Pomodoro runs the player is looking at the timer, not at the farm, so the 3D scene needs far
 * less work. It redraws about 20 times a second instead of every screen refresh, refreshes the shadows only twice a second and
 * renders at a pixel ratio of 1; and when the tab is not visible nothing is drawn at all.
 */
export type RenderMode = 'always' | 'demand' | 'never'

export const FOCUS_FPS = 20
export const FOCUS_FRAME_MS = 1000 / FOCUS_FPS
/** In Modo Foco the shadow map is redrawn this often instead of on every frame. */
export const FOCUS_SHADOW_MS = 500

/**
 * How the scene should be drawn right now:
 *  - `never`: the tab is hidden, nothing is drawn;
 *  - `demand`: Modo Foco (a Pomodoro is running and the player left Modo Foco on): a timer asks for ~20 frames a second;
 *  - `always`: every frame the browser offers (up to the screen's refresh rate).
 */
export function renderMode(hidden: boolean, running: boolean, focusEnabled: boolean): RenderMode {
  if (hidden) return 'never'
  return running && focusEnabled ? 'demand' : 'always'
}

/** True while the tab is not visible to the player. */
export function usePageHidden(): boolean {
  const [hidden, setHidden] = useState(() => typeof document !== 'undefined' && document.visibilityState === 'hidden')
  useEffect(() => {
    const on = () => setHidden(document.visibilityState === 'hidden')
    document.addEventListener('visibilitychange', on)
    return () => document.removeEventListener('visibilitychange', on)
  }, [])
  return hidden
}
