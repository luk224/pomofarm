import { MapControls, OrthographicCamera } from '@react-three/drei'
import { useRef } from 'react'
import type { OrthographicCamera as ThreeOrtho } from 'three'

// Fixed isometric angle: 45° around Y, ~35.264° elevation (true isometric).
const DIST = 30
const ELEV = Math.atan(1 / Math.SQRT2)
const POS: [number, number, number] = [
  DIST * Math.cos(ELEV) * Math.SQRT1_2,
  DIST * Math.sin(ELEV),
  DIST * Math.cos(ELEV) * Math.SQRT1_2,
]

interface Props {
  /** Ground-plane centre the camera starts on (the farm centre). */
  center?: [number, number]
  /** How far from `center` the view may be panned, in world units. */
  panLimit?: number
  minZoom?: number
  maxZoom?: number
  initialZoom?: number
}

/**
 * Fixed isometric orthographic camera with zoom (wheel / pinch) and ground-plane pan
 * (drag). No rotation. The target is clamped so the farm can never be dragged off screen.
 */
export function IsoCamera({ center = [1.5, 1.5], panLimit = 6, minZoom = 30, maxZoom = 140, initialZoom = 70 }: Props) {
  const cam = useRef<ThreeOrtho>(null)
  const [cx, cz] = center

  return (
    <>
      <OrthographicCamera ref={cam} makeDefault zoom={initialZoom} near={0.1} far={200}
        position={[cx + POS[0], POS[1], cz + POS[2]]} />
      <MapControls
        makeDefault
        target={[cx, 0, cz]}
        enableRotate={false}
        screenSpacePanning={false}
        enableDamping
        dampingFactor={0.12}
        minZoom={minZoom}
        maxZoom={maxZoom}
        zoomSpeed={0.8}
        onChange={(e) => {
          const c = e?.target as { target: { x: number; z: number; y: number }; object: ThreeOrtho } | undefined
          if (!c) return
          const dx = c.target.x - cx
          const dz = c.target.z - cz
          const d = Math.hypot(dx, dz)
          if (d > panLimit) {
            const k = panLimit / d
            const nx = cx + dx * k
            const nz = cz + dz * k
            c.object.position.x += nx - c.target.x
            c.object.position.z += nz - c.target.z
            c.target.x = nx
            c.target.z = nz
          }
          if (import.meta.env.DEV) {
            ;(window as unknown as { __iso?: unknown }).__iso = { zoom: c.object.zoom, x: c.target.x, z: c.target.z }
          }
        }}
      />
    </>
  )
}
