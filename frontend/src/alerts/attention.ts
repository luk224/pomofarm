import { useEffect } from 'react'
import { useGame } from '../store/game'
import { atRiskPlots, attentionMessage, loadMemory, saveMemory, shouldNotify, attentionKey } from '../store/attention'
import { usePrefs } from '../store/prefs'
import { useUi } from '../store/ui'

const CHECK_EVERY_MS = 5 * 60 * 1000

/**
 * "Tu granja necesita atención" (GDD §4): checks, whenever the farm state changes and every few minutes, whether several plants
 * are about to wither, and says so once, softly. Never during a Pomodoro (the player is concentrating), never while another
 * message is on screen, and never with sound. If the tab is hidden and the player allowed notifications, it is a silent system
 * notification; otherwise a small message in the page. The server never pushes anything: this only runs while the app is open.
 */
export function useFarmAttention() {
  useEffect(() => {
    const check = () => {
      if (!usePrefs.getState().farmAttention) return
      const { state, fetchedAt } = useGame.getState()
      if (!state || state.pomodoro) return
      if (useUi.getState().toasts.length > 0) return // do not pile on: it will be looked at again later
      const risk = atRiskPlots(state.plots, state.server_time, fetchedAt, performance.now())
      const memory = loadMemory()
      if (!shouldNotify(risk, memory, Date.now())) return
      saveMemory({ key: attentionKey(risk), at: Date.now() })
      const text = attentionMessage(risk.length)
      const prefs = usePrefs.getState()
      const away = document.hidden || !document.hasFocus()
      if (away && prefs.notify && typeof Notification !== 'undefined' && Notification.permission === 'granted') {
        try {
          const n = new Notification('Tu granja pide un poco de atención', {
            body: `${risk.length} plantas dejarán de producir pronto. Cuando quieras, retíralas y vuelve a sembrar.`,
            tag: 'pomofarm-attention', // replaces itself, never piles up
            silent: true,
          })
          n.onclick = () => {
            window.focus()
            n.close()
          }
          return
        } catch {
          /* some browsers only allow notifications from a service worker: fall back to the message in the page */
        }
      }
      useUi.getState().toast(text)
    }
    check()
    const unsubscribe = useGame.subscribe((s, prev) => {
      if (s.state !== prev.state) check()
    })
    const id = window.setInterval(check, CHECK_EVERY_MS)
    return () => {
      unsubscribe()
      window.clearInterval(id)
    }
  }, [])
}
