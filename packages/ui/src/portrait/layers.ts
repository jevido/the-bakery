// The hand-drawn pieces portraits are put together from, on a 16×16 grid.
// Drawn for The Bakery; nothing here is taken from a game.
//
// A layer is a list of rows at a height: [y, row]. One character is one
// pixel: '.' leaves what is below, 's'/'S' skin and its shade, 'h'/'H' hair
// and its shade, 'c'/'C' clothes and their shade, 'e' eyes, 'm' mouth.

export type Layer = [y: number, row: string][]

export const HEADS: Layer[] = [
  // round
  [
    [2, '.....ssssss.....'],
    [3, '....ssssssss....'],
    [4, '...ssssssssss...'],
    [5, '...ssssssssss...'],
    [6, '..ssssssssssss..'],
    [7, '..ssssssssssss..'],
    [8, '...ssssssssss...'],
    [9, '...ssssssssss...'],
    [10, '....ssssssss....'],
    [11, '.....SSSSSS.....'],
    [12, '......ssss......'],
  ],
  // square
  [
    [2, '....ssssssss....'],
    [3, '...ssssssssss...'],
    [4, '...ssssssssss...'],
    [5, '...ssssssssss...'],
    [6, '..ssssssssssss..'],
    [7, '..ssssssssssss..'],
    [8, '...ssssssssss...'],
    [9, '...ssssssssss...'],
    [10, '...ssssssssss...'],
    [11, '....SSSSSSSS....'],
    [12, '......ssss......'],
  ],
  // narrow
  [
    [2, '.....ssssss.....'],
    [3, '....ssssssss....'],
    [4, '....ssssssss....'],
    [5, '....ssssssss....'],
    [6, '...ssssssssss...'],
    [7, '...ssssssssss...'],
    [8, '....ssssssss....'],
    [9, '....ssssssss....'],
    [10, '....ssssssss....'],
    [11, '.....SSSSSS.....'],
    [12, '......ssss......'],
  ],
]

export const NOSE: Layer = [[7, '.......S........']]

export const EYES: Layer[] = [
  [[6, '.....e....e.....']],
  [[6, '.....ee..ee.....']],
  // heavy-lidded
  [
    [5, '.....SS..SS.....'],
    [6, '.....ee..ee.....'],
  ],
  // brows
  [
    [5, '....hhh..hhh....'],
    [6, '.....e....e.....'],
  ],
]

export const MOUTHS: Layer[] = [
  [[9, '......mmmm......']],
  [[9, '.......mm.......']],
  [
    [9, '.....m....m.....'],
    [10, '......mmmm......'],
  ],
  [[9, '......mmm.......']],
]

// Hair goes over the head. An empty layer is a bald colonist.
export const HAIR: Layer[] = [
  // short
  [
    [1, '.....hhhhhh.....'],
    [2, '....hhhhhhhh....'],
    [3, '...hhhhhhhhhh...'],
    [4, '...hh......hh...'],
  ],
  // long
  [
    [1, '.....hhhhhh.....'],
    [2, '....hhhhhhhh....'],
    [3, '...hhhhhhhhhh...'],
    [4, '..hhhH....Hhhh..'],
    [5, '..hh........hh..'],
    [6, '..hh........hh..'],
    [7, '..hh........hh..'],
    [8, '..hh........hh..'],
    [9, '..hH........Hh..'],
    [10, '..hH........Hh..'],
    [11, '..HH........HH..'],
  ],
  // crest
  [
    [0, '.......hh.......'],
    [1, '.......hh.......'],
    [2, '......hhhh......'],
    [3, '......hHHh......'],
  ],
  // bald
  [],
  // bun
  [
    [0, '......hhhh......'],
    [1, '......hHHh......'],
    [2, '....hhhhhhhh....'],
    [3, '...hhhhhhhhhh...'],
    [4, '...h........h...'],
  ],
  // side part
  [
    [1, '....hhhhhhhh....'],
    [2, '...hhhhhhhhhh...'],
    [3, '...hhhhhhh.Hh...'],
    [4, '...hh.......h...'],
  ],
  // cap, in the clothes' colour
  [
    [1, '....cccccccc....'],
    [2, '...cccccccccc...'],
    [3, '..CCCCCCCCCCCC..'],
  ],
  // curls
  [
    [1, '....hHhHhHhH....'],
    [2, '...hHhHhHhHhH...'],
    [3, '..hHhhhhhhhhhH..'],
    [4, '..hh........hh..'],
    [5, '..H..........H..'],
  ],
]

// Drawn under the mouth, over the head, for some colonists.
export const BEARD: Layer = [
  [8, '...h........h...'],
  [9, '...hh......hh...'],
  [10, '....hhhhhhhh....'],
  [11, '.....hHHHHh.....'],
]

export const CLOTHES: Layer[] = [
  // tunic
  [
    [13, '...cccccccccc...'],
    [14, '..cccccccccccc..'],
    [15, '.cccccccccccccc.'],
  ],
  // open collar
  [
    [13, '...ccccssccccc..'],
    [14, '..ccccCssCcccc..'],
    [15, '.ccccccCCcccccc.'],
  ],
  // strap
  [
    [13, '...cCcccccccc...'],
    [14, '..ccCccccccccc..'],
    [15, '.ccccCcccccccc..'],
  ],
]

export const SKIN = ['#f1c9a0', '#dcae82', '#c08b5c', '#9a6441', '#6e452c', '#4a2f20']
export const HAIR_COLOURS = ['#2a221c', '#4e3322', '#7d3c1e', '#b8904c', '#d7c79a', '#8f8d86', '#dedad0', '#3f4a2c']
export const CLOTH = ['#7c8a42', '#6a8cab', '#a55a36', '#c9b27a', '#7a4f6a', '#5c6670', '#8a6242', '#4f7a6a']
export const BACKGROUNDS = ['#2c2e28', '#3a3326', '#26303a', '#35282a', '#2e3a2c', '#33302c']
export const EYE = '#1c1d1a'
export const MOUTH = '#6e3a2c'
