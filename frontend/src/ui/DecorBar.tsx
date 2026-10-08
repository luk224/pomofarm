import { useEffect } from 'react'
import { formatPrice } from '../store/automation'
import { useGame } from '../store/game'
import { useUi } from '../store/ui'
import { freeCells } from '../store/decorCursor'
import { DECOR_NAMES, removeDecorPiece, stopDecor } from './actions'

/**
 * Decorating: tap a free cell to place the chosen piece (cheap, so one tap is enough and you can keep going);
 * tap a placed piece to move it or take it away. Escape or "Terminar" ends it.
 */
export function DecorBar() {
  const mode = useUi((s) => s.decorMode)
  const cost = useGame((s) => s.state?.decor.catalog.find((c) => c.kind === mode?.piece)?.cost)
  const cursor = useUi((s) => s.decorCursor)
  const decor = useGame((s) => s.state?.decor)
  const nothingFree = !!decor && freeCells(decor, mode?.itemId ?? null).length === 0
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
        {nothingFree ? ' No quedan sitios libres: quita alguna pieza.' : ''}
      </p>
      <p className="hint" data-testid="decor-keys">
        Teclado: flechas para mover el cursor, Intro para {moving ? 'mover aquí' : 'colocar'}, Esc para salir.
        <span className="sr-only" role="status" aria-live="polite" data-testid="decor-cursor-text">
          {cursor ? ` Cursor en la columna ${cursor[0] + 3} y la fila ${cursor[1] + 3} del terreno.` : ''}
        </span>
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
