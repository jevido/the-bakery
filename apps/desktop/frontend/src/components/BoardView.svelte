<script lang="ts">
  import Panel from './Panel.svelte'
  import TaskCard from './TaskCard.svelte'
  import TextField from './TextField.svelte'
  import { COLUMN_TITLES, type Colony } from '../lib/colony.svelte'

  let { colony }: { colony: Colony } = $props()

  let draggedId = $state<number | null>(null)
  // Where the dragged task would land: a column and an index among the
  // column's other tasks.
  let dropTarget = $state<{ column: string; index: number } | null>(null)
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
  function dragOver(event: DragEvent, column: string) {
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
  function indicatorAt(column: string, tasks: { id: number }[]): number {
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
  <div class="board">
    {#each colony.view.columns as col (col.column)}
      {@const at = indicatorAt(col.column, col.tasks)}
      <Panel title={`${COLUMN_TITLES[col.column] ?? col.column} · ${col.tasks.length}`}>
        <ul
          class={['tasks', { over: dropTarget?.column === col.column }]}
          data-column={col.column}
          ondragover={(e) => dragOver(e, col.column)}
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
              onrename={(title) => colony.renameTask(task.id, title)}
              ondelete={() => colony.deleteTask(task.id)}
            />
          {/each}
          {#if at === col.tasks.length}<li class="indicator" aria-hidden="true"></li>{/if}
        </ul>
        {#if col.column === 'backlog'}
          <div class="add">
            <TextField placeholder="New task, then Enter" maxlength={200} bind:value={newTitle} onkeydown={addTask} />
          </div>
        {/if}
      </Panel>
    {/each}
  </div>
{/if}

<style>
  .board {
    display: grid;
    grid-template-columns: repeat(4, minmax(180px, 1fr));
    gap: var(--gap);
    height: 100%;
    min-height: 0;
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
