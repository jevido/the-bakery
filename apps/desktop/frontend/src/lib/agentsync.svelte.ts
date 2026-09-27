// The agents' sync as the app shows it: its state, and conflicts waiting
// for the member to choose a version. AgentsService does the syncing in Go
// and says when anything changes, so nothing here polls.

import { Events } from '@wailsio/runtime'
import { AgentsService, messageOf } from './bindings'

export type SyncState = 'idle' | 'syncing' | 'offline' | 'error' | 'signed-out'

export type SyncStatus = {
  state: SyncState
  last_sync: string | null
  conflicts: number
  error?: string
  problems: string[]
}

export type FileDiff = {
  path: string
  local: string
  server: string
  missing_local: boolean
  missing_server: boolean
}

export type Conflict = { slug: string; files: FileDiff[] }

export class AgentSync {
  status = $state<SyncStatus>({ state: 'idle', last_sync: null, conflicts: 0, problems: [] })
  conflicts = $state.raw<Conflict[]>([])
  error = $state('')
  // Bumped whenever the roster may have changed, for screens to reload on.
  changes = $state(0)

  // listen follows the sync until the returned function is called.
  listen(): () => void {
    const offs = [
      Events.On('agents:status', (e) => {
        this.status = e.data as SyncStatus
        if (this.status.conflicts !== this.conflicts.length) this.loadConflicts()
      }),
      Events.On('agents:changed', () => {
        this.changes++
        this.loadConflicts()
      }),
    ]
    AgentsService.Status().then((s) => (this.status = s as SyncStatus))
    this.loadConflicts()
    AgentsService.SyncNow()
    return () => offs.forEach((off) => off())
  }

  async loadConflicts() {
    try {
      this.conflicts = ((await AgentsService.Conflicts()) ?? []).map((c) => ({ slug: c.slug, files: c.files ?? [] }))
    } catch (err) {
      this.error = messageOf(err)
    }
  }

  async resolve(slug: string, choice: 'local' | 'server' | 'both'): Promise<boolean> {
    try {
      await AgentsService.ResolveConflict(slug, choice)
      this.error = ''
      await this.loadConflicts()
      return true
    } catch (err) {
      this.error = messageOf(err)
      return false
    }
  }
}
