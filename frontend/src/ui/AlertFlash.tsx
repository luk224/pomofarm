import { useUi } from '../store/ui'

/** Visual alert (GDD §2.4): a golden frame and a label when a Pomodoro or rest ends and the sound is off. */
export function AlertFlash() {
  const flash = useUi((s) => s.flash)
  if (!flash) return null
  return (
    <>
      <div className="flash" aria-hidden="true" data-testid="alert-flash" />
      <div className="flash__label" role="alert" data-testid="alert-flash-label">{flash.text}</div>
    </>
  )
}
