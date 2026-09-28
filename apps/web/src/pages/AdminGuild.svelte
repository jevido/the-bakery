<script lang="ts">
  import { Panel, Button, TextField } from '@bakery/ui'
  import Confirm from '../components/Confirm.svelte'
  import InvitesTab from '../components/InvitesTab.svelte'
  import ReportForm from '../components/ReportForm.svelte'
  import { guilds, joinedOn, type Guild, type GuildMember } from '../lib/guilds'
  import { session } from '../lib/session.svelte'
  import { router } from '../lib/router.svelte'
  import { ApiError } from '../lib/api'

  let { id }: { id: number } = $props()

  type Tab = 'members' | 'invites' | 'settings'
  let tab = $state<Tab>('members')
  let guild = $state<Guild | null | undefined>(undefined)
  let members = $state<GuildMember[]>([])
  let error = $state('')
  let busy = $state(false)

  let renaming = $state(false)
  let newName = $state('')
  let confirmRemove = $state<number | null>(null)
  let confirmLeave = $state(false)
  let confirmArchive = $state(false)
  // The member being reported (or 0 for the guild itself), or null.
  let reporting = $state<number | null>(null)

  function show(err: unknown) {
    error = err instanceof ApiError ? err.message : String(err)
  }

  async function load() {
    try {
      guild = await guilds.get(id)
      members = await guilds.members(id)
      error = ''
    } catch (err) {
      if (err instanceof ApiError && (err.status === 403 || err.status === 404)) guild = null
      else show(err)
    }
  }

  $effect(() => {
    id // reload when the route changes
    load()
  })

  // run performs one change, shows its error, and reloads.
  async function run(f: () => Promise<unknown>) {
    busy = true
    try {
      await f()
      error = ''
      await load()
    } catch (err) {
      show(err)
    } finally {
      busy = false
      confirmRemove = null
      confirmLeave = false
      confirmArchive = false
    }
  }

  function startRename() {
    newName = guild?.name ?? ''
    renaming = true
  }

  async function rename(event: SubmitEvent) {
    event.preventDefault()
    await run(() => guilds.rename(id, newName))
    renaming = false
  }

  async function leave() {
    busy = true
    try {
      await guilds.leave(id)
      router.navigate('/admin')
    } catch (err) {
      show(err)
      confirmLeave = false
    } finally {
      busy = false
    }
  }
</script>

<svelte:head>
  <title>{guild ? `${guild.name} — The Bakery` : 'Guild — The Bakery'}</title>
</svelte:head>

<p class="back"><a href="/admin">← Your guilds</a></p>

