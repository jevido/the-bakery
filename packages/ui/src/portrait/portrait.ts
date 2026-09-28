// Seeded pixel portraits for members and agents: the same seed always draws
// the same face. Pure and offline: a seed goes in, an SVG string comes out.

import { BACKGROUNDS, BEARD, CLOTH, CLOTHES, EYE, EYES, HAIR, HAIR_COLOURS, HEADS, MOUTH, MOUTHS, NOSE, SKIN, type Layer } from './layers'

export const GRID = 16

// hash is FNV-1a over the seed, the start of the random stream.
function hash(s: string): number {
  let h = 0x811c9dc5
  for (let i = 0; i < s.length; i++) {
    h ^= s.charCodeAt(i)
    h = Math.imul(h, 0x01000193) >>> 0
  }
  return h
}

// mulberry32: a small, good-enough PRNG for picking parts.
function random(seed: number): () => number {
  let a = seed >>> 0
  return () => {
    a = (a + 0x6d2b79f5) >>> 0
    let t = a
    t = Math.imul(t ^ (t >>> 15), t | 1)
    t ^= t + Math.imul(t ^ (t >>> 7), t | 61)
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296
  }
}

// shade darkens a #rrggbb colour.
export function shade(hex: string, by = 0.78): string {
  const n = parseInt(hex.slice(1), 16)
  const c = [n >> 16, (n >> 8) & 255, n & 255].map((v) => Math.round(v * by))
  return '#' + c.map((v) => v.toString(16).padStart(2, '0')).join('')
}

export type Face = {
  head: number
  eyes: number
  mouth: number
  hair: number
  // bald: the hair layer is empty.
  bald: boolean
  beard: boolean
  clothes: number
  skin: string
  hairColour: string
  cloth: string
  background: string
}

// face picks the parts for a seed.
export function face(seed: string): Face {
  const next = random(hash(seed || 'colonist'))
  const pick = <T,>(list: T[]): number => Math.floor(next() * list.length)
  const hair = pick(HAIR)
  return {
    head: pick(HEADS),
    eyes: pick(EYES),
    mouth: pick(MOUTHS),
    hair,
    bald: HAIR[hair].length === 0,
    beard: next() < 0.25,
    clothes: pick(CLOTHES),
    skin: SKIN[pick(SKIN)],
    hairColour: HAIR_COLOURS[pick(HAIR_COLOURS)],
    cloth: CLOTH[pick(CLOTH)],
    background: BACKGROUNDS[pick(BACKGROUNDS)],
  }
}

// pixels paints a face onto a 16×16 grid of colours ('' is background).
export function pixels(f: Face): string[][] {
  const grid = Array.from({ length: GRID }, () => Array<string>(GRID).fill(''))
  const colours: Record<string, string> = {
    s: f.skin,
    S: shade(f.skin),
    h: f.hairColour,
    H: shade(f.hairColour),
    c: f.cloth,
    C: shade(f.cloth),
    e: EYE,
    m: MOUTH,
  }
  const draw = (layer: Layer) => {
    for (const [y, row] of layer) {
      for (let x = 0; x < GRID; x++) {
        const ch = row[x]
        if (ch && ch !== '.') grid[y][x] = colours[ch]
      }
    }
  }
  draw(CLOTHES[f.clothes])
  draw(HEADS[f.head])
  draw(NOSE)
  if (f.beard) draw(BEARD)
  draw(EYES[f.eyes])
  draw(MOUTHS[f.mouth])
  draw(HAIR[f.hair])
  return grid
}

// portrait draws the seed's face as an SVG string, size pixels square.
// Runs of one colour on a row are one rect, to keep the markup small.
export function portrait(seed: string, size = 64): string {
  const f = face(seed)
  const grid = pixels(f)
  let rects = `<rect width="${GRID}" height="${GRID}" fill="${f.background}"/>`
  for (let y = 0; y < GRID; y++) {
    let x = 0
    while (x < GRID) {
      const fill = grid[y][x]
      let end = x + 1
      while (end < GRID && grid[y][end] === fill) end++
      if (fill) rects += `<rect x="${x}" y="${y}" width="${end - x}" height="1" fill="${fill}"/>`
      x = end
    }
  }
  return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 ${GRID} ${GRID}" width="${size}" height="${size}" shape-rendering="crispEdges">${rects}</svg>`
}
