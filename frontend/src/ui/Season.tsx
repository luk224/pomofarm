import { useEffect, useRef, useState } from 'react'
import { BIOMES, biomePalette } from '../scene/biomes'
import { formatCoins } from '../store/economy'
import { useGame } from '../store/game'
import { useUi } from '../store/ui'
import { playEffect } from '../audio/effects'

const REQ_LABELS: Record<string, string> = { plots: 'Parcelas', silo: 'Silo', hives: 'Colmenas', dog: 'Perro Pastor' }

/** Starts the new season (GDD §4.9): asks for a second tap, then resets and tells the player what changed. */
async function startSeason() {
  const before = useGame.getState().state?.player.season ?? 1
  await useGame.getState().prestige()
  const st = useGame.getState().state
  if (st && st.player.season > before) {
    playEffect('buy')
    useUi.getState().toast(`${biomePalette(st.player.biome).name}: estación ${st.player.season}. Tus 🪙 valen un ${st.prestige.bonus_pct}% más.`)
  }
}

/** "Las Estaciones": what the farm still needs, what starts over, what stays, and the button (two taps, never by accident). */
export function Season({ onDone }: { onDone: () => void }) {
  const p = useGame((s) => s.state?.prestige)
  const [armed, setArmed] = useState(false)
  const timer = useRef<number | undefined>(undefined)
  useEffect(() => () => window.clearTimeout(timer.current), [])
  if (!p) return null
  const next = biomePalette(p.next_biome)
  const press = () => {
    if (!p.ready) return
    if (armed) {
      window.clearTimeout(timer.current)
      setArmed(false)
      onDone()
      void startSeason()
    } else {
      setArmed(true)
      timer.current = window.setTimeout(() => setArmed(false), 6000)
    }
  }
  return (
    <section className="season" aria-label="Nueva estación" data-testid="season">
      <p className="hint offer__group">Estación {p.next_season}: {next.name}</p>
      <p className="hint">Un bioma nuevo y +{p.next_bonus_pct}% de 🪙 para siempre. La granja empieza de nuevo con lo que ya tenías desbloqueado.</p>
      <ul className="season__reqs" aria-label="Requisitos">
        {p.requirements.map((r) => (
          <li key={r.key} className={r.met ? 'season__req season__req--met' : 'season__req'} data-testid={`season-req-${r.key}`}>
            <span aria-hidden="true">{r.met ? '✓' : '○'}</span>
            <span>{REQ_LABELS[r.key] ?? r.key}: {r.have} de {r.need}</span>
            <span className="sr-only">{r.met ? ' (cumplido)' : ' (falta)'}</span>
          </li>
        ))}
      </ul>
      <details className="season__details">
        <summary>Qué se reinicia y qué se queda</summary>
        <p><strong>Empiezan de cero:</strong> plantas, colmenas, Perro, decoración y 🪙{p.would_lose_coins_milli > 0 ? ` (ahora tienes ${formatCoins(p.would_lose_coins_milli)})` : ''}.</p>
        <p><strong>Se queda:</strong> tus 💧, semillas y animales desbloqueados, parcelas, el nivel del Silo y todo el Libro de Cosechas.{p.unharvested > 0 ? ` Las ${p.unharvested} plantas listas se cosechan antes, para no perder sus 💧.` : ''}</p>
      </details>
      <button type="button" className={`btn ${armed ? 'btn--danger' : 'btn--primary'}`} data-testid="start-season" disabled={!p.ready} onClick={press}>
        {armed ? `¿Seguro? Empieza la estación ${p.next_season}` : `Empezar la estación ${p.next_season}`}
      </button>
      {!p.ready && <p className="hint">Cuando la granja esté completa podrás empezar la siguiente estación.</p>}
      <p className="hint" aria-hidden="true">{Object.values(BIOMES).map((b) => b.name).join(' · ')}</p>
    </section>
  )
}
