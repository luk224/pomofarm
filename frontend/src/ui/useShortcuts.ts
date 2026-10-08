import { useEffect } from 'react'
import { canCollect } from '../store/economy'
import { useGame } from '../store/game'
import { usePrefs } from '../store/prefs'
import { useUi } from '../store/ui'
import { collectSilo, decorateAtCursor, harvestableId, harvestPlot, moveDecorCursor, plantSelected, togglePause } from './actions'
import { stepPlot } from './selection'

const TYPING = new Set(['INPUT', 'TEXTAREA', 'SELECT'])

/** Space pauses/resumes, Enter plants the chosen seed, H harvests, C empties the Silo, M mutes everything, arrows pick a plot (GDD §2.4). Never fires while typing. */
export function useShortcuts() {
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      const el = e.target as HTMLElement | null
      if (e.ctrlKey || e.metaKey || e.altKey || (el && (TYPING.has(el.tagName) || el.isContentEditable))) return
      const onButton = el?.tagName === 'BUTTON'
      if (useUi.getState().decorMode) {
        // While decorating, the arrows move a cursor over the free cells and Enter places there.
        const arrows: Record<string, [number, number]> = { ArrowRight: [1, 0], ArrowLeft: [-1, 0], ArrowDown: [0, 1], ArrowUp: [0, -1] }
        if (arrows[e.key]) {
          e.preventDefault()
          moveDecorCursor(...arrows[e.key])
          return
        }
        if (e.key === 'Enter' && !onButton) {
          e.preventDefault()
          void decorateAtCursor()
          return
        }
      }
      if (e.code === 'Space' && !onButton && useGame.getState().state?.pomodoro) {
        e.preventDefault()
        void togglePause()
      } else if (e.key === 'Enter' && !onButton) {
        void plantSelected()
      } else if (e.key === 'ArrowRight' || e.key === 'ArrowLeft' || e.key === 'ArrowDown' || e.key === 'ArrowUp') {
        // Choose which plot the dock talks about, without a mouse. Left/up go back, right/down go forward.
        const plots = useGame.getState().state?.plots ?? []
        const id = stepPlot(plots, useUi.getState().selectedPlotId, e.key === 'ArrowRight' || e.key === 'ArrowDown' ? 1 : -1)
        if (id !== null && plots.length > 1) {
          e.preventDefault()
          useUi.getState().selectPlot(id)
        }
      } else if (e.key.toLowerCase() === 'm') {
        const { muted, setMuted } = usePrefs.getState()
        setMuted(!muted)
        useUi.getState().toast(muted ? 'Audio activado.' : 'Sin audio. Te avisaré con un marco dorado.')
      } else if (e.key.toLowerCase() === 'c') {
        if (canCollect(useGame.getState().state?.silo.content_milli ?? 0)) void collectSilo()
      } else if (e.key.toLowerCase() === 'h') {
        const id = harvestableId()
        if (id !== null) void harvestPlot(id)
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [])
}
