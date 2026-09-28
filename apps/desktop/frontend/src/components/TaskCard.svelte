<script lang="ts">
  import Portrait from './Portrait.svelte'
  import { Button } from '@bakery/ui'
  import type { Task } from '../lib/bindings'
  import { workTypeStyle } from '../lib/worktype'

  let {
    task,
    dragging = false,
    selected = false,
    workTypeName = '',
    working = false,
    claimant = null,
    prioritizedFor = '',
    onmenu,
    ondragstart,
    ondragend,
    onopen,
    ondelete,
  }: {
    task: Task
    dragging?: boolean
    selected?: boolean
    workTypeName?: string
    // working marks a task an agent is on right now.
    working?: boolean
    // claimant is the agent holding the task (a name and a portrait seed;
    // "?" for another member's); prioritizedFor the agent it is set aside for.
    claimant?: { name: string; seed: string } | null
    prioritizedFor?: string
    onmenu?: (event: MouseEvent) => void
    ondragstart: (event: DragEvent) => void
    ondragend: () => void
    onopen: () => void
    ondelete: () => void
  } = $props()

  let confirming = $state(false)
  let allDone = $derived(task.subtasks_total > 0 && task.subtasks_done === task.subtasks_total)
</script>

<li
  class={['card', { dragging, selected, forbidden: task.forbidden, claimed: !!claimant }]}
  draggable={!confirming}
  data-task-id={task.id}
  {ondragstart}
  {ondragend}
  oncontextmenu={(e) => {
    if (!onmenu) return
    e.preventDefault()
    onmenu(e)
  }}
>
  {#if confirming}
    <div class="confirm">
      <span>Delete “{task.title}”?</span>
      <div class="confirm-actions">
        <Button variant="danger" onclick={ondelete}>Delete</Button>
        <Button onclick={() => (confirming = false)}>Keep</Button>
      </div>
    </div>
  {:else}
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <span class="title" ondblclick={onopen} title="Double-click to open">{task.title}</span>
    {#if claimant}
      <span class="claimant" title={`${claimant.name} is working on this`}>
        <Portrait seed={claimant.seed} size={16} />
      </span>
    {:else if working}<span class="working" title="An agent is working on this">Working</span>{/if}
    {#if prioritizedFor}<span class="flag" title={`Set aside for ${prioritizedFor}`}>⚑</span>{/if}
    {#if task.forbidden}<span class="forbid" title="Forbidden for agents">⊘</span>{/if}
    {#if task.work_type}
      <span class="wt" style={workTypeStyle(task.work_type)} title="Work type">{workTypeName || task.work_type}</span>
    {/if}
    {#if task.subtasks_total > 0}
      <span class={['badge', { full: allDone }]} title="Subtasks done">{task.subtasks_done}/{task.subtasks_total}</span>
    {/if}
    <button class="delete" aria-label="Delete task" onclick={() => (confirming = true)}>×</button>
  {/if}
</li>

<style>
  .card {
    display: flex;
    align-items: flex-start;
    gap: 6px;
    padding: 6px 8px;
    background: var(--panel-raised);
    border: 1px solid var(--frame-dim);
    border-radius: var(--radius);
    box-shadow: inset 0 1px 0 #ffffff10, var(--shadow-outer);
    cursor: grab;
  }

  .card:hover {
    border-color: var(--frame);
  }

  .dragging {
    opacity: 0.4;
  }

  .selected {
    border-color: var(--steel-bright);
  }

  .wt {
    padding: 0 5px;
    font-size: 11px;
    line-height: 16px;
    white-space: nowrap;
    color: var(--wt-fg);
    background: var(--wt-bg);
    border: 1px solid var(--wt-border);
    border-radius: var(--radius);
  }

  .claimant {
    display: inline-flex;
  }

  /* A thin moving stripe along the bottom while an agent holds the card. */
  .card.claimed {
    position: relative;
    overflow: hidden;
  }

  .card.claimed::after {
    content: '';
    position: absolute;
    left: 0;
    right: 0;
    bottom: 0;
    height: 2px;
    background: linear-gradient(90deg, transparent, var(--olive-bright), transparent);
    background-size: 50% 100%;
    background-repeat: no-repeat;
    animation: stripe 1.8s linear infinite;
  }

  @keyframes stripe {
    from {
      background-position: -50% 0;
    }
    to {
      background-position: 150% 0;
    }
  }

  .card.forbidden {
    border-color: var(--rust);
  }

  .flag {
    font-size: 12px;
    line-height: 16px;
    color: var(--steel-bright);
  }

  .forbid {
    font-size: 13px;
    line-height: 16px;
    font-weight: 700;
    color: var(--rust-bright);
  }

  .working {
    padding: 0 5px;
    font-size: 11px;
    line-height: 16px;
    color: var(--olive-bright);
    border: 1px solid var(--olive);
    border-radius: var(--radius);
    animation: pulse 1.6s ease-in-out infinite;
  }

  @keyframes pulse {
    50% {
      opacity: 0.5;
    }
  }

  @media (prefers-reduced-motion: reduce) {
    .working,
    .card.claimed::after {
      animation: none;
    }
  }

  .badge {
    padding: 0 5px;
    font-size: 11px;
    line-height: 16px;
    color: var(--text-dim);
    background: var(--panel-inset);
    border: 1px solid var(--frame-dim);
    border-radius: var(--radius);
  }

  .badge.full {
    color: var(--bg-deep);
    background: var(--olive);
    border-color: var(--olive-bright);
  }

  .title {
    flex: 1;
    overflow-wrap: anywhere;
  }

  .delete {
    visibility: hidden;
    padding: 0 4px;
    font-size: 15px;
    line-height: 1.2;
    color: var(--text-faint);
    background: none;
    border: none;
    cursor: pointer;
  }

  .card:hover .delete,
  .delete:focus-visible {
    visibility: visible;
  }

  .delete:hover {
    color: var(--rust-bright);
  }

  .confirm {
    display: flex;
    flex-direction: column;
    gap: 6px;
    width: 100%;
  }

  .confirm-actions {
    display: flex;
    gap: 6px;
  }
</style>
