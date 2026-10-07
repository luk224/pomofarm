import {
  BoxGeometry, BufferGeometry, Color, ConeGeometry, CylinderGeometry, Euler, Float32BufferAttribute, IcosahedronGeometry, Matrix4,
  OctahedronGeometry, Quaternion, SphereGeometry, Vector3,
} from 'three'
import { mergeGeometries } from 'three/examples/jsm/utils/BufferGeometryUtils.js'
import { palette as P } from '../palette'
import type { PlantKind } from './kinds'

/**
 * Each plant is ONE mesh: its parts (stem, leaves, petals, fruit…) are merged into a single geometry with a
 * colour per vertex. 16 plants then cost 16 draw calls instead of ~250, which is what keeps a full farm smooth
 * on a phone. The shapes are exactly the low-poly ones the plants always had.
 */

type Shape = 'cyl' | 'sph' | 'ico' | 'cone' | 'oct' | 'box'
export interface Part {
  shape: Shape
  args: number[]
  color: string
  pos?: [number, number, number]
  rot?: [number, number, number]
  scale?: [number, number, number]
}

function shapeGeometry(p: Part): BufferGeometry {
  const a = p.args
  switch (p.shape) {
    case 'cyl': return new CylinderGeometry(a[0], a[1], a[2], a[3])
    case 'sph': return new SphereGeometry(a[0], a[1], a[2])
    case 'ico': return new IcosahedronGeometry(a[0], a[1])
    case 'cone': return new ConeGeometry(a[0], a[1], a[2])
    case 'oct': return new OctahedronGeometry(a[0], a[1])
    case 'box': return new BoxGeometry(a[0], a[1], a[2])
  }
}

function compose(pos: [number, number, number] = [0, 0, 0], rot: [number, number, number] = [0, 0, 0], scale: [number, number, number] = [1, 1, 1]): Matrix4 {
  return new Matrix4().compose(new Vector3(...pos), new Quaternion().setFromEuler(new Euler(...rot)), new Vector3(...scale))
}

/** A part to build, or a geometry that is already merged and coloured (e.g. the sunflower head). */
export type Piece = Part | BufferGeometry

/** Merges the pieces into one non-indexed geometry carrying a colour attribute. `parent` transforms the whole set. */
export function mergeParts(pieces: Piece[], parent: Matrix4 = new Matrix4()): BufferGeometry {
  const geos = pieces.map((piece) => {
    if (piece instanceof BufferGeometry) return piece.clone().applyMatrix4(parent)
    const g = shapeGeometry(piece).toNonIndexed()
    g.deleteAttribute('uv')
    g.applyMatrix4(parent.clone().multiply(compose(piece.pos, piece.rot, piece.scale)))
    const c = new Color(piece.color)
    const n = g.getAttribute('position').count
    const colors = new Float32Array(n * 3)
    for (let i = 0; i < n; i++) c.toArray(colors, i * 3)
    g.setAttribute('color', new Float32BufferAttribute(colors, 3))
    return g
  })
  const merged = mergeGeometries(geos, false)
  geos.forEach((g) => g.dispose())
  return merged
}

const tint = (withered: boolean, color: string) => (withered ? P.withered : color)

function daisy(mature: boolean, w: boolean): Piece[] {
  const parts: Piece[] = [
    { shape: 'cyl', args: [0.025, 0.03, 0.4, 5], pos: [0, 0.2, 0], color: tint(w, P.stem) },
    { shape: 'sph', args: [0.08, 5, 4], pos: [0.1, 0.12, 0], rot: [0, 0, -0.9], color: tint(w, P.leaf) },
    { shape: 'sph', args: [0.06, 5, 4], pos: [-0.08, 0.2, 0.02], rot: [0, 0, 0.9], color: tint(w, P.leaf) },
  ]
  if (!mature) return [...parts, { shape: 'sph', args: [0.06, 5, 4], pos: [0, 0.42, 0], color: P.leaf }]
  for (let i = 0; i < 8; i++) {
    const a = (i / 8) * Math.PI * 2
    parts.push({ shape: 'sph', args: [0.065, 5, 4], pos: [Math.cos(a) * 0.11, 0.42, Math.sin(a) * 0.11], color: tint(w, P.petal) })
  }
  parts.push({ shape: 'sph', args: [0.07, 6, 5], pos: [0, 0.43, 0], color: tint(w, P.yellow) })
  return parts
}

const TOMATOES: [number, number, number][] = [[0.15, 0.2, 0.11], [-0.13, 0.28, 0.09], [0.02, 0.17, -0.16], [-0.05, 0.38, -0.05]]
function tomato(mature: boolean, w: boolean): Piece[] {
  const parts: Piece[] = [
    { shape: 'cyl', args: [0.02, 0.03, 0.4, 5], pos: [0, 0.2, 0], color: tint(w, P.stem) },
    { shape: 'ico', args: [0.24, 0], pos: [0, 0.3, 0], color: tint(w, P.leaf) },
    { shape: 'ico', args: [0.16, 0], pos: [0, 0.5, 0], color: tint(w, P.stem) },
  ]
  if (mature) TOMATOES.forEach((p) => parts.push({ shape: 'sph', args: [0.075, 6, 5], pos: p, color: tint(w, P.red) }))
  return parts
}

