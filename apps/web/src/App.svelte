<script lang="ts">
  import type { Component } from 'svelte'
  import Layout from './components/Layout.svelte'
  import Landing from './pages/Landing.svelte'
  import Desktop from './pages/Desktop.svelte'
  import NotFound from './pages/NotFound.svelte'
  import { router } from './lib/router.svelte'

  const routes: Record<string, Component> = {
    '/': Landing,
    '/desktop': Desktop,
  }

  let Page = $derived(routes[router.path.replace(/\/+$/, '') || '/'] ?? NotFound)
</script>

<svelte:document onclick={router.onclick} />

<Layout>
  <Page />
</Layout>
