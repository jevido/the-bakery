<script lang="ts">
  import { Panel, Tabs } from '@bakery/ui'
  import { api, ApiError, type Names, type Report } from '../lib/api'
  import ReportsTable from '../components/ReportsTable.svelte'

  let status = $state('open')
  let reports = $state.raw<Report[]>([])
  let names = $state.raw<Names>({ members: {}, guilds: {} })
  let error = $state('')

  async function load() {
    error = ''
    try {
      const res = await api<{ reports: Report[]; names: Names }>('GET', `/api/console/reports?status=${status}`)
      // The queue shows the oldest open report first; handled ones newest first.
      reports = status === 'open' ? res.reports : [...res.reports].reverse()
      names = res.names
    } catch (err) {
      error = err instanceof ApiError ? err.message : String(err)
    }
  }

  load()
</script>

<svelte:head><title>Reports · Console</title></svelte:head>

<Panel title="Reports">
  <Tabs
    tabs={[
      { id: 'open', label: 'Open' },
      { id: 'actioned', label: 'Actioned' },
      { id: 'dismissed', label: 'Dismissed' },
    ]}
    label="Report status"
    bind:active={() => status, (v) => ((status = v), load())}
  >
    {#snippet children()}
      {#if error}<p class="error" role="alert">{error}</p>{/if}
      <ReportsTable {reports} {names} onchange={load} />
    {/snippet}
  </Tabs>
</Panel>
