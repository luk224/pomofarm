import { describe, expect, it } from 'vitest'
import { messageFor } from './messages'

describe('messageFor', () => {
  it('translates known codes', () => {
    expect(messageFor('pomodoro_active')).toContain('otro dispositivo')
    expect(messageFor('insufficient_focus')).toContain('💧')
  })
  it('never exposes raw codes for unknown errors', () => {
    expect(messageFor('totally_new_code')).not.toContain('totally_new_code')
    expect(messageFor('')).toBe('Algo ha fallado. Inténtalo de nuevo.')
  })
  it('never apologises', () => {
    for (const c of ['network', 'no_player', 'version_conflict', 'x']) expect(messageFor(c).toLowerCase()).not.toMatch(/lo siento|perd[oó]n|disculp/)
  })
})
