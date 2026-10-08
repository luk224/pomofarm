import { playBowl, unlockAudio } from '../audio/bowl'
import { usePrefs } from '../store/prefs'
import { useUi } from '../store/ui'
import { SEED_NAMES } from '../ui/names'
import type { Completion } from './completion'

/** Call from a click/key handler when a Pomodoro starts: unlocks audio and asks for notification permission once. */
export function prepareAlerts(): void {
  unlockAudio()
  if (usePrefs.getState().notify && typeof Notification !== 'undefined' && Notification.permission === 'default') {
    void Notification.requestPermission().catch(() => undefined)
  }
}

/**
 * A Pomodoro finished. Always shows an in-page message (so a muted player still sees it); adds the
 * chime if sound is on, and a system notification only when the player is not looking at the tab.
 */
export function announceCompletion(c: Completion): void {
  const name = SEED_NAMES[c.plantType] ?? c.plantType
  useUi.getState().toast(`Tu ${name.toLowerCase()} está lista. Tócala para cosechar.`)

  const prefs = usePrefs.getState()
  if (prefs.sound && !prefs.muted) playBowl()
  if (prefs.muted || !prefs.sound) useUi.getState().showFlash(`Tu ${name.toLowerCase()} está lista`) // no sound: a visual alert instead

  const away = document.hidden || !document.hasFocus()
  if (prefs.notify && away && typeof Notification !== 'undefined' && Notification.permission === 'granted') {
    try {
      const n = new Notification('Pomodoro terminado', {
        body: `Tu ${name.toLowerCase()} está lista para cosechar.`,
        tag: `pomofarm-${c.pomodoroId}`, // one notification per Pomodoro, never duplicates
        silent: true, // we play our own chime (or the player muted sound)
      })
      n.onclick = () => {
        window.focus()
        n.close()
      }
    } catch {
      /* some browsers only allow notifications from a service worker: the in-page toast remains */
    }
  }
}

/** The optional rest ended by itself: a gentle invitation, never a demand. */
export function announceRestEnd(): void {
  useUi.getState().toast('Descanso terminado. Cuando quieras, siembra otro Pomodoro.')
  const prefs = usePrefs.getState()
  if (prefs.sound && !prefs.muted) playBowl(0.7)
  if (prefs.muted || !prefs.sound) useUi.getState().showFlash('Descanso terminado')
  const away = document.hidden || !document.hasFocus()
  if (prefs.notify && away && typeof Notification !== 'undefined' && Notification.permission === 'granted') {
    try {
      const n = new Notification('Descanso terminado', { body: 'Cuando quieras, siembra otro Pomodoro.', tag: 'pomofarm-rest', silent: true })
      n.onclick = () => {
        window.focus()
        n.close()
      }
    } catch {
      /* the in-page toast remains */
    }
  }
}
