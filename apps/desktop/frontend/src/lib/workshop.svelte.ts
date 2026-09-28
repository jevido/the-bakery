// This machine's runs: the ones this app started (from WorkshopService),
// and which tasks have a run going anywhere (from the board's run events).

import { Events } from '@wailsio/runtime'
import { WorkshopService, messageOf, type RunInfo } from './bindings'

export class Workshop {
  // This app's runs since it opened, newest first.
  runs = $state.raw<RunInfo[]>([])
  // Runs going on any machine, as the board's events told: run id to task id.
  #elsewhere = $state<Record<number, number>>({})

  listen() {
    WorkshopService.Runs()
      .then((r) => (this.runs = r ?? []))
      .catch(() => {})
    const offs = [
      Events.On('workshop:runs', (e) => (this.runs = (e.data as RunInfo[]) ?? [])),
      // What an agent is doing changes often; only that run is patched.
      Events.On('agent:state', (e) => {
        const s = e.data as { run: string; state: string }
        this.runs = this.runs.map((r) => (r.id === s.run ? { ...r, state: s.state } : r))
      }),
    ]
    return () => offs.forEach((off) => off())
  }

  // A board event about a run, from any member.
  onRunEvent(type: string, data: Record<string, unknown>) {
    const run = Number(data.run_id)
    if (type === 'run.started') this.#elsewhere[run] = Number(data.task_id)
    else delete this.#elsewhere[run]
  }

  // Whether an agent is working on the task, here or elsewhere.
  isRunning(taskId: number): boolean {
    if (this.runs.some((r) => r.task_id === taskId && r.status === 'running')) return true
    return Object.values(this.#elsewhere).includes(taskId)
  }

  // This app's run behind an API run id, if it started it.
  local(apiRunId: number): RunInfo | undefined {
    return this.runs.find((r) => r.api_run_id === apiRunId)
  }

  byId(id: string): RunInfo | undefined {
    return this.runs.find((r) => r.id === id)
  }

  async start(boardId: number, taskId: number, agentSlug: string): Promise<{ run?: RunInfo; error?: string }> {
    try {
      return { run: await WorkshopService.StartRun(boardId, taskId, agentSlug) }
    } catch (err) {
      return { error: messageOf(err) }
    }
  }
}
