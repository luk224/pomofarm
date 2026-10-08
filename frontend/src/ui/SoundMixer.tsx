import { AMBIENT_KINDS, isAmbientKind } from '../audio/ambient'
import { playBowl } from '../audio/bowl'
import { playEffect } from '../audio/effects'
import { unlockAudio, type Layer } from '../audio/engine'
import { usePrefs } from '../store/prefs'

const AMBIENT_LABELS: Record<string, string> = { off: 'Sin ambiente', rain: 'Lluvia', forest: 'Bosque', fire: 'Fuego' }
const LAYER_LABELS: Record<Layer, string> = { ambient: 'Ambiente', effects: 'Efectos', alerts: 'Alertas' }
/** Plays a short sample of the layer, so the player hears what the slider just set. */
const PREVIEWS: Partial<Record<Layer, () => void>> = { effects: () => playEffect('coin'), alerts: () => playBowl(0.5) }

/**
 * The sound mixer, inside the music window: the local ambient sound (rain, forest, fire) and the three independent volumes
 * (ambient, effects, alerts) (GDD §2, ASMR). Saved on this device.
 */
export function SoundMixer() {
  const volumes = usePrefs((s) => s.volumes)
  const ambient = usePrefs((s) => s.ambient)
  const setVolume = usePrefs((s) => s.setVolume)
  const setAmbient = usePrefs((s) => s.setAmbient)
  return (
    <fieldset className="restset" data-testid="sound-settings">
      <legend>Sonido</legend>
      <label className="restfield">
        <span>Ambiente</span>
        <select value={ambient} data-testid="ambient-kind"
          onChange={(e) => { unlockAudio(); if (isAmbientKind(e.target.value)) setAmbient(e.target.value) }}>
          {AMBIENT_KINDS.map((k) => <option key={k} value={k}>{AMBIENT_LABELS[k]}</option>)}
        </select>
      </label>
      {(Object.keys(LAYER_LABELS) as Layer[]).map((l) => (
        <label key={l} className="slider">
          <span>{LAYER_LABELS[l]}</span>
          <input type="range" min={0} max={100} step={1} value={Math.round(volumes[l] * 100)} data-testid={`volume-${l}`}
            aria-label={`Volumen de ${LAYER_LABELS[l].toLowerCase()}`} aria-valuetext={`${Math.round(volumes[l] * 100)} por ciento`}
            onChange={(e) => { unlockAudio(); setVolume(l, Number(e.target.value) / 100) }}
            onPointerUp={() => PREVIEWS[l]?.()} onKeyUp={() => PREVIEWS[l]?.()} />
          <output>{Math.round(volumes[l] * 100)}</output>
        </label>
      ))}
    </fieldset>
  )
}
