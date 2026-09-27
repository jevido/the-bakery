<script lang="ts">
  import { Panel, TextField } from '@bakery/ui'
  import TaskCard from './TaskCard.svelte'
  import ColumnHeader from './ColumnHeader.svelte'
  import type { Colony, LiveState } from '../lib/colony.svelte'

  let { colony }: { colony: Colony } = $props()

  let draggedId = $state<number | null>(null)
  // Where the dragged task would land: a column and an index among the
  // column's other tasks.
  let dropTarget = $state<{ column: number; index: number } | null>(null)

  const LIVE: Record<LiveState, string> = {
    off: '',
    connecting: 'Connecting…',
    live: 'Live',
    reconnecting: 'Reconnecting…',
    'signed-out': 'Signed out',
  }
  let newTitle = $state('')

  // Columns are dragged by their header, with their own data type so a
  // column never lands in a task list and a card never moves a column.
  const COLUMN_TYPE = 'application/x-bakery-column'
  let draggedColumn = $state<number | null>(null)
  // Where the dragged column would land, among the other columns.
  let columnDrop = $state<number | null>(null)
  let addingColumn = $state(false)
  let newColumn = $state('')

  function columnDragStart(event: DragEvent, id: number) {
    draggedColumn = id
    event.dataTransfer?.setData(COLUMN_TYPE, String(id))
    if (event.dataTransfer) event.dataTransfer.effectAllowed = 'move'
  }

  function columnDragEnd() {
    draggedColumn = null
    columnDrop = null
  }

  // The drop index is the number of other columns whose middle is left of
  // the pointer.
  function columnDragOver(event: DragEvent) {
    if (draggedColumn === null) return
    event.preventDefault()
    const others = [...(event.currentTarget as HTMLElement).querySelectorAll<HTMLElement>('[data-column-id]')].filter(
      (el) => Number(el.dataset.columnId) !== draggedColumn,
    )
    columnDrop = others.filter((el) => {
      const r = el.getBoundingClientRect()
      return r.left + r.width / 2 < event.clientX
    }).length
  }

  function columnDropped(event: DragEvent) {
    if (draggedColumn === null) return
    event.preventDefault()
    const id = draggedColumn
    const index = columnDrop
    columnDragEnd()
    if (index !== null) colony.moveColumn(id, index)
  }

  // Which side of which rendered column shows the drop marker.
  function columnMarker(index: number, columns: { id: number }[]): 'before' | 'after' | null {
    if (columnDrop === null || draggedColumn === null) return null
    const others = columns.filter((c) => c.id !== draggedColumn)
    if (columnDrop < others.length) return others[columnDrop].id === columns[index].id ? 'before' : null
    return others[others.length - 1]?.id === columns[index].id ? 'after' : null
  }

  async function addColumn(event: KeyboardEvent) {
    if (event.key === 'Escape') {
      addingColumn = false
      newColumn = ''
      return
    }
    if (event.key !== 'Enter') return
    const name = newColumn.trim()
    if (name && (await colony.addColumn(name))) {
      newColumn = ''
      addingColumn = false
    }
  }

  function focus(node: HTMLInputElement) {
    node.focus()
  }

  function dragStart(event: DragEvent, id: number) {
    draggedId = id
    event.dataTransfer?.setData('text/plain', String(id))
    if (event.dataTransfer) event.dataTransfer.effectAllowed = 'move'
  }

  function dragEnd() {
    draggedId = null
    dropTarget = null
  }

  // The drop index is the number of other cards whose middle is above the
  // pointer.
  function dragOver(event: DragEvent, column: number) {
    if (draggedId === null) return
    event.preventDefault()
    if (event.dataTransfer) event.dataTransfer.dropEffect = 'move'
    const list = event.currentTarget as HTMLElement
    const cards = [...list.querySelectorAll<HTMLElement>('[data-task-id]')].filter(
      (el) => Number(el.dataset.taskId) !== draggedId,
    )
    const index = cards.filter((el) => {
      const r = el.getBoundingClientRect()
      return r.top + r.height / 2 < event.clientY
    }).length
    dropTarget = { column, index }
  }

  function dragLeave(event: DragEvent) {
    const list = event.currentTarget as HTMLElement
    if (!list.contains(event.relatedTarget as Node | null)) dropTarget = null
  }

  function drop(event: DragEvent) {
    event.preventDefault()
    const id = draggedId
    const target = dropTarget
    dragEnd()
    if (id !== null && target) colony.moveTask(id, target.column, target.index)
  }

  async function addTask(event: KeyboardEvent) {
    if (event.key !== 'Enter') return
    const title = newTitle.trim()
    if (title && (await colony.createTask(title))) newTitle = ''
  }

  // Index of the indicator among the rendered cards (which still include the
  // dragged one when it is in this column).
  function indicatorAt(column: number, tasks: { id: number }[]): number {
    if (!dropTarget || dropTarget.column !== column) return -1
    let seen = 0
    for (let i = 0; i < tasks.length; i++) {
      if (tasks[i].id === draggedId) continue
      if (seen === dropTarget.index) return i
      seen++
    }
    return tasks.length
  }
</script>

