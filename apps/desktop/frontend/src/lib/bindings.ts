// One import point for the generated Wails bindings (`wails3 generate
// bindings`, run by `task desktop:dev` and `task desktop:check`).
export { SessionService, BoardsService, TaskService, LiveService, AgentsService, UpdateService, WebsiteService } from '../../bindings/github.com/jevido/the-bakery/apps/desktop'
export type {
  Member,
  Guild,
  Board,
  Task,
  Column,
  BoardView,
  TaskDetail,
  Comment,
  Activity,
} from '../../bindings/github.com/jevido/the-bakery/apps/desktop/internal/api/models'

// Go errors reach the frontend as rejected promises; this pulls out the
// message to show, as a sentence.
export function messageOf(err: unknown): string {
  let message: string
  if (err instanceof Error) message = err.message
  else if (typeof err === 'object' && err !== null && 'message' in err) message = String(err.message)
  else message = String(err)
  return message.charAt(0).toUpperCase() + message.slice(1)
}

// BoardsService rejects with this when the API no longer accepts the token.
export function isSignedOut(err: unknown): boolean {
  return messageOf(err).toLowerCase().startsWith('signed out')
}

export type { AgentDetail, AgentSummary, SkillEntry, TraitList } from '../../bindings/github.com/jevido/the-bakery/apps/desktop/models'
export type { Manifest } from '../../bindings/github.com/jevido/the-bakery/apps/desktop/internal/agents/models'
export type { Agent, Trait } from '../../bindings/github.com/jevido/the-bakery/apps/desktop/internal/api/models'
