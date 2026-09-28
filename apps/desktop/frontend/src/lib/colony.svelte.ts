// The colony screen's state: which guild and board are open, and the open
// board's columns and tasks, kept in step with the board's live events.
// Talks to the API only through BoardsService and LiveService.

import { Events } from '@wailsio/runtime'
import { AgentsService, BoardsService, LiveService, WorkshopService, isSignedOut, messageOf, type AgentSummary, type Board, type BoardSettings, type BoardView, type Guild, type Task, type WorkType } from './bindings'
import { OpenTask } from './task.svelte'
import { Workshop } from './workshop.svelte'

export type ColumnState = { id: number; name: string; tasks: Task[] }

// The open board, with Go's nil slices turned into empty arrays.
export type BoardState = {
  board: Board
  columns: ColumnState[]
}

function normalise(view: BoardView): BoardState {
  return {
    board: view.board,
    columns: (view.columns ?? []).map((c) => ({ id: c.id, name: c.name, tasks: c.tasks ?? [] })),
  }
}

// A board event as the API's stream sends it (see
// docs/domain/contexts/boards/README.md), passed on by LiveService.
export type BoardEvent = {
  type: string
  board_id: number
  actor_id: number
  data: Record<string, unknown>
}

// The board stream's state, for the header's indicator.
export type LiveState = 'off' | 'connecting' | 'live' | 'reconnecting' | 'signed-out'

const LAST_GUILD = 'bakery.lastGuild'
const LAST_BOARD = 'bakery.lastBoard'

// localStorage can throw (private mode, blocked storage); remembering the
// last board is a convenience, so failures are ignored.
function remember(key: string, value: number | null) {
  try {
    if (value === null) localStorage.removeItem(key)
    else localStorage.setItem(key, String(value))
  } catch {}
}

function recall(key: string): number | null {
  try {
    const v = Number(localStorage.getItem(key))
    return Number.isInteger(v) && v > 0 ? v : null
  } catch {
    return null
  }
}

