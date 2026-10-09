/** True on phones and tablets that have no mouse: the primary input is a finger and nothing can hover. */
export function isTouchOnly(match: (q: string) => { matches: boolean } = (q) => matchMedia(q)): boolean {
  try {
    return match('(hover: none) and (pointer: coarse)').matches
  } catch {
    return false
  }
}
