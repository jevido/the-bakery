<script lang="ts">
  import type { Snippet } from 'svelte'

  // Tabs over one panel of content: the caller renders the active tab's
  // content in `children`, given the active tab's id. Arrow keys move
  // between tabs.
  let {
    tabs,
    active = $bindable(tabs[0]?.id ?? ''),
    label,
    children,
  }: {
    tabs: { id: string; label: string }[]
    active?: string
    label: string
    children: Snippet<[string]>
  } = $props()

  const uid = Math.random().toString(36).slice(2, 8)

  function onkeydown(event: KeyboardEvent) {
    const i = tabs.findIndex((t) => t.id === active)
    let next = -1
    if (event.key === 'ArrowRight') next = (i + 1) % tabs.length
    if (event.key === 'ArrowLeft') next = (i - 1 + tabs.length) % tabs.length
    if (next < 0) return
    event.preventDefault()
    active = tabs[next].id
    document.getElementById(`tab-${uid}-${active}`)?.focus()
  }
</script>

<div class="tabs">
  <div class="tablist" role="tablist" aria-label={label} tabindex="-1" {onkeydown}>
    {#each tabs as t (t.id)}
      <button
        id={`tab-${uid}-${t.id}`}
        class="tab"
        role="tab"
        aria-selected={t.id === active}
        aria-controls={`panel-${uid}`}
        tabindex={t.id === active ? 0 : -1}
        onclick={() => (active = t.id)}
      >
        {t.label}
      </button>
    {/each}
  </div>
  <div id={`panel-${uid}`} class="tabpanel" role="tabpanel" aria-labelledby={`tab-${uid}-${active}`}>
    {@render children(active)}
  </div>
</div>

<style>
  .tabs {
    display: flex;
    flex-direction: column;
    min-height: 0;
  }

  .tablist {
    display: flex;
    gap: 2px;
    border-bottom: 1px solid var(--frame-dim);
  }

  .tab {
    font: inherit;
    font-size: 13px;
    color: var(--text-dim);
    padding: 4px 10px;
    margin-bottom: -1px;
    background: none;
    border: 1px solid transparent;
    border-radius: var(--radius) var(--radius) 0 0;
    cursor: pointer;
  }

  .tab:hover {
    color: var(--text);
  }

  .tab[aria-selected='true'] {
    color: var(--text);
    background: var(--panel-raised);
    border-color: var(--frame-dim);
    border-bottom-color: var(--panel-raised);
  }

  .tabpanel {
    padding-top: 8px;
    min-height: 0;
  }
</style>
