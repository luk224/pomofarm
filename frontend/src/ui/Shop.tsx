import { useEffect, useRef, useState } from 'react'
import { useGame } from '../store/game'
import { buyPlot, upgradeSilo } from './actions'
import { DropIcon, SiloIcon } from './icons'

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
  focus: number
  onBuy: () => void
  testId: string
}

function Offer({ icon, title, detail, cost, focus, onBuy, testId }: OfferProps) {
  const missing = cost - focus
  return (
    <div className="offer">
      <span className="offer__icon" aria-hidden="true">{icon}</span>
      <div className="offer__text">
        <strong>{title}</strong>
        <span>{detail}</span>
        {missing > 0 && <span className="offer__missing">Te faltan {missing} <DropIcon size={12} /></span>}
      </div>
      <button type="button" className="btn btn--primary offer__buy" data-testid={testId} disabled={missing > 0} onClick={onBuy}
        aria-label={`Comprar ${title} por ${cost} gotas`}>
        {cost} <DropIcon size={14} />
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

  if (!shop || !silo) return null
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
              detail={`Tienes ${shop.plots_owned} de ${shop.plots_max}`} cost={shop.next_plot.cost} focus={focus}
              onBuy={() => void buyPlot()} />
          ) : (
            <p className="hint">Tienes las {shop.plots_max} parcelas.</p>
          )}
          {shop.silo_upgrade ? (
            <Offer testId="upgrade-silo" icon={<SiloIcon size={22} />} title="Ampliar el Silo"
              detail={`${silo.capacity_hours} h → ${shop.silo_upgrade.capacity_hours} h de producción`} cost={shop.silo_upgrade.cost} focus={focus}
              onBuy={() => void upgradeSilo()} />
          ) : (
            <p className="hint">El Silo está al máximo ({silo.capacity_hours} h).</p>
          )}
        </div>
      )}
    </div>
  )
}
