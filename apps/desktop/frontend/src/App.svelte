<script lang="ts">
  import Panel from './components/Panel.svelte'
  import Button from './components/Button.svelte'
  import Login from './screens/Login.svelte'
  import { SessionService, messageOf, type Member } from './lib/bindings'

  // undefined while the saved session is being checked.
  let member = $state<Member | null | undefined>(undefined)
  let error = $state('')

  SessionService.Me().then(
    (m) => (member = m),
    (err) => {
      member = null
      error = messageOf(err)
    },
  )

  async function clockOut() {
    try {
      await SessionService.Logout()
      member = null
    } catch (err) {
      error = messageOf(err)
    }
  }
</script>

{#if member === undefined}
  <div class="loading">Waking the colonists…</div>
{:else if member === null}
  <Login onsignedin={(m) => (member = m)} />
{:else}
  <main>
    <Panel title="First Colony — Colony log">
      {#snippet actions()}
        <Button onclick={clockOut}>Clock out</Button>
      {/snippet}
      <p>Day 1 of Aprimay. {member.display_name} wakes beside the crashed pod, with little more than a plan.</p>
      {#if error}<p class="error">{error}</p>{/if}
    </Panel>
  </main>
{/if}

<style>
  main {
    height: 100%;
    padding: 16px;
    display: grid;
    place-items: start center;
  }

  main :global(.panel) {
    width: min(640px, 100%);
  }

  p {
    margin: 0;
    color: var(--text-dim);
  }

  .loading {
    height: 100%;
    display: grid;
    place-items: center;
    color: var(--text-faint);
    font-family: var(--font-display);
  }

  .error {
    margin-top: 8px;
    color: var(--rust-bright);
  }
</style>
