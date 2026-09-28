<script lang="ts">
  import { untrack } from 'svelte'
  import { Button, TextField } from '@bakery/ui'
  import { TaskService, messageOf, type WorkType } from '../lib/bindings'

  // An agent's proposed subtasks, for a person to prune before anything is
  // saved: edit, drop and reorder the rows, then Accept adds exactly those,
  // in that order; Discard leaves the task as it was.
  type Row = { id: number; title: string; description: string; work_type: string }

  let {
    taskId,
    agentName,
    plan,
    proposed,
    workTypes,
    cost,
    onaccepted,
    ondiscard,
  }: {
    taskId: number
    agentName: string
    plan: string
    proposed: { title: string; description: string; work_type: string }[]
    workTypes: WorkType[]
    cost: number
    onaccepted: () => void
    ondiscard: () => void
  } = $props()

  let rows = $state<Row[]>(untrack(() => proposed.map((p, i) => ({ id: i, title: p.title, description: p.description ?? '', work_type: p.work_type }))))
  let dragged = $state<number | null>(null)
  let saving = $state(false)
  let error = $state('')

  function drop(index: number) {
    if (dragged === null) return
    const from = rows.findIndex((r) => r.id === dragged)
    const [row] = rows.splice(from, 1)
    rows.splice(index, 0, row)
    dragged = null
  }

  const valid = $derived(rows.length > 0 && rows.every((r) => r.title.trim()))

  async function accept() {
    saving = true
    try {
      await TaskService.ExpandTask(
        taskId,
        rows.map((r) => ({ title: r.title.trim(), description: r.description.trim(), work_type: r.work_type })),
      )
      onaccepted()
    } catch (err) {
      error = messageOf(err)
    }
    saving = false
  }
</script>

<div class="review" aria-label="Proposed plan">
  <header>
    <strong>{agentName}'s plan</strong>
    <span class="dim">${cost.toFixed(2)}</span>
  </header>
  {#if plan}<p class="plan">{plan}</p>{/if}
  {#if error}<p class="error" role="alert">{error}</p>{/if}
  <ol class="rows">
    {#each rows as r, i (r.id)}
      <li
        class={{ dragging: dragged === r.id }}
        ondragover={(e) => dragged !== null && e.preventDefault()}
        ondrop={(e) => {
          e.preventDefault()
          drop(i)
        }}
      >
        <span
          class="handle"
          draggable="true"
          role="button"
          tabindex="-1"
          aria-label="Drag to reorder"
          ondragstart={(e) => {
            dragged = r.id
            e.dataTransfer?.setData('application/x-bakery-plan-row', String(r.id))
          }}
          ondragend={() => (dragged = null)}>⋮⋮</span
        >
        <div class="fields">
          <div class="line">
            <TextField aria-label="Title" maxlength={200} bind:value={r.title} />
            <select aria-label="Work type" bind:value={r.work_type}>
              <option value="">None</option>
              {#each workTypes as w (w.key)}<option value={w.key}>{w.name}</option>{/each}
            </select>
            <button class="remove" aria-label="Drop this subtask" onclick={() => (rows = rows.filter((x) => x.id !== r.id))}>×</button>
          </div>
          <textarea aria-label="Description" placeholder="What done means" bind:value={r.description}></textarea>
        </div>
      </li>
    {/each}
  </ol>
  <div class="actions">
    <Button variant="confirm" disabled={saving || !valid} onclick={accept}>Accept {rows.length}</Button>
    <Button disabled={saving} onclick={ondiscard}>Discard</Button>
  </div>
</div>

<style>
  .review {
    display: flex;
    flex-direction: column;
    gap: 6px;
    padding: 8px;
    background: var(--panel-inset);
    border: 1px solid var(--frame);
    border-radius: var(--radius);
  }

  header {
    display: flex;
    justify-content: space-between;
  }

  .plan {
    margin: 0;
    white-space: pre-wrap;
  }

  .rows {
    list-style: none;
    margin: 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .rows li {
    display: flex;
    gap: 6px;
    align-items: flex-start;
  }

  .rows li.dragging {
    opacity: 0.4;
  }

  .handle {
    padding-top: 6px;
    font-size: 11px;
    letter-spacing: -2px;
    color: var(--text-faint);
    cursor: grab;
  }

  .fields {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 3px;
  }

  .line {
    display: flex;
    gap: 4px;
    align-items: center;
  }

  .line :global(.field) {
    flex: 1;
  }

  select,
  textarea {
    font: inherit;
    font-size: 12px;
    color: var(--text);
    padding: 4px 6px;
    background: var(--panel-raised);
    border: 1px solid var(--frame-dim);
    border-radius: var(--radius);
  }

  textarea {
    min-height: 34px;
    resize: vertical;
    user-select: text;
  }

  .remove {
    font: inherit;
    font-size: 15px;
    padding: 0 4px;
    color: var(--rust-bright);
    background: none;
    border: none;
    cursor: pointer;
  }

  .actions {
    display: flex;
    gap: 6px;
  }

  .dim {
    color: var(--text-dim);
    font-size: 12px;
  }

  .error {
    margin: 0;
    padding: 5px 8px;
    background: color-mix(in srgb, var(--rust) 35%, transparent);
    border: 1px solid var(--rust);
    border-radius: var(--radius);
  }
</style>
