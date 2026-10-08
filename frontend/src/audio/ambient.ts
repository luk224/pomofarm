import { layerBus, noiseBuffer, runningContext } from './engine'

/**
 * Local ambient sound, synthesised: rain, forest and fire. Doubles as the fallback when the Lofi iframe fails (GDD §2).
 * It plays through the "ambient" layer.
 */
export type AmbientKind = 'off' | 'rain' | 'forest' | 'fire'
export const AMBIENT_KINDS: readonly AmbientKind[] = ['off', 'rain', 'forest', 'fire']

export function isAmbientKind(v: unknown): v is AmbientKind {
  return typeof v === 'string' && (AMBIENT_KINDS as readonly string[]).includes(v)
}

interface Running {
  kind: AmbientKind
  stop: () => void
}
let running: Running | null = null

const FADE = 0.8

function loop(c: AudioContext, out: AudioNode, kind: 'white' | 'brown', filter: BiquadFilterType, freq: number, q: number, level: number) {
  const src = c.createBufferSource()
  src.buffer = noiseBuffer(c, 4, kind)
  src.loop = true
  const f = c.createBiquadFilter()
  f.type = filter
  f.frequency.value = freq
  f.Q.value = q
  const g = c.createGain()
  g.gain.value = level
  src.connect(f).connect(g).connect(out)
  src.start()
  return src
}

function chirp(c: AudioContext, out: AudioNode) {
  const t = c.currentTime + 0.02
  const base = 2200 + Math.random() * 1800
  for (let i = 0; i < 2 + Math.floor(Math.random() * 3); i++) {
    const o = c.createOscillator()
    const g = c.createGain()
    const at = t + i * 0.11
    o.type = 'sine'
    o.frequency.setValueAtTime(base, at)
    o.frequency.exponentialRampToValueAtTime(base * (1.15 + Math.random() * 0.3), at + 0.08)
    g.gain.setValueAtTime(0.0001, at)
    g.gain.linearRampToValueAtTime(0.05, at + 0.012)
    g.gain.exponentialRampToValueAtTime(0.0001, at + 0.1)
    o.connect(g).connect(out)
    o.start(at)
    o.stop(at + 0.14)
  }
}

function crackle(c: AudioContext, out: AudioNode) {
  const t = c.currentTime + 0.01
  const src = c.createBufferSource()
  src.buffer = noiseBuffer(c, 0.1, 'white')
  const f = c.createBiquadFilter()
  f.type = 'highpass'
  f.frequency.value = 1800 + Math.random() * 2500
  const g = c.createGain()
  g.gain.setValueAtTime(0.0001, t)
  g.gain.linearRampToValueAtTime(0.12 + Math.random() * 0.12, t + 0.002)
  g.gain.exponentialRampToValueAtTime(0.0001, t + 0.03 + Math.random() * 0.04)
  src.connect(f).connect(g).connect(out)
  src.start(t)
  src.stop(t + 0.1)
}

/** Starts (or switches to) an ambient sound; 'off' fades it out. Returns false if audio is not running. */
export function setAmbient(kind: AmbientKind): boolean {
  const c = runningContext()
  const bus = layerBus('ambient')
  if (kind === 'off') {
    stopAmbient()
    return true
  }
  if (!c || !bus) return false
  if (running?.kind === kind) return true
  stopAmbient()
  const mix = c.createGain() // this sound's own fade, below the layer's volume
  mix.gain.setValueAtTime(0.0001, c.currentTime)
  mix.gain.linearRampToValueAtTime(1, c.currentTime + FADE)
  mix.connect(bus)
  const sources: AudioScheduledSourceNode[] = []
  const timers: ReturnType<typeof setTimeout>[] = []
  const every = (minMs: number, maxMs: number, fn: () => void) => {
    const tick = () => {
      fn()
      timers.push(setTimeout(tick, minMs + Math.random() * (maxMs - minMs)))
    }
    timers.push(setTimeout(tick, minMs))
  }
  if (kind === 'rain') {
    sources.push(loop(c, mix, 'white', 'highpass', 1200, 0.5, 0.18), loop(c, mix, 'white', 'lowpass', 6000, 0.5, 0.14), loop(c, mix, 'brown', 'lowpass', 400, 0.7, 0.35))
  } else if (kind === 'forest') {
    sources.push(loop(c, mix, 'brown', 'lowpass', 700, 0.5, 0.3), loop(c, mix, 'white', 'bandpass', 3500, 0.4, 0.03))
    every(2500, 7000, () => chirp(c, mix))
  } else {
    sources.push(loop(c, mix, 'brown', 'lowpass', 500, 0.6, 0.45), loop(c, mix, 'brown', 'bandpass', 1100, 1.2, 0.12))
    every(120, 900, () => crackle(c, mix))
  }
  running = {
    kind,
    stop: () => {
      timers.forEach(clearTimeout)
      const t = c.currentTime
      mix.gain.cancelScheduledValues(t)
      mix.gain.setValueAtTime(mix.gain.value, t)
      mix.gain.linearRampToValueAtTime(0.0001, t + FADE)
      sources.forEach((s) => s.stop(t + FADE + 0.05))
      setTimeout(() => mix.disconnect(), (FADE + 0.2) * 1000)
    },
  }
  return true
}

export function stopAmbient(): void {
  running?.stop()
  running = null
}

export function currentAmbient(): AmbientKind {
  return running?.kind ?? 'off'
}
