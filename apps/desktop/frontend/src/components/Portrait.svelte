<script lang="ts">
  // A stand-in portrait: a symmetric pixel block drawn from the agent's
  // portrait seed, so every agent has a face of its own until real portraits
  // come. The same seed always draws the same block.
  let { seed, size = 48 }: { seed: string; size?: number } = $props()

  const GRID = 7
  // Earthy colony tones: one for the block, one for its shading.
  const PALETTE = [
    ['#b98a5e', '#8a6242'],
    ['#7c8a42', '#5a6530'],
    ['#6a8cab', '#4b6781'],
    ['#a55a36', '#7a4228'],
    ['#9c8f76', '#6f6553'],
    ['#c9b27a', '#96834f'],
  ]

  // A small, stable hash of the seed (FNV-1a), stretched into enough bits.
  function bits(s: string): number[] {
    let h = 0x811c9dc5
    const out: number[] = []
    for (let round = 0; out.length < 64; round++) {
      for (const ch of s + round) {
        h ^= ch.charCodeAt(0)
        h = Math.imul(h, 0x01000193) >>> 0
      }
      for (let i = 0; i < 32; i++) out.push((h >>> i) & 1)
    }
    return out
  }

  let cells = $derived.by(() => {
    const b = bits(seed || 'agent')
    const colours = PALETTE[(b[0] + 2 * b[1] + 4 * b[2]) % PALETTE.length]
    const out: { x: number; y: number; fill: string }[] = []
    const half = Math.ceil(GRID / 2)
    let i = 3
    for (let y = 0; y < GRID; y++) {
      for (let x = 0; x < half; x++) {
        const on = b[i++ % b.length]
        if (!on) continue
        const fill = b[i++ % b.length] ? colours[0] : colours[1]
        out.push({ x, y, fill })
        if (x !== GRID - 1 - x) out.push({ x: GRID - 1 - x, y, fill })
      }
    }
    return out
  })
</script>

<svg class="portrait" width={size} height={size} viewBox={`-1 -1 ${GRID + 2} ${GRID + 2}`} aria-hidden="true">
  <rect x="-1" y="-1" width={GRID + 2} height={GRID + 2} fill="var(--panel-inset)" />
  {#each cells as c, i (i)}
    <rect x={c.x} y={c.y} width="1" height="1" fill={c.fill} />
  {/each}
</svg>

<style>
  .portrait {
    flex-shrink: 0;
    border: 1px solid var(--frame-dim);
    border-radius: var(--radius);
    image-rendering: pixelated;
  }
</style>
