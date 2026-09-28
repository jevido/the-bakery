<script lang="ts">
  import type { Component } from 'svelte'
  import Layout from './components/Layout.svelte'
  import Landing from './pages/Landing.svelte'
  import Desktop from './pages/Desktop.svelte'
  import SignIn from './pages/SignIn.svelte'
  import SignUp from './pages/SignUp.svelte'
  import Admin from './pages/Admin.svelte'
  import AdminGuild from './pages/AdminGuild.svelte'
  import AdminTokens from './pages/AdminTokens.svelte'
  import NotFound from './pages/NotFound.svelte'
  import Handoff from './pages/Handoff.svelte'
  import Join from './pages/Join.svelte'
  import HowItWorks from './pages/HowItWorks.svelte'
  import Contributors from './pages/Contributors.svelte'
  import { router } from './lib/router.svelte'
  import { session } from './lib/session.svelte'

  const routes: Record<string, Component> = {
    '/': Landing,
    '/desktop': Desktop,
    '/how-it-works': HowItWorks,
    '/contributors': Contributors,
    '/signin': SignIn,
    '/signup': SignUp,
    '/admin': Admin,
    '/admin/tokens': AdminTokens,
    '/handoff': Handoff,
  }

  session.load()

  let path = $derived(router.path.replace(/\/+$/, '') || '/')
  let needsMember = $derived(path === '/admin' || path.startsWith('/admin/'))
  let Page = $derived(routes[path] ?? NotFound)
  // /admin/guilds/{id}
  let guildId = $derived(Number(path.match(/^\/admin\/guilds\/(\d+)$/)?.[1] ?? 0))
  // /join/{code}
  let joinCode = $derived(path.match(/^\/join\/([A-Za-z0-9]+)$/)?.[1] ?? '')

  // Signed-out visitors of /admin go to sign in and come back afterwards.
  $effect(() => {
    if (needsMember && session.member === null) {
      router.replace(`/signin?next=${encodeURIComponent(window.location.pathname + window.location.search)}`)
    }
  })
</script>

<svelte:document onclick={router.onclick} />

<Layout>
  {#if session.refusal}
    <p class="refusal" role="alert">{session.refusal} <a href="https://jevidocs.jevido.app/p/bakery/app/help">What this means</a></p>
  {/if}
  {#if needsMember && !session.member && !session.refusal}
    <p class="dim">Checking your papers…</p>
  {:else if joinCode}
    {#key joinCode}<Join code={joinCode} />{/key}
  {:else if guildId}
    {#key guildId}<AdminGuild id={guildId} />{/key}
  {:else}
    <Page />
  {/if}
</Layout>

<style>
  .refusal {
    margin: 0 0 16px;
    padding: 10px 12px;
    background: color-mix(in srgb, var(--rust) 30%, transparent);
    border: 1px solid var(--rust);
    border-radius: var(--radius);
  }

  .dim {
    color: var(--text-faint);
  }
</style>
