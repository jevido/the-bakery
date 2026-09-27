<script lang="ts">
  import { Panel, Button, TextField } from '@bakery/ui'
  import { guilds, type Guild } from '../lib/guilds'
  import { router } from '../lib/router.svelte'
  import { ApiError } from '../lib/api'

  let showArchived = $state(false)
  let list = $state<Guild[] | undefined>(undefined)
  let error = $state('')
  let name = $state('')
  let founding = $state(false)

  async function load() {
    try {
      list = await guilds.list(showArchived)
      error = ''
    } catch (err) {
      error = err instanceof ApiError ? err.message : String(err)
    }
  }

  $effect(() => {
    showArchived // reload when the toggle changes
    load()
  })

  async function found(event: SubmitEvent) {
    event.preventDefault()
    founding = true
    try {
      const g = await guilds.found(name)
      router.navigate(`/admin/guilds/${g.id}`)
    } catch (err) {
      error = err instanceof ApiError ? err.message : String(err)
    } finally {
      founding = false
    }
  }
</script>

<svelte:head>
  <title>Your guilds — The Bakery</title>
</svelte:head>

<h1>Your guilds</h1>

{#if error}<p class="error" role="alert">{error}</p>{/if}

<div class="grid">
  <Panel title="Guilds">
    {#snippet actions()}
      <label class="toggle"><input type="checkbox" bind:checked={showArchived} /> Show archived</label>
    {/snippet}
    {#if list === undefined}
      <p class="dim">Counting heads…</p>
    {:else if list.length === 0}
      <p class="dim">No guilds yet. Every colony starts with a few people and a crash-landing.</p>
    {:else}
      <ul class="list">
        {#each list as g (g.id)}
          <li>
            <a href={`/admin/guilds/${g.id}`}>{g.name}</a>
            {#if g.archived}<span class="tag">archived</span>{/if}
          </li>
        {/each}
      </ul>
    {/if}
  </Panel>

  <Panel title="Found a guild">
    <form onsubmit={found}>
      <TextField label="Name" required maxlength={60} placeholder="First Colony" bind:value={name} />
      <Button type="submit" variant="confirm" disabled={founding || !name.trim()}>Found it</Button>
    </form>
    <p class="dim small">You become its first member. Invite the others once it stands.</p>
  </Panel>
</div>

<style>
  h1 {
    margin: 0 0 12px;
    font-family: var(--font-display);
    font-weight: 600;
    font-size: 28px;
  }

  .grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
    gap: var(--gap);
    align-items: start;
  }

  .list {
    list-style: none;
    margin: 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .list a {
    color: var(--text);
  }

  .tag {
    margin-left: 6px;
    padding: 0 6px;
    font-size: 12px;
    color: var(--text-dim);
    border: 1px solid var(--frame-dim);
    border-radius: var(--radius);
  }

  .toggle {
    font-size: 12px;
    color: var(--text-dim);
  }

  form {
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  p {
    margin: 0;
  }

  .dim {
    color: var(--text-dim);
  }

  .small {
    margin-top: 10px;
    font-size: 12px;
  }

  .error {
    margin: 0 0 10px;
    padding: 5px 8px;
    background: color-mix(in srgb, var(--rust) 35%, transparent);
    border: 1px solid var(--rust);
    border-radius: var(--radius);
  }
</style>
