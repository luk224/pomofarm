import type { SeedState } from '../api/types'
import { DropIcon, SeedArt } from './icons'
import { SEED_NAMES } from './names'

interface Props {
  seed: SeedState
  tilt: number
  selected: boolean
  canAfford: boolean
  onSelect: () => void
  onUnlock: () => void
}

/** A seed packet: kraft-paper envelope with folded flap. Locked ones show the unlock price. */
export function SeedPacket({ seed, tilt, selected, canAfford, onSelect, onUnlock }: Props) {
  const name = SEED_NAMES[seed.key] ?? seed.key
  const pair = seed.compatible.map((k) => SEED_NAMES[k] ?? k)
  return (
    <div className={`packet${selected ? ' packet--selected' : ''}${seed.unlocked ? '' : ' packet--locked'}`}
      style={{ ['--tilt' as string]: `${tilt}deg` }}>
      <button type="button" role="radio" aria-checked={selected} disabled={!seed.unlocked} className="packet__body"
        onClick={onSelect} aria-label={`${name}, ${seed.key === 'oak' ? 'de 60 a 120 minutos' : `${seed.duration_min} minutos`}, recompensa ${seed.reward} gotas, combina con ${pair.join(' y ')}${seed.unlocked ? '' : ', bloqueada'}`}>
        <span className="packet__flap" aria-hidden="true" />
        <SeedArt kind={seed.key} />
        <span className="packet__name">{name}</span>
        <span className="packet__pair" title={`Combina con ${pair.join(' y ')}`} aria-hidden="true">
          {seed.compatible.map((k) => <SeedArt key={k} kind={k} size={17} />)}
        </span>
        <span className="packet__meta">{seed.key === 'oak' ? '60–120 min' : `${seed.duration_min} min`}</span>
      </button>
      {seed.unlocked ? (
        <span className="packet__stamp" aria-hidden="true">+{seed.reward}<DropIcon size={11} /></span>
      ) : (
        <button type="button" className="packet__unlock" disabled={!canAfford} onClick={onUnlock}
          aria-label={`Desbloquear ${name} por ${seed.unlock_cost} gotas`}>
          Desbloquear {seed.unlock_cost}<DropIcon size={12} />
        </button>
      )}
    </div>
  )
}
