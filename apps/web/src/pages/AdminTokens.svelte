<script lang="ts">
  import { Panel, Button, TextField } from '@bakery/ui'
  import AdminNav from '../components/AdminNav.svelte'
  import Confirm from '../components/Confirm.svelte'
  import { tokens, mcpAddCommand, type PersonalToken } from '../lib/tokens'
  import { ApiError } from '../lib/api'

  let list = $state.raw<PersonalToken[] | undefined>(undefined)
  let error = $state('')
  let name = $state('')
  let busy = $state(false)
  // The secret of the token just made; it is never shown again.
  let fresh = $state<{ name: string; secret: string } | null>(null)
  let copied = $state<'token' | 'command' | null>(null)
  let revoking = $state<number | null>(null)

  function fail(err: unknown) {
    error = err instanceof ApiError ? err.message : String(err)
  }

  async function load() {
    try {
      list = await tokens.list()
    } catch (err) {
      fail(err)
    }
  }
  load()

  async function create(event: SubmitEvent) {
    event.preventDefault()
    busy = true
    error = ''
    try {
      const r = await tokens.create(name.trim())
      fresh = { name: r.personal_token.name, secret: r.token }
      name = ''
      await load()
    } catch (err) {
      fail(err)
    } finally {
      busy = false
    }
  }

  async function revoke(t: PersonalToken) {
    busy = true
    try {
      await tokens.revoke(t.id)
      revoking = null
      await load()
    } catch (err) {
      fail(err)
    } finally {
      busy = false
    }
  }

  async function copy(what: 'token' | 'command', text: string) {
    try {
      await navigator.clipboard.writeText(text)
      copied = what
      setTimeout(() => (copied = null), 1500)
    } catch {
      error = 'Copying failed; select the text and copy it yourself.'
    }
  }

  function when(iso: string | null): string {
    return iso ? new Date(iso).toLocaleString(undefined, { dateStyle: 'medium', timeStyle: 'short' }) : 'never'
  }
</script>

<svelte:head>
  <title>Tokens — The Bakery</title>
</svelte:head>

<AdminNav />

<h1>Personal tokens</h1>
<p class="intro">
  A personal token lets Claude, or anything else you give it to, act as you in your guilds: read boards and add,
  change, move and delete tasks. Make one per place you use it, and revoke it when you stop.
</p>

{#if error}<p class="error" role="alert">{error}</p>{/if}

<div class="grid">
  <div class="stack">
    <Panel title="New token">
      <form onsubmit={create}>
        <TextField label="Name" required maxlength={60} placeholder="Claude on my laptop" bind:value={name} />
        <Button type="submit" variant="confirm" disabled={busy || !name.trim()}>Make token</Button>
      </form>
    </Panel>

    {#if fresh}
      <Panel title={`Token “${fresh.name}”`}>
        <div class="fresh">
          <p class="warn">Copy it now. It will not be shown again.</p>
          <code class="secret">{fresh.secret}</code>
          <Button onclick={() => copy('token', fresh!.secret)}>{copied === 'token' ? 'Copied' : 'Copy token'}</Button>
          <p class="dim">Connect Claude Code by pasting this in a terminal:</p>
          <code class="secret">{mcpAddCommand(fresh.secret)}</code>
          <Button onclick={() => copy('command', mcpAddCommand(fresh!.secret))}>
            {copied === 'command' ? 'Copied' : 'Copy command'}
          </Button>
          <Button onclick={() => (fresh = null)}>I have it</Button>
        </div>
      </Panel>
    {/if}
  </div>

  <Panel title={`Tokens · ${list?.length ?? 0}`}>
    {#if list === undefined}
      <p class="dim">Counting keys…</p>
    {:else if list.length === 0}
      <p class="dim">No tokens yet.</p>
    {:else}
      <ul class="tokens">
        {#each list as t (t.id)}
          <li>
            <span class="name">{t.name}</span>
            <span class="meta">made {when(t.created_at)} · last used {when(t.last_used_at)}</span>
            {#if revoking === t.id}
              <Confirm
                question={`Revoke “${t.name}”? Whatever uses it stops working.`}
                action="Revoke"
                {busy}
                onconfirm={() => revoke(t)}
                oncancel={() => (revoking = null)}
              />
            {:else}
              <span><Button variant="danger" onclick={() => (revoking = t.id)}>Revoke</Button></span>
            {/if}
          </li>
        {/each}
      </ul>
    {/if}
  </Panel>
</div>

<style>
  h1 {
    margin: 0 0 6px;
    font-family: var(--font-display);
    font-weight: 600;
    font-size: 28px;
  }

  .intro {
    margin: 0 0 12px;
    max-width: 65ch;
    color: var(--text-dim);
  }

  .grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
    gap: var(--gap);
    align-items: start;
  }

  .stack {
    display: flex;
    flex-direction: column;
    gap: var(--gap);
    min-width: 0;
  }

  form,
  .fresh {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: 10px;
  }

  form {
    align-items: stretch;
  }

  .secret {
    display: block;
    width: 100%;
    box-sizing: border-box;
    padding: 6px 8px;
    font-size: 13px;
    overflow-wrap: anywhere;
    user-select: all;
    background: var(--panel-inset);
    border: 1px solid var(--frame-dim);
    border-radius: var(--radius);
  }

  .warn {
    padding: 5px 8px;
    background: color-mix(in srgb, var(--rust) 35%, transparent);
    border: 1px solid var(--rust);
    border-radius: var(--radius);
  }

  .tokens {
    list-style: none;
    margin: 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  li {
    display: flex;
    flex-direction: column;
    gap: 4px;
    padding-bottom: 10px;
    border-bottom: 1px solid var(--frame-dim);
  }

  li:last-child {
    border-bottom: none;
    padding-bottom: 0;
  }

  .name {
    font-weight: 600;
  }

  .meta {
    font-size: 12px;
    color: var(--text-dim);
  }

  p {
    margin: 0;
  }

  .dim {
    color: var(--text-dim);
  }

  .error {
    margin: 0 0 10px;
    padding: 5px 8px;
    background: color-mix(in srgb, var(--rust) 35%, transparent);
    border: 1px solid var(--rust);
    border-radius: var(--radius);
  }
</style>
