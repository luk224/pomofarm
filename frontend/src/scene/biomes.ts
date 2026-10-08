/**
 * Biomes (GDD §4.9): each season repaints the sky and the grass. The server decides which biome the farm has
 * (spring, summer, autumn, winter, spring…); this only maps the name to colours. Plants, soil and buildings keep their own
 * colours, so a plant never disappears into the ground.
 */
export type Biome = 'spring' | 'summer' | 'autumn' | 'winter'

export interface BiomePalette {
  name: string
  sky: string
  grassA: string
  grassB: string
}

export const BIOMES: Record<Biome, BiomePalette> = {
  spring: { name: 'Primavera', sky: '#cfe9f5', grassA: '#8fc65a', grassB: '#82bb50' },
  summer: { name: 'Verano dorado', sky: '#fbe6b4', grassA: '#d9c45c', grassB: '#ccb850' },
  autumn: { name: 'Otoño', sky: '#f2d8cc', grassA: '#c98a45', grassB: '#bd7e3b' },
  winter: { name: 'Invierno nevado', sky: '#dce6ee', grassA: '#eef3f7', grassB: '#e0e8ef' },
}

export function isBiome(v: unknown): v is Biome {
  return typeof v === 'string' && Object.hasOwn(BIOMES, v)
}

/** The palette for a biome name from the server; anything unknown reads as spring. */
export function biomePalette(name: unknown): BiomePalette {
  return isBiome(name) ? BIOMES[name] : BIOMES.spring
}
