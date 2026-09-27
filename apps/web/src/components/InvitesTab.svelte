<script lang="ts">
  import { Panel, Button } from '@bakery/ui'
  import { invites, inviteLink, type Invite } from '../lib/invites'
  import { ApiError } from '../lib/api'

  let { guildId, archived }: { guildId: number; archived: boolean } = $props()

  let list = $state<Invite[]>([])
  let expires = $state<'never' | '24' | '168'>('168')
  // A number input binds as a number (or null/undefined when empty).
  let maxUses = $state<number | null>(null)
  let error = $state('')
  let busy = $state(false)
  let copied = $state<number | null>(null)

  async function load() {
    try {
      list = await invites.list(guildId)
    } catch (err) {
      error = err instanceof ApiError ? err.message : String(err)
    }
  }
  load()

  async function create(event: SubmitEvent) {
    event.preventDefault()
    busy = true
    error = ''
    try {
      const uses = typeof maxUses === 'number' && !Number.isNaN(maxUses) ? maxUses : null
      await invites.create(guildId, expires === 'never' ? null : Number(expires), uses)
      maxUses = null
      await load()
    } catch (err) {
      error = err instanceof ApiError ? err.message : String(err)
    } finally {
      busy = false
    }
  }

  async function revoke(inv: Invite) {
    try {
      await invites.revoke(guildId, inv.id)
      await load()
    } catch (err) {
      error = err instanceof ApiError ? err.message : String(err)
    }
  }

  async function copy(inv: Invite) {
    try {
      await navigator.clipboard.writeText(inviteLink(inv.code))
      copied = inv.id
      setTimeout(() => (copied = null), 1500)
    } catch {
      error = 'Copying failed; select the link and copy it yourself.'
    }
  }

  const STATES: Record<Invite['state'], string> = {
    active: 'active',
    expired: 'expired',
    used_up: 'used up',
    revoked: 'withdrawn',
  }

  function until(iso: string | null): string {
    return iso ? new Date(iso).toLocaleString(undefined, { dateStyle: 'medium', timeStyle: 'short' }) : 'no end'
  }
</script>

{#if error}<p class="error" role="alert">{error}</p>{/if}

{#if !archived}
  <Panel title="New invite link">
    <form onsubmit={create}>
      <label>
        <span>Lasts</span>
        <select bind:value={expires}>
          <option value="24">1 day</option>
          <option value="168">7 days</option>
          <option value="never">No end</option>
        </select>
      </label>
      <label>
        <span>Uses</span>
        <input type="number" min="1" placeholder="No limit" bind:value={maxUses} />
      </label>
      <Button type="submit" variant="confirm" disabled={busy}>Make link</Button>
    </form>
  </Panel>
{/if}

<Panel title={`Invite links · ${list.length}`}>
  {#if list.length === 0}
    <p class="dim">No invite links yet. Make one and send it to whoever should join.</p>
  {:else}
    <ul class="invites">
      {#each list as inv (inv.id)}
        <li class={{ inactive: inv.state !== 'active' }}>
          <code class="link">{inviteLink(inv.code)}</code>
          <span class="meta">
            {STATES[inv.state]} · {inv.uses}{inv.max_uses ? ` of ${inv.max_uses}` : ''} used · until {until(inv.expires_at)}
          </span>
          {#if inv.state === 'active'}
            <span class="row">
              <Button onclick={() => copy(inv)}>{copied === inv.id ? 'Copied' : 'Copy'}</Button>
              <Button variant="danger" onclick={() => revoke(inv)}>Withdraw</Button>
            </span>
          {/if}
        </li>
      {/each}
    </ul>
  {/if}
</Panel>

<style>
  form {
    display: flex;
    flex-wrap: wrap;
    align-items: flex-end;
    gap: 10px;
  }

  label {
    display: flex;
    flex-direction: column;
    gap: 3px;
  }

  label span {
    font-size: 12px;
    color: var(--text-dim);
  }

  select,
  input {
    font: inherit;
    color: var(--text);
    padding: 5px 8px;
    width: 9em;
    background: var(--panel-inset);
    border: 1px solid var(--frame-dim);
    border-radius: var(--radius);
  }

  .invites {
    list-style: none;
    margin: 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  li {
    display: flex;
    flex-direction: column;
    gap: 4px;
    padding-bottom: 10px;
    border-bottom: 1px solid var(--frame-dim);
  }

  li:last-child {
    border-bottom: none;
    padding-bottom: 0;
  }

  .inactive {
    opacity: 0.6;
  }

  .link {
    overflow-wrap: anywhere;
    font-size: 13px;
  }

  .meta {
    font-size: 12px;
    color: var(--text-dim);
  }

  .row {
    display: flex;
    gap: 6px;
  }

  p {
    margin: 0;
  }

  .dim {
    color: var(--text-dim);
  }

  .error {
    margin-bottom: 8px;
    padding: 5px 8px;
    background: color-mix(in srgb, var(--rust) 35%, transparent);
    border: 1px solid var(--rust);
    border-radius: var(--radius);
  }

  :global(.panel) + :global(.panel) {
    margin-top: var(--gap);
  }
</style>
