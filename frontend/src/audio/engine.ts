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
    // resume() is asynchronous: listeners (e.g. starting the ambient sound) must run once it is really running
    if (ctx.state === 'suspended') void ctx.resume().then(notify, () => undefined)
    else notify()
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
