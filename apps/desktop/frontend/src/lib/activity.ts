// How a task's history reads. Every sentence is built in activityLine, from
// the text library in flavor/lines.ts: plain, or with the colony's dry
// narration unless quiet colony is on.

import type { Activity } from './bindings'
import { ACTIVITY, pick } from './flavor/lines'

// Activity keeps each column's name as it was then.
const column = (c: unknown) => String(c ?? '')
const quoted = (s: unknown) => `“${String(s ?? '')}”`

function line(e: Activity, kind: string, w: { who: string; from?: string; to?: string; what?: string; agent?: string; title?: string }): string {
  const entry = ACTIVITY[kind]
  return pick(e.id, entry.plain(w), entry.flavored.map((f) => f(w)))
}

export function activityLine(e: Activity): string {
  const who = e.actor_name || 'Someone'
  const d = e.data ?? {}
  switch (e.kind) {
    case 'created':
      return line(e, 'created', { who, to: column(d.column) })
    case 'edited': {
      const what = [d.title && 'the title', d.description && 'the description'].filter(Boolean).join(' and ')
      return line(e, 'edited', { who, what: what || 'this' })
    }
    case 'moved':
      return d.from === d.to
        ? line(e, 'reordered', { who, to: column(d.to) })
        : line(e, 'moved', { who, from: column(d.from), to: column(d.to) })
    case 'commented':
      return line(e, 'commented', { who })
    case 'subtask_added':
      return line(e, 'subtask_added', { who, title: quoted(d.title) })
    case 'subtask_done':
      return line(e, 'subtask_done', { who, title: quoted(d.title) })
    case 'run_started':
      return line(e, 'run_started', { who, agent: String(d.agent_name ?? 'an agent') })
    case 'run_finished': {
      const agent = String(d.agent_name ?? 'The agent')
      const kind = `run_${String(d.status)}`
      return ACTIVITY[kind] ? line(e, kind, { who, agent }) : `${agent} finished`
    }
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
