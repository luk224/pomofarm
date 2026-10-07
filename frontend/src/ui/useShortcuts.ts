import { useEffect } from 'react'
import { useGame } from '../store/game'
import { collectSilo, harvestableId, harvestPlot, plantSelected, togglePause } from './actions'

const TYPING = new Set(['INPUT', 'TEXTAREA', 'SELECT'])

/** Space pauses/resumes, Enter plants the chosen seed, H harvests, C empties the Silo (GDD §2.4). Never fires while typing. */
export function useShortcuts() {
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      const el = e.target as HTMLElement | null
      if (e.ctrlKey || e.metaKey || e.altKey || (el && (TYPING.has(el.tagName) || el.isContentEditable))) return
      const onButton = el?.tagName === 'BUTTON'
      if (e.code === 'Space' && !onButton && useGame.getState().state?.pomodoro) {
        e.preventDefault()
        void togglePause()
      } else if (e.key === 'Enter' && !onButton) {
        void plantSelected()
      } else if (e.key.toLowerCase() === 'c') {
        if ((useGame.getState().state?.silo.content_milli ?? 0) >= 1) void collectSilo()
      } else if (e.key.toLowerCase() === 'h') {
        const id = harvestableId()
        if (id !== null) void harvestPlot(id)
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [])
}
