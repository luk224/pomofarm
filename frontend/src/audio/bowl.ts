/**
 * A soft singing-bowl chime synthesised with Web Audio (no audio files, nothing to license).
 * A bowl is a few inharmonic partials that each fade at their own pace; two strikes make it a "ding… ding".
 */
export interface Partial {
  ratio: number
  gain: number
  /** Seconds to fade to silence. */
  decay: number
}

export const BASE_HZ = 392
export const BOWL_PARTIALS: readonly Partial[] = [
  { ratio: 1, gain: 1, decay: 5 },
  { ratio: 2.71, gain: 0.45, decay: 3.2 },
  { ratio: 5.2, gain: 0.2, decay: 1.8 },
  { ratio: 8.6, gain: 0.08, decay: 1 },
]
/** Strikes: delay in seconds and relative loudness. */
export const STRIKES = [
  { at: 0, level: 1 },
  { at: 1.5, level: 0.55 },
] as const

/** Peak of the summed partials stays well below clipping. */
export const MASTER_GAIN = 0.22

let ctx: AudioContext | null = null

/**
 * Create/resume the audio context. Browsers keep it suspended until a user gesture, so this must be
 * called from a click or key handler (we call it when the player starts a Pomodoro).
 */
export function unlockAudio(): void {
  try {
    ctx ??= new AudioContext()
    if (ctx.state === 'suspended') void ctx.resume()
  } catch {
    ctx = null // no Web Audio: alerts stay visual
  }
}

/** Plays the chime. Returns false if audio was never unlocked or is unavailable. */
export function playBowl(volume = 1): boolean {
  if (!ctx || ctx.state !== 'running') return false
  const t0 = ctx.currentTime + 0.05
  const master = ctx.createGain()
  master.gain.value = MASTER_GAIN * Math.min(1, Math.max(0, volume))
  master.connect(ctx.destination)
  for (const strike of STRIKES) {
    for (const p of BOWL_PARTIALS) {
      const osc = ctx.createOscillator()
      const g = ctx.createGain()
      const start = t0 + strike.at
      osc.type = 'sine'
      osc.frequency.value = BASE_HZ * p.ratio
      g.gain.setValueAtTime(0.0001, start)
      g.gain.linearRampToValueAtTime(p.gain * strike.level, start + 0.012)
      g.gain.exponentialRampToValueAtTime(0.0001, start + p.decay)
      osc.connect(g).connect(master)
      osc.start(start)
      osc.stop(start + p.decay + 0.1)
    }
  }
  return true
}

/**
 * Browsers only start audio after a user gesture, and a reload forgets it. Any first click or key press is a
 * gesture, so the alerts (end of Pomodoro or rest) work even if the player never planted in this page load.
 * Returns a function that removes the listeners.
 */
export function unlockAudioOnFirstGesture(): () => void {
  const events = ['pointerdown', 'keydown'] as const
  const unlock = () => {
    unlockAudio()
    off()
  }
  const off = () => events.forEach((e) => window.removeEventListener(e, unlock, true))
  events.forEach((e) => window.addEventListener(e, unlock, true))
  return off
}
