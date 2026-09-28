<script lang="ts">
  import type { Snippet } from 'svelte'
  import { router } from '../lib/router.svelte'
  import { session } from '../lib/session.svelte'

  let { children }: { children: Snippet } = $props()

  const links = [
    { href: '/members', label: 'Members' },
    { href: '/guilds', label: 'Guilds' },
    { href: '/reports', label: 'Reports' },
    { href: '/signals', label: 'Signals' },
    { href: '/audit', label: 'Audit log' },
  ]
</script>

<div class="shell">
  <header>
    <strong class="brand">The Bakery · Console</strong>
    {#if session.operator}
      <nav>
        {#each links as l (l.href)}
          <a href={l.href} class={{ here: router.path.startsWith(l.href) }}>{l.label}</a>
        {/each}
      </nav>
      <span class="who">
        {session.operator.email}
        <button class="link" onclick={() => session.signOut()}>Sign out</button>
      </span>
    {/if}
  </header>
  <main>{@render children()}</main>
</div>

<style>
  .shell {
    max-width: 1100px;
    margin: 0 auto;
    padding: 16px;
  }

  header {
    display: flex;
    flex-wrap: wrap;
    gap: 16px;
    align-items: center;
    padding-bottom: 12px;
    margin-bottom: 16px;
    border-bottom: 1px solid var(--frame);
  }

  .brand {
    font-family: var(--font-display);
    color: var(--rust-bright);
  }

  nav {
    display: flex;
    gap: 12px;
    flex: 1;
  }

  nav a {
    text-decoration: none;
  }

  nav a.here {
    color: var(--text);
    text-decoration: underline;
  }

  .who {
    color: var(--text-dim);
    font-size: 13px;
  }

  .link {
    font: inherit;
    color: var(--steel-bright);
    background: none;
    border: none;
    cursor: pointer;
    text-decoration: underline;
  }
</style>
