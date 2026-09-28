// Event letters: the storyteller's notes about moments on the open board
// (WorkshopService sends each as story:letter). Unread ones wait in the
// stack of letters until OK; all stay in the board's history. They never
// block anything. With quiet colony on, none are shown.

import { Events } from '@wailsio/runtime'
import { WorkshopService, type StoredEventLetter } from './bindings'
import { settings } from './settings.svelte'

export class Story {
  boardId = $state<number | null>(null)
  history = $state.raw<StoredEventLetter[]>([])
  openId = $state<string | null>(null)
  showHistory = $state(false)

  unread = $derived(settings.quiet ? [] : this.history.filter((l) => !l.read))
  open = $derived(this.history.find((l) => l.id === this.openId) ?? null)

  async load(boardId: number | null) {
    this.boardId = boardId
    this.openId = null
    this.showHistory = false
    this.history = boardId ? ((await WorkshopService.EventLetters(boardId).catch(() => [])) ?? []) : []
  }

  listen() {
    return Events.On('story:letter', (e) => {
      const l = e.data as StoredEventLetter
      if (l.board_id === this.boardId) this.history = [l, ...this.history.filter((h) => h.id !== l.id)]
    })
  }

  // ok closes a letter and marks it read.
  ok(l: StoredEventLetter) {
    this.openId = null
    this.history = this.history.map((h) => (h.id === l.id ? { ...h, read: true } : h))
    WorkshopService.ReadEventLetter(l.board_id, l.id).catch(() => {})
  }
}
