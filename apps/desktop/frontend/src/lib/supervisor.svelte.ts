// The open board's supervisor chat, from SupervisorService. Sending waits
// for the answer (one Claude call); approving a proposal runs it through
// the same services as the member's own clicks.

import { SupervisorService, messageOf, type ChatEntry } from './bindings'

type Draft = { title: string; description: string; work_type: string }

export class SupervisorChat {
  boardId = $state<number | null>(null)
  entries = $state<ChatEntry[]>([])
  thinking = $state(false)
  error = $state('')
  // Which proposal is being carried out, as "entry:index".
  busy = $state('')

  async load(boardId: number | null) {
    if (boardId !== this.boardId) this.error = ''
    this.boardId = boardId
    this.entries = boardId ? ((await SupervisorService.History(boardId).catch(() => [])) ?? []) : []
  }

  async send(text: string): Promise<boolean> {
    const board = this.boardId
    if (!board || this.thinking || !text.trim()) return false
    this.error = ''
    this.thinking = true
    // Shown at once; the history on disk has it too.
    this.entries = [...this.entries, { id: 'pending', from: 'member', text: text.trim(), at: new Date().toISOString() }]
    let ok = true
    try {
      await SupervisorService.Send(board, text)
    } catch (err) {
      this.error = messageOf(err)
      ok = false
    }
    this.thinking = false
    if (this.boardId === board) await this.load(board)
    return ok
  }

  async approve(entryId: string, index: number, subtasks: Draft[] | null) {
    const board = this.boardId
    if (!board) return
    this.busy = `${entryId}:${index}`
    this.error = ''
    try {
      await SupervisorService.Approve(board, entryId, index, subtasks)
    } catch (err) {
      // The refusal is on the card too, from the history.
      this.error = messageOf(err)
    }
    this.busy = ''
    await this.load(board)
  }

  async decline(entryId: string, index: number) {
    const board = this.boardId
    if (!board) return
    await SupervisorService.Decline(board, entryId, index).catch((err) => (this.error = messageOf(err)))
    await this.load(board)
  }

  async forget() {
    const board = this.boardId
    if (!board) return
    await SupervisorService.Forget(board).catch((err) => (this.error = messageOf(err)))
    await this.load(board)
  }
}
