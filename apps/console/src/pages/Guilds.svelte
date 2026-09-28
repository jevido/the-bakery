<script lang="ts">
  import { Panel, TextField } from '@bakery/ui'
  import { api, ApiError, type GuildCard, type Sanction } from '../lib/api'
  import { sanctionLine, when } from '../lib/format'

  type Row = GuildCard & { sanction: Sanction | null }

  let q = $state(new URLSearchParams(window.location.search).get('q') ?? '')
  let rows = $state.raw<Row[]>([])
  let error = $state('')
  let timer: ReturnType<typeof setTimeout> | undefined

  async function search() {
    error = ''
    try {
      rows = (await api<{ guilds: Row[] }>('GET', `/api/console/guilds?q=${encodeURIComponent(q)}`)).guilds
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

<svelte:head><title>Guilds · Console</title></svelte:head>

<Panel title="Guilds">
  <div class="search">
    <TextField aria-label="Search guilds" placeholder="Guild name" bind:value={q} oninput={typed} />
  </div>
  {#if error}<p class="error" role="alert">{error}</p>{/if}
  <table>
    <thead><tr><th>Name</th><th>Members</th><th>Founded</th><th>Standing</th></tr></thead>
    <tbody>
      {#each rows as m (m.id)}
        <tr>
          <td><a href="/guilds/{m.id}">{m.name}</a>{#if m.archived}<span class="dim"> (archived)</span>{/if}</td>
          <td>{m.member_count}</td>
          <td class="dim">{when(m.founded_at)}</td>
          <td>{#if m.sanction}<span class={['tag', m.sanction.kind]}>{sanctionLine(m.sanction)}</span>{/if}</td>
        </tr>
      {:else}
        <tr><td colspan="4" class="dim">No guild matches.</td></tr>
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
