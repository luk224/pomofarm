import { describe, expect, it } from 'vitest'
import type { DecorState } from '../api/types'
import { freeCells, startCursor, stepCursor } from './decorCursor'

const decor = (items: DecorState['items'] = []): DecorState => ({
  items, catalog: [], min: -2, max: 5, hat: { owned: false, available: false, cost: 600 },
  blocked: [...Array.from({ length: 16 }, (_, i): [number, number] => [i % 4, Math.floor(i / 4)]), [-1, 4], [0, 4]],
})

describe('freeCells', () => {
  it('is the 8×8 area minus the field (16) and the Silo and Dog cells (2)', () => expect(freeCells(decor())).toHaveLength(64 - 18))
  it('leaves out cells already holding a piece, unless that piece is the one being moved', () => {
    const d = decor([{ id: 1, kind: 'path', x: 4, y: 1 }])
    expect(freeCells(d)).toHaveLength(45)
    expect(freeCells(d, 1)).toHaveLength(46)
  })
})

describe('stepCursor', () => {
  const free = freeCells(decor())
  it('moves one cell along an axis', () => expect(stepCursor(free, [4, 1], 1, 0, -2, 5)).toEqual([5, 1]))
  it('jumps over the field instead of entering it', () => expect(stepCursor(free, [-1, 1], 1, 0, -2, 5)).toEqual([4, 1]))
  it('stays put at the edge of the area', () => {
    expect(stepCursor(free, [5, 1], 1, 0, -2, 5)).toEqual([5, 1])
    expect(stepCursor(free, [1, -2], 0, -1, -2, 5)).toEqual([1, -2])
  })
  it('stays put when nothing free lies that way', () => expect(stepCursor([[0, 0]], [0, 0], 1, 0, -2, 5)).toEqual([0, 0]))
  it('skips occupied cells', () => {
    const d = freeCells(decor([{ id: 1, kind: 'path', x: 5, y: 1 }]))
    expect(stepCursor(d, [4, 1], 1, 0, -2, 5)).toEqual([4, 1])
  })
})

describe('startCursor', () => {
  it('starts on a free cell close to the field', () => {
    const c = startCursor(freeCells(decor()))!
    expect(Math.hypot(c[0] - 1.5, c[1] - 1.5)).toBeLessThan(3.2)
  })
  it('is null when nothing is free', () => expect(startCursor([])).toBeNull())
})
