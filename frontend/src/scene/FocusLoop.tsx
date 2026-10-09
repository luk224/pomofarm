import { useThree } from '@react-three/fiber'
import { useEffect } from 'react'
import { FOCUS_FRAME_MS, FOCUS_SHADOW_MS, type RenderMode } from './renderMode'

/**
 * Drives the scene while it is not on the browser's own "always" loop. In Modo Foco (`demand`) it asks for a frame every
 * 50 ms and redraws the shadow map only every 500 ms; leaving Modo Foco puts the shadows back to updating every frame.
 */
export function FocusLoop({ mode }: { mode: RenderMode }) {
  const invalidate = useThree((s) => s.invalidate)
  const get = useThree((s) => s.get)

  useEffect(() => {
    const { gl } = get() // read at the moment of use: the renderer is configured here, not owned by this component
    if (mode !== 'demand') {
      gl.shadowMap.autoUpdate = true
      gl.shadowMap.needsUpdate = true
      return
    }
    gl.shadowMap.autoUpdate = false
    gl.shadowMap.needsUpdate = true
    const frames = window.setInterval(() => invalidate(), FOCUS_FRAME_MS)
    const shadows = window.setInterval(() => {
      gl.shadowMap.needsUpdate = true
    }, FOCUS_SHADOW_MS)
    return () => {
      window.clearInterval(frames)
      window.clearInterval(shadows)
      gl.shadowMap.autoUpdate = true
      gl.shadowMap.needsUpdate = true
    }
  }, [mode, invalidate, get])
  return null
}
