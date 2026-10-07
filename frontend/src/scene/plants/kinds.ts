export type PlantKind = 'daisy' | 'tomato' | 'sunflower' | 'apple' | 'oak'

export function isPlantKind(k: string | null): k is PlantKind {
  return k === 'daisy' || k === 'tomato' || k === 'sunflower' || k === 'apple' || k === 'oak'
}

/** Height of each plant when fully grown, to float the sparkle and the timer above it. */
export const PLANT_HEIGHT: Record<PlantKind, number> = { daisy: 0.7, tomato: 0.75, sunflower: 1.15, apple: 1.35, oak: 1.6 }
