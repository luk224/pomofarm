/**
 * One AudioContext and three independent layers (GDD §1, "Volumen independiente por capa"):
 * ambient (background sound), effects (digging, harvesting, coins) and alerts (the bowl chime).
 * Everything is synthesised with Web Audio: no audio files, nothing to license.
 */
export type Layer = 'ambient' | 'effects' | 'alerts'
export const LAYERS: readonly Layer[] = ['ambient', 'effects', 'alerts']

/** The slider is linear, hearing is not: squaring makes the lower half of the slider usable. */
export function volumeToGain(v: number): number {
  const x = Number.isFinite(v) ? Math.min(1, Math.max(0, v)) : 0
  return x * x
}

let ctx: AudioContext | null = null
const buses: Partial<Record<Layer, GainNode>> = {}
let master: GainNode | null = null
let muted = false
const volumes: Record<Layer, number> = { ambient: 0.5, effects: 0.7, alerts: 0.8 }
const unlockListeners = new Set<() => void>()

/** Create/resume the context. Browsers keep it suspended until a user gesture. Returns the context, or null without Web Audio. */
export function unlockAudio(): AudioContext | null {
  try {
    if (!ctx) {
      playbackSession()
      ctx = new AudioContext()
      master = ctx.createGain() // one switch for everything: "Sin audio" (GDD §2.4)
      master.gain.value = muted ? 0 : 1
      master.connect(ctx.destination)
      for (const l of LAYERS) {
        const g = ctx.createGain()
        g.gain.value = volumeToGain(volumes[l])
        g.connect(master)
        buses[l] = g
      }
      if (import.meta.env.DEV) (window as unknown as { __audioEngine?: unknown }).__audioEngine = { ctx, buses, volumes, get master() { return master } }
    }
    const notify = () => unlockListeners.forEach((f) => f())
    if (ctx.state !== 'running') {
      // "suspended" (not unlocked yet) or "interrupted" (iOS after a call or a trip to the background). resume() is asynchronous:
      // listeners (e.g. starting the ambient sound) must run once it is really running.
      void ctx.resume().then(notify, () => undefined)
      kick(ctx)
    } else notify()
  } catch {
    ctx = null // no Web Audio: everything stays visual
  }
  return ctx
}

/** Silences everything at once (all three layers), smoothly. Remembered for when audio is first unlocked. */
export function setMasterMuted(on: boolean): void {
  muted = on
  if (master && ctx) master.gain.setTargetAtTime(on ? 0 : 1, ctx.currentTime, 0.02)
}

/**
 * Phones are strict. Chrome on Android only counts a tap as permission to play sound when it ENDS (pointerdown does not count for
 * touch), and Safari on iPhone wants audio to be started from a touchend/click handler, plays Web Audio through the ringer switch
 * (silent mode = no sound) unless the page says its audio is "playback", and may "interrupt" the context later. So:
 *  - the context is (re)started on every kind of tap or key press until it is running, not only on the first pointerdown;
 *  - a one-sample silent buffer is played inside that gesture (the classic iOS unlock);
 *  - the Audio Session API is set to "playback" where it exists, so the ringer switch does not mute the farm.
 */
export const UNLOCK_EVENTS = ['pointerdown', 'pointerup', 'touchend', 'click', 'keydown'] as const

function kick(c: AudioContext): void {
  try {
    const src = c.createBufferSource()
    src.buffer = c.createBuffer(1, 1, 22050)
    src.connect(c.destination)
    src.start(0)
  } catch {
    /* harmless: the resume() above is the real unlock */
  }
}

function playbackSession(): void {
  const nav = navigator as Navigator & { audioSession?: { type: string } }
  try {
    if (nav.audioSession) nav.audioSession.type = 'playback'
  } catch {
    /* not supported */
  }
}

/** Listens for taps and key presses and unlocks (or re-unlocks) the audio from inside them. Returns a function that stops. */
export function watchAudioUnlock(): () => void {
  const handler = () => {
    if (!ctx || ctx.state !== 'running') unlockAudio()
  }
  for (const e of UNLOCK_EVENTS) window.addEventListener(e, handler, { capture: true, passive: true })
  return () => {
    for (const e of UNLOCK_EVENTS) window.removeEventListener(e, handler, true)
  }
}

/** The context if audio is running (unlocked by a gesture), otherwise null. */
export function runningContext(): AudioContext | null {
  return ctx && ctx.state === 'running' ? ctx : null
}

export function layerBus(layer: Layer): GainNode | null {
  return buses[layer] ?? null
}

/** Sets a layer's volume (0..1). Smoothed, so dragging a slider never clicks. */
export function setLayerVolume(layer: Layer, v: number): void {
  volumes[layer] = Math.min(1, Math.max(0, Number.isFinite(v) ? v : 0))
  const bus = buses[layer]
  if (bus && ctx) bus.gain.setTargetAtTime(volumeToGain(volumes[layer]), ctx.currentTime, 0.03)
}

/** Called each time audio gets unlocked (first gesture), e.g. to start the ambient sound the player chose. */
export function onUnlock(f: () => void): () => void {
  unlockListeners.add(f)
  return () => unlockListeners.delete(f)
}

/** A buffer of noise, `seconds` long: white, or brown (random walk, deep like rain on a roof or a fire). */
export function noiseBuffer(c: AudioContext, seconds: number, kind: 'white' | 'brown'): AudioBuffer {
  const buf = c.createBuffer(1, Math.max(1, Math.floor(c.sampleRate * seconds)), c.sampleRate)
  const d = buf.getChannelData(0)
  let last = 0
  for (let i = 0; i < d.length; i++) {
    const w = Math.random() * 2 - 1
    if (kind === 'white') d[i] = w
    else {
      last = (last + 0.02 * w) / 1.02
      d[i] = last * 3.5
    }
  }
  return buf
}
