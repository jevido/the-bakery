// Pixel art for the colony view, drawn from small character maps so there
// are no image files to keep in step. One character is one pixel; '.' is
// see-through. Each sprite names its colours in a palette.

import { face, shade } from '@bakery/ui'

export type Sprite = { rows: string[]; palette: Record<string, string> }

// A colonist, 10×14. 's' skin, 'h' hair, 'b' the body colour and 'd' its
// shade come from the agent's portrait (colonistColours).
export const COLONIST: Sprite = {
  rows: [
    '...hhhh...',
    '..hhhhhh..',
    '..hssssh..',
    '..sesses..',
    '..ssssss..',
    '...ssss...',
    '..bbbbbb..',
    '.bbbbbbbb.',
    '.sbbddbbs.',
    '.sbbddbbs.',
    '..bbbbbb..',
    '..ll..ll..',
    '..ll..ll..',
    '..kk..kk..',
  ],
  palette: { h: '#4a3526', s: '#d9a877', e: '#1c1d1a', b: '#7c8a42', d: '#5a6530', l: '#3d3a33', k: '#26241f' },
}

// The supervisor, 10×14: a long coat, a cap and a clipboard.
export const SUPERVISOR: Sprite = {
  rows: [
    '..cccccc..',
    '.cccccccc.',
    '..hssssh..',
    '..sesses..',
    '..ssssss..',
    '...ssss...',
    '..oooooo..',
    '.oooooooop',
    '.soowwoopp',
    '.soowwoopp',
    '..oooooo..',
    '..oo..oo..',
    '..ll..ll..',
    '..kk..kk..',
  ],
  palette: { c: '#3d4a5c', h: '#4a3526', s: '#d9a877', e: '#1c1d1a', o: '#5c6670', w: '#e8e2cf', p: '#c9b27a', l: '#3d3a33', k: '#26241f' },
}

// A note being handed over, 4×4.
export const NOTE: Sprite = {
  rows: ['pppp', 'plpp', 'pplp', 'pppp'],
  palette: { p: '#e8e2cf', l: '#8c8676' },
}

// A workbench, 20×10.
export const BENCH: Sprite = {
  rows: [
    '....................',
    '.tttttttttttttttttt.',
    '.TTTTTTTTTTTTTTTTTT.',
    '.w.gg.........ii..w.',
    '.w.gg.........ii..w.',
    '.w................w.',
    '.w................w.',
    '.w................w.',
    '.w................w.',
    '....................',
  ],
  palette: { t: '#9a7a52', T: '#6f5537', w: '#5a4430', g: '#6a8cab', i: '#a55a36' },
}

// The table idle colonists stand around, 16×8.
export const TABLE: Sprite = {
  rows: [
    '................',
    '.tttttttttttttt.',
    '.TTTTTTTTTTTTTT.',
    '..w..........w..',
    '..w..........w..',
    '..w..........w..',
    '..w..........w..',
    '................',
  ],
  palette: { t: '#8a7a5e', T: '#5f5340', w: '#4a3f30' },
}

// Small marks over a colonist's head, 5×5.
export const MARKS: Record<string, Sprite> = {
  waiting: {
    rows: ['.yyy.', 'y...y', '...y.', '..y..', '..y..'],
    palette: { y: '#e5c14a' },
  },
  done: {
    rows: ['....g', '...g.', 'g.g..', '.g...', '.....'],
    palette: { g: '#98a856' },
  },
  failed: {
    rows: ['r...r', '.r.r.', '..r..', '.r.r.', 'r...r'],
    palette: { r: '#c06c43' },
  },
}

// Mood marks, 3×3, beside a colonist's head. An okay agent has none.
export const MOODS: Record<string, Sprite> = {
  content: { rows: ['g.g', '...', 'ggg'], palette: { g: '#98a856' } },
  stressed: { rows: ['.a.', 'aaa', '.a.'], palette: { a: '#b8913a' } },
  breaking: { rows: ['r.r', '.r.', 'r.r'], palette: { r: '#c06c43' } },
}

// A hammer, 4×4, swung while editing or running a command.
export const HAMMER: Sprite = {
  rows: ['mmm.', 'mmm.', '.w..', '.w..'],
  palette: { m: '#9aa3a8', w: '#6f5537' },
}

// The floor tile, 8×8.
export const FLOOR: Sprite = {
  rows: ['aaaaaaab', 'aaaaaaab', 'aaaaaaab', 'aaaaaaab', 'aaaaaaab', 'aaaaaaab', 'aaaaaaab', 'bbbbbbbb'],
  palette: { a: '#2c2b25', b: '#252420' },
}

export function draw(ctx: CanvasRenderingContext2D, s: Sprite, x: number, y: number, scale: number, override: Record<string, string> = {}) {
  for (let r = 0; r < s.rows.length; r++) {
    const row = s.rows[r]
    for (let c = 0; c < row.length; c++) {
      const ch = row[c]
      if (ch === '.') continue
      ctx.fillStyle = override[ch] ?? s.palette[ch]
      ctx.fillRect(Math.round(x + c * scale), Math.round(y + r * scale), scale, scale)
    }
  }
}

export function size(s: Sprite, scale: number) {
  return { w: s.rows[0].length * scale, h: s.rows.length * scale }
}

// colonistColours colours a colonist like the agent's portrait: the same
// skin, hair and clothes, so the figure and the portrait read as one person.
export function colonistColours(seed: string): Record<string, string> {
  const f = face(seed || 'agent')
  return { s: f.skin, h: f.bald ? shade(f.skin) : f.hairColour, b: f.cloth, d: shade(f.cloth) }
}
