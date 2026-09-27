<script lang="ts">
  import Panel from '../components/Panel.svelte'
  import Button from '../components/Button.svelte'
  import TextField from '../components/TextField.svelte'
  import { SessionService, messageOf, type Member } from '../lib/bindings'

  let { onsignedin }: { onsignedin: (member: Member) => void } = $props()

  let registering = $state(false)
  let email = $state('')
  let displayName = $state('')
  let password = $state('')
  let error = $state('')
  let busy = $state(false)

  async function submit(event: SubmitEvent) {
    event.preventDefault()
    busy = true
    error = ''
    try {
      const member = registering
        ? await SessionService.Register(email, displayName, password)
        : await SessionService.Login(email, password)
      onsignedin(member)
    } catch (err) {
      error = messageOf(err)
    } finally {
      busy = false
    }
  }

  function toggle() {
    registering = !registering
    error = ''
  }
</script>

<div class="screen">
  <Panel title={registering ? 'Join the colony' : 'Report for duty'}>
    <form onsubmit={submit}>
      <TextField label="Email" type="email" autocomplete="email" required bind:value={email} />
      {#if registering}
        <TextField label="Display name" autocomplete="nickname" required maxlength={60} bind:value={displayName} />
      {/if}
      <TextField
        label="Password"
        type="password"
        autocomplete={registering ? 'new-password' : 'current-password'}
        required
        bind:value={password}
      />
      {#if error}<p class="error" role="alert">{error}</p>{/if}
      <div class="row">
        <Button type="button" onclick={toggle} disabled={busy}>
          {registering ? 'I have an account' : 'New colonist'}
        </Button>
        <Button type="submit" variant="confirm" disabled={busy}>
          {registering ? 'Register' : 'Sign in'}
        </Button>
      </div>
    </form>
  </Panel>
</div>

<style>
  .screen {
    height: 100%;
    display: grid;
    place-items: center;
    padding: 16px;
  }

  .screen :global(.panel) {
    width: min(360px, 100%);
  }

  form {
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .row {
    display: flex;
    justify-content: space-between;
    margin-top: 4px;
  }

  .error {
    margin: 0;
    padding: 5px 8px;
    color: var(--text);
    background: color-mix(in srgb, var(--rust) 35%, transparent);
    border: 1px solid var(--rust);
    border-radius: var(--radius);
  }
</style>
