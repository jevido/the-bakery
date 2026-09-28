<script lang="ts">
  import { empty } from '../lib/flavor/lines'
  import { Panel, Button, TextField, Tabs, Portrait } from '@bakery/ui'
  import CharacterCard from './CharacterCard.svelte'
  import { Roster } from '../lib/roster.svelte'
  import type { AgentSync } from '../lib/agentsync.svelte'
  import type { Agent, Guild } from '../lib/bindings'

  // The roster: my agents on the left, the open one's character card on the
  // right, and the agents my guild shares, to recruit.
  let { sync, guilds, guildId }: { sync: AgentSync; guilds: Guild[]; guildId: number | null } = $props()

  const roster = new Roster()
  roster.load()
  // Reload whenever the sync changed the folders.
  $effect(() => {
    if (sync.changes > 0) roster.load()
  })

  const TABS = [
    { id: 'roster', label: 'Roster' },
    { id: 'recruit', label: 'Recruit' },
  ]
  let tab = $state('roster')

  let newName = $state('')
  let newTitle = $state('')
  let creating = $state(false)

  async function create(event: SubmitEvent) {
    event.preventDefault()
    if (!newName.trim()) return
    if (await roster.create(newName.trim(), newTitle.trim())) {
      newName = ''
      newTitle = ''
      creating = false
    }
  }

  let shared = $state.raw<Agent[] | null>(null)
  let recruited = $state<number[]>([])
  let guild = $derived(guilds.find((g) => g.id === guildId) ?? null)

  async function loadShared() {
    shared = guildId === null ? [] : await roster.guildAgents(guildId)
  }

  $effect(() => {
    if (tab === 'recruit') loadShared()
  })

  async function recruit(a: Agent) {
    if (guildId !== null && (await roster.recruit(a.id, guildId))) recruited = [...recruited, a.id]
  }
</script>

<div class="agents">
  <Tabs label="Agents" tabs={TABS} bind:active={tab}>
    {#snippet children(active)}
      {#if active === 'roster'}
        <div class="split">
          <Panel title={`Colonists · ${roster.agents.length}`}>
            {#if roster.loaded && roster.agents.length === 0}
              <p class="dim">{empty('roster')}</p>
            {/if}
            <ul class="list">
              {#each roster.agents as a (a.slug)}
                <li>
                  <button class={['item', { active: a.slug === roster.selected }]} onclick={() => roster.open(a.slug)}>
                    <Portrait seed={a.portrait_seed || a.slug} size={32} />
                    <span class="who">
                      <span class="name">{a.name || a.slug}</span>
                      <span class="title">{a.title}</span>
                    </span>
                    {#if a.conflict}<span class="dot warn" title="Changed in two places"></span>
                    {:else if !a.synced}<span class="dot" title="Not synced yet"></span>{/if}
                  </button>
                </li>
              {/each}
            </ul>
            {#if creating}
              <form class="new" onsubmit={create}>
                <TextField placeholder="Name" maxlength={40} bind:value={newName} />
                <TextField placeholder="Title (optional)" maxlength={60} bind:value={newTitle} />
                <div class="row">
                  <Button type="submit" variant="confirm" disabled={!newName.trim()}>Create</Button>
                  <Button onclick={() => (creating = false)}>Cancel</Button>
                </div>
              </form>
            {:else}
              <Button onclick={() => (creating = true)}>+ New agent</Button>
            {/if}
          </Panel>
          <div class="card">
            {#if roster.detail}
              <CharacterCard {roster} {guilds} />
            {:else if roster.loaded}
              <Panel><p class="dim">Pick an agent, or make a new one.</p></Panel>
            {/if}
          </div>
        </div>
      {:else}
        <Panel title={guild ? `Shared with ${guild.name}` : 'Shared agents'}>
          {#if guildId === null}
            <p class="dim">Open a guild first.</p>
          {:else if shared === null}
            <p class="dim">Asking around…</p>
          {:else if shared.length === 0}
            <p class="dim">Nobody in {guild?.name} has shared an agent yet.</p>
          {:else}
            <ul class="shared">
              {#each shared as a (a.id)}
                <li>
                  <Portrait seed={a.portrait_seed || a.slug} size={40} />
                  <div class="who">
                    <span class="name">{a.name} <span class="dim">— from {a.owner_name}</span></span>
                    <span class="title">{a.title}</span>
                    <span class="dim small">
                      {(a.traits ?? []).join(', ') || 'no traits'} · {a.model} ·
                      {(a.skills ?? []).map((s) => s.name).join(', ') || 'no skills'}
                    </span>
                  </div>
                  {#if recruited.includes(a.id)}
                    <span class="dim">Recruited</span>
                  {:else}
                    <Button onclick={() => recruit(a)}>Recruit</Button>
                  {/if}
                </li>
              {/each}
            </ul>
          {/if}
        </Panel>
      {/if}
    {/snippet}
  </Tabs>
</div>

<style>
  .agents {
    display: flex;
    flex-direction: column;
    min-height: 0;
    height: 100%;
  }

  .agents :global(.tabs) {
    min-height: 0;
    flex: 1;
  }

  .split {
    display: grid;
    grid-template-columns: 260px 1fr;
    gap: var(--gap);
    align-items: start;
  }

  .card {
    min-width: 0;
  }

  .list,
  .shared {
    list-style: none;
    margin: 0 0 8px;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .item {
    width: 100%;
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 4px 6px;
    font: inherit;
    text-align: left;
    color: var(--text-dim);
    background: none;
    border: 1px solid transparent;
    border-radius: var(--radius);
    cursor: pointer;
  }

  .item:hover {
    color: var(--text);
    background: var(--panel-raised);
  }

  .item.active {
    color: var(--text);
    background: var(--panel-title);
    border-color: var(--frame-dim);
  }

  .who {
    display: flex;
    flex-direction: column;
    min-width: 0;
    flex: 1;
  }

  .name {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .title {
    font-size: 12px;
    color: var(--text-faint);
  }

  .dot {
    width: 7px;
    height: 7px;
    border-radius: 50%;
    background: var(--steel);
  }

  .dot.warn {
    background: var(--rust-bright);
  }

  .shared li {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 6px 0;
    border-bottom: 1px solid var(--frame-dim);
  }

  .shared li:last-child {
    border-bottom: none;
  }

  .new {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .row {
    display: flex;
    gap: 6px;
  }

  p {
    margin: 0;
  }

  .dim {
    color: var(--text-dim);
  }

  .small {
    font-size: 12px;
  }
</style>
