<script lang="ts">
  import { Panel } from '@bakery/ui'
  import { api, ApiError, type Member } from '../lib/api'
  import { session } from '../lib/session.svelte'
  import { router } from '../lib/router.svelte'

  // The desktop app opens /handoff?code=...&next=/admin to sign its member in
  // here too. The code works once, for a minute.
  const params = new URLSearchParams(window.location.search)
  const code = params.get('code') ?? ''
  const nextParam = params.get('next') ?? ''
  const next = nextParam.startsWith('/') && !nextParam.startsWith('//') ? nextParam : '/admin'

  // Drop the code from the address bar and history right away.
  router.replace(`/handoff?next=${encodeURIComponent(next)}`)

  let error = $state('')

  api<{ member: Member }>('POST', '/api/web/handoff/redeem', { code }).then(
    (res) => {
      session.adopt(res.member)
      router.replace(next)
    },
    async (err) => {
      // An old link opened by someone already signed in here: just go on.
      await session.load()
      if (session.member) router.replace(next)
      else error = err instanceof ApiError ? err.message : String(err)
    },
  )
</script>

<svelte:head>
  <title>Signing you in — The Bakery</title>
</svelte:head>

{#if error}
  <Panel title="Sign in on the website">
    <p>{error}</p>
    <p><a href={`/signin?next=${encodeURIComponent(next)}`}>Sign in</a></p>
  </Panel>
{:else}
  <p class="dim">Walking you over from the desktop…</p>
{/if}

<style>
  p {
    margin: 0 0 8px;
  }

  .dim {
    color: var(--text-faint);
  }
</style>
