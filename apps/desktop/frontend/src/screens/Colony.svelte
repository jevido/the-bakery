<script lang="ts">
  import { empty } from '../lib/flavor/lines'
  import { Panel, Button, TextField, Portrait } from '@bakery/ui'
  import BoardView from '../components/BoardView.svelte'
  import TaskPanel from '../components/TaskPanel.svelte'
  import ConflictDialog from '../components/ConflictDialog.svelte'
  import AgentsScreen from '../components/AgentsScreen.svelte'
  import WorkTab from '../components/WorkTab.svelte'
  import BoardSettings from '../components/BoardSettings.svelte'
  import RunPanel from '../components/RunPanel.svelte'
  import Letters from '../components/Letters.svelte'
  import ReportGuild from '../components/ReportGuild.svelte'
  import AlertsColumn from '../components/AlertsColumn.svelte'
  import SettingsScreen from '../components/SettingsScreen.svelte'
  import { loadSettings } from '../lib/settings.svelte'
  import { Alerts } from '../lib/alerts.svelte'
  import { Story } from '../lib/story.svelte'
  import { Letters as LetterBox } from '../lib/letters.svelte'
  import { AgentSync } from '../lib/agentsync.svelte'
  import { Colony } from '../lib/colony.svelte'
  import { needs } from '../lib/needs.svelte'
  import { SessionService, WebsiteService, WorkshopService, messageOf, type Alert, type Member } from '../lib/bindings'

  let {
    member,
    onclockout,
    onsignedout,
  }: { member: Member; onclockout: () => void; onsignedout: () => void } = $props()

  // The member's own portrait seed; a re-roll replaces it here.
  let seed = $derived(member.portrait_seed)
  let rerolling = $state(false)

  async function reroll() {
    rerolling = true
    try {
      seed = (await SessionService.RerollPortrait()).portrait_seed
    } catch (err) {
      colony.error = messageOf(err)
    }
    rerolling = false
  }

  const colony = new Colony(() => onsignedout())
  colony.load()
  // Follow the open board's live events while this screen is up.
  $effect(() => colony.listen())
  // Every agent's needs and mood, for the agent card and the colony view.
  $effect(() => needs.listen())
  // What needs attention, at the right edge.
  const alerts = new Alerts()
  $effect(() => alerts.listen())
  // The storyteller's event letters for the open board.
  const story = new Story()
  $effect(() => story.listen())
  $effect(() => {
    story.load(colony.boardId)
  })

  // goToAlert opens what an alert is about.
  async function goToAlert(a: Alert) {
    if (a.guild_id && a.guild_id !== colony.guildId) await colony.openGuild(a.guild_id)
    if (a.kind === 'uncovered_work') {
      if (a.board_id !== colony.boardId) await colony.openBoard(a.board_id)
      view = 'work'
      return
    }
    view = 'board'
    if (a.board_id && a.board_id !== colony.boardId) await colony.openBoard(a.board_id)
    if (a.kind === 'question_waiting' && a.letter) letters.openId = a.letter
    else if (a.kind === 'run_failed' && a.run_id) colony.openRun(a.run_id)
    else if (a.task_id) colony.openTask(a.task_id)
  }
  // The guild list's right-click menu, and the guild being reported.
  let guildMenu = $state<{ guild: { id: number; name: string }; x: number; y: number } | null>(null)
  let reporting = $state<{ id: number; name: string } | null>(null)

  // Runs asking for permission or with a question.
  const letters = new LetterBox()
  $effect(() => letters.listen())

  // Agent folders sync in the background; the sidebar shows how it goes.
  const agentSync = new AgentSync()
  $effect(() => agentSync.listen())
  let showConflicts = $state(false)

  // The main area shows the board or the roster.
  let view = $state<'board' | 'agents' | 'work' | 'settings'>('board')
  loadSettings()

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
            <button
              class={['item', { active: g.id === colony.guildId }]}
              title="Right-click to report this guild"
              onclick={() => colony.openGuild(g.id)}
              oncontextmenu={(e) => {
                e.preventDefault()
                guildMenu = { guild: g, x: e.clientX, y: e.clientY }
              }}
            >
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
      <button class="face" title="Re-roll portrait" aria-label="Re-roll portrait" disabled={rerolling} onclick={reroll}>
        <Portrait {seed} size={28} />
      </button>
      <span>{member.display_name}</span>
      <Button onclick={onclockout}>Clock out</Button>
      <button class={['settings', { active: view === 'settings' }]} title="Settings" aria-label="Settings" onclick={() => (view = 'settings')}>⚙</button>
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
    {:else if view === 'settings'}
      <SettingsScreen />
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
        <p class="dim">{empty('board', colony.guild.id)}</p>
      </Panel>
    {/if}
  </main>

  <AlertsColumn {alerts} ongo={goToAlert} />
</div>

{#if showConflicts}
  <ConflictDialog sync={agentSync} onclose={() => (showConflicts = false)} />
{/if}

<Letters {letters} {colony} {story} />

{#if guildMenu}
  <!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
  <div class="menu-veil" onclick={() => (guildMenu = null)}></div>
  <div class="guild-menu" role="menu" style:left="{guildMenu.x}px" style:top="{guildMenu.y}px">
    <button
      role="menuitem"
      onclick={() => {
        reporting = guildMenu!.guild
        guildMenu = null
      }}>Report this guild…</button
    >
  </div>
{/if}
{#if reporting}
  <ReportGuild guild={reporting} onclose={() => (reporting = null)} />
{/if}

<style>
  .colony {
    display: grid;
    grid-template-columns: 200px 1fr auto;
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

  .settings {
    font: inherit;
    padding: 2px 7px;
    color: var(--text-dim);
    background: var(--panel);
    border: 1px solid var(--frame-dim);
    border-radius: var(--radius);
    cursor: pointer;
  }

  .settings.active {
    color: var(--text);
    border-color: var(--frame);
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
    gap: 6px;
    padding: 4px 2px;
    color: var(--text-dim);
  }

  .who > span {
    flex: 1;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
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

  .menu-veil {
    position: fixed;
    inset: 0;
    z-index: 20;
  }

  .guild-menu {
    position: fixed;
    z-index: 21;
    display: flex;
    flex-direction: column;
    padding: 4px;
    background: var(--panel-raised);
    border: 1px solid var(--frame);
    border-radius: var(--radius);
    box-shadow: 0 4px 12px rgb(0 0 0 / 0.4);
  }

  .guild-menu button {
    font: inherit;
    font-size: 13px;
    text-align: left;
    padding: 4px 10px;
    color: var(--text);
    background: none;
    border: none;
    border-radius: var(--radius);
    cursor: pointer;
  }

  .guild-menu button:hover {
    background: var(--panel-inset);
  }

  .face {
    padding: 0;
    line-height: 0;
    background: none;
    border: none;
    cursor: pointer;
  }

  .face:disabled {
    opacity: 0.6;
  }
</style>
