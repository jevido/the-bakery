// The colony screen's state: which guild and board are open, and the open
// board's tasks. Talks to the API only through BoardsService.

import { BoardsService, isSignedOut, messageOf, type Board, type BoardView, type Guild, type Task } from './bindings'

// The open board, with Go's nil slices turned into empty arrays.
export type BoardState = {
  board: Board
  columns: { column: string; tasks: Task[] }[]
}

function normalise(view: BoardView): BoardState {
  return {
    board: view.board,
    columns: (view.columns ?? []).map((c) => ({ column: c.column, tasks: c.tasks ?? [] })),
  }
}

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

export const COLUMN_TITLES: Record<string, string> = {
  backlog: 'Backlog',
  todo: 'To do',
  doing: 'Doing',
  done: 'Done',
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

  guild = $derived(this.guilds.find((g) => g.id === this.guildId) ?? null)

  #onSignedOut: () => void

  constructor(onSignedOut: () => void) {
    this.#onSignedOut = onSignedOut
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
    this.boards = (await this.#run(() => BoardsService.ListBoards(id))) ?? []
    const last = recall(LAST_BOARD)
    const pick = this.boards.find((b) => b.id === last) ?? this.boards[0]
    if (pick) await this.openBoard(pick.id)
    else {
      this.boardId = null
      this.view = null
    }
  }

  async openBoard(id: number) {
    this.boardId = id
    remember(LAST_BOARD, id)
    await this.reloadBoard()
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
    this.view?.columns.find((c) => c.column === 'backlog')?.tasks.push(task)
    return true
  }

  async renameTask(id: number, title: string) {
    const task = await this.#run(() => BoardsService.UpdateTask(id, title, null))
    if (!task) return this.reloadBoard()
    for (const col of this.view?.columns ?? []) {
      const t = col.tasks.find((t) => t.id === id)
      if (t) t.title = task.title
    }
  }

  async deleteTask(id: number) {
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

  // moveTask moves the task to index in column right away, then tells the
  // API its new neighbours. If the API refuses, the board is reloaded.
  async moveTask(id: number, column: string, index: number) {
    const view = this.view
    if (!view) return
    let task
    for (const col of view.columns) {
      const i = col.tasks.findIndex((t) => t.id === id)
      if (i >= 0) [task] = col.tasks.splice(i, 1)
    }
    const target = view.columns.find((c) => c.column === column)
    if (!task || !target) return this.reloadBoard()
    index = Math.max(0, Math.min(index, target.tasks.length))
    task.column = column
    target.tasks.splice(index, 0, task)

    const afterId = target.tasks[index - 1]?.id ?? null
    const beforeId = target.tasks[index + 1]?.id ?? null
    const moved = await this.#run(() => BoardsService.MoveTask(id, column, afterId, beforeId))
    if (!moved) await this.reloadBoard()
    else task.position = moved.position
  }
}
