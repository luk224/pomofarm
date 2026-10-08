import { usePrefs } from '../store/prefs'
import { palette as P } from './palette'

/**
 * Colours that carry meaning, in the default palette and in the colour-blind-safe one (GDD §2.4). The safe palette is the
 * Okabe-Ito set, chosen to stay apart under protanopia, deuteranopia and tritanopia: blue and orange instead of green and
 * red/gold. Colour is never the only signal anyway (badges say the number, plants differ in shape and when withered they droop).
 */
export interface Accents {
  /** Fill of a plain bonus badge, and its text colour. */
  badge: string
  badgeText: string
  /** Fill of the Huerto-completo badge and its text colour. */
  gold: string
  goldText: string
  ringRunning: string
  ringPaused: string
  /** Amber overlay on the plots a hive would cover. */
  coverage: string
}

export const ACCENTS: Record<'default' | 'cb', Accents> = {
  default: { badge: '#2f7d32', badgeText: '#ffffff', gold: P.sparkle, goldText: '#3b2a1f', ringRunning: P.ringRunning, ringPaused: P.ringPaused, coverage: '#ff9f1c' },
  cb: { badge: '#0072B2', badgeText: '#ffffff', gold: '#E69F00', goldText: '#1a1209', ringRunning: '#0072B2', ringPaused: '#7a7f88', coverage: '#E69F00' },
}

export function accentsFor(palette: 'default' | 'cb'): Accents {
  return ACCENTS[palette] ?? ACCENTS.default
}

export function useAccents(): Accents {
  return accentsFor(usePrefs((s) => s.palette))
}
