import { useEffect, useState } from 'react'
import { api } from '../api/client'
import type { FlowRow } from '../api/types'

export const FLOW_MIN = 60
export const FLOW_MAX = 120

export function clampFlow(minutes: number): number {
  if (!Number.isFinite(minutes)) return FLOW_MIN
  return Math.min(FLOW_MAX, Math.max(FLOW_MIN, Math.round(minutes)))
}

/** The table row for a duration (clamped to the Flow range), or undefined while the table is loading. */
export function flowRow(table: FlowRow[] | null, minutes: number): FlowRow | undefined {
  return table?.find((r) => r.duration_min === clampFlow(minutes))
}

let cache: FlowRow[] | null = null

/** The Flow table, fetched once and kept: the server never changes it while it runs. */
export function useFlowTable(enabled: boolean): FlowRow[] | null {
  const [table, setTable] = useState<FlowRow[] | null>(cache)
  useEffect(() => {
    if (!enabled || cache) return
    let live = true
    api.flow().then((rows) => {
      cache = rows
      if (live) setTable(rows)
    }).catch(() => undefined) // without it the slider still works; only the preview is missing
    return () => {
      live = false
    }
  }, [enabled])
  return table ?? cache
}

/** "5 h", "174,6 h", "216 h". */
export function formatHours(h: number): string {
  return `${new Intl.NumberFormat('es-ES', { maximumFractionDigits: 1 }).format(h)} h`
}
