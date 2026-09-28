<script lang="ts">
  import { Button, Panel, TextField } from '@bakery/ui'
  import { api, ApiError } from '../lib/api'
  import { session } from '../lib/session.svelte'

  // Two steps: the password gives a short-lived challenge, and the challenge
  // with a code from the authenticator app gives the operator's token.
  let email = $state('')
  let password = $state('')
  let code = $state('')
  let challenge = $state('')
  let busy = $state(false)
  let error = $state('')

  async function submitPassword(event: SubmitEvent) {
    event.preventDefault()
    busy = true
    error = ''
    try {
      const res = await api<{ challenge: string }>('POST', '/api/console/login', { email, password }, '')
      challenge = res.challenge
      password = ''
    } catch (err) {
      error = err instanceof ApiError ? err.message : String(err)
    }
    busy = false
  }

  async function submitCode(event: SubmitEvent) {
    event.preventDefault()
    busy = true
    error = ''
    try {
      const res = await api<{ token: string }>('POST', '/api/console/totp/verify', { challenge, code: code.trim() }, '')
      await session.signIn(res.token)
    } catch (err) {
      error = err instanceof ApiError ? err.message : String(err)
      if (err instanceof ApiError && err.status === 401) code = ''
    }
    busy = false
  }
</script>

<svelte:head><title>Sign in · Console</title></svelte:head>

<div class="login">
  <Panel title="Operator sign-in">
    {#if error}<p class="error" role="alert">{error}</p>{/if}
    {#if !challenge}
      <form onsubmit={submitPassword}>
        <TextField label="Email" type="email" autocomplete="username" required bind:value={email} />
        <TextField label="Password" type="password" autocomplete="current-password" required bind:value={password} />
        <Button variant="confirm" type="submit" disabled={busy}>Continue</Button>
      </form>
    {:else}
      <form onsubmit={submitCode}>
        <p class="dim">Enter the six-digit code from your authenticator app.</p>
        <TextField label="Code" inputmode="numeric" autocomplete="one-time-code" maxlength={6} required bind:value={code} />
        <div class="row">
          <Button variant="confirm" type="submit" disabled={busy}>Sign in</Button>
          <Button type="button" onclick={() => ((challenge = ''), (code = ''))}>Back</Button>
        </div>
      </form>
    {/if}
  </Panel>
</div>

<style>
  .login {
    max-width: 360px;
    margin: 60px auto;
  }

  form {
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .row {
    display: flex;
    gap: 8px;
  }

  p {
    margin: 0 0 8px;
  }
</style>
