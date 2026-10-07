import type { ReactElement } from 'react'
import { palette as P } from '../scene/palette'

/** Seed illustrations: same shapes and colours as the 3D plants, so the packet shows what you will grow. */
const ART: Record<string, ReactElement> = {
  daisy: (
    <>
      <rect x="19" y="26" width="2.4" height="12" rx="1" fill={P.stem} />
      <ellipse cx="25" cy="33" rx="4.5" ry="2.5" fill={P.leaf} transform="rotate(-25 25 33)" />
      {Array.from({ length: 8 }, (_, i) => (
        <circle key={i} cx={20 + Math.cos((i / 8) * 6.283) * 7} cy={19 + Math.sin((i / 8) * 6.283) * 7} r="4.2" fill="#fff" stroke="#d9d4c3" strokeWidth="0.8" />
      ))}
      <circle cx="20" cy="19" r="4.6" fill={P.yellow} />
    </>
  ),
  tomato: (
    <>
      <circle cx="20" cy="24" r="12" fill={P.red} />
      <path d="M12 15 L20 19 L28 15 L24 21 L20 17 L16 21 Z" fill={P.leaf} />
      <rect x="19" y="10" width="2.4" height="6" rx="1" fill={P.stem} />
      <circle cx="15.5" cy="22" r="2" fill="#fff" opacity="0.45" />
    </>
  ),
  sunflower: (
    <>
      <rect x="19" y="26" width="2.6" height="12" rx="1" fill={P.stem} />
      <ellipse cx="14.5" cy="32" rx="4.5" ry="2.4" fill={P.leaf} transform="rotate(30 14.5 32)" />
      {Array.from({ length: 12 }, (_, i) => (
        <ellipse key={i} cx="20" cy="9.5" rx="2.6" ry="5" fill={P.yellow} transform={`rotate(${i * 30} 20 19)`} />
      ))}
      <circle cx="20" cy="19" r="6.2" fill={P.brown} />
    </>
  ),
  apple: (
    <>
      <rect x="18.2" y="24" width="3.6" height="13" rx="1.4" fill={P.trunk} />
      <circle cx="20" cy="17" r="11.5" fill={P.canopy} />
      <circle cx="26" cy="21" r="6" fill={P.leafDark} />
      <circle cx="15" cy="14" r="2.2" fill={P.red} />
      <circle cx="23.5" cy="12" r="2.2" fill={P.red} />
      <circle cx="19" cy="21" r="2.2" fill={P.red} />
    </>
  ),
  oak: (
    <>
      <rect x="17.4" y="24" width="5.2" height="13" rx="1.8" fill={P.trunk} />
      <circle cx="20" cy="16.5" r="13" fill={P.oakCanopy} />
      <circle cx="12" cy="20" r="6" fill={P.canopy} />
      <path d="M7 8 l1.6 3.2 3.2 1.6 -3.2 1.6 -1.6 3.2 -1.6 -3.2 -3.2 -1.6 3.2 -1.6z" fill={P.magic} transform="translate(2 -3) scale(.8)" />
      <path d="M33 14 l1.2 2.4 2.4 1.2 -2.4 1.2 -1.2 2.4 -1.2 -2.4 -2.4 -1.2 2.4 -1.2z" fill={P.magic} />
    </>
  ),
}

export function SeedArt({ kind, size = 44 }: { kind: string; size?: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 40 40" aria-hidden="true" focusable="false">
      {ART[kind] ?? null}
    </svg>
  )
}

export function DropIcon({ size = 16 }: { size?: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 16 16" aria-hidden="true" focusable="false">
      <path d="M8 1.2C8 1.2 3 6.4 3 9.8a5 5 0 0 0 10 0C13 6.4 8 1.2 8 1.2z" fill="#1f84c9" />
      <path d="M6 10.2a2.2 2.2 0 0 0 1.8 2" stroke="#fff" strokeWidth="1.2" strokeLinecap="round" fill="none" opacity="0.8" />
    </svg>
  )
}
