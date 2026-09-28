import { describe, expect, test } from 'bun:test'
import { BEARD, CLOTHES, EYES, HAIR, HEADS, MOUTHS, NOSE } from './layers'
import { GRID, portrait } from './portrait'

describe('portrait', () => {
  test('every layer row is 16 pixels, inside the grid', () => {
    for (const layer of [...HEADS, ...EYES, ...MOUTHS, ...HAIR, ...CLOTHES, BEARD, NOSE]) {
      for (const [y, row] of layer) {
        expect(row.length).toBe(GRID)
        expect(y).toBeGreaterThanOrEqual(0)
        expect(y).toBeLessThan(GRID)
      }
    }
  })

  test('the same seed draws the same portrait', () => {
    expect(portrait('ada-3f9c')).toBe(portrait('ada-3f9c'))
    expect(portrait('', 32)).toBe(portrait('', 32))
  })

  test('100 seeds draw at least 90 different portraits', () => {
    const seen = new Set<string>()
    for (let i = 0; i < 100; i++) seen.add(portrait(`seed-${i}`))
    expect(seen.size).toBeGreaterThanOrEqual(90)
  })

  test('size sets the SVG size, not the grid', () => {
    const svg = portrait('x', 24)
    expect(svg).toContain('width="24" height="24"')
    expect(svg).toContain('viewBox="0 0 16 16"')
  })
})
