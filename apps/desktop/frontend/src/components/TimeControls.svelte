<script lang="ts">
  import type { Colony } from '../lib/colony.svelte'

  // Time controls, like RimWorld's: ⏸ ▶ ▶▶ for the open board on this
  // machine, and the keys Space (pause or resume), 1 (normal) and 2 (fast).
  // Not while typing, and not with a modifier held. Only for a linked board.
  let { colony }: { colony: Colony } = $props()

  const SPEEDS = [
    { id: 'paused', label: '⏸', title: 'Pause (Space): no new runs start' },
    { id: 'normal', label: '▶', title: 'Normal speed (1)' },
    { id: 'fast', label: '▶▶', title: 'Fast (2): more runs at once' },
  ] as const
  const speed = $derived(colony.settings?.config.speed ?? 'paused')
  let lastSpeed: 'normal' | 'fast' = 'normal'

  function typing(target: EventTarget | null): boolean {
    const el = target as HTMLElement | null
    return !!el && (el.isContentEditable || ['INPUT', 'TEXTAREA', 'SELECT'].includes(el.tagName))
  }

  function speedKeys(e: KeyboardEvent) {
    if (e.ctrlKey || e.metaKey || e.altKey || typing(e.target) || !colony.settings?.linked) return
    if (e.key === ' ') {
      e.preventDefault()
      if (speed === 'paused') colony.setSpeed(lastSpeed)
      else {
        lastSpeed = speed === 'fast' ? 'fast' : 'normal'
        colony.setSpeed('paused')
      }
    } else if (e.key === '1') colony.setSpeed('normal')
    else if (e.key === '2') colony.setSpeed('fast')
  }
</script>

<svelte:window onkeydown={speedKeys} />

{#if colony.settings?.linked}
  <div class="speed" role="group" aria-label="Time controls">
    {#each SPEEDS as s (s.id)}
      <button class={{ on: speed === s.id }} title={s.title} aria-pressed={speed === s.id} onclick={() => colony.setSpeed(s.id)}
        >{s.label}</button
      >
    {/each}
  </div>
  {#if speed === 'paused'}<span class="paused">Paused</span>{/if}
{/if}

<style>
  .speed {
    display: flex;
    border: 1px solid var(--frame-dim);
    border-radius: var(--radius);
    overflow: hidden;
  }

  .speed button {
    font: inherit;
    font-size: 11px;
    min-width: 26px;
    padding: 1px 6px;
    color: var(--text-dim);
    background: var(--panel-inset);
    border: none;
    border-right: 1px solid var(--frame-dim);
    cursor: pointer;
  }

  .speed button:last-child {
    border-right: none;
  }

  .speed button.on {
    color: var(--bg-deep);
    background: var(--olive);
  }

  .paused {
    font-family: var(--font-display);
    font-size: 12px;
    letter-spacing: 0.08em;
    text-transform: uppercase;
    color: var(--rust-bright);
  }
</style>
