import type { PlantKind } from './plants/kinds'
import { PlantView } from './plants/PlantView'

const KINDS: PlantKind[] = ['daisy', 'tomato', 'sunflower', 'apple', 'oak']
const STAGES = [
  { key: 'sprout', growth: 0.05, mature: false, withered: false },
  { key: 'growing', growth: 0.6, mature: false, withered: false },
  { key: 'mature', growth: 1, mature: true, withered: false },
  { key: 'withered', growth: 1, mature: true, withered: true },
] as const

/** Dev-only (?lab=plants): every plant at every stage, to eyeball them and count triangles. */
export function Gallery() {
  return (
    <>
      {KINDS.map((k, i) =>
        STAGES.map((s, j) => (
          <group key={`${k}-${s.key}`} position={[i * 1.7, 0, j * 1.7]} name={`gallery-${k}-${s.key}`}>
            <PlantView kind={k} growth={s.growth} mature={s.mature} withered={s.withered} ready={s.key === 'mature'} />
          </group>
        )),
      )}
    </>
  )
}
