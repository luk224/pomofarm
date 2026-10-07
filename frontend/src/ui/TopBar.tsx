import { useEffect } from 'react'
import { useGame } from '../store/game'
import { useRemainingMs } from '../store/hooks'
import { tabTitle } from '../store/time'
import { formatCoins } from '../store/economy'
import { CoinIcon, DropIcon } from './icons'
import { Settings } from './Settings'
import { Shop } from './Shop'

export function TopBar() {
  const state = useGame((s) => s.state)
  const ms = useRemainingMs()
  const pomodoro = state?.pomodoro ?? null
  const ready = !!state?.plots.some((p) => p.state === 'mature' && !p.harvested)

  useEffect(() => {
    document.title = tabTitle(pomodoro, ms ?? 0, ready)
  }, [pomodoro, ms, ready])

  if (!state) return null
  return (
    <div className="topbar">
      <div className="chip" data-testid="focus" role="status" aria-label={`${state.player.focus_points} gotas de enfoque`}>
        <DropIcon size={18} />
        <span>{state.player.focus_points}</span>
      </div>
      {(state.player.coins_milli > 0 || state.silo.rate_milli_per_h > 0 || state.silo.content_milli > 0) && (
        <div className="chip" data-testid="coins" role="status" aria-label={`${formatCoins(state.player.coins_milli)} monedas`}>
          <CoinIcon size={18} />
          <span>{formatCoins(state.player.coins_milli)}</span>
        </div>
      )}
      <Shop />
      <Settings />
    </div>
  )
}
