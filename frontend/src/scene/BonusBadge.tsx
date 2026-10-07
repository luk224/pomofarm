import { useMemo } from 'react'
import { CanvasTexture, LinearFilter } from 'three'
import { palette as P } from './palette'

const cache = new Map<string, CanvasTexture>()

function draw(ctx: CanvasRenderingContext2D, text: string, gold: boolean) {
  const w = 128, h = 64
  ctx.clearRect(0, 0, w, h)
  ctx.fillStyle = gold ? P.sparkle : '#2f7d32'
  ctx.beginPath()
  ctx.roundRect(4, 8, w - 8, h - 16, 24)
  ctx.fill()
  ctx.lineWidth = 4
  ctx.strokeStyle = '#ffffff'
  ctx.stroke()
  ctx.fillStyle = gold ? '#3b2a1f' : '#ffffff'
  ctx.font = '600 34px "Fredoka Variable", system-ui, sans-serif'
  ctx.textAlign = 'center'
  ctx.textBaseline = 'middle'
  ctx.fillText(text, w / 2, h / 2 + 1)
}

/** One texture per label, shared by every badge that shows it. */
function badgeTexture(text: string, gold: boolean): CanvasTexture {
  const key = `${gold ? 'g' : 'n'}:${text}`
  let tex = cache.get(key)
  if (!tex) {
    const canvas = document.createElement('canvas')
    canvas.width = 128
    canvas.height = 64
    const ctx = canvas.getContext('2d')
    if (ctx) draw(ctx, text, gold)
    tex = new CanvasTexture(canvas)
    tex.minFilter = LinearFilter
    tex.userData = { text, gold, canvas }
    cache.set(key, tex)
    // The webfont may arrive after the first draw: repaint once it is there.
    void document.fonts?.load('600 34px "Fredoka Variable"').then(() => {
      const c2 = canvas.getContext('2d')
      if (c2 && tex) {
        draw(c2, text, gold)
        tex.needsUpdate = true
      }
    })
  }
  return tex
}

interface Props {
  /** Position inside the parent group. */
  position?: [number, number, number]
  /** Width in world units (the label is twice as wide as tall). */
  size?: number
  text: string
  /** Gold for the Huerto completo; green for plain adjacency. */
  gold: boolean
  y?: number
}

/**
 * A small floating label over a producing plant that earns a synergy bonus (GDD §4.6). Always faces the camera
 * and is drawn on top, so a tall neighbour never hides it. Colour is not the only signal: the text says the number.
 */
export function BonusBadge({ text, gold, y = 0, position, size = 0.62 }: Props) {
  const tex = useMemo(() => badgeTexture(text, gold), [text, gold])
  return (
    <sprite name="bonus-badge" position={position ?? [0, y, 0]} scale={[size, size / 2, 1]} renderOrder={10} userData={{ text, gold }}>
      <spriteMaterial map={tex} transparent depthTest={false} />
    </sprite>
  )
}
