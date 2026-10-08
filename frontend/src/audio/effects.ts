import { layerBus, noiseBuffer, runningContext } from './engine'

/**
 * Interaction sounds for the "effects" layer. Short, soft and crunchy; each is a few noise bursts and sine blips.
 * They return false when audio is not running (never unlocked, or the tab is muted), so callers can ignore it.
 */
export type EffectName = 'dig' | 'harvest' | 'coin' | 'buy' | 'place'

/** Peak level per effect, kept low so the layer slider alone sets how loud they are. */
export const EFFECT_LEVEL: Record<EffectName, number> = { dig: 0.5, harvest: 0.45, coin: 0.35, buy: 0.4, place: 0.4 }

let white: AudioBuffer | null = null

function burst(c: AudioContext, out: AudioNode, at: number, dur: number, freq: number, q: number, level: number, type: BiquadFilterType = 'bandpass') {
  white ??= noiseBuffer(c, 0.5, 'white')
  const src = c.createBufferSource()
  src.buffer = white
  const f = c.createBiquadFilter()
  f.type = type
  f.frequency.value = freq
  f.Q.value = q
  const g = c.createGain()
  g.gain.setValueAtTime(0.0001, at)
  g.gain.linearRampToValueAtTime(level, at + 0.006)
  g.gain.exponentialRampToValueAtTime(0.0001, at + dur)
  src.connect(f).connect(g).connect(out)
  src.start(at, Math.random() * 0.3)
  src.stop(at + dur + 0.05)
}

function blip(c: AudioContext, out: AudioNode, at: number, hz: number, dur: number, level: number, endHz = hz) {
  const o = c.createOscillator()
  const g = c.createGain()
  o.type = 'sine'
  o.frequency.setValueAtTime(hz, at)
  if (endHz !== hz) o.frequency.exponentialRampToValueAtTime(endHz, at + dur)
  g.gain.setValueAtTime(0.0001, at)
  g.gain.linearRampToValueAtTime(level, at + 0.008)
  g.gain.exponentialRampToValueAtTime(0.0001, at + dur)
  o.connect(g).connect(out)
  o.start(at)
  o.stop(at + dur + 0.05)
}

export function playEffect(name: EffectName): boolean {
  const c = runningContext()
  const bus = layerBus('effects')
  if (!c || !bus) return false
  const t = c.currentTime + 0.01
  const L = EFFECT_LEVEL[name]
  switch (name) {
    case 'dig': // a scoop of soil: a low thud and two gritty crunches
      blip(c, bus, t, 150, 0.14, L * 0.8, 70)
      burst(c, bus, t, 0.12, 900, 0.8, L)
      burst(c, bus, t + 0.09, 0.1, 1500, 1.2, L * 0.7)
      break
    case 'harvest': // a fresh crunch and a rising pop
      burst(c, bus, t, 0.07, 2600, 1.5, L)
      burst(c, bus, t + 0.06, 0.09, 1800, 1.1, L * 0.8)
      blip(c, bus, t + 0.04, 520, 0.16, L * 0.5, 880)
      break
    case 'coin': // two bright pings
      blip(c, bus, t, 1319, 0.22, L)
      blip(c, bus, t + 0.09, 1760, 0.35, L * 0.9)
      break
    case 'buy': // a soft thunk and a ping
      blip(c, bus, t, 220, 0.12, L, 120)
      blip(c, bus, t + 0.07, 988, 0.28, L * 0.7)
      break
    case 'place': // a quick tap on the ground
      burst(c, bus, t, 0.08, 600, 1, L)
      blip(c, bus, t, 260, 0.1, L * 0.6, 150)
      break
  }
  return true
}
