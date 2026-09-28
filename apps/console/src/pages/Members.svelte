<script lang="ts">
  import { Panel, TextField } from '@bakery/ui'
  import { api, ApiError, type MemberCard, type Sanction } from '../lib/api'
  import { sanctionLine, when } from '../lib/format'

  type Row = MemberCard & { guild_count: number; sanction: Sanction | null }

  let q = $state(new URLSearchParams(window.location.search).get('q') ?? '')
  let rows = $state.raw<Row[]>([])
  let error = $state('')
  let timer: ReturnType<typeof setTimeout> | undefined

  async function search() {
    error = ''
    try {
      rows = (await api<{ members: Row[] }>('GET', `/api/console/members?q=${encodeURIComponent(q)}`)).members
      window.history.replaceState({}, '', q ? `?q=${encodeURIComponent(q)}` : window.location.pathname)
    } catch (err) {
      error = err instanceof ApiError ? err.message : String(err)
    }
  }

  function typed() {
    clearTimeout(timer)
    timer = setTimeout(search, 250)
  }

  search()
</script>

<svelte:head><title>Members · Console</title></svelte:head>

<Panel title="Members">
  <div class="search">
    <TextField aria-label="Search members" placeholder="Email or name" bind:value={q} oninput={typed} />
  </div>
  {#if error}<p class="error" role="alert">{error}</p>{/if}
  <table>
    <thead><tr><th>Name</th><th>Email</th><th>Guilds</th><th>Joined</th><th>Standing</th></tr></thead>
    <tbody>
      {#each rows as m (m.id)}
        <tr>
          <td><a href="/members/{m.id}">{m.display_name}</a></td>
          <td>{m.email}</td>
          <td>{m.guild_count}</td>
          <td class="dim">{when(m.joined_at)}</td>
          <td>{#if m.sanction}<span class={['tag', m.sanction.kind]}>{sanctionLine(m.sanction)}</span>{/if}</td>
        </tr>
      {:else}
        <tr><td colspan="5" class="dim">No member matches.</td></tr>
      {/each}
    </tbody>
  </table>
</Panel>

<style>
  .search {
    max-width: 360px;
    margin-bottom: 10px;
  }
</style>
