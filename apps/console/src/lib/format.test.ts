import { describe, expect, test } from 'bun:test'
import { sanctionLine, targetPath, untilAfter, when } from './format'

describe('format', () => {
  test('untilAfter adds whole days', () => {
    expect(untilAfter(1, new Date('2026-09-28T10:00:00Z'))).toBe('2026-09-29T10:00:00.000Z')
  })

  test('when prints UTC to the minute and hides empty times', () => {
    expect(when('2026-09-28T10:05:59Z')).toBe('2026-09-28 10:05 UTC')
    expect(when(null)).toBe('')
    expect(when('0001-01-01T00:00:00Z')).toBe('')
  })

  test('targetPath links members and guilds only', () => {
    expect(targetPath('member', 3)).toBe('/members/3')
    expect(targetPath('guild', 4)).toBe('/guilds/4')
    expect(targetPath('operator', 1)).toBe('')
  })

  test('sanctionLine says whether it still holds', () => {
    expect(sanctionLine({ kind: 'ban', until: null, active: true, lifted_at: null })).toBe('Banned')
    expect(sanctionLine({ kind: 'suspension', until: '2026-09-29T10:00:00Z', active: false, lifted_at: null })).toBe(
      'Suspended until 2026-09-29 10:00 UTC (ended)',
    )
    expect(sanctionLine({ kind: 'ban', until: null, active: false, lifted_at: '2026-09-28T11:00:00Z' })).toBe(
      'Banned (lifted 2026-09-28 11:00 UTC)',
    )
  })
})
