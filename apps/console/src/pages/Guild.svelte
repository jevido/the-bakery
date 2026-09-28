<script lang="ts">
  import { Panel } from '@bakery/ui'
  import { api, ApiError, type AuditEntry, type GuildCard, type MemberCard, type Names, type Report, type Sanction } from '../lib/api'
  import { when } from '../lib/format'
  import AuditTable from '../components/AuditTable.svelte'
  import ReportsTable from '../components/ReportsTable.svelte'
  import Sanctions from '../components/Sanctions.svelte'

  let { id }: { id: number } = $props()

  type Record = {
    guild: GuildCard
    members: MemberCard[]
    board_count: number
    sanctions: Sanction[]
    reports: Report[]
    audit: AuditEntry[]
    names: Names
  }
  let record = $state.raw<Record | null>(null)
  let error = $state('')

  async function load() {
    try {
      record = await api<Record>('GET', `/api/console/guilds/${id}`)
    } catch (err) {
      error = err instanceof ApiError ? err.message : String(err)
    }
  }

  load()
</script>

<svelte:head><title>{record?.guild.name ?? 'Guild'} · Console</title></svelte:head>

{#if error}<p class="error" role="alert">{error}</p>{/if}
{#if record}
  {@const g = record.guild}
  <div class="stack">
    <Panel title={g.name}>
      <dl>
        <dt>Guild id</dt><dd>{g.id}</dd>
        <dt>Founded</dt><dd>{when(g.founded_at)}</dd>
        <dt>Boards</dt><dd>{record.board_count}</dd>
        <dt>State</dt><dd>{g.archived ? 'Archived' : 'Active'}</dd>
      </dl>
    </Panel>
    <Sanctions target="guild:{g.id}" targetName={g.name} sanctions={record.sanctions} onchange={load} />
    <Panel title="Members ({record.members.length})">
      <table>
        <tbody>
          {#each record.members as m (m.id)}
            <tr><td><a href="/members/{m.id}">{m.display_name}</a></td><td>{m.email}</td></tr>
          {/each}
        </tbody>
      </table>
    </Panel>
    <Panel title="Reports about it">
      <ReportsTable reports={record.reports} names={record.names} onchange={load} />
    </Panel>
    <Panel title="Audit log">
      <AuditTable entries={record.audit} names={record.names} />
    </Panel>
  </div>
{:else if !error}
  <p class="dim">Loading…</p>
{/if}

<style>
  .stack {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  dl {
    display: grid;
    grid-template-columns: max-content 1fr;
    gap: 4px 16px;
    margin: 0;
  }

  dt {
    color: var(--text-dim);
  }

  dd {
    margin: 0;
  }
</style>
