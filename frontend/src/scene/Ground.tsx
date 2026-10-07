import { useLayoutEffect, useRef } from 'react'
import { Color, Object3D, type InstancedMesh } from 'three'
import { palette as P } from './palette'

const SIZE = 6 // tiles per side, centred on the 4×4 farm
const FROM = -1

/** Checkerboard grass as ONE instanced mesh (one draw call). */
export function Ground() {
  const ref = useRef<InstancedMesh>(null)
  useLayoutEffect(() => {
    const m = ref.current
    if (!m) return
    const o = new Object3D()
    const a = new Color(P.grassA)
    const b = new Color(P.grassB)
    for (let i = 0; i < SIZE * SIZE; i++) {
      const x = (i % SIZE) + FROM
      const z = Math.floor(i / SIZE) + FROM
      o.position.set(x, -0.05, z)
      o.updateMatrix()
      m.setMatrixAt(i, o.matrix)
      m.setColorAt(i, (x + z) % 2 === 0 ? a : b)
    }
    m.instanceMatrix.needsUpdate = true
    if (m.instanceColor) m.instanceColor.needsUpdate = true
  }, [])
  return (
    <instancedMesh ref={ref} args={[undefined, undefined, SIZE * SIZE]} receiveShadow>
      <boxGeometry args={[1, 0.1, 1]} />
      <meshStandardMaterial flatShading />
    </instancedMesh>
  )
}
