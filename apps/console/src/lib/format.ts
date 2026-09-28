// Small formatting helpers for the console's tables.

// untilAfter is the end of a suspension that lasts this many days from now,
// as an RFC 3339 time.
export function untilAfter(days: number, now = new Date()): string {
  return new Date(now.getTime() + days * 24 * 60 * 60 * 1000).toISOString()
}

// when prints a time in UTC, to the minute: operators compare them with the
// audit log across time zones.
export function when(iso: string | null | undefined): string {
  if (!iso) return ''
  const d = new Date(iso)
  if (Number.isNaN(d.getTime()) || d.getUTCFullYear() < 2000) return ''
  return d.toISOString().slice(0, 16).replace('T', ' ') + ' UTC'
}

// targetPath is the console page of a report's or an entry's target.
export function targetPath(kind: string, id: number): string {
  if (kind === 'member') return `/members/${id}`
  if (kind === 'guild') return `/guilds/${id}`
  return ''
}

// sanctionLine says what a sanction does, for a table cell.
export function sanctionLine(s: { kind: string; until: string | null; active: boolean; lifted_at: string | null }): string {
  const what = s.kind === 'ban' ? 'Banned' : `Suspended until ${when(s.until)}`
  if (s.lifted_at) return `${what} (lifted ${when(s.lifted_at)})`
  if (!s.active) return `${what} (ended)`
  return what
}
