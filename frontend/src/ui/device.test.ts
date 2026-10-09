import { describe, expect, it } from 'vitest'
import { isTouchOnly } from './device'

describe('isTouchOnly', () => {
  it('is true when nothing can hover and the pointer is coarse (a phone)', () => {
    expect(isTouchOnly((q) => ({ matches: q === '(hover: none) and (pointer: coarse)' }))).toBe(true)
  })
  it('is false on a computer with a mouse', () => expect(isTouchOnly(() => ({ matches: false }))).toBe(false))
  it('is false if the browser cannot answer', () => {
    expect(isTouchOnly(() => { throw new Error('no matchMedia') })).toBe(false)
  })
})
