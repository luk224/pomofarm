import { useEffect, useState } from 'react'
import { useGame } from '../store/game'
import { formatCoins, siloFill, siloNow } from '../store/economy'
import { collectSilo } from './actions'
import { CoinIcon, SiloIcon } from './icons'

/** Ticks once a second while the Silo is filling, only to repaint; the server holds the real amount. */
function useSiloAmount(): number {
  const silo = useGame((s) => s.state?.silo)
  const fetchedAt = useGame((s) => s.fetchedAt)
  const [now, setNow] = useState(() => performance.now())
  const filling = !!silo && !silo.full && silo.rate_milli_per_h > 0
  useEffect(() => {
    if (!filling) return
    const id = window.setInterval(() => setNow(performance.now()), 1000)
    return () => window.clearInterval(id)
  }, [filling])
  return silo ? siloNow(silo, fetchedAt, Math.max(now, fetchedAt)) : 0
}

/** Shown once the farm has produced or can produce 🪙. Collecting is a deliberate tap (GDD §3.4). */
export function SiloPanel() {
  const silo = useGame((s) => s.state?.silo)
  const amount = useSiloAmount()
  if (!silo || (silo.rate_milli_per_h <= 0 && silo.content_milli <= 0)) return null
  const fill = siloFill(amount, silo.capacity_milli)
  const collectable = Math.floor(amount) >= 1
  return (
    <section className={`silo${silo.full ? ' silo--full' : ''}`} aria-label="Silo" data-testid="silo">
      <div className="silo__head">
        <SiloIcon />
        <span className="silo__amount" data-testid="silo-amount">
          {formatCoins(amount)}<small> / {formatCoins(silo.capacity_milli)}</small>
        </span>
        <CoinIcon size={16} />
      </div>
      <div className="silo__bar" role="meter" aria-label="Contenido del Silo" aria-valuemin={0}
        aria-valuemax={Math.round(silo.capacity_milli / 1000)} aria-valuenow={Math.round(amount / 1000)}>
        <span style={{ width: `${fill * 100}%` }} />
      </div>
      <p className="silo__note" data-testid="silo-note">
        {silo.full ? 'Lleno: recoge para seguir produciendo.' : `+${formatCoins(silo.rate_milli_per_h)} por hora`}
      </p>
      <button type="button" className="btn btn--sun" data-testid="silo-collect" disabled={!collectable} onClick={() => void collectSilo()}>
        Recoger
      </button>
    </section>
  )
}
