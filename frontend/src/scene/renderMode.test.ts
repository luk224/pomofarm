import { describe, expect, it } from 'vitest'
import { FOCUS_FPS, FOCUS_FRAME_MS, FOCUS_SHADOW_MS, renderMode } from './renderMode'

describe('renderMode', () => {
  it('draws nothing while the tab is hidden, whatever else is going on', () => {
    for (const running of [true, false]) for (const focus of [true, false]) expect(renderMode(true, running, focus)).toBe('never')
  })
  it('uses Modo Foco only while a Pomodoro runs and the player left it on', () => {
    expect(renderMode(false, true, true)).toBe('demand')
    expect(renderMode(false, true, false)).toBe('always')
    expect(renderMode(false, false, true)).toBe('always')
    expect(renderMode(false, false, false)).toBe('always')
  })
  it('stays inside the 15–30 FPS the GDD asks for', () => {
    expect(FOCUS_FPS).toBeGreaterThanOrEqual(15)
    expect(FOCUS_FPS).toBeLessThanOrEqual(30)
    expect(FOCUS_FRAME_MS).toBeCloseTo(1000 / FOCUS_FPS)
  })
  it('refreshes shadows much less often than frames', () => expect(FOCUS_SHADOW_MS).toBeGreaterThanOrEqual(FOCUS_FRAME_MS * 5))
})
