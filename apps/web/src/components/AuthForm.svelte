<script lang="ts">
  import { Panel, Button, TextField } from '@bakery/ui'
  import { session } from '../lib/session.svelte'
  import { router } from '../lib/router.svelte'
  import { ApiError } from '../lib/api'

  let { mode }: { mode: 'signin' | 'signup' } = $props()

  let email = $state('')
  let displayName = $state('')
  let password = $state('')
  let error = $state('')
  let busy = $state(false)

  // Where to go afterwards: ?next= if it is a path on this site.
  function next(): string {
    const n = new URLSearchParams(window.location.search).get('next') ?? ''
    return n.startsWith('/') && !n.startsWith('//') ? n : '/admin'
  }

  function otherPage(): string {
    const q = window.location.search
    return (mode === 'signin' ? '/signup' : '/signin') + q
  }

  async function submit(event: SubmitEvent) {
    event.preventDefault()
    busy = true
    error = ''
    try {
      if (mode === 'signin') await session.signIn(email, password)
      else await session.signUp(email, displayName, password)
      router.navigate(next())
    } catch (err) {
      error = err instanceof ApiError ? err.message : String(err)
    } finally {
      busy = false
    }
  }
</script>

<div class="wrap">
  <Panel title={mode === 'signin' ? 'Report for duty' : 'Join the colony'}>
    <form onsubmit={submit}>
      <TextField label="Email" type="email" autocomplete="email" required bind:value={email} />
      {#if mode === 'signup'}
        <TextField label="Display name" autocomplete="nickname" required maxlength={60} bind:value={displayName} />
      {/if}
      <TextField
        label="Password"
        type="password"
        autocomplete={mode === 'signin' ? 'current-password' : 'new-password'}
        required
        minlength={mode === 'signup' ? 8 : undefined}
        bind:value={password}
      />
      {#if error}<p class="error" role="alert">{error}</p>{/if}
      <Button type="submit" variant="confirm" disabled={busy}>{mode === 'signin' ? 'Sign in' : 'Sign up'}</Button>
    </form>
    <p class="switch">
      {#if mode === 'signin'}
        New here? <a href={otherPage()}>Sign up</a>
      {:else}
        Already have an account? <a href={otherPage()}>Sign in</a>
      {/if}
    </p>
  </Panel>
</div>

<style>
  .wrap {
    display: grid;
    place-items: center;
    padding: 24px 0;
  }

  .wrap :global(.panel) {
    width: min(380px, 100%);
  }

  form {
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .error {
    margin: 0;
    padding: 5px 8px;
    background: color-mix(in srgb, var(--rust) 35%, transparent);
    border: 1px solid var(--rust);
    border-radius: var(--radius);
  }

  .switch {
    margin: 12px 0 0;
    color: var(--text-dim);
  }
</style>
