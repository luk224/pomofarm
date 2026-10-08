import { useId } from 'react'
import type { HarvestUnit } from '../store/book'

/** One harvest icon (hay bale, silo or apple basket); `fill` (0..1) draws a partially filled one for the leftover minutes. */
export function HarvestIcon({ unit, fill = 1 }: { unit: HarvestUnit; fill?: number }) {
  const id = useId()
  return (
    <svg className="harvest__icon" width="32" height="32" viewBox="0 0 32 32" aria-hidden="true" focusable="false">
      <defs>
        <clipPath id={id}>
          <rect x="0" y={32 - 32 * fill} width="32" height={32 * fill} />
        </clipPath>
      </defs>
      {fill < 1 && <g opacity="0.22">{art(unit)}</g>}
      <g clipPath={fill < 1 ? `url(#${id})` : undefined}>{art(unit)}</g>
    </svg>
  )
}

function art(unit: HarvestUnit) {
  if (unit === 'bale') {
    return (
      <>
        <rect x="3" y="9" width="26" height="18" rx="4" fill="#e0b84a" stroke="#a8823a" strokeWidth="1.4" />
        <path d="M3 15h26M3 21h26" stroke="#a8823a" strokeWidth="1.1" />
        <path d="M11 9v18M21 9v18" stroke="#8a5a3b" strokeWidth="1.6" />
      </>
    )
  }
  if (unit === 'silo') {
    return (
      <>
        <rect x="8" y="10" width="16" height="19" rx="2" fill="#d9bd85" stroke="#a8874a" strokeWidth="1.4" />
        <path d="M6 11c0-6 20-6 20 0z" fill="#a8553b" stroke="#7a3d2a" strokeWidth="1.2" />
        <path d="M8 16h16M8 22h16" stroke="#a8874a" strokeWidth="1.1" />
        <rect x="14" y="23" width="4" height="6" rx="1" fill="#7a4a2a" />
      </>
    )
  }
  return (
    <>
      <path d="M5 15h22l-3 13H8z" fill="#c9894a" stroke="#8a5a3b" strokeWidth="1.4" />
      <path d="M8 19h16M9 23h14" stroke="#8a5a3b" strokeWidth="1" />
      <circle cx="11" cy="12" r="4.6" fill="#d14b3a" />
      <circle cx="18" cy="11" r="4.6" fill="#e25b45" />
      <circle cx="23" cy="13" r="4" fill="#d14b3a" />
      <path d="M18 6.5c1.4-2 3-2.4 4-2" stroke="#2f7d32" strokeWidth="1.6" fill="none" strokeLinecap="round" />
    </>
  )
}
