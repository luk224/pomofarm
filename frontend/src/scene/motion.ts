import { useEffect, useState } from 'react'

const QUERY = '(prefers-reduced-motion: reduce)'

/** True when the user asked the OS for reduced motion; a settings toggle joins this in P4-04. */
export function useReducedMotion(): boolean {
  const [reduce, setReduce] = useState(() => (typeof matchMedia === 'function' ? matchMedia(QUERY).matches : false))
  useEffect(() => {
    if (typeof matchMedia !== 'function') return
    const mq = matchMedia(QUERY)
    const on = () => setReduce(mq.matches)
    mq.addEventListener('change', on)
    return () => mq.removeEventListener('change', on)
  }, [])
  return reduce
}

/** Squash-and-stretch multiplier for a "pop" that started `t` seconds ago (1 at rest). Pure, for tests. */
export function popScale(t: number): { y: number; xz: number } {
  if (t < 0 || t > 0.8) return { y: 1, xz: 1 }
  const k = Math.exp(-7 * t) * Math.cos(16 * t)
  return { y: 1 + 0.3 * k, xz: 1 - 0.15 * k } // taller+thinner, then wider+shorter, settling
}
