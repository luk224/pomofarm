import { MeshStandardMaterial } from 'three'

/** Shared material for every merged, vertex-coloured prop. */
export const BODY_MAT = new MeshStandardMaterial({ vertexColors: true, flatShading: true })
