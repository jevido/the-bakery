<script lang="ts">
  import { Panel, Button, TextField } from '@bakery/ui'
  import BoardView from '../components/BoardView.svelte'
  import TaskPanel from '../components/TaskPanel.svelte'
  import ConflictDialog from '../components/ConflictDialog.svelte'
  import AgentsScreen from '../components/AgentsScreen.svelte'
  import WorkTab from '../components/WorkTab.svelte'
  import BoardSettings from '../components/BoardSettings.svelte'
  import RunPanel from '../components/RunPanel.svelte'
  import Letters from '../components/Letters.svelte'
  import { Letters as LetterBox } from '../lib/letters.svelte'
  import { AgentSync } from '../lib/agentsync.svelte'
  import { Colony } from '../lib/colony.svelte'
  import { WebsiteService, WorkshopService, messageOf, type Member } from '../lib/bindings'

  let {
    member,
    onclockout,
    onsignedout,
  }: { member: Member; onclockout: () => void; onsignedout: () => void } = $props()

  const colony = new Colony(() => onsignedout())
  colony.load()
  // Follow the open board's live events while this screen is up.
  $effect(() => colony.listen())
  // Runs asking for permission or with a question.
  const letters = new LetterBox()
  $effect(() => letters.listen())

  // Agent folders sync in the background; the sidebar shows how it goes.
  const agentSync = new AgentSync()
  $effect(() => agentSync.listen())
  let showConflicts = $state(false)

  // The main area shows the board or the roster.
  let view = $state<'board' | 'agents' | 'work'>('board')

  const SYNC_LABEL: Record<string, string> = {
    idle: 'Agents in sync',
    syncing: 'Syncing agents…',
    offline: 'Offline: agent edits wait',
    error: 'Agent sync failed',
    'signed-out': 'Agents not synced',
  }

  let openColumn = $derived(
    colony.view?.columns.find((c) => c.id === colony.task.task?.column_id)?.name ?? '',
  )

  let newBoard = $state('')

  async function openAdmin() {
    try {
      await WebsiteService.OpenAdmin()
    } catch (err) {
      colony.error = messageOf(err)
    }
  }

  // Puts an agent to work on the open task; says why not, or "".
  async function assign(agentSlug: string): Promise<string> {
    if (colony.boardId === null || colony.openTaskId === null) return 'Open a task first.'
    const { run, error } = await colony.workshop.start(colony.boardId, colony.openTaskId, agentSlug)
    if (error || !run) return error ?? 'The run did not start.'
    colony.openRun(run.id)
    return ''
  }

  // Has an agent propose subtasks for the open task.
  async function plan(agentSlug: string) {
    if (colony.boardId === null || colony.openTaskId === null) return { error: 'Open a task first.' }
    try {
      const proposal = await WorkshopService.PlanTask(colony.boardId, colony.openTaskId, agentSlug)
      return { proposal, agentName: colony.agents.find((a) => a.slug === agentSlug)?.name }
    } catch (err) {
      return { error: messageOf(err) }
    }
  }

  async function addBoard(event: SubmitEvent) {
    event.preventDefault()
    const name = newBoard.trim()
    if (name && (await colony.createBoard(name))) newBoard = ''
  }
</script>

