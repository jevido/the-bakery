// Letters: runs asking a person, like RimWorld's letters down the right edge.
// Fed by WorkshopService (letter:new, letter:closed); answering resumes the
// run.

import { Events } from '@wailsio/runtime'
import { WorkshopService, messageOf, type Letter } from './bindings'

export type Choice = 'allow' | 'allow_always' | 'deny'

export class Letters {
  // Open letters, oldest first (the newest sits at the bottom).
  list = $state.raw<Letter[]>([])
  // The letter open in the dialog.
  openId = $state<string | null>(null)
  error = $state('')

  open = $derived(this.list.find((l) => l.id === this.openId) ?? null)

  listen() {
    WorkshopService.OpenLetters()
      .then((l) => (this.list = l ?? []))
      .catch(() => {})
    const offs = [
      Events.On('letter:new', (e) => {
        const l = e.data as Letter
        if (!this.list.some((x) => x.id === l.id)) this.list = [...this.list, l]
      }),
      Events.On('letter:closed', (e) => {
        const l = e.data as Letter
        this.list = this.list.filter((x) => x.id !== l.id)
        if (this.openId === l.id) this.openId = null
      }),
    ]
    return () => offs.forEach((off) => off())
  }

  async answer(id: string, choice: Choice, message = '', answers: Record<string, string> = {}): Promise<boolean> {
    try {
      await WorkshopService.AnswerLetter(id, { choice, message, answers })
      this.error = ''
      this.list = this.list.filter((x) => x.id !== id)
      if (this.openId === id) this.openId = null
      return true
    } catch (err) {
      this.error = messageOf(err)
      return false
    }
  }
}
