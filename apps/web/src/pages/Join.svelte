<script lang="ts">
  import { Panel, Button } from '@bakery/ui'
  import { invites, type InviteInfo } from '../lib/invites'
  import { session } from '../lib/session.svelte'
  import { router } from '../lib/router.svelte'
  import { ApiError } from '../lib/api'

  let { code }: { code: string } = $props()

  let info = $state<InviteInfo | null | undefined>(undefined)
  let error = $state('')
  let busy = $state(false)

  $effect(() => {
    invites.info(code).then(
      (i) => (info = i),
      () => (info = null),
    )
  })

  let here = $derived(`/join/${code}`)

  async function join() {
    busy = true
    error = ''
    try {
      const g = await invites.accept(code)
      router.navigate(`/admin/guilds/${g.id}`)
    } catch (err) {
      error = err instanceof ApiError ? err.message : String(err)
    } finally {
      busy = false
    }
  }
</script>

<svelte:head>
  <title>{info ? `Join ${info.guild_name} — The Bakery` : 'Join a guild — The Bakery'}</title>
</svelte:head>

<div class="wrap">
  {#if info === undefined}
    <p class="dim">Reading the invitation…</p>
  {:else if info === null || !info.valid}
    <Panel title="This invite leads nowhere">
      <p>{info?.problem ? `${info.problem.charAt(0).toUpperCase()}${info.problem.slice(1)}.` : 'This invite link does not exist.'}</p>
      <p class="dim">Ask whoever sent it for a new link.</p>
    </Panel>
  {:else}
    <Panel title={`Join ${info.guild_name}`}>
      <p>{info.member_count} {info.member_count === 1 ? 'member' : 'members'} already work here.</p>
      {#if error}<p class="error" role="alert">{error}</p>{/if}
      {#if session.member}
        <Button variant="confirm" onclick={join} disabled={busy}>Join as {session.member.display_name}</Button>
      {:else if session.member === null}
        <p class="dim">You need an account to join.</p>
        <div class="row">
          <a class="cta" href={`/signup?next=${encodeURIComponent(here)}`}>Sign up</a>
          <a href={`/signin?next=${encodeURIComponent(here)}`}>I have an account</a>
        </div>
      {/if}
    </Panel>
  {/if}
</div>

<style>
  .wrap {
    display: grid;
    place-items: center;
    padding: 24px 0;
  }

  .wrap :global(.panel) {
    width: min(420px, 100%);
  }

  p {
    margin: 0 0 10px;
  }

  .dim {
    color: var(--text-dim);
  }

  .row {
    display: flex;
    align-items: center;
    gap: 16px;
  }

  .cta {
    font-family: var(--font-display);
    color: var(--text);
    text-decoration: none;
    padding: 5px 14px;
    background: var(--olive);
    border: 1px solid var(--frame);
    border-radius: var(--radius);
  }

  .error {
    padding: 5px 8px;
    background: color-mix(in srgb, var(--rust) 35%, transparent);
    border: 1px solid var(--rust);
    border-radius: var(--radius);
  }
</style>
