<script lang="ts">
  import { Button, Portrait } from '@bakery/ui'
  import { AgentsService, messageOf, type AgentSummary } from '../lib/bindings'

  // Choosing who works a task, the way RimWorld lists colonists for a job:
  // the ones who want this kind of work most come first. Agents who never do
  // it come last, greyed, and can still be put on it.
  let {
    workType,
    workTypeName,
    busy,
    error,
    onpick,
    oncancel,
  }: {
    workType: string | null
    workTypeName: string
    busy: boolean
    error: string
    onpick: (slug: string) => void
    oncancel: () => void
  } = $props()

  let agents = $state.raw<AgentSummary[]>([])
  let loadError = $state('')
  let loaded = $state(false)

  AgentsService.List()
    .then((a) => (agents = a ?? []))
    .catch((err) => (loadError = messageOf(err)))
    .finally(() => (loaded = true))

  function priority(a: AgentSummary): number {
    return workType ? (a.work_priorities?.[workType] ?? 0) : 0
  }

  // 1 first, 4 last, off after them; by name within a priority.
  const sorted = $derived(
    [...agents].sort((a, b) => {
      const pa = priority(a) || 9
      const pb = priority(b) || 9
      return pa - pb || a.name.localeCompare(b.name)
    }),
  )

  const ORDINAL = ['', 'first', 'second', 'third', 'last']
</script>

<div class="picker" role="dialog" aria-label="Assign an agent">
  <header>
    <strong>Assign an agent</strong>
    <button class="close" aria-label="Cancel" onclick={oncancel}>×</button>
  </header>
  {#if error}<p class="error" role="alert">{error}</p>{/if}
  {#if loadError}<p class="error">{loadError}</p>{/if}
  {#if loaded && agents.length === 0}
    <p class="dim">No agents yet. Make one on the Agents screen.</p>
  {/if}
  <ul>
    {#each sorted as a (a.slug)}
      {@const p = priority(a)}
      <li class={{ off: workType !== null && p === 0 }}>
        <Portrait seed={a.portrait_seed || a.slug} size={28} />
        <div class="who">
          <span class="name">{a.name || a.slug}</span>
          <span class="dim">
            {a.title || 'Agent'} · {a.model}
            {#if workType}
              · {p ? `does ${workTypeName} ${ORDINAL[p]}` : `Not allowed to do ${workTypeName}`}
            {/if}
          </span>
        </div>
        <Button variant={p === 1 ? 'confirm' : undefined} disabled={busy || !a.synced} onclick={() => onpick(a.slug)}>
          {a.synced ? 'Assign' : 'Not synced'}
        </Button>
      </li>
    {/each}
  </ul>
</div>

<style>
  .picker {
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
    align-items: center;
  }

  ul {
    list-style: none;
    margin: 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: 4px;
    max-height: 280px;
    overflow: auto;
  }

  li {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  li.off {
    opacity: 0.55;
  }

  .who {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
  }

  .who .dim {
    font-size: 12px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .close {
    font: inherit;
    font-size: 16px;
    padding: 0 4px;
    color: var(--text-dim);
    background: none;
    border: none;
    cursor: pointer;
  }

  p {
    margin: 0;
  }

  .dim {
    color: var(--text-dim);
  }

  .error {
    padding: 5px 8px;
    background: color-mix(in srgb, var(--rust) 35%, transparent);
    border: 1px solid var(--rust);
    border-radius: var(--radius);
  }
</style>
