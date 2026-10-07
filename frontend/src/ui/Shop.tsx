import { useEffect, useRef, useState } from 'react'
import { useGame } from '../store/game'
import { formatPrice } from '../store/automation'
import { buyDog, buyPlot, startPlacing, unlockAnimal, upgradeSilo } from './actions'
import { CoinIcon, DropIcon, SiloIcon } from './icons'

function PlotIcon() {
  return (
    <svg width="22" height="22" viewBox="0 0 22 22" aria-hidden="true" focusable="false">
      <path d="M11 3.5 19 8v6.5L11 19l-8-4.5V8z" fill="#8a5a3b" stroke="#6f4429" strokeWidth="1.2" />
      <path d="M6.5 10.5l4.5 2.5 4.5-2.5M6.5 13l4.5 2.5 4.5-2.5" stroke="#6f4429" strokeWidth="1.1" fill="none" />
      <path d="M11 4.2v-2M9.8 2.9h2.4" stroke="#2f7d32" strokeWidth="1.3" strokeLinecap="round" />
    </svg>
  )
}

interface OfferProps {
  icon: React.ReactNode
  title: string
  detail: string
  cost: number
  /** What the player has of the currency this costs. */
  funds: number
  currency?: 'drop' | 'coin'
  onBuy: () => void
  testId: string
}

function Offer({ icon, title, detail, cost, funds, currency = 'drop', onBuy, testId }: OfferProps) {
  const missing = cost - funds
  const Icon = currency === 'drop' ? DropIcon : CoinIcon
  const word = currency === 'drop' ? 'gotas' : 'monedas'
  return (
    <div className="offer">
      <span className="offer__icon" aria-hidden="true">{icon}</span>
      <div className="offer__text">
        <strong>{title}</strong>
        <span>{detail}</span>
        {missing > 0 && <span className="offer__missing">Te faltan {formatPrice(missing)} <Icon size={12} /></span>}
      </div>
      <button type="button" className="btn btn--primary offer__buy" data-testid={testId} disabled={missing > 0} onClick={onBuy}
        aria-label={`Comprar ${title} por ${cost} ${word}`}>
        {formatPrice(cost)} <Icon size={14} />
      </button>
    </div>
  )
}

/** The farm's shop: a new plot and the next Silo level, both paid with 💧 (GDD §4.4, §4.5). */
export function Shop() {
  const [open, setOpen] = useState(false)
  const shop = useGame((s) => s.state?.shop)
  const focus = useGame((s) => s.state?.player.focus_points ?? 0)
  const silo = useGame((s) => s.state?.silo)
  const auto = useGame((s) => s.state?.automation)
  const coins = useGame((s) => Math.floor((s.state?.player.coins_milli ?? 0) / 1000))
  const root = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!open) return
    const onKey = (e: KeyboardEvent) => e.key === 'Escape' && setOpen(false)
    const onDown = (e: PointerEvent) => !root.current?.contains(e.target as Node) && setOpen(false)
    window.addEventListener('keydown', onKey)
    window.addEventListener('pointerdown', onDown)
    return () => {
      window.removeEventListener('keydown', onKey)
      window.removeEventListener('pointerdown', onDown)
    }
  }, [open])

  if (!shop || !silo || !auto) return null
  const affordable = (shop.next_plot && focus >= shop.next_plot.cost) || (shop.silo_upgrade && focus >= shop.silo_upgrade.cost)
  return (
    <div className="settings" ref={root}>
      <button type="button" className={`chip chip--button${affordable ? ' chip--attention' : ''}`} data-testid="shop-button"
        aria-expanded={open} aria-controls="shop-panel" onClick={() => setOpen(!open)}>
        <PlotIcon />
        <span className="chip__label">Mejoras</span>
      </button>
      {open && (
        <div id="shop-panel" className="panelbox panelbox--wide" role="group" aria-label="Mejoras de la granja" data-testid="shop-panel">
          {shop.next_plot ? (
            <Offer testId="buy-plot" icon={<PlotIcon />} title={`Parcela ${shop.next_plot.number}`}
              detail={`Tienes ${shop.plots_owned} de ${shop.plots_max}`} cost={shop.next_plot.cost} funds={focus}
              onBuy={() => void buyPlot()} />
          ) : (
            <p className="hint">Tienes las {shop.plots_max} parcelas.</p>
          )}
          {shop.silo_upgrade ? (
            <Offer testId="upgrade-silo" icon={<SiloIcon size={22} />} title="Ampliar el Silo"
              detail={`${silo.capacity_hours} h → ${shop.silo_upgrade.capacity_hours} h de producción`} cost={shop.silo_upgrade.cost} funds={focus}
              onBuy={() => void upgradeSilo()} />
          ) : (
            <p className="hint">El Silo está al máximo ({silo.capacity_hours} h).</p>
          )}
          <p className="hint offer__group">Con 🪙 del Silo</p>
          {!auto.bees.unlocked ? (
            <Offer testId="unlock-bees" icon={<span>🐝</span>} title="Desbloquear las Abejas"
              detail="Luego compras colmenas con 🪙" cost={auto.bees.unlock_cost} funds={focus} onBuy={() => void unlockAnimal('bees')} />
          ) : auto.bees.next_cost !== null ? (
            <Offer testId="buy-hive" icon={<span>🍯</span>} title={`Colmena ${auto.bees.hives.length + 1}`}
              detail={`+25% a las 9 parcelas de alrededor. Tienes ${auto.bees.hives.length} de ${auto.bees.max}`}
              cost={auto.bees.next_cost} funds={coins} currency="coin"
              onBuy={() => { setOpen(false); startPlacing(null) }} />
          ) : (
            <p className="hint">Tienes las {auto.bees.max} colmenas.</p>
          )}
          {auto.bees.hives.length > 0 && (
            <p className="hint">Toca una colmena en la granja para moverla gratis.</p>
          )}
          {!auto.dog.unlocked ? (
            <Offer testId="unlock-dog" icon={<span>🐕</span>} title="Desbloquear el Perro Pastor"
              detail="Recoge el Silo solo" cost={auto.dog.unlock_cost} funds={focus} onBuy={() => void unlockAnimal('dog')} />
          ) : !auto.dog.owned ? (
            <Offer testId="buy-dog" icon={<span>🐕</span>} title="Perro Pastor"
              detail={`Recoge el Silo solo y guarda ${auto.dog.silo_bonus_hours} h más`}
              cost={auto.dog.cost} funds={coins} currency="coin" onBuy={() => void buyDog()} />
          ) : (
            <p className="hint">El Perro Pastor vigila el Silo.</p>
          )}
        </div>
      )}
    </div>
  )
}