{#if colony.view}
  <!-- One element, so the screen's grid (board beside the task panel) sees
       one cell, not the header and the columns separately. -->
  <section class="board-view">
  <header class="board-head">
    <h1>{colony.view.board.name}</h1>
    {#if colony.live !== 'off'}
      <span class={['live', colony.live]} title="Changes by others show up here as they happen">{LIVE[colony.live]}</span>
    {/if}
  </header>
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div
    class="board"
    style:--columns={colony.view.columns.length}
    ondragover={columnDragOver}
    ondrop={columnDropped}
  >
    {#each colony.view.columns as col, c (col.id)}
      {@const at = indicatorAt(col.id, col.tasks)}
      {@const marker = columnMarker(c, colony.view.columns)}
      <div
        class={['column', marker && `drop-${marker}`, { dragging: draggedColumn === col.id }]}
        data-column-id={col.id}
      >
      <Panel>
        {#snippet header()}
          <ColumnHeader
            column={col}
            last={colony.view!.columns.length === 1}
            ondragstart={(e) => columnDragStart(e, col.id)}
            ondragend={columnDragEnd}
            onrename={(name) => colony.renameColumn(col.id, name)}
            ondelete={() => colony.deleteColumn(col.id)}
          />
        {/snippet}
        <ul
          class={['tasks', { over: dropTarget?.column === col.id }]}
          data-column={col.id}
          ondragover={(e) => dragOver(e, col.id)}
          ondragleave={dragLeave}
          ondrop={drop}
        >
          {#each col.tasks as task, i (task.id)}
            {#if at === i}<li class="indicator" aria-hidden="true"></li>{/if}
            <TaskCard
              {task}
              dragging={draggedId === task.id}
              ondragstart={(e) => dragStart(e, task.id)}
              ondragend={dragEnd}
              selected={colony.openTaskId === task.id}
              onopen={() => colony.openTask(task.id)}
              ondelete={() => colony.deleteTask(task.id)}
            />
          {/each}
          {#if at === col.tasks.length}<li class="indicator" aria-hidden="true"></li>{/if}
        </ul>
        {#if c === 0}
          <div class="add">
            <TextField placeholder="New task, then Enter" maxlength={200} bind:value={newTitle} onkeydown={addTask} />
          </div>
        {/if}
      </Panel>
      </div>
    {/each}
    <div class="add-column">
      {#if addingColumn}
        <input
          class="column-input"
          aria-label="New column name"
          placeholder="Column name, then Enter"
          maxlength={40}
          bind:value={newColumn}
          onkeydown={addColumn}
          onblur={() => !newColumn.trim() && (addingColumn = false)}
          {@attach focus}
        />
      {:else}
        <button class="add-column-button" onclick={() => (addingColumn = true)}>+ Column</button>
      {/if}
    </div>
  </div>
  </section>
{/if}

<style>
  .board-view {
    display: flex;
    flex-direction: column;
    min-width: 0;
    min-height: 0;
  }

  .board-head {
    display: flex;
    align-items: center;
    gap: var(--gap);
    padding: 0 2px 6px;
  }

  h1 {
    margin: 0;
    font-family: var(--font-display);
    font-size: 16px;
    font-weight: 600;
  }

  .live {
    padding: 0 6px;
    font-size: 11px;
    line-height: 16px;
    color: var(--text-dim);
    border: 1px solid var(--frame-dim);
    border-radius: var(--radius);
  }

  .live.live {
    color: var(--olive-bright);
    border-color: var(--olive);
  }

  .live.reconnecting,
  .live.signed-out {
    color: var(--rust-bright);
    border-color: var(--rust);
  }

  /* As many columns as the board has; past the window's width it scrolls
     sideways. */
  .board {
    display: grid;
    grid-template-columns: repeat(var(--columns), minmax(160px, 1fr)) 100px;
    gap: var(--gap);
    flex: 1;
    min-height: 0;
    overflow-x: auto;
  }

  .column {
    display: flex;
    flex-direction: column;
    min-height: 0;
    min-width: 0;
    border-radius: var(--radius);
  }

  .column :global(.panel) {
    flex: 1;
    min-height: 0;
  }

  .column.dragging {
    opacity: 0.4;
  }

  .column.drop-before {
    box-shadow: -5px 0 0 -2px var(--steel-bright);
  }

  .column.drop-after {
    box-shadow: 5px 0 0 -2px var(--steel-bright);
  }

  .add-column {
    min-width: 0;
  }

  .add-column-button {
    width: 100%;
    font: inherit;
    color: var(--text-dim);
    padding: 6px 8px;
    text-align: left;
    background: none;
    border: 1px dashed var(--frame-dim);
    border-radius: var(--radius);
    cursor: pointer;
  }

  .add-column-button:hover {
    color: var(--text);
    border-color: var(--frame);
  }

  .column-input {
    width: 100%;
    box-sizing: border-box;
    font: inherit;
    color: var(--text);
    padding: 6px 8px;
    background: var(--panel-inset);
    border: 1px solid var(--frame);
    border-radius: var(--radius);
    user-select: text;
  }

  .board :global(.panel > .body) {
    display: flex;
    flex-direction: column;
    gap: var(--gap);
    flex: 1;
  }

  .tasks {
    list-style: none;
    margin: 0;
    padding: 2px;
    display: flex;
    flex-direction: column;
    gap: 6px;
    flex: 1;
    min-height: 48px;
    border: 1px dashed transparent;
    border-radius: var(--radius);
  }

  .tasks.over {
    border-color: var(--frame-dim);
    background: #00000018;
  }

  .indicator {
    height: 2px;
    margin: -4px 0;
    background: var(--steel-bright);
    box-shadow: 0 0 4px var(--steel);
  }
</style>
