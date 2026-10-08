/** Minimal types for the part of the YouTube IFrame Player API that the player uses. */
export interface YTPlayer {
  playVideo(): void
  pauseVideo(): void
  setVolume(v: number): void
  loadVideoById(id: string): void
  destroy(): void
}
export interface YTEvents {
  onReady?: () => void
  onStateChange?: (e: { data: number }) => void
  onError?: (e: { data: number }) => void
}
export interface YTNamespace {
  Player: new (el: HTMLElement, opts: {
    videoId: string
    width?: number | string
    height?: number | string
    host?: string
    playerVars?: Record<string, string | number>
    events?: YTEvents
  }) => YTPlayer
  PlayerState: { PLAYING: number; PAUSED: number; ENDED: number; BUFFERING: number }
}

declare global {
  interface Window {
    YT?: YTNamespace
    onYouTubeIframeAPIReady?: () => void
  }
}

export const API_URL = 'https://www.youtube.com/iframe_api'

let loading: Promise<YTNamespace> | null = null

/**
 * Loads YouTube's player script the first time it is needed, never at page load (nothing is requested from YouTube until
 * the player presses play). Rejects if the script cannot load or does not start within `timeoutMs`; the caller then falls
 * back to the local ambient sound. A failed load is forgotten so "Reintentar" tries again.
 */
export function loadYouTubeApi(timeoutMs = 10000): Promise<YTNamespace> {
  if (window.YT?.Player) return Promise.resolve(window.YT)
  if (loading) return loading
  loading = new Promise<YTNamespace>((resolve, reject) => {
    const script = document.createElement('script')
    const timer = setTimeout(() => fail(new Error('timeout')), timeoutMs)
    const fail = (e: Error) => {
      clearTimeout(timer)
      script.remove()
      loading = null
      reject(e)
    }
    const previous = window.onYouTubeIframeAPIReady
    window.onYouTubeIframeAPIReady = () => {
      previous?.()
      clearTimeout(timer)
      if (window.YT?.Player) resolve(window.YT)
      else fail(new Error('no player'))
    }
    script.src = API_URL
    script.async = true
    script.onerror = () => fail(new Error('network'))
    document.head.appendChild(script)
  })
  return loading
}

/** YouTube error codes that mean "this video will not play here" (invalid id, not found, embedding disabled). */
export function isFatalError(code: number): boolean {
  return [2, 5, 100, 101, 150].includes(code)
}
