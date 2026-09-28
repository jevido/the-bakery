// Sound cues, off by default. Each is preloaded once and played without
// waiting: the action that triggers it never waits for it, and a sound
// that cannot play is simply skipped. Silent with quiet colony on.

import taskDone from '../assets/sounds/task-done.wav'
import letter from '../assets/sounds/letter.wav'
import runFinished from '../assets/sounds/run-finished.wav'
import runFailed from '../assets/sounds/run-failed.wav'
import { settings } from './settings.svelte'

const FILES = { 'task-done': taskDone, letter, 'run-finished': runFinished, 'run-failed': runFailed }
export type Cue = keyof typeof FILES

const cache = new Map<Cue, HTMLAudioElement>()

function audio(cue: Cue): HTMLAudioElement {
  let a = cache.get(cue)
  if (!a) {
    a = new Audio(FILES[cue])
    a.preload = 'auto'
    cache.set(cue, a)
  }
  return a
}

// preload fetches every cue ahead, once sound is on.
export function preload() {
  for (const cue of Object.keys(FILES) as Cue[]) audio(cue)
}

export function play(cue: Cue) {
  if (!settings.sound || settings.quiet) return
  queueMicrotask(() => {
    const a = audio(cue)
    a.volume = settings.volume
    a.currentTime = 0
    a.play().catch(() => {})
  })
}
