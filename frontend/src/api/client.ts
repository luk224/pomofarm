import type { GameState, PlantRequest } from './types'

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
  const data = await res.json().catch(() => ({}))
  if (!res.ok) throw new ApiError(res.status, typeof data?.error === 'string' ? data.error : 'unknown')
  return data as T
}

export const api = {
  state: () => request<GameState>('GET', '/api/state'),
  plant: (req: PlantRequest) => request<GameState>('POST', '/api/pomodoros', req),
  pause: () => request<GameState>('POST', '/api/pomodoros/active/pause'),
  resume: () => request<GameState>('POST', '/api/pomodoros/active/resume'),
  cancel: () => request<GameState>('POST', '/api/pomodoros/active/cancel'),
  harvest: (plotId: number) =>
    request<{ reward_focus: number; state: GameState }>('POST', `/api/plots/${plotId}/harvest`),
  clearPlot: (plotId: number, confirm: boolean) =>
    request<GameState>('POST', `/api/plots/${plotId}/clear`, { confirm }),
  setSetting: (key: string, value: string) => request<GameState>('POST', '/api/settings', { key, value }),
  unlockSeed: (key: string) => request<GameState>('POST', '/api/unlocks', { kind: 'seed', key }),
}
