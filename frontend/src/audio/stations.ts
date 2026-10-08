/**
 * Lofi Girl live streams offered by default. These IDs are configuration, not logic: YouTube retires and replaces live
 * streams now and then, and the channel's main "lofi hip hop radio" does NOT allow embedding (the player answers error 150),
 * so these are streams that do. Verified with the real player on 2026-10-08. If one stops working the player falls back
 * to the local ambient sound, and the player can paste their own link (see `parseVideoId`).
 */
export interface Station {
  id: string
  name: string
  /** True for links the player pasted (they can be removed). */
  custom?: boolean
}

export const DEFAULT_STATIONS: readonly Station[] = [
  { id: 'lTRiuFIWV54', name: 'Lofi · sesión de estudio' },
  { id: '4xDzrJKXOOY', name: 'Synthwave · jugar' },
  { id: 'S_MOd40zlYU', name: 'Ambiente oscuro · soñar' },
]

const ID = /^[A-Za-z0-9_-]{11}$/

/**
 * The video ID in whatever the player pasted: a bare ID, watch?v=…, youtu.be/…, /live/…, /embed/… or /shorts/….
 * Returns null for anything else (other sites, playlists without a video, junk), so nothing arbitrary reaches the iframe.
 */
export function parseVideoId(input: string): string | null {
  const text = input.trim()
  if (ID.test(text)) return text
  let url: URL
  try {
    url = new URL(/^[a-z]+:\/\//i.test(text) ? text : `https://${text}`)
  } catch {
    return null
  }
  if (url.protocol !== 'https:' && url.protocol !== 'http:') return null
  const host = url.hostname.replace(/^(www\.|m\.|music\.)/, '')
  let id: string | null = null
  if (host === 'youtu.be') id = url.pathname.split('/')[1] ?? null
  else if (host === 'youtube.com' || host === 'youtube-nocookie.com') {
    const [, kind, rest] = url.pathname.split('/')
    if (kind === 'watch') id = url.searchParams.get('v')
    else if (kind === 'live' || kind === 'embed' || kind === 'shorts' || kind === 'v') id = rest ?? null
  }
  return id && ID.test(id) ? id : null
}
