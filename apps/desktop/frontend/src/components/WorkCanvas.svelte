<script lang="ts" module>
  import type { World as W } from '../lib/choreography'
  // Each board's world outlives the canvas, so a run started from the
  // Boards screen is still handed out when you come back to Work mode.
  const worlds = new Map<number, W>()
</script>

<script lang="ts">
  import { untrack } from 'svelte'
  import { Portrait } from '@bakery/ui'
  import { BENCH, COLONIST, FLOOR, HAMMER, MARKS, MOODS, NOTE, SUPERVISOR, TABLE, colonistColours, draw, size } from '../lib/sprites'
  import { moving, newWorld, step, type Room, type RunIn, type World } from '../lib/choreography'
  import { needs } from '../lib/needs.svelte'
  import { notesLine } from '../lib/flavor/lines'
  import { settings } from '../lib/settings.svelte'
  import type { Colony } from '../lib/colony.svelte'
  import type { Letters } from '../lib/letters.svelte'
  import type { RunInfo } from '../lib/bindings'

  // Work mode's canvas: the board's agents in a room with a bench per work
  // type and the supervisor at a desk. It acts out the runs that happen
  // (lib/choreography.ts) and reads nothing else; drawing it costs no
  // tokens. Hover a figure for who it is and what it does; click one at work
  // to open its run, or its letter while it waits for you.
  // thinking: the supervisor chat is waiting for an answer.
  let {
    colony,
    letters,
    thinking = false,
    onopenrun,
  }: { colony: Colony; letters: Letters; thinking?: boolean; onopenrun: (id: string) => void } = $props()

  const S = 3 // one sprite pixel is 3×3 screen pixels
  const FRAME = 1000 / 30
  const reduced = typeof matchMedia === 'function' && matchMedia('(prefers-reduced-motion: reduce)').matches

  let canvas = $state<HTMLCanvasElement | null>(null)
  let width = $state(800)
  let height = $state(500)
  let hover = $state<{ x: number; y: number; slug: string; doing: string } | null>(null)

  const bench = size(BENCH, S)
  const person = size(COLONIST, S)
  const table = size(TABLE, S)

  const benches = $derived([...colony.workTypes.map((w) => ({ key: w.key, name: w.name })), { key: '', name: 'Other work' }])

  // The room, fitted to the canvas: benches along the top, the supervisor's
  // desk at the bottom, the floor between them to stroll on.
  const room = $derived.by((): Room => {
    const gap = 28
    const perRow = Math.max(1, Math.floor((width - gap) / (bench.w + gap)))
    const rows = Math.ceil(benches.length / perRow)
    const spots: Record<string, { x: number; y: number }> = {}
    benches.forEach((b, i) => {
      const x = gap + (i % perRow) * (bench.w + gap)
      const y = 20 + Math.floor(i / perRow) * (bench.h + person.h + 30)
      spots[b.key] = { x: x + bench.w / 2, y: y + bench.h + person.h - 4 }
    })
    const top = 20 + rows * (bench.h + person.h + 30) + person.h
    const desk = { x: Math.round(width / 2), y: Math.max(top + 60, height - 30) }
    return {
      width,
      height,
      desk,
      benches: spots,
      wander: { x: 30, y: top, w: Math.max(40, width - 60), h: Math.max(20, desk.y - top - person.h - 20) },
    }
  })

  const boardRuns = $derived(colony.workshop.runs.filter((r) => r.board_id === colony.boardId && r.kind !== 'plan'))

  function notesOf(r: RunInfo): string {
    return notesLine(r.id, r.status, r.moved_to, r.cost_usd)
  }

  const input = $derived.by(() => {
    const tasks = new Map((colony.view?.columns ?? []).flatMap((c) => c.tasks).map((t) => [t.id, t]))
    // Oldest first, so the newest run of an agent decides.
    const runs: RunIn[] = [...boardRuns].reverse().map((r) => ({
      id: r.id,
      agent: r.agent_slug,
      status: r.status,
      workType: tasks.get(r.task_id)?.work_type ?? '',
      notes: notesOf(r),
    }))
    return { agents: colony.settings?.config.agents ?? [], runs, room, quiet: settings.quiet || reduced, thinking }
  })

  // The world is redrawn every frame, not reactive state: the template
  // reads only what the hover copied out of it.
  const boardId = untrack(() => colony.boardId ?? 0)
  let world: World = worlds.get(boardId) ?? newWorld(untrack(() => room))
  let last = performance.now()

  function agentOf(slug: string) {
    return colony.agents.find((a) => a.slug === slug)
  }
  function runOf(slug: string): RunInfo | undefined {
    return boardRuns.find((r) => r.agent_slug === slug)
  }

  // Hit boxes of the last frame, for hover and click.
  let boxes: { x: number; y: number; w: number; h: number; slug: string }[] = []

  // The room itself (floor, benches, desk) changes only with its size and
  // the work types, so it is drawn once into its own canvas and copied in
  // each frame.
  const backdrop = $derived.by(() => {
    const bg = document.createElement('canvas')
    bg.width = width
    bg.height = height
    const ctx = bg.getContext('2d')!
    ctx.imageSmoothingEnabled = false
    const tile = document.createElement('canvas')
    tile.width = tile.height = 8 * S
    draw(tile.getContext('2d')!, FLOOR, 0, 0, S)
    ctx.fillStyle = ctx.createPattern(tile, 'repeat')!
    ctx.fillRect(0, 0, width, height)
    ctx.font = '600 12px "Pixelify Sans", monospace'
    ctx.textAlign = 'center'
    for (const b of benches) {
      const p = room.benches[b.key]
      draw(ctx, BENCH, p.x - bench.w / 2, p.y - person.h - bench.h + 4, S)
      ctx.fillStyle = '#b3ad9c'
      ctx.fillText(b.name, p.x, p.y + 16)
    }
    draw(ctx, TABLE, room.desk.x - table.w / 2, room.desk.y - table.h + 2, S)
    ctx.fillStyle = '#8c8676'
    ctx.fillText('Supervisor', room.desk.x, room.desk.y + 16)
    return bg
  })

  function render(now: number) {
    const ctx = canvas?.getContext('2d')
    if (!ctx || !canvas) return
    world = step(world, { ...input, now }, Math.max(0, Math.min(100, now - last)))
    worlds.set(boardId, world)
    last = now
    ctx.imageSmoothingEnabled = false
    ctx.drawImage(backdrop, 0, 0)
    ctx.font = '600 12px "Pixelify Sans", monospace'
    ctx.textAlign = 'center'

    const quiet = input.quiet
    boxes = []
    const phase = now / 1000
    const order = Object.values(world.figures).sort((a, b) => a.pos.y - b.pos.y)
    for (const f of order) {
      const x = Math.round(f.pos.x - person.w / 2)
      let y = Math.round(f.pos.y - person.h)
      const mood = quiet ? undefined : needs.bySlug[f.slug]?.mood
      if (f.state === 'wander' && mood === 'stressed') y += S
      const agent = agentOf(f.slug)
      draw(ctx, COLONIST, x, y, S, colonistColours(agent?.portrait_seed || f.slug))
      const run = runOf(f.slug)
      const headX = x + person.w / 2 - 2.5 * S
      if (f.state === 'working' && run) {
        if (run.state === 'waiting') {
          if (quiet || Math.floor(phase * 2) % 2 === 0) draw(ctx, MARKS.waiting, headX, y - 7 * S, S)
        } else if (run.state === 'editing' || run.state === 'running') {
          const up = quiet || Math.floor(phase * 5) % 2 === 0
          draw(ctx, HAMMER, x + person.w - S, y + (up ? 5 : 8) * S, S)
        } else if (!quiet) {
          ctx.fillStyle = '#b3ad9c'
          for (let i = 0; i < Math.floor(phase * 2) % 4; i++) ctx.fillRect(headX + i * 2 * S, y - 4 * S, S, S)
        }
      }
      if (mood && MOODS[mood]) draw(ctx, MOODS[mood], x + person.w, y - 3 * S, S)
      if (f.state === 'to_supervisor') draw(ctx, NOTE, x + person.w - 2 * S, y + 6 * S, S)
      if (f.state === 'handing_in' && f.notes && !quiet) bubble(ctx, f.notes, f.pos.x, y - 8)
      boxes.push({ x, y: y - 8 * S, w: person.w, h: person.h + 8 * S, slug: f.slug })
    }
    const sup = world.supervisor
    const sx = Math.round(sup.pos.x - person.w / 2)
    const sy = Math.round(sup.pos.y - person.h)
    draw(ctx, SUPERVISOR, sx, sy, S)
    if (sup.state === 'handing') draw(ctx, NOTE, sx + person.w, sy + 6 * S, S)
    if (sup.thinking) {
      ctx.fillStyle = '#b3ad9c'
      for (let i = 0; i < Math.floor(phase * 2) % 4; i++) ctx.fillRect(sx + person.w / 2 - 2.5 * S + i * 2 * S, sy - 4 * S, S, S)
    }
  }

  function bubble(ctx: CanvasRenderingContext2D, text: string, cx: number, bottom: number) {
    ctx.font = '12px "Pixelify Sans", monospace'
    const w = Math.min(260, ctx.measureText(text).width + 14)
    const x = Math.max(4, Math.min(width - w - 4, cx - w / 2))
    const y = bottom - 22
    ctx.fillStyle = '#e8e2cf'
    ctx.fillRect(x, y, w, 20)
    ctx.fillStyle = '#1c1d1a'
    ctx.textAlign = 'left'
    ctx.fillText(text, x + 7, y + 14, w - 14)
    ctx.textAlign = 'center'
  }

  // Draw at 30 frames a second while anything moves or works, else only
  // when something changes. Quiet colony and reduced motion never loop.
  $effect(() => {
    if (!canvas) return
    canvas.width = width
    canvas.height = height
    void input
    void backdrop
    void needs.bySlug
    render(performance.now())
    if (input.quiet) return
    let raf = 0
    let prev = 0
    const loop = (t: number) => {
      raf = requestAnimationFrame(loop)
      if (document.hidden || t - prev < FRAME) return
      prev = t
      const busy = moving(world, t) || Object.values(world.figures).some((f) => f.state === 'working')
      if (busy) render(t)
      else last = t
    }
    raf = requestAnimationFrame(loop)
    return () => cancelAnimationFrame(raf)
  })

  // In development, bakeryCanvas.advance(ms) draws ms of frames at once, so
  // the choreography can be checked while the window is covered and the
  // browser draws no frames of its own.
  if (import.meta.env.DEV) {
    let clock = 0
    ;(globalThis as { bakeryCanvas?: { advance(ms: number): void } }).bakeryCanvas = {
      advance(ms: number) {
        clock = Math.max(clock, performance.now())
        for (let t = 0; t < ms; t += FRAME) {
          clock += FRAME
          render(clock)
        }
      },
    }
  }

  function hit(e: MouseEvent) {
    const rect = canvas!.getBoundingClientRect()
    const x = e.clientX - rect.left
    const y = e.clientY - rect.top
    return boxes.find((b) => x >= b.x && x <= b.x + b.w && y >= b.y && y <= b.y + b.h)
  }

  function move(e: MouseEvent) {
    const b = hit(e)
    hover = b ? { x: Math.min(b.x + b.w + 6, width - 230), y: b.y, slug: b.slug, doing: world.figures[b.slug]?.state ?? 'wander' } : null
  }

  function click(e: MouseEvent) {
    const b = hit(e)
    const run = b && runOf(b.slug)
    if (!run || world.figures[b.slug]?.state !== 'working') return
    const letter = letters.list.find((l) => l.run_id === run.id)
    if (letter) letters.openId = letter.id
    else onopenrun(run.id)
  }

  const WORDS: Record<string, string> = {
    wander: 'Dilly-dallying',
    waiting: 'Waiting for the supervisor',
    to_bench: 'Off to the bench',
    working: 'Working',
    to_supervisor: 'Bringing back the notes',
    handing_in: 'Handing in the notes',
  }