{#if guild === undefined}
  <p class="dim">Counting heads…</p>
{:else if guild === null}
  <Panel title="No such guild">
    <p>This guild does not exist, or you are not one of its members.</p>
  </Panel>
{:else}
  <header>
    {#if renaming}
      <form class="rename" onsubmit={rename}>
        <TextField aria-label="Guild name" required maxlength={60} bind:value={newName} />
        <Button type="submit" variant="confirm" disabled={busy}>Save</Button>
        <Button type="button" onclick={() => (renaming = false)}>Cancel</Button>
      </form>
    {:else}
      <h1>{guild.name}</h1>
      {#if !guild.archived}<button class="link" onclick={startRename}>Rename</button>{/if}
    {/if}
    {#if guild.archived}<span class="tag">archived: read-only</span>{/if}
  </header>

  {#if error}<p class="error" role="alert">{error}</p>{/if}

  <nav class="tabs" aria-label="Guild">
    {#each [['members', 'Members'], ['invites', 'Invites'], ['settings', 'Settings']] as [key, label] (key)}
      <button class={['tab', { active: tab === key }]} onclick={() => (tab = key as Tab)}>{label}</button>
    {/each}
  </nav>

  {#if tab === 'members'}
    <Panel title={`Members · ${members.length}`}>
      <table>
        <thead><tr><th>Name</th><th>Joined</th><th></th></tr></thead>
        <tbody>
          {#each members as m (m.id)}
            <tr>
              <td>{m.display_name}{#if m.id === session.member?.id} <span class="dim">(you)</span>{/if}</td>
              <td class="dim">{joinedOn(m.joined_at)}</td>
              <td class="actions">
                {#if !guild.archived}
                  {#if m.id === session.member?.id}
                    <button class="link danger" onclick={() => (confirmLeave = true)}>Leave guild</button>
                  {:else}
                    <button class="link danger" onclick={() => (confirmRemove = m.id)}>Remove</button>
                  {/if}
                {/if}
                {#if m.id !== session.member?.id}
                  <button class="link" onclick={() => (reporting = m.id)}>Report</button>
                {/if}
              </td>
            </tr>
            {#if confirmRemove === m.id}
              <tr><td colspan="3">
                <Confirm question={`Remove ${m.display_name} from ${guild.name}?`} action="Remove" {busy}
                  onconfirm={() => run(() => guilds.remove(id, m.id))} oncancel={() => (confirmRemove = null)} />
              </td></tr>
            {/if}
            {#if reporting === m.id}
              <tr><td colspan="3">
                <ReportForm targetKind="member" targetId={m.id} name={m.display_name} onclose={() => (reporting = null)} />
              </td></tr>
            {/if}
            {#if confirmLeave && m.id === session.member?.id}
              <tr><td colspan="3">
                <Confirm question={`Leave ${guild.name}?`} action="Leave" {busy} onconfirm={leave}
                  oncancel={() => (confirmLeave = false)} />
              </td></tr>
            {/if}
          {/each}
        </tbody>
      </table>
    </Panel>
  {:else if tab === 'invites'}
    <InvitesTab guildId={id} archived={guild.archived} />
  {:else}
    <Panel title="Settings">
      {#if guild.archived}
        <p>This guild is archived: hidden from lists, and nothing in it can change. Any member can bring it back.</p>
        <Button variant="confirm" onclick={() => run(() => guilds.restore(id))} disabled={busy}>Restore guild</Button>
      {:else if confirmArchive}
        <Confirm question={`Archive ${guild.name}? It disappears from lists and becomes read-only until someone restores it.`}
          action="Archive" {busy} onconfirm={() => run(() => guilds.archive(id))} oncancel={() => (confirmArchive = false)} />
      {:else}
        <p>Archive a guild nobody works in any more. Nothing is deleted, and any member can restore it.</p>
        <Button variant="danger" onclick={() => (confirmArchive = true)}>Archive guild</Button>
      {/if}
    </Panel>
    <Panel title="Report this guild">
      {#if reporting === 0}
        <ReportForm targetKind="guild" targetId={id} name={guild.name} onclose={() => (reporting = null)} />
      {:else}
        <p>If this guild breaks the rules (spam, scams, abuse), tell the operators.</p>
        <Button onclick={() => (reporting = 0)}>Report {guild.name}</Button>
      {/if}
    </Panel>
  {/if}
{/if}

<style>
  .back {
    margin: 0 0 8px;
  }

  header {
    display: flex;
    flex-wrap: wrap;
    align-items: baseline;
    gap: 6px 14px;
    margin-bottom: 12px;
  }

  h1 {
    margin: 0;
    font-family: var(--font-display);
    font-weight: 600;
    font-size: 28px;
    overflow-wrap: anywhere;
  }

  .rename {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
    align-items: center;
  }

  .tabs {
    display: flex;
    gap: 4px;
    margin-bottom: 8px;
  }

  .tab {
    font-family: var(--font-display);
    font-size: 14px;
    color: var(--text-dim);
    padding: 4px 12px;
    background: var(--panel);
    border: 1px solid var(--frame-dim);
    border-radius: var(--radius);
    cursor: pointer;
  }

  .tab.active {
    color: var(--text);
    background: var(--panel-title);
    border-color: var(--frame);
  }

  table {
    width: 100%;
    border-collapse: collapse;
  }

  th {
    text-align: left;
    font-weight: normal;
    font-size: 12px;
    color: var(--text-faint);
    padding: 0 6px 6px 0;
  }

  td {
    padding: 6px 6px 6px 0;
    border-top: 1px solid var(--frame-dim);
    overflow-wrap: anywhere;
  }

  .actions {
    text-align: right;
    white-space: nowrap;
  }

  .actions .link + .link {
    margin-left: 12px;
  }

  .link {
    font: inherit;
    padding: 0;
    color: var(--steel-bright);
    background: none;
    border: none;
    cursor: pointer;
  }

  .link.danger {
    color: var(--rust-bright);
  }

  .tag {
    padding: 0 6px;
    font-size: 12px;
    color: var(--text-dim);
    border: 1px solid var(--frame-dim);
    border-radius: var(--radius);
  }

  p {
    margin: 0 0 10px;
  }

  .dim {
    color: var(--text-dim);
  }

  .error {
    padding: 5px 8px;
    background: color-mix(in srgb, var(--rust) 35%, transparent);
    border: 1px solid var(--rust);
    border-radius: var(--radius);
  }
</style>
