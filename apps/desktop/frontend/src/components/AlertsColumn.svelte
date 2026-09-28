<script lang="ts">
  import type { Alert } from '../lib/bindings'
  import type { Alerts } from '../lib/alerts.svelte'

  // RimWorld's alerts, at the right edge: one small framed tab per
  // standing condition, most urgent first. Hover for the detail, click to
  // go to its cause. Collapsed, only the icons show.
  let { alerts, ongo }: { alerts: Alerts; ongo: (a: Alert) => void } = $props()

  const ICONS: Record<string, string> = {
    question_waiting: '?',
    run_failed: '✕',
    awaiting_review: '◷',
    uncovered_work: '∅',
    idle_agent: 'z',
  }
</script>

{#if alerts.list.length}
  <aside class={['alerts', { collapsed: alerts.collapsed }]} aria-label="Alerts">
    <button class="fold" title={alerts.collapsed ? 'Show alerts' : 'Fold alerts'} onclick={() => alerts.toggle()}>
      {alerts.collapsed ? '‹' : '›'}
    </button>
    <ul>
      {#each alerts.list as a (a.id)}
        <li>
          <button class={['alert', a.severity, a.kind]} title={a.detail} onclick={() => ongo(a)}>
            <span class="icon" aria-hidden="true">{ICONS[a.kind] ?? '!'}</span>
            {#if !alerts.collapsed}<span class="title">{a.title}</span>{/if}
          </button>
        </li>
      {/each}
    </ul>
  </aside>
{/if}

<style>
  .alerts {
    width: 190px;
    display: flex;
    flex-direction: column;
    gap: 4px;
    min-height: 0;
    overflow-y: auto;
  }

  .alerts.collapsed {
    width: 34px;
  }

  .fold {
    align-self: flex-end;
    font: inherit;
    padding: 0 6px;
    color: var(--text-dim);
    background: none;
    border: 1px solid var(--frame-dim);
    border-radius: var(--radius);
    cursor: pointer;
  }

  ul {
    list-style: none;
    margin: 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: 3px;
  }

  .alert {
    width: 100%;
    display: flex;
    gap: 6px;
    align-items: center;
    padding: 4px 6px;
    font: inherit;
    font-size: 12px;
    text-align: left;
    color: var(--text);
    background: var(--panel);
    border: 1px solid var(--frame);
    border-left: 3px solid var(--amber);
    border-radius: var(--radius);
    box-shadow: var(--shadow-outer);
    cursor: pointer;
  }

  .alert:hover {
    background: var(--panel-raised);
  }

  .alert.high {
    border-left-color: var(--rust-bright);
    background: color-mix(in srgb, var(--rust) 22%, var(--panel));
  }

  .alert.run_failed {
    border-left-color: var(--rust);
  }

  .icon {
    width: 14px;
    flex-shrink: 0;
    text-align: center;
    font-family: var(--font-display);
    color: var(--text-dim);
  }

  .title {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
</style>
