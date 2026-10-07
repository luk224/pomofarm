import { useEffect } from 'react'
import { formatPrice } from '../store/automation'
import { useGame } from '../store/game'
import { useUi } from '../store/ui'
import { DECOR_NAMES, removeDecorPiece, stopDecor } from './actions'

/**
 * Decorating: tap a free cell to place the chosen piece (cheap, so one tap is enough and you can keep going);
 * tap a placed piece to move it or take it away. Escape or "Terminar" ends it.
 */
export function DecorBar() {
  const mode = useUi((s) => s.decorMode)
  const cost = useGame((s) => s.state?.decor.catalog.find((c) => c.kind === mode?.piece)?.cost)
  const coins = useGame((s) => Math.floor((s.state?.player.coins_milli ?? 0) / 1000))

  useEffect(() => {
    if (!mode) return
    const onKey = (e: KeyboardEvent) => e.key === 'Escape' && stopDecor()
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [mode])

  if (!mode || cost === undefined) return null
  const moving = mode.itemId !== null
  const name = DECOR_NAMES[mode.piece] ?? mode.piece
  return (
    <section className="placing" role="group" aria-label="Decorar" data-testid="decor-bar">
      <p className="placing__text" data-testid="decor-text">
        {moving ? `Toca un sitio libre para mover: ${name}. Moverla es gratis.` : `${name}: ${formatPrice(cost)} 🪙 cada una. Toca un sitio libre.`}
        {!moving && coins < cost ? ' Te faltan 🪙.' : ''}
      </p>
      <div className="placing__buttons">
        <button type="button" className="btn btn--primary" data-testid="decor-done" onClick={stopDecor}>
          {moving ? 'Cancelar' : 'Terminar'}
        </button>
        {moving && (
          <button type="button" className="btn" data-testid="decor-remove" onClick={() => void removeDecorPiece()}>
            Quitar (sin reembolso)
          </button>
        )}
      </div>
    </section>
  )
}
