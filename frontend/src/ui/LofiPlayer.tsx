import { useState } from 'react'
import { chooseStation, setHost, setMusicVolume, stopMusic, togglePauseMusic } from '../audio/lofi'
import { stationsOf, useMusic, type MusicStatus } from '../store/music'
import { usePrefs } from '../store/prefs'
import { isTouchOnly } from './device'
import { SoundMixer } from './SoundMixer'

const STATUS_TEXT: Record<MusicStatus, string> = {
  idle: 'Elige una emisión y pulsa Reproducir.',
  loading: 'Conectando con YouTube…',
  playing: 'Sonando.',
  paused: 'En pausa.',
  fallback: '',
}

function MusicIcon() {
  return (
    <svg width="20" height="20" viewBox="0 0 20 20" aria-hidden="true" focusable="false">
      <path d="M8 3.5v9.2a2.6 2.6 0 1 0 1.6 2.4V7.4l5-1.2v4.5a2.6 2.6 0 1 0 1.6 2.4V3.5z" fill="currentColor" />
    </svg>
  )
}

/** The chip in the top bar that opens the player. */
export function LofiChip() {
  const open = useMusic((s) => s.open)
  const status = useMusic((s) => s.status)
  const setOpen = useMusic((s) => s.setOpen)
  return (
    <button type="button" className={`chip chip--button${status === 'playing' ? ' chip--attention' : ''}`} data-testid="music-button"
      aria-expanded={open} aria-controls="lofi-player" aria-label="Música" onClick={() => setOpen(!open)}>
      <MusicIcon />
      <span className="chip__label">Música</span>
    </button>
  )
}

/**
 * Corner widget (GDD §6.6): YouTube's player for the Lofi Girl streams, with its own volume, a selector, and room for
 * the player's own link. Nothing is requested from YouTube until Reproducir is pressed; if it fails, the local ambient
 * sound takes over and the widget says so.
 */
export function LofiPlayer() {
  const { open, status, failure, custom, selected, volume, setOpen, addCustom, removeCustom } = useMusic()
  const muted = usePrefs((s) => s.muted)
  // Phones ignore the player's volume (the device's own buttons decide it), so the slider would do nothing there
  const [touchOnly] = useState(() => isTouchOnly())
  const [link, setLink] = useState('')
  const [linkError, setLinkError] = useState(false)
  const stations = stationsOf(custom)
  const showVideo = status === 'loading' || status === 'playing' || status === 'paused'

  // The video lives in this panel. Closing the panel must not destroy it, or the music would stop while the store still says
  // "playing" and reopening would show an empty box: so while something is loading, playing or paused the panel stays mounted
  // (out of sight when closed) and only goes away for good once the music is stopped.
  const alive = status !== 'idle'
  if (!open && !alive) return null
  const add = () => {
    const id = addCustom(link)
    setLinkError(id === null)
    if (id) {
      setLink('')
      chooseStation(id)
    }
  }
  return (
    <section id="lofi-player" className={`lofi${open ? '' : ' lofi--closed'}`} inert={!open} role="group" aria-label="Reproductor de música" data-testid="lofi-player">
      <div className={`lofi__video${showVideo ? '' : ' lofi__video--hidden'}`} ref={(el) => setHost(el)} data-testid="lofi-video" />
      <p className="lofi__status" role="status" data-testid="lofi-status" data-status={status}>
        {status === 'fallback'
          ? failure === 'video'
            ? 'Esa emisión no se puede reproducir aquí. Suena el ambiente local.'
            : 'No se pudo conectar con YouTube. Suena el ambiente local.'
          : STATUS_TEXT[status]}
      </p>
      {muted && <p className="hint" data-testid="lofi-muted">Sin audio activado: quítalo en Ajustes o con la tecla M para oír la música.</p>}
      <div className="lofi__row">
        <button type="button" className="btn btn--primary" data-testid="lofi-play" disabled={muted} onClick={togglePauseMusic}>
          {status === 'playing' ? 'Pausa' : status === 'paused' ? 'Reanudar' : status === 'fallback' ? 'Reintentar' : 'Reproducir'}
        </button>
        <button type="button" className="btn" data-testid="lofi-stop" onClick={() => { stopMusic(); setOpen(false) }}>
          {status === 'idle' ? 'Cerrar' : 'Parar'}
        </button>
      </div>
      <label className="restfield lofi__select">
        <span className="sr-only">Emisión</span>
        <select value={selected} data-testid="lofi-station" onChange={(e) => chooseStation(e.target.value)}>
          {stations.map((s) => <option key={s.id} value={s.id}>{s.name}</option>)}
        </select>
      </label>
      {touchOnly ? (
        <p className="hint" data-testid="lofi-volume-hint">Volumen de la música: botones del móvil.</p>
      ) : (
      <label className="slider lofi__volume">
        <span>Volumen</span>
        <input type="range" min={0} max={100} value={Math.round(volume * 100)} data-testid="lofi-volume" aria-label="Volumen de la música"
          onChange={(e) => setMusicVolume(Number(e.target.value) / 100)} />
        <output>{Math.round(volume * 100)}</output>
      </label>
      )}
      <div className="lofi__link">
        <input type="text" value={link} placeholder="Pega un enlace de YouTube" aria-label="Enlace de YouTube propio" data-testid="lofi-link"
          aria-invalid={linkError} onChange={(e) => { setLink(e.target.value); setLinkError(false) }}
          onKeyDown={(e) => e.key === 'Enter' && add()} />
        <button type="button" className="btn" data-testid="lofi-add" onClick={add} disabled={!link.trim()}>Añadir</button>
      </div>
      {linkError && <p className="hint lofi__error" role="alert">Eso no parece un enlace de vídeo de YouTube.</p>}
      <SoundMixer />
      {custom.length > 0 && (
        <ul className="lofi__custom" aria-label="Tus enlaces">
          {custom.map((s) => (
            <li key={s.id}>
              <span>{s.name}</span>
              <button type="button" className="btn" aria-label={`Quitar ${s.name}`} data-testid={`lofi-remove-${s.id}`} onClick={() => removeCustom(s.id)}>Quitar</button>
            </li>
          ))}
        </ul>
      )}
    </section>
  )
}