export class Colony {
  guilds = $state.raw<Guild[]>([])
  guildId = $state<number | null>(null)
  boards = $state.raw<Board[]>([])
  boardId = $state<number | null>(null)
  // Deep state: drag and drop edits it in place before the API answers.
  view = $state<BoardState | null>(null)
  loaded = $state(false)
  error = $state('')
  // The task open in the side panel.
  openTaskId = $state<number | null>(null)
  task: OpenTask
  live = $state<LiveState>('off')
  // The open guild's work types, in its order.
  workTypes = $state.raw<WorkType[]>([])
  // How this machine works the open board (never sent to the API), and
  // whether its settings panel is open in place of the task panel.
  settings = $state.raw<BoardSettings | null>(null)
  settingsOpen = $state(false)
  // This machine's runs, and which tasks have one going anywhere.
  workshop = new Workshop()
  // The member's own agents, to name and draw them on cards and menus.
  agents = $state.raw<AgentSummary[]>([])
  // The run shown in the side panel in place of the task, by its id here.
  openRunId = $state<string | null>(null)
  // Open streams on the board, by stream id: who has it open right now.
  #streams = $state<Record<string, { id: number; name: string; seed: string }>>({})
  // One entry per member, however many windows they have open.
  present = $derived.by(() => {
    const seen = new Map<number, { id: number; name: string; seed: string }>()
    for (const m of Object.values(this.#streams)) seen.set(m.id, m)
    return [...seen.values()]
  })
  #reloadTimer: ReturnType<typeof setTimeout> | undefined

  guild = $derived(this.guilds.find((g) => g.id === this.guildId) ?? null)

  #onSignedOut: () => void

  constructor(onSignedOut: () => void) {
    this.#onSignedOut = onSignedOut
    this.task = new OpenTask(
      () => this.reloadBoard(),
      () => this.#onSignedOut(),
    )
  }

  // listen follows the open board's live events until the returned function
  // is called (when the colony screen goes away).
  listen(): () => void {
    const offs = [
      this.workshop.listen(),
      Events.On('board:event', (e) => this.apply(e.data as BoardEvent)),
      Events.On('board:status', (e) => {
        const s = e.data as { board_id: number; state: LiveState }
        if (s.board_id !== this.boardId) return
        this.live = s.state
        if (s.state === 'signed-out') this.#onSignedOut()
      }),
      Events.On('board:resync', (e) => {
        if ((e.data as { board_id: number }).board_id === this.boardId) this.#resync()
      }),
    ]
    return () => {
      offs.forEach((off) => off())
      LiveService.Unwatch()
      this.live = 'off'
    }
  }

  #resync() {
    this.reloadBoard()
    this.task.reload()
  }

  // reloadSoon refetches the board once a burst of events has passed.
  #reloadSoon() {
    clearTimeout(this.#reloadTimer)
    this.#reloadTimer = setTimeout(() => this.reloadBoard(), 150)
  }

  // apply puts one live event on the board. Moves, deletes and renames are
  // applied in place; anything that needs data the event does not carry
  // refetches the board. Events for changes this app made itself find the
  // board already changed and do nothing.
  apply(ev: BoardEvent) {
    const view = this.view
    if (!view || ev.board_id !== this.boardId) return
    const taskId = Number(ev.data.task_id ?? 0)
    if (taskId && taskId === this.openTaskId) {
      if (ev.type === 'task.deleted') this.closeTask()
      else this.task.reload()
    }
    switch (ev.type) {
      case 'presence': {
        const d = ev.data
        if (d.state === 'snapshot') {
          const streams: Record<string, { id: number; name: string; seed: string }> = {}
          for (const p of (d.present as { conn_id: string; member_id: number; display_name: string; portrait_seed?: string }[]) ?? []) {
            streams[p.conn_id] = { id: p.member_id, name: p.display_name, seed: p.portrait_seed || `member-${p.member_id}` }
          }
          this.#streams = streams
        } else if (d.state === 'joined') {
          this.#streams[String(d.conn_id)] = {
            id: Number(d.member_id),
            name: String(d.display_name ?? ''),
            seed: String(d.portrait_seed || `member-${d.member_id}`),
          }
        } else if (d.state === 'left') {
          delete this.#streams[String(d.conn_id)]
        }
        return
      }
      case 'task.moved': {
        const target = view.columns.find((c) => c.id === Number(ev.data.to))
        const from = view.columns.find((c) => c.tasks.some((t) => t.id === taskId))
        const task = from?.tasks.find((t) => t.id === taskId)
        const position = String(ev.data.position)
        if (!target || !from || !task) return this.#reloadSoon()
        if (from === target && task.position === position) return
        from.tasks.splice(from.tasks.indexOf(task), 1)
        task.column_id = target.id
        task.position = position
        // Positions sort as plain strings (the keys are ASCII).
        const at = target.tasks.findIndex((t) => t.position > position)
        target.tasks.splice(at < 0 ? target.tasks.length : at, 0, task)
        return
      }
      case 'task.deleted':
        for (const col of view.columns) {
          const i = col.tasks.findIndex((t) => t.id === taskId)
          if (i >= 0) col.tasks.splice(i, 1)
        }
        return
      case 'column.updated': {
        const col = view.columns.find((c) => c.id === Number(ev.data.column_id))
        if (col) col.name = String(ev.data.name)
        else this.#reloadSoon()
        return
      }
      case 'column.deleted': {
        const i = view.columns.findIndex((c) => c.id === Number(ev.data.column_id))
        if (i >= 0 && view.columns[i].tasks.length === 0) view.columns.splice(i, 1)
        else if (i >= 0) this.#reloadSoon()
        return
      }
      case 'task.claimed':
      case 'task.released': {
        const task = view.columns.flatMap((c) => c.tasks).find((t) => t.id === taskId)
        if (!task) return this.#reloadSoon()
        task.claim =
          ev.type === 'task.claimed'
            ? {
                id: Number(ev.data.claim_id),
                agent_id: Number(ev.data.agent_id),
                member_id: Number(ev.data.member_id),
                machine_id: '',
                expires_at: String(ev.data.expires_at),
              }
            : null
        return
      }
      case 'run.started':
      case 'run.finished':
        // The card's working mark follows; an open task panel reloaded above.
        this.workshop.onRunEvent(ev.type, ev.data)
        return
      default:
        // task.created, task.updated, column.created, column.moved, and
        // types this version does not know.
        this.#reloadSoon()
    }
  }

