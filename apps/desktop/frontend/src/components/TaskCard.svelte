<script lang="ts">
  import { Button } from '@bakery/ui'
  import type { Task } from '../lib/bindings'

  let {
    task,
    dragging = false,
    selected = false,
    ondragstart,
    ondragend,
    onopen,
    ondelete,
  }: {
    task: Task
    dragging?: boolean
    selected?: boolean
    ondragstart: (event: DragEvent) => void
    ondragend: () => void
    onopen: () => void
    ondelete: () => void
  } = $props()

  let confirming = $state(false)
  let allDone = $derived(task.subtasks_total > 0 && task.subtasks_done === task.subtasks_total)
</script>

<li
  class={['card', { dragging, selected }]}
  draggable={!confirming}
  data-task-id={task.id}
  {ondragstart}
  {ondragend}
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
