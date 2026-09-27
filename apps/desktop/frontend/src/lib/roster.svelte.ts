// The Agents screen's state: the roster (read from the agent folders), the
// agent open on the character card, and what the card needs (traits, the
// skills in ~/.claude/skills, a guild's shared agents). Everything goes
// through AgentsService; saving writes the folder and the sync sends it.

import {
  AgentsService,
  messageOf,
  type Agent,
  type AgentDetail,
  type AgentSummary,
  type Manifest,
  type SkillEntry,
  type Trait,
} from './bindings'

// A card being edited: the manifest with arrays and maps never null.
export type Draft = {
  name: string
  title: string
  backstory: string
  traits: string[]
  model: string
  permission_mode: string
  allowed_tools: string
  portrait_seed: string
  work_priorities: Record<string, number>
}

function draftOf(m: Manifest): Draft {
  return {
    name: m.name ?? '',
    title: m.title ?? '',
    backstory: m.backstory ?? '',
    traits: m.traits ?? [],
    model: m.model || 'sonnet',
    permission_mode: m.permission_mode || 'manual',
    allowed_tools: (m.allowed_tools ?? []).join('\n'),
    portrait_seed: m.portrait_seed ?? '',
    work_priorities: (m.work_priorities ?? {}) as Record<string, number>,
  }
}

export function manifestOf(d: Draft): Manifest {
  return {
    name: d.name.trim(),
    title: d.title.trim(),
    backstory: d.backstory,
    traits: d.traits,
    model: d.model,
    permission_mode: d.permission_mode,
    allowed_tools: d.allowed_tools
      .split('\n')
      .map((t) => t.trim())
      .filter(Boolean),
    portrait_seed: d.portrait_seed,
    work_priorities: d.work_priorities,
  } as Manifest
}

export class Roster {
  agents = $state.raw<AgentSummary[]>([])
  selected = $state<string | null>(null)
  detail = $state.raw<AgentDetail | null>(null)
  draft = $state<Draft | null>(null)
  traits = $state.raw<Trait[]>([])
  permissionModes = $state.raw<string[]>(['manual', 'acceptEdits', 'plan', 'dontAsk', 'auto'])
  error = $state('')
  loaded = $state(false)

  async #run<T>(f: () => Promise<T>): Promise<T | undefined> {
    try {
      const v = await f()
      this.error = ''
      return v
    } catch (err) {
      this.error = messageOf(err)
      return undefined
    }
  }

  async load() {
    this.agents = (await this.#run(() => AgentsService.List())) ?? []
    this.loaded = true
    if (this.selected && !this.agents.some((a) => a.slug === this.selected)) this.selected = null
    if (!this.selected && this.agents[0]) await this.open(this.agents[0].slug)
    else if (this.selected) await this.refreshDetail()
    if (this.traits.length === 0) {
      const list = await this.#run(() => AgentsService.Traits())
      if (list) {
        this.traits = list.traits ?? []
        if (list.permission_modes?.length) this.permissionModes = list.permission_modes
      }
    }
  }

  async open(slug: string) {
    this.selected = slug
    const d = await this.#run(() => AgentsService.Get(slug))
    if (!d || this.selected !== slug) return
    this.detail = d
    this.draft = draftOf(d.manifest)
  }

  // refreshDetail reloads the card's facts (skills, sync, sharing) without
  // throwing away what the member is typing.
  // It leaves the error line alone, so a failed action's message stays up.
  async refreshDetail() {
    if (!this.selected) return
    try {
      this.detail = await AgentsService.Get(this.selected)
    } catch {
      // The next load or action reports it.
    }
  }

  get dirty(): boolean {
    if (!this.detail || !this.draft) return false
    return JSON.stringify(manifestOf(this.draft)) !== JSON.stringify(manifestOf(draftOf(this.detail.manifest)))
  }

  async save(): Promise<boolean> {
    if (!this.selected || !this.draft) return false
    const ok = await this.#run(async () => {
      await AgentsService.Save(this.selected!, manifestOf(this.draft!))
      return true
    })
    if (ok) await this.open(this.selected)
    return !!ok
  }

  async create(name: string, title: string): Promise<boolean> {
    const slug = await this.#run(() => AgentsService.Create({ name, title } as Manifest))
    if (!slug) return false
    await this.load()
    await this.open(slug)
    return true
  }

  async remove(slug: string) {
    const ok = await this.#run(async () => {
      await AgentsService.Delete(slug)
      return true
    })
    if (ok) {
      this.selected = null
      this.detail = null
      this.draft = null
      await this.load()
    }
  }

  async claudeSkills(): Promise<SkillEntry[]> {
    return (await this.#run(() => AgentsService.ClaudeSkills())) ?? []
  }

  async importSkills(dirs: string[]): Promise<boolean> {
    if (!this.selected) return false
    const ok = await this.#run(async () => {
      await AgentsService.ImportSkills(this.selected!, dirs)
      return true
    })
    await this.refreshDetail()
    return !!ok
  }

  async importFolder() {
    if (!this.selected) return
    await this.#run(() => AgentsService.ImportSkillFolder(this.selected!))
    await this.refreshDetail()
  }

  async removeSkill(skill: string) {
    if (!this.selected) return
    await this.#run(() => AgentsService.RemoveSkill(this.selected!, skill))
    await this.refreshDetail()
  }

  async openFolder() {
    if (this.selected) await this.#run(() => AgentsService.OpenFolder(this.selected!))
  }

  async share(guildId: number, on: boolean) {
    if (!this.selected) return
    await this.#run(() => AgentsService.Share(this.selected!, guildId, on))
    await this.refreshDetail()
  }

  async pullOrigin() {
    if (!this.selected) return
    await this.#run(() => AgentsService.PullOrigin(this.selected!))
    await this.refreshDetail()
  }

  async guildAgents(guildId: number): Promise<Agent[]> {
    return (await this.#run(() => AgentsService.GuildAgents(guildId))) ?? []
  }

  async recruit(agentId: number, guildId: number): Promise<boolean> {
    const ok = await this.#run(async () => {
      await AgentsService.Recruit(agentId, guildId)
      return true
    })
    return !!ok
  }
}
