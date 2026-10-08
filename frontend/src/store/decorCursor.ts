import type { DecorState } from '../api/types'

export type Cell = [number, number]

/** Every background cell where a piece could go right now: inside the area, not blocked, not already taken. */
export function freeCells(decor: DecorState, ignoreId: number | null = null): Cell[] {
  const blocked = new Set(decor.blocked.map(([x, y]) => `${x},${y}`))
  const taken = new Set(decor.items.filter((i) => i.id !== ignoreId).map((i) => `${i.x},${i.y}`))
  const out: Cell[] = []
  for (let x = decor.min; x <= decor.max; x++)
    for (let y = decor.min; y <= decor.max; y++) if (!blocked.has(`${x},${y}`) && !taken.has(`${x},${y}`)) out.push([x, y])
  return out
}

/**
 * Moves the keyboard cursor one step along an axis, skipping cells that are not free (the field, the Silo, pieces already
 * there). Stays put if nothing free lies that way, so the cursor can never leave the allowed area.
 */
export function stepCursor(free: Cell[], from: Cell, dx: number, dy: number, min: number, max: number): Cell {
  const ok = new Set(free.map(([x, y]) => `${x},${y}`))
  let [x, y] = from
  for (;;) {
    x += dx
    y += dy
    if (x < min || x > max || y < min || y > max) return from
    if (ok.has(`${x},${y}`)) return [x, y]
  }
}

/** Where the cursor starts: the free cell nearest to the middle of the area (so it is on screen), or null if none is free. */
export function startCursor(free: Cell[]): Cell | null {
  if (free.length === 0) return null
  const cx = 1.5, cy = 1.5 // the centre of the 4×4 field
  return [...free].sort((a, b) => Math.hypot(a[0] - cx, a[1] - cy) - Math.hypot(b[0] - cx, b[1] - cy))[0]
}