function sunflower(mature: boolean, w: boolean): Piece[] {
  const parts: Piece[] = [
    { shape: 'cyl', args: [0.03, 0.04, 0.8, 5], pos: [0, 0.4, 0], color: tint(w, P.stem) },
    { shape: 'sph', args: [0.12, 5, 4], pos: [0.13, 0.3, 0], rot: [0, 0, -0.8], color: tint(w, P.leaf) },
    { shape: 'sph', args: [0.12, 5, 4], pos: [-0.13, 0.5, 0], rot: [0, 0, 0.8], color: tint(w, P.leaf) },
  ]
  if (!mature) return parts
  // The head faces the isometric camera: turn 45° around Y, then tilt it up.
  const head = compose([0, 0.85, 0], [0, Math.PI / 4, 0]).multiply(compose([0, 0, 0], [0.45, 0, 0]))
  const headParts: Part[] = []
  for (let i = 0; i < 12; i++) {
    const a = (i / 12) * Math.PI * 2
    headParts.push({ shape: 'cone', args: [0.085, 0.2, 4], pos: [Math.cos(a) * 0.27, Math.sin(a) * 0.27, 0], rot: [0, 0, a], color: tint(w, P.yellow) })
  }
  headParts.push({ shape: 'cyl', args: [0.22, 0.22, 0.07, 10], rot: [Math.PI / 2, 0, 0], color: tint(w, P.yellow) })
  headParts.push({ shape: 'cyl', args: [0.15, 0.15, 0.07, 8], pos: [0, 0, 0.05], rot: [Math.PI / 2, 0, 0], color: tint(w, P.brown) })
  return [...parts, mergeParts(headParts, head)]
}

const APPLES: [number, number, number][] = [[0.3, 0.85, 0.25], [-0.25, 0.95, 0.3], [0.05, 1.1, -0.3], [-0.3, 0.8, -0.2]]
function tree(oak: boolean, mature: boolean, w: boolean): Piece[] {
  const r = oak ? 0.44 : 0.36
  const crown = oak ? P.oakCanopy : P.canopy
  const parts: Piece[] = [
    { shape: 'cyl', args: [oak ? 0.1 : 0.07, oak ? 0.14 : 0.1, 0.6, 6], pos: [0, 0.3, 0], color: tint(w, P.trunk) },
    { shape: 'ico', args: [r, 1], pos: [0, 0.8, 0], color: tint(w, crown) },
    { shape: 'ico', args: [r * 0.6, 0], pos: [r * 0.55, 0.65, r * 0.2], color: tint(w, P.leafDark) },
    { shape: 'ico', args: [r * 0.55, 0], pos: [-r * 0.5, 0.95, -r * 0.2], color: tint(w, P.leafDark) },
  ]
  if (!oak && mature) APPLES.forEach((p) => parts.push({ shape: 'sph', args: [0.065, 6, 5], pos: p, color: tint(w, P.red) }))
  return parts
}

function partsFor(kind: PlantKind, mature: boolean, withered: boolean): Piece[] {
  switch (kind) {
    case 'daisy': return daisy(mature, withered)
    case 'tomato': return tomato(mature, withered)
    case 'sunflower': return sunflower(mature, withered)
    case 'apple': return tree(false, mature, withered)
    case 'oak': return tree(true, mature, withered)
  }
}

const cache = new Map<string, BufferGeometry>()

/** The merged geometry of a plant in a given stage; built once and shared by every plant of that kind and stage. */
export function plantGeometry(kind: PlantKind, mature: boolean, withered: boolean): BufferGeometry {
  const key = `${kind}:${mature ? 1 : 0}:${withered ? 1 : 0}`
  let g = cache.get(key)
  if (!g) {
    g = mergeParts(partsFor(kind, mature, withered))
    cache.set(key, g)
  }
  return g
}

/** The three orbs that circle a mature Oak, as one geometry (the group is rotated each frame). */
let orbs: BufferGeometry | undefined
export function orbsGeometry(): BufferGeometry {
  if (!orbs) {
    orbs = mergeParts([0, 2.1, 4.2].map((a) => ({
      shape: 'oct' as const, args: [0.09, 0], pos: [Math.cos(a) * 0.7, Math.sin(a * 2) * 0.12, Math.sin(a) * 0.7] as [number, number, number], color: P.magic,
    })))
  }
  return orbs
}

/** Number of triangles of a plant in a given stage (for the polygon budget). */
export function triangleCount(kind: PlantKind, mature: boolean, withered: boolean): number {
  return plantGeometry(kind, mature, withered).getAttribute('position').count / 3
}
