// How long a board action takes from the click to the changed board on
// screen, kept for checking that flavor never slows one (the last 50, on
// globalThis.bakeryTimings). Measuring waits for the next frame and never
// holds up the action.

import { tick } from 'svelte'

type Timing = { action: string; ms: number }
const kept: Timing[] = []
;(globalThis as { bakeryTimings?: Timing[] }).bakeryTimings = kept

export function measure(action: string) {
  const start = performance.now()
  tick().then(() =>
    requestAnimationFrame(() => {
      kept.push({ action, ms: performance.now() - start })
      if (kept.length > 50) kept.shift()
    }),
  )
}
