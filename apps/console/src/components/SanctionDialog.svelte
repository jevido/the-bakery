<script lang="ts">
  import { untrack } from 'svelte'
  import { Button, TextField } from '@bakery/ui'
  import { api, ApiError, type Sanction } from '../lib/api'
  import { untilAfter } from '../lib/format'

  // Suspends or bans a member or a guild. The reason is required: the
  // target reads it in the refusal, and it goes in the audit log.
  let {
    target,
    targetName,
    kind: initialKind = 'suspension',
    reason: initialReason = '',
    onsanctioned,
    oncancel,
  }: {
    target: string
    targetName: string
    kind?: 'suspension' | 'ban'
    reason?: string
    onsanctioned: (s: Sanction) => void
    oncancel: () => void
  } = $props()

  let kind = $state(untrack(() => initialKind))
  let reason = $state(untrack(() => initialReason))
  let days = $state(1)
  let busy = $state(false)
  let error = $state('')

  const valid = $derived(reason.trim().length > 0 && (kind === 'ban' || days > 0))

  async function submit(event: SubmitEvent) {
    event.preventDefault()
    busy = true
    error = ''
    try {
      const { sanction } = await api<{ sanction: Sanction }>('POST', '/api/console/sanctions', {
        target,
        kind,
        reason: reason.trim(),
        ...(kind === 'suspension' ? { until: untilAfter(days) } : {}),
      })
      onsanctioned(sanction)
    } catch (err) {
      error = err instanceof ApiError ? err.message : String(err)
    }
    busy = false
  }
</script>

<div class="backdrop" role="presentation" onclick={(e) => e.target === e.currentTarget && oncancel()}>
  <div class="dialog" role="dialog" aria-modal="true" aria-label="Sanction {targetName}">
  <form onsubmit={submit}>
    <h2>Sanction {targetName}</h2>
    {#if error}<p class="error" role="alert">{error}</p>{/if}
    <fieldset>
      <label><input type="radio" name="kind" value="suspension" bind:group={kind} /> Suspend</label>
      <label><input type="radio" name="kind" value="ban" bind:group={kind} /> Ban</label>
    </fieldset>
    {#if kind === 'suspension'}
      <label class="days">
        For
        <input type="number" min="1" max="365" bind:value={days} aria-label="Days" />
        day{days === 1 ? '' : 's'}
      </label>
    {/if}
    <TextField label="Reason (the target reads this)" maxlength={500} required bind:value={reason} />
    <div class="row">
      <Button variant="danger" type="submit" disabled={busy || !valid}>{kind === 'ban' ? 'Ban' : 'Suspend'}</Button>
      <Button type="button" onclick={oncancel}>Cancel</Button>
    </div>
  </form>
  </div>
</div>

<svelte:window onkeydown={(e) => e.key === 'Escape' && oncancel()} />

<style>
  .backdrop {
    position: fixed;
    inset: 0;
    display: grid;
    place-items: center;
    background: #0009;
    z-index: 10;
  }

  .dialog {
    width: min(420px, calc(100vw - 32px));
  }

  form {
    display: flex;
    flex-direction: column;
    gap: 10px;
    padding: 16px;
    background: var(--panel);
    border: 1px solid var(--frame);
    border-radius: var(--radius);
    box-shadow: var(--shadow-outer);
  }

  h2 {
    margin: 0;
    font-family: var(--font-display);
    font-size: 16px;
  }

  fieldset {
    display: flex;
    gap: 16px;
    margin: 0;
    padding: 0;
    border: none;
  }

  .days input {
    width: 60px;
    font: inherit;
    color: var(--text);
    background: var(--panel-raised);
    border: 1px solid var(--frame-dim);
    border-radius: var(--radius);
    padding: 2px 4px;
  }

  .row {
    display: flex;
    gap: 8px;
  }
</style>
