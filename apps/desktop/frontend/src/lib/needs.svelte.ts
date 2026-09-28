// Every agent's needs and mood on this machine, from WorkshopService (it
// sends them again whenever a run changes, and every 30 seconds).
// Presentation only: nothing here changes what an agent does.

import { Events } from '@wailsio/runtime'
import { WorkshopService } from './bindings'

export type Mood = 'content' | 'okay' | 'stressed' | 'breaking'
export type AgentNeeds = {
  agent: string
  needs: { budget: number; focus: number; morale: number; rest: number }
  mood: Mood
  reason: string
}

class Needs {
  bySlug = $state.raw<Record<string, AgentNeeds>>({})

  #set(list: AgentNeeds[] | null) {
    this.bySlug = Object.fromEntries((list ?? []).map((n) => [n.agent, n]))
  }

  listen() {
    WorkshopService.AllNeeds()
      .then((l) => this.#set(l as AgentNeeds[]))
      .catch(() => {})
    return Events.On('agent:needs', (e) => this.#set(e.data as AgentNeeds[]))
  }
}

export const needs = new Needs()

// What each need measures, for the bars' hints.
export const NEED_HINTS = {
  budget: "Money left of today's budget",
  focus: 'Context left in the current or last run',
  morale: 'How the last day of runs went',
  rest: 'Time since a break',
}
