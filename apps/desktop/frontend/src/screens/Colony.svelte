<script lang="ts">
  import { Panel, Button, TextField } from '@bakery/ui'
  import BoardView from '../components/BoardView.svelte'
  import { Colony } from '../lib/colony.svelte'
  import type { Member } from '../lib/bindings'

  let {
    member,
    onclockout,
    onsignedout,
  }: { member: Member; onclockout: () => void; onsignedout: () => void } = $props()

  const colony = new Colony(() => onsignedout())
  colony.load()

  let newBoard = $state('')

  async function addBoard(event: SubmitEvent) {
    event.preventDefault()
    const name = newBoard.trim()
    if (name && (await colony.createBoard(name))) newBoard = ''
  }
</script>

<div class="colony">
  <aside>
    <Panel title="Guilds">
      {#if colony.guilds.length === 0 && colony.loaded}
        <p class="dim">No guild yet.</p>
      {/if}
      <ul class="list">
        {#each colony.guilds as g (g.id)}
          <li>
            <button class={['item', { active: g.id === colony.guildId }]} onclick={() => colony.openGuild(g.id)}>
              {g.name}
            </button>
          </li>
        {/each}
      </ul>
    </Panel>

    {#if colony.guild}
      <Panel title="Boards">
        <ul class="list">
          {#each colony.boards as b (b.id)}
            <li>
              <button class={['item', { active: b.id === colony.boardId }]} onclick={() => colony.openBoard(b.id)}>
                {b.name}
              </button>
            </li>
          {/each}
        </ul>
        <form class="new-board" onsubmit={addBoard}>
          <TextField placeholder="New board" maxlength={60} bind:value={newBoard} />
          <Button type="submit" disabled={!newBoard.trim()}>Add</Button>
        </form>
      </Panel>
    {/if}

    <div class="who">
      <span>{member.display_name}</span>
      <Button onclick={onclockout}>Clock out</Button>
    </div>
  </aside>

  <main>
    {#if colony.error}
      <p class="error" role="alert">{colony.error}</p>
    {/if}

    {#if !colony.loaded}
      <p class="dim">Waking the colonists…</p>
    {:else if colony.guilds.length === 0}
      <Panel title="No guild yet">
        <p>Guilds are founded on the website, which is coming later.</p>
        <p class="dim">
          For now, in development, seed one: <code>task api:seed</code> creates “First Colony” with
          ada@, bram@ and cas@bakery.test (password <code>password</code>). Or found one through the
          API with <code>POST /api/guilds</code>.
        </p>
      </Panel>
    {:else if colony.view}
      <BoardView {colony} />
    {:else if colony.guild}
      <Panel title={colony.guild.name}>
        <p class="dim">No boards yet. Add one on the left.</p>
      </Panel>
    {/if}
  </main>
</div>

<style>
  .colony {
    display: grid;
    grid-template-columns: 200px 1fr;
    gap: var(--gap);
    height: 100%;
    padding: var(--gap);
  }

  aside {
    display: flex;
    flex-direction: column;
    gap: var(--gap);
    min-height: 0;
  }

  main {
    display: flex;
    flex-direction: column;
    gap: var(--gap);
    min-width: 0;
    min-height: 0;
  }

  .list {
    list-style: none;
    margin: 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .item {
    width: 100%;
    text-align: left;
    font: inherit;
    color: var(--text-dim);
    padding: 4px 8px;
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

  .new-board {
    display: flex;
    gap: 6px;
    margin-top: var(--gap);
  }

  .new-board :global(.field) {
    flex: 1;
  }

  .who {
    margin-top: auto;
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 4px 2px;
    color: var(--text-dim);
  }

  p {
    margin: 0 0 6px;
  }

  .dim {
    color: var(--text-dim);
  }

  code {
    font-size: 12px;
    padding: 0 3px;
    background: var(--panel-inset);
    border-radius: var(--radius);
    user-select: text;
  }

  .error {
    margin: 0;
    padding: 5px 8px;
    background: color-mix(in srgb, var(--rust) 35%, transparent);
    border: 1px solid var(--rust);
    border-radius: var(--radius);
  }
</style>
