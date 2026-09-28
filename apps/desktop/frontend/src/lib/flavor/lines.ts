// The text library: every sentence that may be told with a colony's dry
// narration, each with its plain form. A flavored line is picked from its
// variants by a stable key (an activity entry's id, a column's id), so it
// does not change when the page draws again. With quiet colony on, the
// plain form is always used. All lines are written for The Bakery.

import { settings } from '../settings.svelte'

// pick chooses one of variants by key, or plain when the colony is quiet.
export function pick(key: string | number, plain: string, variants: string[]): string {
  if (settings.quiet || variants.length === 0) return plain
  let h = 0x811c9dc5
  for (const ch of String(key)) {
    h ^= ch.charCodeAt(0)
    h = Math.imul(h, 0x01000193) >>> 0
  }
  return variants[h % variants.length]
}

type Words = { who: string; from?: string; to?: string; what?: string; agent?: string; title?: string }

// ACTIVITY is how each kind of task history entry can read.
export const ACTIVITY: Record<string, { plain: (w: Words) => string; flavored: ((w: Words) => string)[] }> = {
  created: {
    plain: (w) => `${w.who} created this in ${w.to}`,
    flavored: [
      (w) => `${w.who} set this down in ${w.to}.`,
      (w) => `${w.who} drew this up and left it in ${w.to}.`,
      (w) => `${w.who} brought this into the world, in ${w.to}.`,
    ],
  },
  edited: {
    plain: (w) => `${w.who} changed ${w.what}`,
    flavored: [
      (w) => `${w.who} tinkered with ${w.what}.`,
      (w) => `${w.who} had second thoughts about ${w.what}.`,
      (w) => `${w.who} reworded ${w.what}.`,
    ],
  },
  moved: {
    plain: (w) => `${w.who} moved this from ${w.from} to ${w.to}`,
    flavored: [
      (w) => `${w.who} hauled this to ${w.to}.`,
      (w) => `${w.who} carried this from ${w.from} to ${w.to}.`,
      (w) => `${w.who} dragged this over to ${w.to}.`,
      (w) => `${w.who} decided this belongs in ${w.to}.`,
    ],
  },
  reordered: {
    plain: (w) => `${w.who} reordered this in ${w.to}`,
    flavored: [(w) => `${w.who} shuffled this along ${w.to}.`, (w) => `${w.who} gave this a new place in the queue.`],
  },
  commented: {
    plain: (w) => `${w.who} commented`,
    flavored: [(w) => `${w.who} had something to say.`, (w) => `${w.who} left a note.`, (w) => `${w.who} weighed in.`],
  },
  subtask_added: {
    plain: (w) => `${w.who} added the subtask ${w.title}`,
    flavored: [(w) => `${w.who} added ${w.title} to the list of chores.`, (w) => `${w.who} noted another job: ${w.title}.`],
  },
  subtask_done: {
    plain: (w) => `${w.who} ticked off ${w.title}`,
    flavored: [(w) => `${w.who} finished ${w.title}. One fewer.`, (w) => `${w.who} crossed off ${w.title}.`],
  },
  run_started: {
    plain: (w) => `${w.who} put ${w.agent} to work on this`,
    flavored: [(w) => `${w.who} sent ${w.agent} to the workbench.`, (w) => `${w.who} handed this to ${w.agent}.`],
  },
  run_succeeded: {
    plain: (w) => `${w.agent} finished working on this`,
    flavored: [(w) => `${w.agent} finished the job and put down the tools.`, (w) => `${w.agent} is done here.`],
  },
  run_failed: {
    plain: (w) => `${w.agent} could not finish this`,
    flavored: [(w) => `${w.agent} gave up on this, for now.`, (w) => `${w.agent} walked away from the bench, defeated.`],
  },
  run_stopped: {
    plain: (w) => `${w.agent} was stopped`,
    flavored: [(w) => `${w.agent} was called off the job.`, (w) => `${w.agent} was told to down tools.`],
  },
}

// EMPTY is what an empty place says.
export const EMPTY = {
  column: { plain: 'No tasks here.', flavored: ['Nothing here but dust.', 'An empty stretch of floor.', 'Quiet. For now.'] },
  board: {
    plain: 'No boards yet. Add one on the left.',
    flavored: ['No boards yet. The colony waits for a plan; add one on the left.', 'Bare ground. Add a board on the left to start building.'],
  },
  runs: {
    plain: 'No agent has worked on this yet.',
    flavored: ['No agent has touched this yet.', 'Untouched by any colonist so far.'],
  },
  roster: {
    plain: 'No one has joined the colony yet.',
    flavored: ['No one has joined the colony yet. It is very quiet.', 'The barracks stand empty.'],
  },
  pickAgents: {
    plain: 'No agents yet. Make one on the Agents screen.',
    flavored: ['No colonists to send. Make one on the Agents screen.'],
  },
}

// empty says what an empty place says, keyed so it stays put.
export function empty(place: keyof typeof EMPTY, key: string | number = place): string {
  return pick(key, EMPTY[place].plain, EMPTY[place].flavored)
}

// NOTES is what an agent tells the supervisor when it hands in its notes.
const NOTES: Record<string, { plain: (w: { to: string }) => string; flavored: ((w: { to: string }) => string)[] }> = {
  succeeded: {
    plain: (w) => (w.to ? `Done, moved to ${w.to}` : 'Done'),
    flavored: [
      (w) => (w.to ? `All done. It's in ${w.to} now.` : 'All done.'),
      (w) => (w.to ? `Finished. Left it in ${w.to}.` : 'Finished.'),
      () => 'Job done, boss.',
    ],
  },
  failed: {
    plain: () => 'Could not finish it',
    flavored: [() => 'It beat me, this time.', () => "Couldn't crack it.", () => 'No luck. Notes are on the task.'],
  },
  stopped: {
    plain: () => 'Was called off',
    flavored: [() => 'Called off. Back to the yard.', () => 'Stopped halfway.'],
  },
}

// notesLine is an agent's notes on a finished run, keyed by the run so it
// stays the same; cost is added as it is.
export function notesLine(runId: string, status: string, movedTo: string, cost: number): string {
  const entry = NOTES[status]
  if (!entry) return ''
  const w = { to: movedTo }
  const line = pick(runId, entry.plain(w), entry.flavored.map((f) => f(w)))
  return cost ? `${line} · $${cost.toFixed(2)}` : line
}
