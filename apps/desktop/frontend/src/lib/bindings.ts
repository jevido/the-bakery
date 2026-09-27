// One import point for the generated Wails bindings (`wails3 generate
// bindings`, run by `task desktop:dev` and `task desktop:check`).
export { SessionService } from '../../bindings/github.com/jevido/the-bakery/apps/desktop'
export type { Member } from '../../bindings/github.com/jevido/the-bakery/apps/desktop/internal/api/models'

// Go errors reach the frontend as rejected promises; this pulls out the
// message to show, as a sentence.
export function messageOf(err: unknown): string {
  let message: string
  if (err instanceof Error) message = err.message
  else if (typeof err === 'object' && err !== null && 'message' in err) message = String(err.message)
  else message = String(err)
  return message.charAt(0).toUpperCase() + message.slice(1)
}
