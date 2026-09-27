<script lang="ts">
  import type { Snippet } from 'svelte'
  import { router } from '../lib/router.svelte'
  import { session } from '../lib/session.svelte'

  let { children }: { children: Snippet } = $props()

  async function signOut() {
    await session.signOut()
    router.navigate('/')
  }

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
    <div class="account">
      {#if session.member}
        <a href="/admin">{session.member.display_name}</a>
        <button class="link" onclick={signOut}>Sign out</button>
      {:else if session.member === null}
        <a href="/signin">Sign in</a>
        <a class="cta" href="/signup">Sign up</a>
      {/if}
    </div>
  </header>

  <main>
    {@render children()}
  </main>

  <footer>
    <span>The Bakery is open source.</span>
    <a href="https://github.com/jevido/the-bakery">Source on GitHub</a>
    <a href="https://github.com/jevido/the-bakery/blob/main/CONTRIBUTING.md">Contributing</a>
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

  .account {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .account a,
  .link {
    color: var(--text-dim);
    text-decoration: none;
  }

  .link {
    font: inherit;
    padding: 0;
    background: none;
    border: none;
    cursor: pointer;
  }

  .account a:hover,
  .link:hover {
    color: var(--text);
  }

  .account .cta {
    padding: 2px 10px;
    color: var(--text);
    background: var(--olive);
    border: 1px solid var(--frame);
    border-radius: var(--radius);
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
