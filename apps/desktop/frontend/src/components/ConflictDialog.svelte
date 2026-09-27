<script lang="ts">
  import { Panel, Button } from '@bakery/ui'
  import type { AgentSync } from '../lib/agentsync.svelte'

  // Shows each conflicted agent's differing files, this device's version
  // beside the server's, and the three ways out. Nothing is resolved without
  // the member choosing.
  let { sync, onclose }: { sync: AgentSync; onclose: () => void } = $props()

  let busy = $state(false)

  async function choose(slug: string, choice: 'local' | 'server' | 'both') {
    busy = true
    await sync.resolve(slug, choice)
    busy = false
    if (sync.conflicts.length === 0) onclose()
  }
</script>

<svelte:window onkeydown={(e) => e.key === 'Escape' && onclose()} />

<div class="backdrop" role="presentation" onclick={(e) => e.target === e.currentTarget && onclose()}>
  <div class="dialog" role="dialog" aria-modal="true" aria-label="Agents changed in two places">
    <Panel title="Changed in two places">
      {#snippet actions()}
        <button class="close" aria-label="Close" onclick={onclose}>×</button>
      {/snippet}
      <p class="dim">
        These agents were changed on this device and somewhere else before they could sync. Nothing has been
        overwritten; choose which version each keeps.
      </p>
      {#if sync.error}<p class="error" role="alert">{sync.error}</p>{/if}
      {#each sync.conflicts as c (c.slug)}
        <section class="conflict">
          <h3>{c.slug}</h3>
          {#each c.files as f (f.path)}
            <div class="file">
              <div class="path">{f.path}</div>
              <div class="sides">
                <div>
                  <span class="side">This device</span>
                  <pre>{f.missing_local ? '(not here)' : f.local}</pre>
                </div>
                <div>
                  <span class="side">Server</span>
                  <pre>{f.missing_server ? '(not there)' : f.server}</pre>
                </div>
              </div>
            </div>
          {/each}
          <div class="choices">
            <Button variant="confirm" disabled={busy} onclick={() => choose(c.slug, 'local')}>Keep this device's</Button>
            <Button disabled={busy} onclick={() => choose(c.slug, 'server')}>Take the server's</Button>
            <Button disabled={busy} onclick={() => choose(c.slug, 'both')}>Keep both</Button>
          </div>
          <p class="dim small">"Keep both" keeps the server's version under {c.slug} and adds this device's as {c.slug}-local.</p>
        </section>
      {/each}
    </Panel>
  </div>
</div>

<style>
  .backdrop {
    position: fixed;
    inset: 0;
    z-index: 50;
    display: grid;
    place-items: center;
    background: #000000a0;
  }

  .dialog {
    width: min(960px, calc(100vw - 32px));
    max-height: calc(100vh - 48px);
    display: flex;
  }

  .dialog :global(.panel) {
    flex: 1;
    min-height: 0;
  }

  .close {
    padding: 0 4px;
    font-size: 16px;
    line-height: 1;
    color: var(--text-dim);
    background: none;
    border: none;
    cursor: pointer;
  }

  .conflict {
    display: flex;
    flex-direction: column;
    gap: 8px;
    padding-top: 12px;
    margin-top: 12px;
    border-top: 1px solid var(--frame-dim);
  }

  h3 {
    margin: 0;
    font-family: var(--font-display);
    font-size: 15px;
  }

  .path {
    font-size: 12px;
    color: var(--text-dim);
    margin-bottom: 4px;
  }

  .sides {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 8px;
  }

  .side {
    font-size: 11px;
    text-transform: uppercase;
    letter-spacing: 0.04em;
    color: var(--text-faint);
  }

  pre {
    margin: 2px 0 0;
    max-height: 240px;
    overflow: auto;
    padding: 6px 8px;
    font-size: 12px;
    white-space: pre-wrap;
    overflow-wrap: anywhere;
    background: var(--panel-inset);
    border: 1px solid var(--frame-dim);
    border-radius: var(--radius);
    user-select: text;
  }

  .choices {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
  }

  p {
    margin: 0;
  }

  .dim {
    color: var(--text-dim);
  }

  .small {
    font-size: 12px;
  }

  .error {
    margin-top: 8px;
    padding: 5px 8px;
    background: color-mix(in srgb, var(--rust) 35%, transparent);
    border: 1px solid var(--rust);
    border-radius: var(--radius);
  }
</style>
