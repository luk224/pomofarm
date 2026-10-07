import { useEffect } from 'react'
import { useGame } from '../store/game'
import { useUi } from '../store/ui'
import { messageFor } from './messages'

/** Turns store errors into readable messages, then shows the queue in an aria-live region. */
export function Toasts() {
  const toasts = useUi((s) => s.toasts)
  const dismiss = useUi((s) => s.dismiss)
  const error = useGame((s) => s.error)

  useEffect(() => {
    if (!error) return
    useUi.getState().toast(messageFor(error), 'error')
    useGame.getState().clearError()
  }, [error])

  return (
    <div className="toasts" role="status" aria-live="polite">
      {toasts.map((t) => (
        <button key={t.id} type="button" className={`toast toast--${t.kind}`} onClick={() => dismiss(t.id)}>
          {t.text}
        </button>
      ))}
    </div>
  )
}
