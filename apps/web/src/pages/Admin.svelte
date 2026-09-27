<script lang="ts">
  import { Panel } from '@bakery/ui'
  import { session } from '../lib/session.svelte'
</script>

<svelte:head>
  <title>Your guilds — The Bakery</title>
</svelte:head>

<h1>Your guilds</h1>
<Panel title="Guild admin">
  <p>Signed in as {session.member?.display_name}. Founding and running guilds lands here next.</p>
</Panel>

<style>
  h1 {
    margin: 0 0 12px;
    font-family: var(--font-display);
    font-weight: 600;
    font-size: 28px;
  }

  p {
    margin: 0;
    color: var(--text-dim);
  }
</style>
