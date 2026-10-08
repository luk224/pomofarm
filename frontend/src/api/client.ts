import type { BookMonth, FlowRow, GameState, PlantRequest } from './types'

/** Error body from the server is {"error": "<code>"}; code is stable, message is for humans. */
export class ApiError extends Error {
  readonly status: number
  readonly code: string
  constructor(status: number, code: string) {
    super(`${status} ${code}`)
    this.status = status
    this.code = code
  }
}

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  let res: Response
  try {
    res = await fetch(path, {
      method,
      headers: body === undefined ? undefined : { 'Content-Type': 'application/json' },
      body: body === undefined ? undefined : JSON.stringify(body),
    })
  } catch {
    throw new ApiError(0, 'network')
  }
  // 502/503/504 come from the proxy (Nginx/Vite) when the game server is down or restarting: that is a
  // connection problem, whatever body the proxy sends.
  if (res.status === 502 || res.status === 503 || res.status === 504) throw new ApiError(res.status, 'network')
  const data = await res.json().catch(() => ({}))
  if (!res.ok) throw new ApiError(res.status, typeof data?.error === 'string' ? data.error : 'unknown')
  return data as T
}

/** The player's own time zone, so the server groups Pomodoros into the days they actually lived. */
export const localTimeZone = (): string => {
  try {
    return Intl.DateTimeFormat().resolvedOptions().timeZone || ''
  } catch {
    return ''
  }
}

export const bookExportUrl = (): string => `/api/book/export.csv?tz=${encodeURIComponent(localTimeZone())}`

export const api = {
  book: (month: string) => request<BookMonth>('GET', `/api/book?month=${encodeURIComponent(month)}&tz=${encodeURIComponent(localTimeZone())}`),
  flow: () => request<FlowRow[]>('GET', '/api/flow'),
  state: () => request<GameState>('GET', '/api/state'),
  plant: (req: PlantRequest) => request<GameState>('POST', '/api/pomodoros', req),
  pause: () => request<GameState>('POST', '/api/pomodoros/active/pause'),
  resume: () => request<GameState>('POST', '/api/pomodoros/active/resume'),
  cancel: () => request<GameState>('POST', '/api/pomodoros/active/cancel'),
  harvest: (plotId: number) =>
    request<{ reward_focus: number; state: GameState }>('POST', `/api/plots/${plotId}/harvest`),
  buyPlot: () => request<GameState>('POST', '/api/plots'),
  upgradeSilo: () => request<GameState>('POST', '/api/silo/upgrade'),
  skipRest: () => request<GameState>('POST', '/api/rest/skip'),
  collectSilo: () => request<{ collected_milli: number; state: GameState }>('POST', '/api/silo/collect'),
  clearPlot: (plotId: number, confirm: boolean) =>
    request<GameState>('POST', `/api/plots/${plotId}/clear`, { confirm }),
  setSetting: (key: string, value: string) => request<GameState>('POST', '/api/settings', { key, value }),
  unlockAnimal: (key: 'bees' | 'dog') => request<GameState>('POST', '/api/unlocks', { kind: 'animal', key }),
  buyHive: (plotId: number) => request<GameState>('POST', '/api/structures', { kind: 'hive', plot_id: plotId }),
  buyDog: () => request<GameState>('POST', '/api/structures', { kind: 'dog' }),
  moveHive: (hiveId: number, plotId: number) =>
    request<GameState>('POST', `/api/structures/${hiveId}/move`, { plot_id: plotId }),
  buyDecor: (kind: string, x: number, y: number) => request<GameState>('POST', '/api/decor', { kind, x, y }),
  buyHat: () => request<GameState>('POST', '/api/decor', { kind: 'hat' }),
  moveDecor: (id: number, x: number, y: number) => request<GameState>('POST', `/api/decor/${id}/move`, { x, y }),
  removeDecor: (id: number) => request<GameState>('DELETE', `/api/decor/${id}`),
  unlockSeed: (key: string) => request<GameState>('POST', '/api/unlocks', { kind: 'seed', key }),
}
