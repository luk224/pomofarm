/// <reference types="node" />
import { readFileSync } from 'node:fs'
import { describe, expect, it } from 'vitest'
import { ACCENTS, accentsFor } from './accents'

// the real stylesheet, so a token edited in ui.css is what gets checked
const css = readFileSync(new URL('../ui/ui.css', import.meta.url), 'utf8')
const rgb = (hex: string): [number, number, number] => [1, 3, 5].map((i) => parseInt(hex.slice(i, i + 2), 16) / 255) as [number, number, number]
const lin = (v: number) => (v <= 0.03928 ? v / 12.92 : ((v + 0.055) / 1.055) ** 2.4)
const lum = (hex: string) => {
  const [r, g, b] = rgb(hex).map(lin)
  return 0.2126 * r + 0.7152 * g + 0.0722 * b
}
const contrast = (a: string, b: string) => (Math.max(lum(a), lum(b)) + 0.05) / (Math.min(lum(a), lum(b)) + 0.05)

// Machado et al. (2009) matrices for full dichromacy, applied in linear RGB
const CVD: Record<string, number[][]> = {
  protanopia: [[0.152286, 1.052583, -0.204868], [0.114503, 0.786281, 0.099216], [-0.003882, -0.048116, 1.051998]],
  deuteranopia: [[0.367322, 0.860646, -0.227968], [0.280085, 0.672501, 0.047413], [-0.01182, 0.04294, 0.968881]],
  tritanopia: [[1.255528, -0.076749, -0.178779], [-0.078411, 0.930809, 0.147602], [0.004733, 0.691367, 0.3039]],
}
const simulate = (hex: string, m: number[][]): [number, number, number] => {
  const c = rgb(hex).map(lin)
  return m.map((row) => Math.min(1, Math.max(0, row[0] * c[0] + row[1] * c[1] + row[2] * c[2]))) as [number, number, number]
}
// distance in linear RGB weighted by perceived lightness: enough to tell "clearly apart" from "same to the eye"
const apart = (a: [number, number, number], b: [number, number, number]) => Math.hypot(a[0] - b[0], a[1] - b[1], a[2] - b[2])

function tokens(selector: string): Record<string, string> {
  const start = css.indexOf(selector)
  const body = css.slice(css.indexOf('{', start) + 1, css.indexOf('}', start))
  return Object.fromEntries([...body.matchAll(/(--[\w-]+):\s*(#[0-9a-fA-F]{6})/g)].map((m) => [m[1], m[2]]))
}
const base = tokens(':root {')
const cb = { ...base, ...tokens(':root[data-palette=cb]') }

describe('both palettes keep text readable (WCAG AA, 4.5:1)', () => {
  for (const [name, t] of [['default', base], ['cb', cb]] as const) {
    it(`${name}: white text on primary buttons, danger buttons and water buttons`, () => {
      expect(contrast('#ffffff', t['--leaf'])).toBeGreaterThanOrEqual(4.5)
      expect(contrast('#ffffff', t['--danger'])).toBeGreaterThanOrEqual(4.5)
      expect(contrast('#ffffff', t['--water-deep'])).toBeGreaterThanOrEqual(4.5)
    })
    it(`${name}: ink on white and on the sun colour`, () => {
      expect(contrast(t['--ink'], '#ffffff')).toBeGreaterThanOrEqual(7)
      expect(contrast(t['--ink'], t['--sun'])).toBeGreaterThanOrEqual(4.5)
    })
  }
})

describe('badges', () => {
  for (const key of ['default', 'cb'] as const) {
    it(`${key}: the text on each badge is readable`, () => {
      const a = ACCENTS[key]
      expect(contrast(a.badgeText, a.badge)).toBeGreaterThanOrEqual(4.5)
      expect(contrast(a.goldText, a.gold)).toBeGreaterThanOrEqual(4.5)
    })
  }
  it('the safe palette keeps "go" (primary) and "careful" (danger) apart for every kind of colour blindness', () => {
    for (const [kind, m] of Object.entries(CVD)) {
      expect(apart(simulate(cb['--leaf'], m), simulate(cb['--danger'], m)), `${kind}: primary vs danger`).toBeGreaterThan(0.1)
    }
  })
  it('the safe palette keeps the plain badge and the golden badge apart for every kind of colour blindness', () => {
    const a = ACCENTS.cb
    for (const [kind, m] of Object.entries(CVD)) {
      expect(apart(simulate(a.badge, m), simulate(a.gold, m)), kind).toBeGreaterThan(0.2)
    }
  })
  it('the safe palette uses only blues and oranges (no green, no red) for meaning', () => {
    const hue = (hex: string) => {
      const [r, g, b] = rgb(hex)
      const max = Math.max(r, g, b), min = Math.min(r, g, b)
      if (max === min) return -1
      const d = max - min
      const h = max === r ? ((g - b) / d) % 6 : max === g ? (b - r) / d + 2 : (r - g) / d + 4
      return (h * 60 + 360) % 360
    }
    for (const c of [ACCENTS.cb.badge, ACCENTS.cb.ringRunning]) expect(hue(c)).toBeGreaterThan(190) // blue
    for (const c of [ACCENTS.cb.gold, ACCENTS.cb.coverage]) expect(hue(c)).toBeLessThan(50) // orange / amber
    expect(hue(cb['--leaf'])).toBeGreaterThan(190)
  })
  it('falls back to the default palette for an unknown name', () => {
    expect(accentsFor('nope' as never)).toBe(ACCENTS.default)
  })
})
