<script lang="ts">
  import { Panel, Button, TextField } from '@bakery/ui'
  import Portrait from './Portrait.svelte'
  import { AgentsService, BoardsService, messageOf, type AgentSummary } from '../lib/bindings'
  import type { Colony } from '../lib/colony.svelte'
  import type { AgentSync } from '../lib/agentsync.svelte'

  // RimWorld's work tab: one row per agent, one column per work type of the
  // open guild, each cell a priority from 1 (first) to 4 (last), or empty
  // for never. Priorities for work types this guild doesn't have are left
  // alone; they count in other guilds.
  let { colony, sync }: { colony: Colony; sync: AgentSync } = $props()

  let agents = $state<AgentSummary[]>([])
  let error = $state('')
  let editing = $state(false)

  async function load() {
    try {
      agents = (await AgentsService.List()) ?? []
      error = ''
    } catch (err) {
      error = messageOf(err)
    }
  }
  load()
  $effect(() => {
    if (sync.changes > 0) load()
  })

  // Clicks are shown at once and saved a second after the last one.
  const pending = new Map<string, Record<string, number>>()
  const timers = new Map<string, ReturnType<typeof setTimeout>>()

  function priorityOf(a: AgentSummary, key: string): number {
    return a.work_priorities?.[key] ?? 0
  }

  function step(a: AgentSummary, key: string, back: boolean) {
    const now = priorityOf(a, key)
    // off → 1 → 2 → 3 → 4 → off, or the other way.
    const next = back ? (now + 4) % 5 : (now + 1) % 5
    a.work_priorities = { ...(a.work_priorities ?? {}) }
    if (next === 0) delete a.work_priorities[key]
    else a.work_priorities[key] = next
    const changes = pending.get(a.slug) ?? {}
    changes[key] = next
    pending.set(a.slug, changes)
    clearTimeout(timers.get(a.slug))
    timers.set(
      a.slug,
      setTimeout(() => save(a.slug), 1000),
    )
  }

  async function save(slug: string) {
    const changes = pending.get(slug)
    pending.delete(slug)
    timers.delete(slug)
    if (!changes) return
    try {
      await AgentsService.SetWorkPriorities(slug, changes)
    } catch (err) {
      error = messageOf(err)
      load()
    }
  }

  const ORDINAL = ['', 'first', 'second', 'third', 'last']

  function hint(a: AgentSummary, name: string, key: string): string {
    const p = priorityOf(a, key)
    return p ? `${a.name} will do ${name} ${ORDINAL[p]}` : `${a.name} never does ${name}`
  }

  // The work type editor.
  let newKey = $state('')
  let newName = $state('')
  let dragKey = $state<string | null>(null)

  async function guarded(f: () => Promise<unknown>) {
    try {
      await f()
      error = ''
    } catch (err) {
      error = messageOf(err)
    }
    await colony.reloadWorkTypes()
  }

  async function add(event: SubmitEvent) {
    event.preventDefault()
    if (colony.guildId === null) return
    const key = newKey.trim()
    await guarded(() => BoardsService.AddWorkType(colony.guildId!, key, newName.trim()))
    if (!error) {
      newKey = ''
      newName = ''
    }
  }

  function rename(key: string, current: string, event: Event) {
    const input = event.currentTarget as HTMLInputElement
    const name = input.value.trim()
    if (!name) input.value = current
    else if (name !== current && colony.guildId !== null) guarded(() => BoardsService.RenameWorkType(colony.guildId!, key, name))
  }

  function dropOn(index: number) {
    const key = dragKey
    dragKey = null
    if (key && colony.guildId !== null) guarded(() => BoardsService.MoveWorkType(colony.guildId!, key, index))
  }
</script>