</script>

<div class="work-canvas" bind:clientWidth={width} bind:clientHeight={height}>
  <canvas bind:this={canvas} onmousemove={move} onmouseleave={() => (hover = null)} onclick={click}></canvas>
  {#if hover}
    {@const agent = agentOf(hover.slug)}
    {@const run = runOf(hover.slug)}
    <div class="tip" style:left="{hover.x}px" style:top="{hover.y}px">
      <span class="who"><Portrait seed={agent?.portrait_seed || hover.slug} size={32} /><strong>{agent?.name ?? hover.slug}</strong></span>
      <span>{WORDS[hover.doing]}</span>
      {#if run && hover.doing === 'working'}<span class="dim">{run.task_title}</span>{/if}
    </div>
  {/if}
  {#if (colony.settings?.config.agents ?? []).length === 0}
    <p class="empty">No agents on this board yet. Enable some in Board settings (⚙ on the Boards screen).</p>
  {/if}
</div>

<style>
  .work-canvas {
    position: relative;
    height: 100%;
    min-height: 320px;
    overflow: hidden;
    background: var(--bg-deep);
    border: 1px solid var(--frame-dim);
    border-radius: var(--radius);
  }

  canvas {
    position: absolute;
    inset: 0;
    display: block;
    image-rendering: pixelated;
  }

  .tip {
    position: absolute;
    z-index: 5;
    display: flex;
    flex-direction: column;
    gap: 1px;
    max-width: 220px;
    padding: 5px 8px;
    font-size: 12px;
    pointer-events: none;
    background: var(--panel-raised);
    border: 1px solid var(--frame);
    border-radius: var(--radius);
  }

  .who {
    display: flex;
    gap: 6px;
    align-items: center;
  }

  .dim {
    color: var(--text-dim);
  }

  .empty {
    position: absolute;
    left: 16px;
    bottom: 12px;
    margin: 0;
    color: var(--text-dim);
  }
</style>
