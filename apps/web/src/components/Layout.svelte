<script lang="ts">
  import type { Snippet } from 'svelte'
  import { router } from '../lib/router.svelte'

  let { children }: { children: Snippet } = $props()

  const nav = [
    { href: '/', label: 'Home' },
    { href: '/desktop', label: 'Desktop app' },
  ]
</script>

<div class="site">
  <header>
    <a class="brand" href="/">The Bakery</a>
    <nav aria-label="Main">
      {#each nav as item (item.href)}
        <a href={item.href} aria-current={router.path === item.href ? 'page' : undefined}>{item.label}</a>
      {/each}
      <a href="https://jevidocs.jevido.app/p/bakery">Docs</a>
      <a href="https://github.com/jevido/the-bakery">GitHub</a>
    </nav>
    <!-- Room for the account menu (phase 03). -->
  </header>

  <main>
    {@render children()}
  </main>

  <footer>
    <span>The Bakery is open source.</span>
    <a href="https://github.com/jevido/the-bakery">Source on GitHub</a>
  </footer>
</div>

<style>
  .site {
    min-height: 100vh;
    display: flex;
    flex-direction: column;
  }

  header {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    justify-content: space-between;
    gap: 8px 16px;
    padding: 10px 16px;
    background: var(--panel-title);
    border-bottom: 1px solid var(--frame);
    box-shadow: var(--shadow-outer);
  }

  .brand {
    font-family: var(--font-display);
    font-weight: 600;
    font-size: 20px;
    color: var(--text);
    text-decoration: none;
  }

  nav {
    display: flex;
    flex-wrap: wrap;
    gap: 4px 14px;
  }

  nav a {
    color: var(--text-dim);
    text-decoration: none;
  }

  nav a:hover,
  nav a[aria-current='page'] {
    color: var(--text);
  }

  nav a[aria-current='page'] {
    text-decoration: underline;
    text-underline-offset: 4px;
  }

  main {
    flex: 1;
    width: min(960px, 100%);
    margin: 0 auto;
    padding: 24px 16px 40px;
  }

  footer {
    display: flex;
    flex-wrap: wrap;
    justify-content: center;
    gap: 6px 16px;
    padding: 14px 16px;
    color: var(--text-faint);
    border-top: 1px solid var(--frame-dim);
  }
</style>
