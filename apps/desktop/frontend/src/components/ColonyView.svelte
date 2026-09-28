<script lang="ts">
  import { Portrait } from '@bakery/ui'
  import { BENCH, COLONIST, FLOOR, HAMMER, MARKS, MOODS, TABLE, colonistColours, draw, size } from '../lib/sprites'
  import { needs, type Mood } from '../lib/needs.svelte'
  import { settings } from '../lib/settings.svelte'
  import type { Colony } from '../lib/colony.svelte'
  import type { Letters } from '../lib/letters.svelte'
  import type { RunInfo } from '../lib/bindings'

  // The colony from above: a workbench per work type of the guild, each
  // agent with a run going standing at the bench of its task's work type,
  // and the agents enabled on this board and idle around a table. Hover an
  // agent for what it does; click it to open its run, or its letter while
  // it waits for you.
  let { colony, letters }: { colony: Colony; letters: Letters } = $props()

  const S = 3 // one sprite pixel is 3×3 screen pixels
  const DONE_FOR = 5000
  const FAILED_FOR = 10000
  const FRAME = 1000 / 30

  let canvas = $state<HTMLCanvasElement | null>(null)
  let width = $state(600)
  let hover = $state<{ x: number; y: number; who: Figure } | null>(null)
  const reduced = typeof matchMedia === 'function' && matchMedia('(prefers-reduced-motion: reduce)').matches

  type Figure = {
    slug: string
    name: string
    seed: string
    state: string // idle, thinking, editing, running, waiting, done, failed
    mood?: Mood
    run?: RunInfo
    bench: number // -1: at the table
  }

  // When each run of this board stopped running, to show its check or cross
  // for a moment.
  const endedAt = new Map<string, number>()
  let now = $state(Date.now())

  $effect(() => {
    for (const r of colony.workshop.runs) {
      if (r.status !== 'running' && !endedAt.has(r.id)) endedAt.set(r.id, Date.now())
    }
  })

  const benches = $derived([...colony.workTypes.map((w) => ({ key: w.key, name: w.name })), { key: '', name: 'Other work' }])

  const figures = $derived.by(() => {
    const board = colony.boardId
    const tasks = new Map((colony.view?.columns ?? []).flatMap((c) => c.tasks).map((t) => [t.id, t]))
    const out: Figure[] = []
    const seen = new Set<string>()
    for (const r of colony.workshop.runs) {
      if (r.board_id !== board || seen.has(r.agent_slug)) continue
      let state = r.state || 'thinking'
      if (r.status !== 'running') {
        const since = now - (endedAt.get(r.id) ?? 0)
        const shown = r.status === 'failed' ? FAILED_FOR : DONE_FOR
        if (since > shown) continue
        state = r.status === 'failed' ? 'failed' : 'done'
      }
      const workType = tasks.get(r.task_id)?.work_type ?? ''
      const bench = benches.findIndex((b) => b.key === workType)
      const agent = colony.agents.find((a) => a.slug === r.agent_slug)
      out.push({
        slug: r.agent_slug,
        name: r.agent_name,
        seed: agent?.portrait_seed || r.agent_slug,
        state,
        mood: needs.bySlug[r.agent_slug]?.mood,
        run: r,
        bench: bench < 0 ? benches.length - 1 : bench,
      })
      seen.add(r.agent_slug)
    }
    for (const slug of colony.settings?.config.agents ?? []) {
      if (seen.has(slug)) continue
      const agent = colony.agents.find((a) => a.slug === slug)
      if (!agent) continue
      out.push({ slug, name: agent.name, seed: agent.portrait_seed || slug, state: 'idle', mood: needs.bySlug[slug]?.mood, bench: -1 })
    }
    return out
  })

  // Layout: benches in rows across the room, the table below them.
  const bench = size(BENCH, S)
  const person = size(COLONIST, S)
  const table = size(TABLE, S)
  const GAP = 36
  const layout = $derived.by(() => {
    const perRow = Math.max(1, Math.floor((width - GAP) / (bench.w + GAP)))
    const rowH = bench.h + person.h + 40
    const spots = benches.map((_, i) => ({
      x: GAP + (i % perRow) * (bench.w + GAP),
      y: 24 + Math.floor(i / perRow) * rowH,
    }))
    const rows = Math.ceil(benches.length / perRow)
    const tableY = 24 + rows * rowH + 10
    return { spots, tableX: GAP, tableY, height: tableY + table.h + person.h + 40 }
  })

  const animated = $derived(
    !reduced && figures.some(
        (f) => ['thinking', 'editing', 'running', 'waiting'].includes(f.state) || (f.state === 'idle' && f.mood === 'breaking' && !settings.quiet),
      ),
  )
  const flashing = $derived(figures.some((f) => f.state === 'done' || f.state === 'failed'))

  // A check or cross goes after a few seconds.
  $effect(() => {
    if (!flashing) return
    const timer = setInterval(() => (now = Date.now()), 1000)
    return () => clearInterval(timer)
  })

  // Hit boxes of the last frame, for hover and click.
  let boxes: { x: number; y: number; w: number; h: number; who: Figure }[] = []

  function render(t: number) {
    const ctx = canvas?.getContext('2d')
    if (!ctx || !canvas) return
    ctx.imageSmoothingEnabled = false
    const w = canvas.width
    const h = canvas.height
    for (let y = 0; y < h; y += 8 * S) for (let x = 0; x < w; x += 8 * S) draw(ctx, FLOOR, x, y, S)
    ctx.font = '600 12px "Pixelify Sans", monospace'
    ctx.textAlign = 'center'
    benches.forEach((b, i) => {
      const p = layout.spots[i]
      draw(ctx, BENCH, p.x, p.y, S)
      ctx.fillStyle = '#b3ad9c'
      ctx.fillText(b.name, p.x + bench.w / 2, p.y + bench.h + person.h + 22)
    })
    draw(ctx, TABLE, layout.tableX, layout.tableY, S)
    ctx.fillStyle = '#8c8676'
    ctx.textAlign = 'left'
    ctx.fillText('Idle', layout.tableX, layout.tableY - 4)

    boxes = []
    const atBench = new Map<number, number>()
    let idle = 0
    for (const f of figures) {
      let x: number, y: number
      if (f.bench >= 0) {
        const n = atBench.get(f.bench) ?? 0
        atBench.set(f.bench, n + 1)
        const p = layout.spots[f.bench]
        x = p.x + 4 * S + n * (person.w + 2 * S)
        y = p.y + bench.h - 2 * S
      } else {
        x = layout.tableX + table.w + 8 + idle * (person.w + 3 * S)
        y = layout.tableY - 2 * S
        idle++
      }
      const phase = t / 1000
      let dy = 0
      if (!reduced && f.state === 'thinking') dy = Math.round(Math.sin(phase * 4) * 1.5) * S
      // An idle agent shows its mood: slumped when stressed, pacing when
      // breaking. Looks only; nothing it does changes.
      // Quiet colony: no mood, no idle acting.
      const mood = settings.quiet ? undefined : f.mood
      if (f.state === 'idle' && mood === 'stressed') dy = S
      if (f.state === 'idle' && mood === 'breaking' && !reduced) x += Math.round(Math.sin(phase * 2) * 3) * S
      draw(ctx, COLONIST, x, y + dy, S, colonistColours(f.seed))
      const mark = mood ? MOODS[mood] : undefined
      if (mark) draw(ctx, mark, x + person.w, y - 3 * S + dy, S)
      const headX = x + person.w / 2 - 2.5 * S
      const headY = y - 7 * S
      if (f.state === 'editing' || f.state === 'running') {
        const up = reduced || Math.floor(phase * 5) % 2 === 0
        draw(ctx, HAMMER, x + person.w - S, y + (up ? 5 : 8) * S, S)
      } else if (f.state === 'thinking' && !reduced) {
        const dots = Math.floor(phase * 2) % 4
        ctx.fillStyle = '#b3ad9c'
        for (let i = 0; i < dots; i++) ctx.fillRect(headX + i * 2 * S, headY + 3 * S, S, S)
      } else if (MARKS[f.state]) {
        const blink = f.state === 'waiting' && !reduced && Math.floor(phase * 2) % 2 === 1
        if (!blink) draw(ctx, MARKS[f.state], headX, headY + dy, S)
      }
      boxes.push({ x, y: y - 8 * S, w: person.w, h: person.h + 8 * S, who: f })
    }
  }

  // Draw now and then only as often as something moves: 30 frames a
  // second while an agent is busy, else once per change. Paused while the
  // window is hidden.
  $effect(() => {
    if (!canvas) return
    canvas.width = width
    canvas.height = layout.height
    void figures
    void settings.quiet
    render(performance.now())
    if (!animated) return
    let raf = 0
    let last = 0
    const loop = (t: number) => {
      raf = requestAnimationFrame(loop)
      if (document.hidden || t - last < FRAME) return
      last = t
      render(t)
    }
    raf = requestAnimationFrame(loop)
    return () => cancelAnimationFrame(raf)
  })

  function hit(e: MouseEvent) {
    const rect = canvas!.getBoundingClientRect()
    const x = e.clientX - rect.left
    const y = e.clientY - rect.top
    return boxes.find((b) => x >= b.x && x <= b.x + b.w && y >= b.y && y <= b.y + b.h)
  }

  function move(e: MouseEvent) {
    const b = hit(e)
    hover = b ? { x: b.x + b.w + 6, y: b.y, who: b.who } : null
  }

  function click(e: MouseEvent) {
    const b = hit(e)
    if (!b?.who.run) return
    const letter = letters.list.find((l) => l.run_id === b.who.run!.id)
    if (b.who.state === 'waiting' && letter) letters.openId = letter.id
    else colony.openRun(b.who.run.id)
  }

  function elapsed(r: RunInfo): string {
    const s = Math.max(0, Math.round((Date.now() - new Date(r.started_at).getTime()) / 1000))
    return s < 60 ? `${s}s` : `${Math.floor(s / 60)}m ${String(s % 60).padStart(2, '0')}s`
  }

  const WORDS: Record<string, string> = {
    idle: 'Idle',
    thinking: 'Thinking',
    editing: 'Editing files',
    running: 'Running a command',
    waiting: 'Waiting for you',
    done: 'Done',
    failed: 'Failed',
  }
