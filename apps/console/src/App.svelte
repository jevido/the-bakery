<script lang="ts">
  import Layout from './components/Layout.svelte'
  import Login from './pages/Login.svelte'
  import Members from './pages/Members.svelte'
  import Member from './pages/Member.svelte'
  import Guilds from './pages/Guilds.svelte'
  import Guild from './pages/Guild.svelte'
  import Reports from './pages/Reports.svelte'
  import Audit from './pages/Audit.svelte'
  import Signals from './pages/Signals.svelte'
  import { router } from './lib/router.svelte'
  import { session } from './lib/session.svelte'

  session.load()

  let path = $derived(router.path.replace(/\/+$/, '') || '/')
  let memberId = $derived(Number(path.match(/^\/members\/(\d+)$/)?.[1] ?? 0))
  let guildId = $derived(Number(path.match(/^\/guilds\/(\d+)$/)?.[1] ?? 0))
</script>

<svelte:document onclick={router.onclick} />

<Layout>
  {#if !session.checked}
    <p class="dim">Checking…</p>
  {:else if !session.operator}
    <Login />
  {:else if memberId}
    {#key memberId}<Member id={memberId} />{/key}
  {:else if guildId}
    {#key guildId}<Guild id={guildId} />{/key}
  {:else if path === '/guilds'}
    <Guilds />
  {:else if path === '/reports'}
    <Reports />
  {:else if path === '/audit'}
    <Audit />
  {:else if path === '/signals'}
    <Signals />
  {:else}
    <Members />
  {/if}
</Layout>
