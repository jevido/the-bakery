<script lang="ts">
  import type { Snippet } from 'svelte'

  // header replaces the title with markup of the caller's own (an editable
  // name, a drag handle); actions stay on the right either way.
  let {
    title,
    header,
    actions,
    children,
  }: { title?: string; header?: Snippet; actions?: Snippet; children: Snippet } = $props()
</script>

<section class="panel">
  {#if title || header || actions}
    <header>
      {#if header}
        <div class="custom">{@render header()}</div>
      {:else if title}<h2>{title}</h2>{/if}
      {#if actions}<div class="actions">{@render actions()}</div>{/if}
    </header>
  {/if}
  <div class="body">
    {@render children()}
  </div>
</section>

<style>
  .panel {
    display: flex;
    flex-direction: column;
    min-height: 0;
    background: var(--panel);
    border: 1px solid var(--frame);
    border-radius: var(--radius);
    box-shadow: var(--shadow-inner), var(--shadow-outer);
  }

  header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--gap);
    padding: 4px 10px;
    background: var(--panel-title);
    border-bottom: 1px solid var(--frame-dim);
  }

  h2 {
    margin: 0;
    font-family: var(--font-display);
    font-weight: 600;
    font-size: 15px;
    letter-spacing: 0.02em;
    color: var(--text);
  }

  .custom {
    display: flex;
    align-items: center;
    gap: 6px;
    flex: 1;
    min-width: 0;
    font-family: var(--font-display);
    font-weight: 600;
    font-size: 15px;
    letter-spacing: 0.02em;
    color: var(--text);
  }

  .actions {
    display: flex;
    gap: 6px;
  }

  .body {
    padding: 10px;
    min-height: 0;
    overflow: auto;
  }
</style>
