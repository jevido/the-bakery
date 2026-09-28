<script lang="ts">
  import { Button, Panel, TextField } from '@bakery/ui'
  import { api, ApiError, type Sanction } from '../lib/api'
  import { sanctionLine, when } from '../lib/format'
  import SanctionDialog from './SanctionDialog.svelte'

  // A target's sanctions, newest first, with Sanction and Lift.
  let {
    target,
    targetName,
    sanctions,
    onchange,
  }: { target: string; targetName: string; sanctions: Sanction[]; onchange: () => void } = $props()

  let open = $state(false)
  let error = $state('')
  // lifting is the sanction whose lift reason is being typed.
  let lifting = $state<number | null>(null)
  let liftReason = $state('')
  const active = $derived(sanctions.find((s) => s.active))

  async function lift(s: Sanction) {
    error = ''
    try {
      await api('POST', `/api/console/sanctions/${s.id}/lift`, { reason: liftReason.trim() })
      lifting = null
      liftReason = ''
      onchange()
    } catch (err) {
      error = err instanceof ApiError ? err.message : String(err)
    }
  }
</script>

<Panel title="Sanctions">
  {#snippet actions()}
    {#if !active}<Button variant="danger" onclick={() => (open = true)}>Sanction</Button>{/if}
  {/snippet}
  {#if error}<p class="error" role="alert">{error}</p>{/if}
  {#if sanctions.length === 0}
    <p class="dim">None.</p>
  {:else}
    <table>
      <thead><tr><th>What</th><th>Reason</th><th>Since</th><th></th></tr></thead>
      <tbody>
        {#each sanctions as s (s.id)}
          <tr>
            <td><span class={['tag', s.active && s.kind]}>{sanctionLine(s)}</span></td>
            <td>{s.reason}</td>
            <td class="dim">{when(s.created_at)}</td>
            <td>
              {#if s.active && lifting === s.id}
                <form class="lift" onsubmit={(e) => (e.preventDefault(), lift(s))}>
                  <TextField aria-label="Why lift it" placeholder="Why (audit log)" bind:value={liftReason} />
                  <Button variant="confirm" type="submit">Lift</Button>
                  <Button type="button" onclick={() => (lifting = null)}>Cancel</Button>
                </form>
              {:else if s.active}
                <Button onclick={() => ((lifting = s.id), (liftReason = ''))}>Lift…</Button>
              {/if}
            </td>
          </tr>
        {/each}
      </tbody>
    </table>
  {/if}
</Panel>

{#if open}
  <SanctionDialog
    {target}
    {targetName}
    onsanctioned={() => ((open = false), onchange())}
    oncancel={() => (open = false)}
  />
{/if}

<style>
  .lift {
    display: flex;
    gap: 4px;
    align-items: center;
  }
</style>
