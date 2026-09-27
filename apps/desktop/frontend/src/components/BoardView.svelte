<script lang="ts">
  import { Panel, TextField } from '@bakery/ui'
  import TaskCard from './TaskCard.svelte'
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
  <header class="board-head">
    <h1>{colony.view.board.name}</h1>
    {#if colony.live !== 'off'}
      <span class={['live', colony.live]} title="Changes by others show up here as they happen">{LIVE[colony.live]}</span>
    {/if}
  </header>
  <div class="board">
    {#each colony.view.columns as col, c (col.id)}
      {@const at = indicatorAt(col.id, col.tasks)}
      <Panel title={`${col.name} · ${col.tasks.length}`}>
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
    {/each}
  </div>
{/if}

<style>
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
    grid-auto-flow: column;
    grid-auto-columns: minmax(200px, 1fr);
    gap: var(--gap);
    height: 100%;
    min-height: 0;
    overflow-x: auto;
  }

  .board :global(.panel) {
    min-height: 0;
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
