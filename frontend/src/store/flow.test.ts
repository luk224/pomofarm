import { describe, expect, it } from 'vitest'
import type { FlowRow } from '../api/types'
import { clampFlow, flowRow, formatHours } from './flow'

const table: FlowRow[] = [60, 61, 90, 120].map((m) => ({ duration_min: m, reward: m, life_h: m * 1.8, yield: m * 8, rest_min: 15 }))

describe('clampFlow', () => {
  it.each([[59, 60], [60, 60], [90.4, 90], [90.6, 91], [120, 120], [500, 120], [-1, 60], [NaN, 60], [Infinity, 60]])('%s -> %s', (i, o) =>
    expect(clampFlow(i)).toBe(o))
})

describe('flowRow', () => {
  it('finds the row for a minute, clamping to the range', () => {
    expect(flowRow(table, 90)?.duration_min).toBe(90)
    expect(flowRow(table, 30)?.duration_min).toBe(60)
    expect(flowRow(table, 999)?.duration_min).toBe(120)
  })
  it('is undefined while the table has not loaded', () => expect(flowRow(null, 90)).toBeUndefined())
})

describe('formatHours', () => {
  it.each([[108, '108 h'], [174.6, '174,6 h'], [109.8, '109,8 h']])('%s', (h, out) => expect(formatHours(h)).toBe(out))
})
