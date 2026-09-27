<script lang="ts">
  import type { Task } from '../lib/bindings'
  import Button from './Button.svelte'

  let {
    task,
    dragging = false,
    ondragstart,
    ondragend,
    onrename,
    ondelete,
  }: {
    task: Task
    dragging?: boolean
    ondragstart: (event: DragEvent) => void
    ondragend: () => void
    onrename: (title: string) => void
    ondelete: () => void
  } = $props()

  let editing = $state(false)
  let confirming = $state(false)
  let draft = $state('')

  function startEdit() {
    draft = task.title
    editing = true
  }

  function finishEdit(save: boolean) {
    if (!editing) return
    editing = false
    const title = draft.trim()
    if (save && title && title !== task.title) onrename(title)
  }

  function onkeydown(event: KeyboardEvent) {
    if (event.key === 'Enter') finishEdit(true)
    if (event.key === 'Escape') finishEdit(false)
  }

  // Focus and select the title field as soon as it appears.
  function focusSelect(node: HTMLInputElement) {
    node.focus()
    node.select()
  }
</script>

<li
  class={['card', { dragging }]}
  draggable={!editing && !confirming}
  data-task-id={task.id}
  {ondragstart}
  {ondragend}
>
  {#if editing}
    <input
      class="title-input"
      maxlength={200}
      bind:value={draft}
      {onkeydown}
      onblur={() => finishEdit(true)}
      {@attach focusSelect}
    />
  {:else if confirming}
    <div class="confirm">
      <span>Delete “{task.title}”?</span>
      <div class="confirm-actions">
        <Button variant="danger" onclick={ondelete}>Delete</Button>
        <Button onclick={() => (confirming = false)}>Keep</Button>
      </div>
    </div>
  {:else}
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <span class="title" ondblclick={startEdit} title="Double-click to rename">{task.title}</span>
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

  .title-input {
    flex: 1;
    font: inherit;
    color: var(--text);
    padding: 2px 4px;
    background: var(--panel-inset);
    border: 1px solid var(--frame);
    border-radius: var(--radius);
    user-select: text;
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
