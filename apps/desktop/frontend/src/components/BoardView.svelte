<script lang="ts">
  import { Panel, TextField, Portrait } from '@bakery/ui'
  import TaskCard from './TaskCard.svelte'
  import ColumnHeader from './ColumnHeader.svelte'
  import type { Colony, LiveState } from '../lib/colony.svelte'
  import type { Letters } from '../lib/letters.svelte'
  import ColonyView from './ColonyView.svelte'

  let { colony, letters }: { colony: Colony; letters: Letters } = $props()

  // The board as cards, or as the colony from above.
  let mode = $state<'board' | 'colony'>('board')

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

  // Time controls, like RimWorld's: Space pauses or resumes, 1 normal, 2
  // fast. Not while typing, and not with a modifier held.
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

  // The card menu: prioritize for one of my agents, forbid or allow.
  let menu = $state<{ taskId: number; x: number; y: number } | null>(null)
  const menuTask = $derived(menu ? colony.view?.columns.flatMap((c) => c.tasks).find((t) => t.id === menu!.taskId) : undefined)
  const enabledAgents = $derived(
    colony.agents.filter((a) => a.agent_id && (colony.settings?.config.agents ?? []).includes(a.slug)),
  )

  function claimantOf(t: { claim?: { agent_id: number } | null }) {
    if (!t.claim) return null
    const a = colony.agentById(t.claim.agent_id)
    return a ? { name: a.name, seed: a.portrait_seed || a.slug } : { name: "Another member's agent", seed: '?' + t.claim.agent_id }
  }

  const MAX_FACES = 5

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

<svelte:window onkeydown={speedKeys} />

