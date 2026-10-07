import { afterEach, describe, expect, it, vi } from 'vitest'
import { api, ApiError } from './client'

const reply = (status: number, body: unknown, jsonOk = true) =>
  vi.fn().mockResolvedValue({ ok: status < 400, status, json: jsonOk ? async () => body : async () => { throw new Error('not json') } })

afterEach(() => vi.unstubAllGlobals())

describe('api client errors', () => {
  it('reads the server error code', async () => {
    vi.stubGlobal('fetch', reply(409, { error: 'pomodoro_active' }))
    await expect(api.state()).rejects.toMatchObject({ status: 409, code: 'pomodoro_active' })
  })
  it('treats proxy 502/503/504 as a connection problem, whatever the body', async () => {
    for (const s of [502, 503, 504]) {
      vi.stubGlobal('fetch', reply(s, null, false))
      await expect(api.state()).rejects.toMatchObject({ status: s, code: 'network' })
    }
  })
  it('treats a failed fetch (server unreachable) as network', async () => {
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new TypeError('Failed to fetch')))
    const err = await api.state().catch((e) => e)
    expect(err).toBeInstanceOf(ApiError)
    expect(err.code).toBe('network')
  })
  it('falls back to "unknown" for an error without a code', async () => {
    vi.stubGlobal('fetch', reply(500, {}))
    await expect(api.state()).rejects.toMatchObject({ code: 'unknown' })
  })
})
