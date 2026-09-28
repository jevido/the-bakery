<script lang="ts">
  import { Panel, Button } from '@bakery/ui'
  import { BoardsService, messageOf } from '../lib/bindings'

  // Report a guild to the operators, with a reason. The guild's members do
  // not see who sent it.
  let { guild, onclose }: { guild: { id: number; name: string }; onclose: () => void } = $props()

  let reason = $state('')
  let busy = $state(false)
  let error = $state('')
  let sent = $state(false)

  async function send(event: SubmitEvent) {
    event.preventDefault()
    busy = true
    try {
      await BoardsService.ReportGuild(guild.id, reason.trim())
      sent = true
      error = ''
    } catch (err) {
      error = messageOf(err)
    }
    busy = false
  }
</script>

<svelte:window onkeydown={(e) => e.key === 'Escape' && onclose()} />

<!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
<div class="veil" onclick={onclose}></div>
<div class="dialog" role="dialog" aria-label={`Report ${guild.name}`}>
  <Panel title={`Report ${guild.name}`}>
    {#if sent}
      <p>Thanks. The operators will look at your report.</p>
      <Button onclick={onclose}>Close</Button>
    {:else}
      <form onsubmit={send}>
        <label>
          <span>What is wrong with this guild? The operators read this; its members do not see who sent it.</span>
          <textarea maxlength={1000} bind:value={reason}></textarea>
        </label>
        {#if error}<p class="error" role="alert">{error}</p>{/if}
        <div class="row">
          <Button type="submit" variant="danger" disabled={busy || !reason.trim()}>Send report</Button>
          <Button type="button" onclick={onclose} disabled={busy}>Cancel</Button>
        </div>
      </form>
    {/if}
  </Panel>
</div>

<style>
  .veil {
    position: fixed;
    inset: 0;
    z-index: 40;
    background: rgb(0 0 0 / 0.35);
  }

  .dialog {
    position: fixed;
    z-index: 41;
    top: 50%;
    left: 50%;
    width: min(460px, calc(100vw - 40px));
    transform: translate(-50%, -50%);
  }

  form,
  label {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  label span {
    font-size: 13px;
    color: var(--text-dim);
  }

  textarea {
    font: inherit;
    color: var(--text);
    min-height: 80px;
    padding: 6px 8px;
    background: var(--panel-inset);
    border: 1px solid var(--frame-dim);
    border-radius: var(--radius);
    user-select: text;
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
