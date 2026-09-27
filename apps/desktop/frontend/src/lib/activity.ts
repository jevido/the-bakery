// How a task's history reads. Every sentence is built in activityLine, so
// the wording (later: flavour text) can change in one place.

import type { Activity } from './bindings'

// Activity keeps each column's name as it was then.
const column = (c: unknown) => String(c ?? '')
const quoted = (s: unknown) => `“${String(s ?? '')}”`

export function activityLine(e: Activity): string {
  const who = e.actor_name || 'Someone'
  const d = e.data ?? {}
  switch (e.kind) {
    case 'created':
      return `${who} created this in ${column(d.column)}`
    case 'edited': {
      const what = [d.title && 'the title', d.description && 'the description'].filter(Boolean).join(' and ')
      return `${who} changed ${what || 'this'}`
    }
    case 'moved':
      return d.from === d.to ? `${who} reordered this in ${column(d.to)}` : `${who} moved this from ${column(d.from)} to ${column(d.to)}`
    case 'commented':
      return `${who} commented`
    case 'subtask_added':
      return `${who} added the subtask ${quoted(d.title)}`
    case 'subtask_done':
      return `${who} ticked off ${quoted(d.title)}`
    default:
      return `${who}: ${e.kind}`
  }
}

const relative = new Intl.RelativeTimeFormat(undefined, { numeric: 'auto' })
const steps: [Intl.RelativeTimeFormatUnit, number][] = [
  ['second', 60],
  ['minute', 60],
  ['hour', 24],
  ['day', 7],
  ['week', 4.35],
  ['month', 12],
  ['year', Infinity],
]

// timeAgo says "just now", "3 minutes ago", "yesterday" and so on. The
// caller's now can lag behind (it ticks every half minute), so anything
// newer than 45 seconds, or seemingly in the future, is "just now".
export function timeAgo(at: string, now = Date.now()): string {
  let n = (new Date(at).getTime() - now) / 1000
  if (n > -45) return 'just now'
  for (const [unit, size] of steps) {
    if (Math.abs(n) < size) return relative.format(Math.round(n), unit)
    n /= size
  }
  return ''
}

export function fullTime(at: string): string {
  return new Date(at).toLocaleString(undefined, { dateStyle: 'medium', timeStyle: 'short' })
}
