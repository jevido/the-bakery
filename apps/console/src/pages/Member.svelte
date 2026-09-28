<script lang="ts">
  import { Panel } from '@bakery/ui'
  import { api, ApiError, type AuditEntry, type GuildCard, type MemberCard, type Names, type Report, type Sanction } from '../lib/api'
  import { when } from '../lib/format'
  import AuditTable from '../components/AuditTable.svelte'
  import ReportsTable from '../components/ReportsTable.svelte'
  import Sanctions from '../components/Sanctions.svelte'

  let { id }: { id: number } = $props()

  type Record = {
    member: MemberCard
    guilds: GuildCard[]
    sanctions: Sanction[]
    reports: Report[]
    audit: AuditEntry[]
    names: Names
  }
  let record = $state.raw<Record | null>(null)
  let error = $state('')

  async function load() {
    try {
      record = await api<Record>('GET', `/api/console/members/${id}`)
    } catch (err) {
      error = err instanceof ApiError ? err.message : String(err)
    }
  }

  load()
</script>

<svelte:head><title>{record?.member.display_name ?? 'Member'} · Console</title></svelte:head>

{#if error}<p class="error" role="alert">{error}</p>{/if}
{#if record}
  {@const m = record.member}
  <div class="stack">
    <Panel title={m.display_name}>
      <dl>
        <dt>Email</dt><dd>{m.email}</dd>
        <dt>Member id</dt><dd>{m.id}</dd>
        <dt>Joined</dt><dd>{when(m.joined_at)}</dd>
      </dl>
    </Panel>
    <Sanctions target="member:{m.id}" targetName={m.display_name} sanctions={record.sanctions} onchange={load} />
    <Panel title="Guilds">
      {#if record.guilds.length === 0}<p class="dim">In no guild.</p>{:else}
        <ul>
          {#each record.guilds as g (g.id)}
            <li><a href="/guilds/{g.id}">{g.name}</a> <span class="dim">{g.member_count} members{g.archived ? ', archived' : ''}</span></li>
          {/each}
        </ul>
      {/if}
    </Panel>
    <Panel title="Reports about them">
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

  ul {
    margin: 0;
    padding-left: 18px;
  }
</style>