  openTask(id: number) {
    this.settingsOpen = false
    this.openRunId = null
    this.openTaskId = id
    this.task.open(id)
  }

  closeTask() {
    this.openTaskId = null
    this.task.close()
  }

  // run shows failures as the colony's error line; a lost session goes back
  // to the login screen.
  async #run<T>(f: () => Promise<T>): Promise<T | undefined> {
    try {
      const v = await f()
      this.error = ''
      return v
    } catch (err) {
      if (isSignedOut(err)) this.#onSignedOut()
      else this.error = messageOf(err)
      return undefined
    }
  }

  async load() {
    const guilds = await this.#run(() => BoardsService.ListGuilds())
    this.guilds = guilds ?? []
    const last = recall(LAST_GUILD)
    const pick = this.guilds.find((g) => g.id === last) ?? this.guilds[0]
    if (pick) await this.openGuild(pick.id)
    this.loaded = true
  }

  async openGuild(id: number) {
    this.guildId = id
    remember(LAST_GUILD, id)
    this.reloadWorkTypes()
    this.boards = (await this.#run(() => BoardsService.ListBoards(id))) ?? []
    const last = recall(LAST_BOARD)
    const pick = this.boards.find((b) => b.id === last) ?? this.boards[0]
    if (pick) await this.openBoard(pick.id)
    else {
      this.closeTask()
      this.boardId = null
      this.view = null
      LiveService.Unwatch()
      this.live = 'off'
    }
  }

  async reloadWorkTypes() {
    if (this.guildId === null) return
    const id = this.guildId
    const wts = await this.#run(() => BoardsService.WorkTypes(id))
    if (wts && this.guildId === id) this.workTypes = wts
  }

  workTypeName(key: string | null | undefined): string {
    if (!key) return ''
    return this.workTypes.find((w) => w.key === key)?.name ?? key
  }

  async openBoard(id: number) {
    if (id !== this.boardId) {
      this.closeTask()
      this.settingsOpen = false
      this.settings = null
    }
    this.boardId = id
    this.reloadSettings()
    this.reloadAgents()
    remember(LAST_BOARD, id)
    this.live = 'connecting'
    this.#streams = {}
    LiveService.Watch(id)
    await this.reloadBoard()
  }

  async reloadAgents() {
    const list = await this.#run(() => AgentsService.List())
    if (list) this.agents = list
  }

  // An agent by its API id: one of the member's, or undefined for someone
  // else's.
  agentById(id: number | null | undefined): AgentSummary | undefined {
    return id ? this.agents.find((a) => a.agent_id === id) : undefined
  }

  async setSpeed(speed: 'paused' | 'normal' | 'fast') {
    if (this.boardId === null) return
    const id = this.boardId
    const s = await this.#run(() => WorkshopService.SetSpeed(id, speed))
    if (s && this.boardId === id) this.settings = s
  }

  // Prioritize a task for an agent (0 clears it) and/or forbid it for
  // agents; the board follows through its task.updated event.
  async draft(taskId: number, prioritizedAgentId: number | null, forbidden: boolean | null) {
    const t = await this.#run(() => BoardsService.DraftTask(taskId, prioritizedAgentId, forbidden))
    if (t) this.#patchTask(t)
  }

  #patchTask(t: Task) {
    for (const col of this.view?.columns ?? []) {
      const i = col.tasks.findIndex((x) => x.id === t.id)
      if (i >= 0) col.tasks[i] = { ...col.tasks[i], prioritized_agent_id: t.prioritized_agent_id, forbidden: t.forbidden }
    }
  }

  async reloadSettings() {
    if (this.boardId === null) return
    const id = this.boardId
    const s = await this.#run(() => WorkshopService.GetBoardConfig(id))
    if (s && this.boardId === id) this.settings = s
  }

  // openRun shows a run; once seen, a failed run is no longer an alert.
  openRun(id: string) {
    this.settingsOpen = false
    this.openRunId = id
    WorkshopService.SeenRun(id).catch(() => {})
  }

  closeRun() {
    this.openRunId = null
  }

  openSettings() {
    this.openRunId = null
    this.closeTask()
    this.settingsOpen = true
  }

  async reloadBoard() {
    if (this.boardId === null) return
    const view = await this.#run(() => BoardsService.GetBoard(this.boardId!))
    if (view) this.view = normalise(view)
  }

  async createBoard(name: string): Promise<boolean> {
    if (this.guildId === null) return false
    const board = await this.#run(() => BoardsService.CreateBoard(this.guildId!, name))
    if (!board) return false
    this.boards = [...this.boards, board].sort((a, b) => a.name.localeCompare(b.name))
    await this.openBoard(board.id)
    return true
  }

  async createTask(title: string): Promise<boolean> {
    if (this.boardId === null) return false
    const task = await this.#run(() => BoardsService.CreateTask(this.boardId!, title))
    if (!task) return false
    const first = this.view?.columns[0]
    if (first && !first.tasks.some((t) => t.id === task.id)) first.tasks.push(task)
    return true
  }

  async deleteTask(id: number) {
    if (this.openTaskId === id) this.closeTask()
    for (const col of this.view?.columns ?? []) {
      const i = col.tasks.findIndex((t) => t.id === id)
      if (i >= 0) col.tasks.splice(i, 1)
    }
    const ok = await this.#run(async () => {
      await BoardsService.DeleteTask(id)
      return true
    })
    if (!ok) await this.reloadBoard()
  }

  // addColumn puts a new column at the end of the board.
  async addColumn(name: string): Promise<boolean> {
    if (this.boardId === null) return false
    const col = await this.#run(() => BoardsService.CreateColumn(this.boardId!, name))
    if (!col) return false
    if (this.view && !this.view.columns.some((c) => c.id === col.id)) {
      this.view.columns.push({ id: col.id, name: col.name, tasks: [] })
    }
    return true
  }

  async renameColumn(id: number, name: string) {
    const col = this.view?.columns.find((c) => c.id === id)
    if (!col || col.name === name) return
    const before = col.name
    col.name = name
    const saved = await this.#run(() => BoardsService.RenameColumn(id, name))
    if (!saved) col.name = before
  }

  // moveColumn puts the column at index among the others right away, then
  // tells the API its new neighbours. If the API refuses, the board is
  // reloaded.
  async moveColumn(id: number, index: number) {
    const view = this.view
    if (!view) return
    const from = view.columns.findIndex((c) => c.id === id)
    if (from < 0) return
    const [col] = view.columns.splice(from, 1)
    index = Math.max(0, Math.min(index, view.columns.length))
    view.columns.splice(index, 0, col)
    if (index === from) return
    const afterId = view.columns[index - 1]?.id ?? null
    const beforeId = view.columns[index + 1]?.id ?? null
    const moved = await this.#run(() => BoardsService.MoveColumn(id, afterId, beforeId))
    if (!moved) await this.reloadBoard()
  }

  // deleteColumn removes an empty column; the API refuses one with tasks.
  async deleteColumn(id: number) {
    const view = this.view
    if (!view) return
    const i = view.columns.findIndex((c) => c.id === id)
    if (i < 0) return
    const [col] = view.columns.splice(i, 1)
    const ok = await this.#run(async () => {
      await BoardsService.DeleteColumn(id)
      return true
    })
    if (!ok) view.columns.splice(i, 0, col)
  }

  // moveTask moves the task to index in the column right away, then tells
  // the API its new neighbours. If the API refuses, the board is reloaded.
  async moveTask(id: number, columnId: number, index: number) {
    const view = this.view
    if (!view) return
    let task
    for (const col of view.columns) {
      const i = col.tasks.findIndex((t) => t.id === id)
      if (i >= 0) [task] = col.tasks.splice(i, 1)
    }
    const target = view.columns.find((c) => c.id === columnId)
    if (!task || !target) return this.reloadBoard()
    index = Math.max(0, Math.min(index, target.tasks.length))
    task.column_id = columnId
    target.tasks.splice(index, 0, task)

    const afterId = target.tasks[index - 1]?.id ?? null
    const beforeId = target.tasks[index + 1]?.id ?? null
    const moved = await this.#run(() => BoardsService.MoveTask(id, columnId, afterId, beforeId))
    if (!moved) await this.reloadBoard()
    else task.position = moved.position
  }
}
