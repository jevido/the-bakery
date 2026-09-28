<script lang="ts">
  import { Button, Panel, TextField } from '@bakery/ui'
  import { api, ApiError, type AuditEntry, type Names } from '../lib/api'
  import AuditTable from '../components/AuditTable.svelte'

  // The whole audit log, newest first, filtered and paged by entry id.
  const params = new URLSearchParams(window.location.search)
  let action = $state(params.get('action') ?? '')
  let actorKind = $state(params.get('actor_kind') ?? '')
  let actorId = $state(params.get('actor_id') ?? '')
  let targetKind = $state(params.get('target_kind') ?? '')
  let targetId = $state(params.get('target_id') ?? '')

  let entries = $state.raw<AuditEntry[]>([])
  let names = $state<Names>({ members: {}, guilds: {} })
  let more = $state(false)
  let error = $state('')
  const pageSize = 50

  function query(before?: number): string {
    const q = new URLSearchParams()
    if (action.trim()) q.set('action', action.trim())
    if (actorKind) q.set('actor_kind', actorKind)
    if (actorId.trim()) q.set('actor_id', actorId.trim())
    if (targetKind) q.set('target_kind', targetKind)
    if (targetId.trim()) q.set('target_id', targetId.trim())
    q.set('limit', String(pageSize))
    if (before) q.set('before', String(before))
    return q.toString()
  }

  async function load(event?: SubmitEvent) {
    event?.preventDefault()
    error = ''
    try {
      const res = await api<{ entries: AuditEntry[]; names: Names }>('GET', `/api/console/audit?${query()}`)
      entries = res.entries
      names = res.names
      more = res.entries.length === pageSize
      const shown = new URLSearchParams(query())
      shown.delete('limit')
      window.history.replaceState({}, '', shown.size ? `?${shown}` : window.location.pathname)
    } catch (err) {
      error = err instanceof ApiError ? err.message : String(err)
    }
  }

  async function older() {
    const last = entries.at(-1)
    if (!last) return
    try {
      const res = await api<{ entries: AuditEntry[]; names: Names }>('GET', `/api/console/audit?${query(last.id)}`)
      entries = [...entries, ...res.entries]
      names = {
        members: { ...names.members, ...res.names.members },
        guilds: { ...names.guilds, ...res.names.guilds },
      }
      more = res.entries.length === pageSize
    } catch (err) {
      error = err instanceof ApiError ? err.message : String(err)
    }
  }

  load()
</script>

<svelte:head><title>Audit log · Console</title></svelte:head>

<Panel title="Audit log">
  <form class="filters" onsubmit={load}>
    <TextField label="Action" placeholder="sanction.ban" bind:value={action} />
    <label class="select">
      <span>Actor</span>
      <select bind:value={actorKind}>
        <option value="">Anyone</option>
        <option value="member">Member</option>
        <option value="operator">Operator</option>
      </select>
    </label>
    <TextField label="Actor id" inputmode="numeric" bind:value={actorId} />
    <label class="select">
      <span>Target</span>
      <select bind:value={targetKind}>
        <option value="">Anything</option>
        <option value="member">Member</option>
        <option value="guild">Guild</option>
        <option value="operator">Operator</option>
        <option value="personal_token">Personal token</option>
      </select>
    </label>
    <TextField label="Target id" inputmode="numeric" bind:value={targetId} />
    <Button type="submit">Filter</Button>
  </form>
  {#if error}<p class="error" role="alert">{error}</p>{/if}
  <AuditTable {entries} {names} />
  {#if more}<div class="more"><Button onclick={older}>Older</Button></div>{/if}
</Panel>

<style>
  .filters {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
    align-items: flex-end;
    margin-bottom: 12px;
  }

  .filters > :global(.field) {
    width: 140px;
  }

  .select {
    display: flex;
    flex-direction: column;
    gap: 3px;
    font-size: 12px;
    color: var(--text-dim);
  }

  select {
    font: inherit;
    font-size: 13px;
    color: var(--text);
    padding: 4px 6px;
    background: var(--panel-raised);
    border: 1px solid var(--frame-dim);
    border-radius: var(--radius);
  }

  .more {
    margin-top: 10px;
  }
</style>
