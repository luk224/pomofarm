import { useEffect } from 'react'
import { useGame } from '../store/game'
import { useRemainingMs } from '../store/hooks'
import { tabTitle } from '../store/time'
import { DropIcon } from './icons'

export function TopBar() {
  const state = useGame((s) => s.state)
  const ms = useRemainingMs()
  const pomodoro = state?.pomodoro ?? null

  useEffect(() => {
    document.title = tabTitle(pomodoro, ms ?? 0)
  }, [pomodoro, ms])

  if (!state) return null
  return (
    <div className="topbar">
      <div className="chip" data-testid="focus" role="status" aria-label={`${state.player.focus_points} gotas de enfoque`}>
        <DropIcon size={18} />
        <span>{state.player.focus_points}</span>
      </div>
    </div>
  )
}