<div class="work">
  <Panel title={`Work · ${colony.guild?.name ?? ''}`}>
    {#snippet actions()}
      <Button onclick={() => (editing = !editing)}>{editing ? 'Done editing' : 'Edit work types'}</Button>
    {/snippet}
    {#if error}<p class="error" role="alert">{error}</p>{/if}

    {#if colony.workTypes.length === 0}
      <p class="dim">This guild has no work types.</p>
    {:else if agents.length === 0}
      <p class="dim">No agents yet. Make some on the Agents screen.</p>
    {:else}
      <div class="scroll">
        <table>
          <thead>
            <tr>
              <th></th>
              {#each colony.workTypes as w (w.key)}
                <th class="type"><span>{w.name}</span></th>
              {/each}
            </tr>
          </thead>
          <tbody>
            {#each agents as a (a.slug)}
              <tr>
                <th class="agent">
                  <Portrait seed={a.portrait_seed || a.slug} size={24} />
                  <span>{a.name || a.slug}</span>
                </th>
                {#each colony.workTypes as w (w.key)}
                  {@const p = priorityOf(a, w.key)}
                  <td>
                    <button
                      class={['cell', `p${p}`]}
                      title={hint(a, w.name, w.key)}
                      aria-label={hint(a, w.name, w.key)}
                      onclick={() => step(a, w.key, false)}
                      oncontextmenu={(e) => {
                        e.preventDefault()
                        step(a, w.key, true)
                      }}>{p || ''}</button
                    >
                  </td>
                {/each}
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
      <p class="dim small">Click a cell to raise it (1 is done first), right-click to lower it. Empty means never.</p>
    {/if}
  </Panel>

  {#if editing}
    <Panel title="Work types">
      <ul class="types">
        {#each colony.workTypes as w, i (w.key)}
          <li
            class={{ dragging: dragKey === w.key }}
            ondragover={(e) => dragKey && e.preventDefault()}
            ondrop={(e) => {
              e.preventDefault()
              dropOn(i)
            }}
          >
            <span
              class="handle"
              draggable="true"
              role="button"
              tabindex="-1"
              aria-label="Drag to reorder"
              ondragstart={(e) => {
                dragKey = w.key
                e.dataTransfer?.setData('application/x-bakery-work-type', w.key)
              }}
              ondragend={() => (dragKey = null)}>⋮⋮</span
            >
            <input class="name" aria-label="Name" maxlength={40} value={w.name} onchange={(e) => rename(w.key, w.name, e)} />
            <code>{w.key}</code>
            <button
              class="link danger"
              onclick={() => colony.guildId !== null && guarded(() => BoardsService.DeleteWorkType(colony.guildId!, w.key))}
              >Delete</button
            >
          </li>
        {/each}
      </ul>
      <form class="add" onsubmit={add}>
        <TextField placeholder="key, e.g. lore" maxlength={32} bind:value={newKey} />
        <TextField placeholder="Name, e.g. Lore" maxlength={40} bind:value={newName} />
        <Button type="submit" variant="confirm" disabled={!newKey.trim() || !newName.trim()}>Add</Button>
      </form>
      <p class="dim small">A work type some task still has cannot be deleted.</p>
    </Panel>
  {/if}
</div>

<style>
  .work {
    display: flex;
    flex-direction: column;
    gap: var(--gap);
    min-height: 0;
  }

  .scroll {
    overflow: auto;
  }

  table {
    border-collapse: separate;
    border-spacing: 2px;
  }

  th.type {
    height: 96px;
    vertical-align: bottom;
    padding: 0 0 4px;
  }

  th.type span {
    display: inline-block;
    writing-mode: vertical-rl;
    transform: rotate(180deg);
    font-weight: 600;
    font-size: 12px;
    color: var(--text-dim);
    white-space: nowrap;
  }

  th.agent {
    display: flex;
    align-items: center;
    gap: 6px;
    padding-right: 10px;
    font-weight: 400;
    text-align: left;
    white-space: nowrap;
  }

  .cell {
    width: 30px;
    height: 26px;
    font: inherit;
    font-weight: 600;
    color: var(--bg-deep);
    border: 1px solid var(--frame-dim);
    border-radius: var(--radius);
    cursor: pointer;
  }

  /* 1 strongest, 4 faintest, off empty. */
  .p0 {
    background: var(--panel-inset);
  }
  .p1 {
    background: var(--olive-bright);
  }
  .p2 {
    background: color-mix(in srgb, var(--olive) 80%, var(--panel-inset));
  }
  .p3 {
    background: color-mix(in srgb, var(--olive) 55%, var(--panel-inset));
    color: var(--text);
  }
  .p4 {
    background: color-mix(in srgb, var(--olive) 30%, var(--panel-inset));
    color: var(--text-dim);
  }

  .types {
    list-style: none;
    margin: 0 0 8px;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .types li {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .types li.dragging {
    opacity: 0.4;
  }

  .handle {
    color: var(--text-faint);
    font-size: 11px;
    letter-spacing: -2px;
    cursor: grab;
  }

  .name {
    font: inherit;
    color: var(--text);
    padding: 2px 6px;
    width: 12em;
    background: var(--panel-inset);
    border: 1px solid var(--frame-dim);
    border-radius: var(--radius);
    user-select: text;
  }

  code {
    font-size: 12px;
    color: var(--text-faint);
  }

  .add {
    display: flex;
    gap: 6px;
    align-items: flex-end;
  }

  .link {
    font: inherit;
    font-size: 12px;
    padding: 0;
    color: var(--steel-bright);
    background: none;
    border: none;
    cursor: pointer;
  }

  .link.danger {
    color: var(--rust-bright);
  }

  p {
    margin: 0;
  }

  .dim {
    color: var(--text-dim);
  }

  .small {
    margin-top: 8px;
    font-size: 12px;
  }

  .error {
    margin-bottom: 8px;
    padding: 5px 8px;
    background: color-mix(in srgb, var(--rust) 35%, transparent);
    border: 1px solid var(--rust);
    border-radius: var(--radius);
  }
</style>
