<script lang="ts">
  import Login from './screens/Login.svelte'
  import Colony from './screens/Colony.svelte'
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
    } catch (err) {
      error = messageOf(err)
    }
    member = null
  }
</script>

{#if member === undefined}
  <div class="loading">Waking the colonists…</div>
{:else if member === null}
  <Login onsignedin={(m) => (member = m)} />
{:else}
  {#key member.id}
    <Colony {member} onclockout={clockOut} onsignedout={() => (member = null)} />
  {/key}
{/if}

{#if error}<p class="error">{error}</p>{/if}

<style>
  .loading {
    height: 100%;
    display: grid;
    place-items: center;
    color: var(--text-faint);
    font-family: var(--font-display);
  }

  .error {
    position: fixed;
    bottom: 8px;
    left: 50%;
    translate: -50% 0;
    margin: 0;
    color: var(--rust-bright);
  }
</style>