</script>

<div class="colony-view" bind:clientWidth={width}>
  <canvas bind:this={canvas} class={{ pointer: !!hover?.who.run }} onmousemove={move} onmouseleave={() => (hover = null)} onclick={click}
  ></canvas>
  {#if hover}
    <div class="tip" style:left="{hover.x}px" style:top="{hover.y}px">
      <span class="who"><Portrait seed={hover.who.seed} size={32} /><strong>{hover.who.name}</strong></span>
      <span>{WORDS[hover.who.state] ?? hover.who.state}</span>
      {#if hover.who.run}
        <span class="dim">{hover.who.run.task_title}</span>
        <span class="dim">{elapsed(hover.who.run)} · ${hover.who.run.cost_usd.toFixed(2)}</span>
      {/if}
    </div>
  {/if}
  {#if figures.length === 0}
    <p class="empty">No agents here yet. Enable agents in Board settings (⚙) and unpause the board.</p>
  {/if}
</div>

<style>
  .colony-view {
    position: relative;
    flex: 1;
    min-height: 0;
    overflow: auto;
    background: var(--bg-deep);
    border: 1px solid var(--frame-dim);
    border-radius: var(--radius);
  }

  canvas {
    display: block;
    image-rendering: pixelated;
  }

  canvas.pointer {
    cursor: pointer;
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

  .who {
    display: flex;
    gap: 6px;
    align-items: center;
  }
</style>
