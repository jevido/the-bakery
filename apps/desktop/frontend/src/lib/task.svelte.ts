// The task panel's state: the open task and its subtasks. Talks to the API
// only through TaskService. After every change it tells the board, so a
// card's subtask badge and title follow.

import { TaskService, isSignedOut, messageOf, type Activity, type Comment, type Run, type Task } from './bindings'

const ACTIVITY_PAGE = 50

export class OpenTask {
  task = $state<Task | null>(null)
  // Deep state: ticking and reordering edit it in place before the API answers.
  subtasks = $state<Task[]>([])
  comments = $state.raw<Comment[]>([])
  activity = $state.raw<Activity[]>([])
  // Runs of agents on the task, by every member, newest first.
  runs = $state.raw<Run[]>([])
  // Whether older activity is left to load.
  moreActivity = $state(false)
  error = $state('')

  #onChanged: () => void
  #onSignedOut: () => void
  // Only the answer to the newest load counts.
  #loading = 0

  constructor(onChanged: () => void, onSignedOut: () => void) {
    this.#onChanged = onChanged
    this.#onSignedOut = onSignedOut
  }

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

  async open(id: number) {
    const n = ++this.#loading
    const [detail, comments, activity, runs] = await Promise.all([
      this.#run(() => TaskService.GetTask(id)),
      this.#run(() => TaskService.ListComments(id)),
      this.#run(() => TaskService.ListActivity(id, 0, ACTIVITY_PAGE)),
      this.#run(() => TaskService.ListRuns(id)),
    ])
    if (n !== this.#loading || !detail) return
    this.task = detail.task
    this.subtasks = detail.subtasks ?? []
    this.comments = comments ?? []
    this.activity = activity ?? []
    this.runs = runs ?? []
    this.moreActivity = this.activity.length === ACTIVITY_PAGE
  }

  async loadMoreActivity() {
    const last = this.activity[this.activity.length - 1]
    if (!this.task || !last) return
    const older = await this.#run(() => TaskService.ListActivity(this.task!.id, last.id, ACTIVITY_PAGE))
    if (!older) return
    this.activity = [...this.activity, ...older]
    this.moreActivity = older.length === ACTIVITY_PAGE
  }

  async addComment(body: string): Promise<boolean> {
    if (!this.task) return false
    const c = await this.#run(() => TaskService.AddComment(this.task!.id, body))
    if (!c) return false
    await this.reload()
    return true
  }

  async editComment(id: number, body: string): Promise<boolean> {
    const c = await this.#run(() => TaskService.EditComment(id, body))
    if (!c) return false
    this.comments = this.comments.map((x) => (x.id === id ? c : x))
    return true
  }

  async deleteComment(id: number) {
    const ok = await this.#run(async () => {
      await TaskService.DeleteComment(id)
      return true
    })
    if (ok) this.comments = this.comments.filter((c) => c.id !== id)
  }

  close() {
    this.#loading++
    this.task = null
    this.subtasks = []
    this.comments = []
    this.activity = []
    this.runs = []
    this.moreActivity = false
    this.error = ''
  }

  async reload() {
    if (this.task) await this.open(this.task.id)
  }

  async #changed() {
    await this.reload()
    this.#onChanged()
  }

  async rename(title: string): Promise<boolean> {
    if (!this.task) return false
    const t = await this.#run(() => TaskService.UpdateTask(this.task!.id, title, null))
    if (!t) return false
    await this.#changed()
    return true
  }

  async describe(description: string): Promise<boolean> {
    if (!this.task) return false
    const t = await this.#run(() => TaskService.UpdateTask(this.task!.id, null, description))
    if (!t) return false
    await this.reload()
    return true
  }

  // setWorkType gives the task one of the guild's work types ("" for none).
  async setWorkType(key: string) {
    if (!this.task) return
    const t = await this.#run(() => TaskService.SetWorkType(this.task!.id, key))
    if (t) await this.#changed()
  }

  async addSubtask(title: string): Promise<boolean> {
    if (!this.task) return false
    const st = await this.#run(() => TaskService.AddSubtask(this.task!.id, title))
    if (!st) return false
    await this.#changed()
    return true
  }

  async renameSubtask(id: number, title: string) {
    const st = await this.#run(() => TaskService.UpdateTask(id, title, null))
    if (st) await this.reload()
  }

  // toggle ticks the box at once and puts it back if the API refuses.
  async toggle(id: number) {
    const st = this.subtasks.find((s) => s.id === id)
    if (!st) return
    st.done = !st.done
    const saved = await this.#run(() => TaskService.SetSubtaskDone(id, st.done))
    if (!saved) st.done = !st.done
    else await this.#changed()
  }

  async deleteSubtask(id: number) {
    const ok = await this.#run(async () => {
      await TaskService.DeleteTask(id)
      return true
    })
    if (ok) await this.#changed()
  }

  // moveSubtask moves it to index at once, then tells the API its new
  // neighbours; a refusal reloads the list.
  async moveSubtask(id: number, index: number) {
    const from = this.subtasks.findIndex((s) => s.id === id)
    if (from < 0) return
    const [st] = this.subtasks.splice(from, 1)
    index = Math.max(0, Math.min(index, this.subtasks.length))
    this.subtasks.splice(index, 0, st)
    const afterId = this.subtasks[index - 1]?.id ?? null
    const beforeId = this.subtasks[index + 1]?.id ?? null
    const moved = await this.#run(() => TaskService.MoveSubtask(id, afterId, beforeId))
    if (!moved) await this.reload()
    else st.position = moved.position
  }
}
