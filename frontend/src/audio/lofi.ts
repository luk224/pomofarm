import { useMusic } from '../store/music'
import { usePrefs } from '../store/prefs'
import { unlockAudio } from './engine'
import { isFatalError, loadYouTubeApi, type YTPlayer } from './youtube'

/**
 * Drives the Lofi player (GDD §6.6): YouTube's player in an iframe, and if it cannot play (no connection, blocked,
 * retired stream, embedding disabled) the local ambient sound takes over. All state lives in the music store;
 * this module only does the work.
 */
const READY_TIMEOUT_MS = 10000

let player: YTPlayer | null = null
let host: HTMLElement | null = null
let readyTimer: ReturnType<typeof setTimeout> | undefined
/** Bumped by every start/stop so late callbacks from an older attempt are ignored. */
let attempt = 0
/** True when the fallback turned the ambient sound on (so stopping turns it off again, but never a sound the player chose). */
let fallbackStartedAmbient = false

/** The widget registers the element the iframe goes in. */
export function setHost(el: HTMLElement | null): void {
  host = el
}

function destroyPlayer() {
  clearTimeout(readyTimer)
  try {
    player?.destroy()
  } catch {
    /* already gone */
  }
  player = null
  host?.replaceChildren()
}

function releaseFallback() {
  if (fallbackStartedAmbient) {
    usePrefs.getState().setAmbient('off')
    fallbackStartedAmbient = false
  }
}

function fallback(reason: 'network' | 'timeout' | 'video') {
  attempt++
  destroyPlayer()
  useMusic.getState().setStatus('fallback', reason)
  unlockAudio()
  const prefs = usePrefs.getState()
  if (prefs.ambient === 'off') {
    prefs.setAmbient('rain')
    fallbackStartedAmbient = true
  }
}

/** Starts the selected stream. Call from a click: it also unlocks audio, as browsers require. */
export async function startMusic(): Promise<void> {
  const mine = ++attempt
  releaseFallback()
  destroyPlayer()
  const music = useMusic.getState()
  music.setStatus('loading')
  music.setOpen(true)
  unlockAudio()
  let YT
  try {
    YT = await loadYouTubeApi()
  } catch (e) {
    if (mine === attempt) fallback(e instanceof Error && e.message === 'timeout' ? 'timeout' : 'network')
    return
  }
  if (mine !== attempt) return
  if (!host) return fallback('network')
  const el = document.createElement('div')
  host.replaceChildren(el)
  readyTimer = setTimeout(() => mine === attempt && fallback('timeout'), READY_TIMEOUT_MS)
  try {
    player = new YT.Player(el, {
      videoId: useMusic.getState().selected,
      width: '100%',
      height: '100%',
      host: 'https://www.youtube-nocookie.com',
      playerVars: { autoplay: 1, controls: 0, playsinline: 1, rel: 0, modestbranding: 1, disablekb: 1, origin: location.origin },
      events: {
        onReady: () => {
          if (mine !== attempt) return
          clearTimeout(readyTimer)
          player?.setVolume(Math.round(useMusic.getState().volume * 100))
          player?.playVideo()
        },
        onStateChange: (e) => {
          if (mine !== attempt) return
          if (e.data === YT.PlayerState.PLAYING) useMusic.getState().setStatus('playing')
          else if (e.data === YT.PlayerState.PAUSED) useMusic.getState().setStatus('paused')
        },
        onError: (e) => {
          console.warn('[lofi] YouTube player error', e.data) // helps to tell "retired stream" (100) from "embedding blocked" (101/150/153)
          if (mine === attempt && isFatalError(e.data)) fallback('video')
          else if (mine === attempt) fallback('network')
        },
      },
    })
  } catch {
    fallback('network')
  }
}

export function togglePauseMusic(): void {
  const { status } = useMusic.getState()
  if (status === 'playing') player?.pauseVideo()
  else if (status === 'paused') player?.playVideo()
  else if (status === 'idle' || status === 'fallback') void startMusic()
}

/** Stops the music and closes the iframe (nothing keeps playing or loading). */
export function stopMusic(): void {
  attempt++
  destroyPlayer()
  releaseFallback()
  useMusic.getState().setStatus('idle')
}

/** Picks another stream; if the player is running it switches at once. */
export function chooseStation(id: string): void {
  const music = useMusic.getState()
  music.select(id)
  const { status } = useMusic.getState()
  if (status === 'playing' || status === 'paused') player?.loadVideoById(id)
  else if (status === 'fallback' || status === 'loading') void startMusic()
}

export function setMusicVolume(v: number): void {
  useMusic.getState().setVolume(v)
  player?.setVolume(Math.round(useMusic.getState().volume * 100))
}
