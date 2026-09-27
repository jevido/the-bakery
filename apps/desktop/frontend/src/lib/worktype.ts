// A muted colour per work type key, the same everywhere the key shows.
export function workTypeHue(key: string): number {
  let h = 0
  for (const ch of key) h = (h * 31 + ch.charCodeAt(0)) >>> 0
  return h % 360
}

export function workTypeStyle(key: string): string {
  const hue = workTypeHue(key)
  return `--wt-bg: hsl(${hue} 22% 26%); --wt-fg: hsl(${hue} 35% 78%); --wt-border: hsl(${hue} 22% 40%)`
}
