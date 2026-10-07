// Runs in a Web Worker: worker timers are not throttled in background tabs the way page timers are,
// so the end-of-Pomodoro check fires on time even when the tab is hidden.
const w = self as unknown as { onmessage: (e: MessageEvent) => void; postMessage: (m: unknown) => void }
let handle: ReturnType<typeof setTimeout> | undefined

w.onmessage = (e: MessageEvent<{ type: 'set'; ms: number } | { type: 'clear' }>) => {
  clearTimeout(handle)
  if (e.data.type === 'set') handle = setTimeout(() => w.postMessage('fire'), Math.max(0, e.data.ms))
}
