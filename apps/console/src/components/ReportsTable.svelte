<script lang="ts">
  import { Button } from '@bakery/ui'
  import { api, ApiError, type Names, type Report, type Sanction } from '../lib/api'
  import { targetPath, when } from '../lib/format'
  import SanctionDialog from './SanctionDialog.svelte'

  // Reports with their handling: open ones can be dismissed, or acted on
  // with a suspension or a ban of their target (a sanction, then the report
  // is marked actioned with it). A ban is two clicks: Ban, then confirm.
  let { reports, names, onchange }: { reports: Report[]; names: Names; onchange: () => void } = $props()

  let acting = $state<{ report: Report; kind: 'suspension' | 'ban' } | null>(null)
  let error = $state('')

  function nameOf(kind: string, id: number): string {
    return (kind === 'member' ? names.members[id] : names.guilds[id]) ?? `${kind} ${id}`
  }

  async function dismiss(r: Report) {
    error = ''
    try {
      await api('POST', `/api/console/reports/${r.id}/dismiss`)
      onchange()
    } catch (err) {
      error = err instanceof ApiError ? err.message : String(err)
    }
  }

  async function sanctioned(r: Report, s: Sanction) {
    acting = null
    try {
      await api('POST', `/api/console/reports/${r.id}/action`, { sanction_id: s.id })
    } catch (err) {
      error = err instanceof ApiError ? err.message : String(err)
    }
    onchange()
  }
</script>

{#if error}<p class="error" role="alert">{error}</p>{/if}
{#if reports.length === 0}
  <p class="dim">No reports.</p>
{:else}
  <table>
    <thead><tr><th>Filed</th><th>About</th><th>By</th><th>Reason</th><th>Status</th><th></th></tr></thead>
    <tbody>
      {#each reports as r (r.id)}
        <tr>
          <td class="dim">{when(r.created_at)}</td>
          <td>{r.target_kind} <a href={targetPath(r.target_kind, r.target_id)}>{nameOf(r.target_kind, r.target_id)}</a></td>
          <td><a href={targetPath('member', r.by_member_id)}>{nameOf('member', r.by_member_id)}</a></td>
          <td class="reason">{r.reason}</td>
          <td>{r.status}{#if r.handled_at}<br /><span class="dim">{when(r.handled_at)}</span>{/if}</td>
          <td>
            {#if r.status === 'open'}
              <div class="actions">
              <Button variant="danger" onclick={() => (acting = { report: r, kind: 'ban' })}>Ban</Button>
              <Button onclick={() => (acting = { report: r, kind: 'suspension' })}>Suspend</Button>
              <Button onclick={() => dismiss(r)}>Dismiss</Button>
              </div>
            {/if}
          </td>
        </tr>
      {/each}
    </tbody>
  </table>
{/if}

{#if acting}
  {@const r = acting.report}
  {#key r.id}
    <SanctionDialog
      target={`${r.target_kind}:${r.target_id}`}
      targetName={nameOf(r.target_kind, r.target_id)}
      kind={acting.kind}
      reason={r.reason}
      onsanctioned={(s) => sanctioned(r, s)}
      oncancel={() => (acting = null)}
    />
  {/key}
{/if}

<style>
  .reason {
    max-width: 360px;
    white-space: pre-wrap;
  }

  .actions {
    display: flex;
    gap: 4px;
    flex-wrap: wrap;
  }
</style>