{#if colony.view}
  <!-- One element, so the screen's grid (board beside the task panel) sees
       one cell, not the header and the columns separately. -->
  <section class="board-view">
  <header class="board-head">
    <h1>{colony.view.board.name}</h1>
    {#if colony.live !== 'off'}
      <span class={['live', colony.live]} title="Changes by others show up here as they happen">{LIVE[colony.live]}</span>
    {/if}
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
    <div class="modes" role="group" aria-label="View">
      <button class={{ on: mode === 'board' }} aria-pressed={mode === 'board'} onclick={() => (mode = 'board')}>Board</button>
      <button class={{ on: mode === 'colony' }} aria-pressed={mode === 'colony'} onclick={() => (mode = 'colony')}>Colony</button>
    </div>
    <span class="spacer"></span>
    {#if colony.settings && !colony.settings.linked}
      <button class="linked" title="Agents cannot run on this board on this machine until it is linked to a repository" onclick={() => colony.openSettings()}>Not linked</button>
    {/if}
    {#if colony.present.length}
      <ul class="faces" aria-label="Who has this board open">
        {#each colony.present.slice(0, MAX_FACES) as m (m.id)}
          <li class="face" title={m.name}><Portrait seed={m.seed} size={22} alt={m.name} /></li>
        {/each}
        {#if colony.present.length > MAX_FACES}
          <li
            class="face more"
            title={colony.present
              .slice(MAX_FACES)
              .map((m) => m.name)
              .join(', ')}
          >
            +{colony.present.length - MAX_FACES}
          </li>
        {/if}
      </ul>
    {/if}
    <button class="gear" aria-label="Board settings" title="Board settings (this machine)" onclick={() => colony.openSettings()}>⚙</button>
  </header>
  {#if mode === 'colony'}
    <ColonyView {colony} {letters} />
  {:else}
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
              workTypeName={colony.workTypeName(task.work_type)}
              working={colony.workshop.isRunning(task.id)}
              claimant={claimantOf(task)}
              prioritizedFor={task.prioritized_agent_id ? (colony.agentById(task.prioritized_agent_id)?.name ?? 'another agent') : ''}
              onmenu={(e) => (menu = { taskId: task.id, x: e.clientX, y: e.clientY })}
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
  {/if}
    {#if menu && menuTask}
    <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
    <div class="menu-veil" onclick={() => (menu = null)} oncontextmenu={(e) => { e.preventDefault(); menu = null }}></div>
    <div class="card-menu" role="menu" style:left="{menu.x}px" style:top="{menu.y}px">
      <span class="menu-head">Prioritize for</span>
      {#each enabledAgents as a (a.slug)}
        <button role="menuitem" class={{ on: menuTask.prioritized_agent_id === a.agent_id }} onclick={() => { colony.draft(menuTask.id, a.agent_id, null); menu = null }}>{a.name}</button>
      {:else}
        <span class="dim">No agents enabled on this board here.</span>
      {/each}
      {#if menuTask.prioritized_agent_id}
        <button role="menuitem" onclick={() => { colony.draft(menuTask.id, 0, null); menu = null }}>Clear priority</button>
      {/if}
      <hr />
      <button role="menuitem" onclick={() => { colony.draft(menuTask.id, null, !menuTask.forbidden); menu = null }}>
        {menuTask.forbidden ? 'Allow for agents' : 'Forbid for agents'}
      </button>
    </div>
  {/if}
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

  .modes {
    display: flex;
    margin-left: 6px;
    border: 1px solid var(--frame-dim);
    border-radius: var(--radius);
    overflow: hidden;
  }

  .modes button {
    font: inherit;
    font-size: 11px;
    padding: 1px 8px;
    color: var(--text-dim);
    background: var(--panel-inset);
    border: none;
    cursor: pointer;
  }

  .modes button.on {
    color: var(--text);
    background: var(--panel-title);
  }

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

  .menu-veil {
    position: fixed;
    inset: 0;
    z-index: 20;
  }

  .card-menu {
    position: fixed;
    z-index: 21;
    display: flex;
    flex-direction: column;
    min-width: 170px;
    padding: 4px;
    background: var(--panel-raised);
    border: 1px solid var(--frame);
    border-radius: var(--radius);
    box-shadow: 0 4px 12px rgb(0 0 0 / 0.4);
  }

  .card-menu button {
    font: inherit;
    font-size: 13px;
    text-align: left;
    padding: 4px 8px;
    color: var(--text);
    background: none;
    border: none;
    border-radius: var(--radius);
    cursor: pointer;
  }

  .card-menu button:hover {
    background: var(--panel-inset);
  }

  .card-menu button.on::before {
    content: '⚑ ';
    color: var(--steel-bright);
  }

  .menu-head {
    padding: 2px 8px;
    font-size: 11px;
    color: var(--text-dim);
    text-transform: uppercase;
  }

  .card-menu hr {
    width: 100%;
    border: none;
    border-top: 1px solid var(--frame-dim);
  }

  .card-menu .dim {
    padding: 2px 8px;
    font-size: 12px;
    color: var(--text-dim);
  }

  .spacer {
    flex: 1;
  }

  .gear {
    font: inherit;
    font-size: 15px;
    line-height: 1;
    padding: 2px 5px;
    color: var(--text-dim);
    background: none;
    border: 1px solid transparent;
    border-radius: var(--radius);
    cursor: pointer;
  }

  .gear:hover {
    color: var(--text);
    border-color: var(--frame-dim);
  }

  .linked {
    padding: 0 6px;
    font: inherit;
    font-size: 11px;
    line-height: 16px;
    color: var(--text-dim);
    background: none;
    border: 1px dashed var(--frame-dim);
    border-radius: var(--radius);
    cursor: pointer;
  }

  .faces {
    display: flex;
    gap: 3px;
    margin: 0;
    padding: 0;
    list-style: none;
  }

  .face {
    display: grid;
    place-items: center;
    cursor: default;
  }

  .face.more {
    width: 22px;
    height: 22px;
    font-size: 10px;
    font-weight: 600;
    color: var(--text-dim);
    background: var(--panel-inset);
    border: 1px solid var(--frame);
    border-radius: var(--radius);
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