<div class="colony">
  <aside>
    <nav class="views" aria-label="Screens">
      <button class={['view', { active: view === 'board' }]} onclick={() => (view = 'board')}>Boards</button>
      <button class={['view', { active: view === 'agents' }]} onclick={() => (view = 'agents')}>Agents</button>
      <button class={['view', { active: view === 'work' }]} onclick={() => (view = 'work')}>Work</button>
    </nav>

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

    <div class="sync">
      {#if agentSync.status.conflicts > 0}
        <button class="sync-conflicts" onclick={() => (showConflicts = true)}>
          {agentSync.status.conflicts === 1 ? '1 agent' : `${agentSync.status.conflicts} agents`} changed in two places
        </button>
      {:else}
        <span
          class={['sync-state', agentSync.status.state]}
          title={agentSync.status.error || agentSync.status.problems.join('\n') || ''}
        >
          {SYNC_LABEL[agentSync.status.state] ?? agentSync.status.state}{agentSync.status.problems.length
            ? ` · ${agentSync.status.problems.length} not sent`
            : ''}
        </span>
      {/if}
    </div>

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
        <p>Guilds are founded and joined on the website. Found one there, or ask a guild member for an invite.</p>
        <Button variant="confirm" onclick={openAdmin}>Found a guild on the website</Button>
        <p class="dim small">Once you are in a guild, it shows up here. {#if colony.loaded}<button class="link" onclick={() => colony.load()}>Check again</button>{/if}</p>
      </Panel>
    {:else if view === 'agents'}
      <AgentsScreen sync={agentSync} guilds={colony.guilds} guildId={colony.guildId} />
    {:else if view === 'work'}
      <WorkTab {colony} sync={agentSync} />
    {:else if colony.view}
      {@const openRun = colony.openRunId ? colony.workshop.byId(colony.openRunId) : undefined}
      <div class={['work', { 'with-panel': colony.openTaskId !== null || colony.settingsOpen || openRun }]}>
        <BoardView {colony} {letters} />
        {#if colony.settingsOpen}
          <BoardSettings {colony} onclose={() => (colony.settingsOpen = false)} />
        {:else if openRun}
          <RunPanel
            run={openRun}
            onback={() => colony.openTask(openRun.task_id)}
            onclose={() => {
              colony.closeRun()
              colony.closeTask()
            }}
          />
        {:else if colony.openTaskId !== null}
          <TaskPanel
            open={colony.task}
            me={member.id}
            column={openColumn}
            workTypes={colony.workTypes}
            workshop={colony.workshop}
            linked={!!colony.settings?.linked}
            onassign={assign}
            onplan={plan}
            onopenrun={(r) => colony.openRun(r.id)}
            onclose={() => colony.closeTask()}
          />
        {/if}
      </div>
    {:else if colony.guild}
      <Panel title={colony.guild.name}>
        <p class="dim">No boards yet. Add one on the left.</p>
      </Panel>
    {/if}
  </main>
</div>

{#if showConflicts}
  <ConflictDialog sync={agentSync} onclose={() => (showConflicts = false)} />
{/if}

<Letters {letters} {colony} />

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

  .views {
    display: flex;
    gap: 4px;
  }

  .view {
    flex: 1;
    font: inherit;
    font-family: var(--font-display);
    padding: 5px 8px;
    color: var(--text-dim);
    background: var(--panel);
    border: 1px solid var(--frame-dim);
    border-radius: var(--radius);
    cursor: pointer;
  }

  .view.active {
    color: var(--text);
    background: var(--panel-title);
    border-color: var(--frame);
  }

  .work {
    display: grid;
    grid-template-columns: 1fr;
    gap: var(--gap);
    flex: 1;
    min-height: 0;
  }

  .work.with-panel {
    grid-template-columns: 1fr minmax(300px, 380px);
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

  .sync {
    margin-top: auto;
    font-size: 12px;
  }

  .sync-state {
    color: var(--text-faint);
  }

  .sync-state.syncing {
    color: var(--text-dim);
  }

  .sync-state.offline,
  .sync-state.error {
    color: var(--rust-bright);
  }

  .sync-conflicts {
    font: inherit;
    padding: 2px 6px;
    color: var(--text);
    background: color-mix(in srgb, var(--rust) 35%, transparent);
    border: 1px solid var(--rust);
    border-radius: var(--radius);
    cursor: pointer;
  }

  .who {
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

  .small {
    margin-top: 10px;
    font-size: 12px;
  }

  .link {
    font: inherit;
    padding: 0;
    color: var(--steel-bright);
    background: none;
    border: none;
    cursor: pointer;
  }


  .error {
    margin: 0;
    padding: 5px 8px;
    background: color-mix(in srgb, var(--rust) 35%, transparent);
    border: 1px solid var(--rust);
    border-radius: var(--radius);
  }
</style>
