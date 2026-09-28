<script lang="ts">
  import { Button } from '@bakery/ui'
  import { reports, type ReportTarget } from '../lib/reports'
  import { ApiError } from '../lib/api'

  // Report a member or a guild to the operators, with a reason. They read
  // every report; the one reported is not told who sent it.
  let {
    targetKind,
    targetId,
    name,
    onclose,
  }: { targetKind: ReportTarget; targetId: number; name: string; onclose: () => void } = $props()

  let reason = $state('')
  let busy = $state(false)
  let error = $state('')
  let sent = $state(false)

  async function send(event: SubmitEvent) {
    event.preventDefault()
    busy = true
    try {
      await reports.file(targetKind, targetId, reason.trim())
      sent = true
      error = ''
    } catch (err) {
      error = err instanceof ApiError ? err.message : String(err)
    } finally {
      busy = false
    }
  }
</script>

<div class="report" role="dialog" aria-label={`Report ${name}`}>
  {#if sent}
    <p>Thanks. The operators will look at your report about {name}.</p>
    <Button onclick={onclose}>Close</Button>
  {:else}
    <form onsubmit={send}>
      <label>
        <span>What is wrong with {name}? The operators read this; {name} does not see who sent it.</span>
        <textarea maxlength={1000} required bind:value={reason}></textarea>
      </label>
      {#if error}<p class="error" role="alert">{error}</p>{/if}
      <div class="row">
        <Button type="submit" variant="danger" disabled={busy || !reason.trim()}>Send report</Button>
        <Button type="button" onclick={onclose} disabled={busy}>Cancel</Button>
      </div>
    </form>
  {/if}
</div>

<style>
  .report {
    display: flex;
    flex-direction: column;
    gap: 8px;
    padding: 10px;
    background: var(--panel-inset);
    border: 1px solid var(--frame-dim);
    border-radius: var(--radius);
  }

  form,
  label {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  label span {
    color: var(--text-dim);
  }

  textarea {
    font: inherit;
    color: var(--text);
    min-height: 70px;
    padding: 6px 8px;
    background: var(--panel-raised);
    border: 1px solid var(--frame-dim);
    border-radius: var(--radius);
  }

  .row {
    display: flex;
    gap: 8px;
  }

  p {
    margin: 0;
  }

  .error {
    color: var(--rust-bright);
  }
</style>
