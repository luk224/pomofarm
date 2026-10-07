import { useFrame } from '@react-three/fiber'
import { useMemo } from 'react'
import { Vector3 } from 'three'
import { useGame } from '../store/game'
import { useRingAnchor } from '../store/anchor'
import { growthFraction, remainingMs } from '../store/time'
import { isPlantKind, PLANT_HEIGHT } from './plants/kinds'

/**
 * Projects the point above the active plant to screen coordinates each frame and hands them to the
 * DOM timer ring. Renders nothing; the ring itself is plain React/DOM (ui/TimerRing).
 */
export function RingAnchor() {
  const v = useMemo(() => new Vector3(), [])
  useFrame(({ camera, size }) => {
    const { state, fetchedAt } = useGame.getState()
    const { pomodoro, plots } = state ?? {}
    const plot = pomodoro ? plots?.find((p) => p.id === pomodoro.plot_id) : undefined
    const set = useRingAnchor.getState().set
    if (!plot || !isPlantKind(plot.plant_type)) return set(false, 0, 0)
    // Float just above the plant as it is NOW (it grows from 30% to 100% of its height).
    const grown = 0.3 + 0.7 * growthFraction(pomodoro!.planned_s, remainingMs(pomodoro!, fetchedAt, performance.now()))
    v.set(plot.x, 0.08 + PLANT_HEIGHT[plot.plant_type] * grown + 0.35, plot.y).project(camera)
    set(v.z < 1, ((v.x + 1) / 2) * size.width, ((1 - v.y) / 2) * size.height)
  })
  return null
}
