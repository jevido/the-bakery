<script lang="ts">
  import type { ColumnState } from '../lib/colony.svelte'

  // One column's title bar: a drag handle, the name (double-click to
  // rename), the task count, and a menu with Delete.
  let {
    column,
    last,
    ondragstart,
    ondragend,
    onrename,
    ondelete,
  }: {
    column: ColumnState
    last: boolean
    ondragstart: (event: DragEvent) => void
    ondragend: () => void
    onrename: (name: string) => void
    ondelete: () => void
  } = $props()

  let editing = $state(false)
  let draft = $state('')
  let menu = $state(false)

  // Why Delete is off, if it is.
  let blocked = $derived(
    column.tasks.length > 0
      ? 'Move its tasks out first: only an empty column can be deleted.'
      : last
        ? 'A board keeps at least one column.'
        : '',
  )

  function startEdit() {
    draft = column.name
    editing = true
  }

  function finishEdit(save: boolean) {
    if (!editing) return
    editing = false
    const name = draft.trim()
    if (save && name && name !== column.name) onrename(name)
  }

  function onkeydown(event: KeyboardEvent) {
    if (event.key === 'Enter') finishEdit(true)
    if (event.key === 'Escape') {
      event.stopPropagation()
      finishEdit(false)
    }
  }

  function focusSelect(node: HTMLInputElement) {
    node.focus()
    node.select()
  }

  function closeMenu(event: MouseEvent) {
    if (menu && !(event.target as HTMLElement).closest('.menu-wrap')) menu = false
  }
</script>

<svelte:window onclick={closeMenu} />

<span
  class="handle"
  draggable="true"
  role="button"
  tabindex="-1"
  aria-label="Drag to move the column"
  title="Drag to move the column"
  {ondragstart}
  {ondragend}>⋮⋮</span
>
{#if editing}
  <input
    class="name-input"
    aria-label="Column name"
    maxlength={40}
    bind:value={draft}
    {onkeydown}
    onblur={() => finishEdit(true)}
    {@attach focusSelect}
  />
{:else}
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <span class="name" ondblclick={startEdit} title="Double-click to rename">{column.name}</span>
  <span class="count">· {column.tasks.length}</span>
{/if}
<span class="menu-wrap">
  <button class="menu-button" aria-label="Column menu" aria-expanded={menu} onclick={() => (menu = !menu)}>⋯</button>
  {#if menu}
    <div class="menu" role="menu">
      <button role="menuitem" onclick={() => ((menu = false), startEdit())}>Rename</button>
      <button
        role="menuitem"
        class="danger"
        disabled={blocked !== ''}
        title={blocked || 'Delete this column'}
        onclick={() => ((menu = false), ondelete())}>Delete column</button
      >
    </div>
  {/if}
</span>

<style>
  .handle {
    color: var(--text-faint);
    font-size: 11px;
    letter-spacing: -2px;
    cursor: grab;
  }

  .name {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .count {
    color: var(--text-dim);
    white-space: nowrap;
  }

  .name-input {
    flex: 1;
    min-width: 0;
    font: inherit;
    color: var(--text);
    padding: 1px 4px;
    background: var(--panel-inset);
    border: 1px solid var(--frame);
    border-radius: var(--radius);
    user-select: text;
  }

  .menu-wrap {
    position: relative;
    margin-left: auto;
  }

  .menu-button {
    padding: 0 4px;
    font-size: 15px;
    line-height: 1;
    color: var(--text-dim);
    background: none;
    border: none;
    cursor: pointer;
  }

  .menu-button:hover {
    color: var(--text);
  }

  .menu {
    position: absolute;
    right: 0;
    top: calc(100% + 4px);
    z-index: 10;
    display: flex;
    flex-direction: column;
    min-width: 140px;
    padding: 4px;
    font-family: var(--font-body);
    font-size: 13px;
    font-weight: 400;
    letter-spacing: 0;
    background: var(--panel-raised);
    border: 1px solid var(--frame);
    border-radius: var(--radius);
    box-shadow: var(--shadow-outer);
  }

  .menu button {
    text-align: left;
    font: inherit;
    color: var(--text);
    padding: 4px 8px;
    background: none;
    border: none;
    border-radius: var(--radius);
    cursor: pointer;
  }

  .menu button:hover:not(:disabled) {
    background: var(--panel-title);
  }

  .menu .danger {
    color: var(--rust-bright);
  }

  .menu button:disabled {
    color: var(--text-faint);
    cursor: not-allowed;
  }
</style>
