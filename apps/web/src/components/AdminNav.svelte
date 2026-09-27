<script lang="ts">
  import { router } from '../lib/router.svelte'

  const links = [
    { href: '/admin', label: 'Guilds', flavor: 'Who you stand with' },
    { href: '/admin/tokens', label: 'Tokens', flavor: 'Keys to the colony' },
  ]

  let path = $derived(router.path.replace(/\/+$/, ''))
</script>

<nav aria-label="Admin">
  {#each links as l (l.href)}
    <a href={l.href} aria-current={path === l.href ? 'page' : undefined}>
      <span class="label">{l.label}</span>
      <span class="flavor">{l.flavor}</span>
    </a>
  {/each}
</nav>

<style>
  nav {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
    margin-bottom: var(--gap);
  }

  a {
    display: flex;
    flex-direction: column;
    padding: 5px 12px;
    color: var(--text);
    text-decoration: none;
    border: 1px solid var(--frame-dim);
    border-radius: var(--radius);
  }

  a[aria-current='page'] {
    background: var(--panel-inset);
    border-color: var(--frame);
  }

  .label {
    font-weight: 600;
  }

  .flavor {
    font-size: 12px;
    color: var(--text-dim);
  }
</style>
