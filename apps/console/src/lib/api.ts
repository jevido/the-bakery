// The console's API client. It sends the operator's bearer token, never a
// cookie, and signs the operator out on a 401 (an expired or revoked token).
// In development VITE_API_URL is empty and Vite proxies /api.

const BASE = (import.meta.env.VITE_API_URL ?? '').replace(/\/$/, '')

export class ApiError extends Error {
  constructor(
    readonly status: number,
    message: string,
    readonly field?: string,
  ) {
    super(message)
  }
}

let token = ''
let onUnauthorized = () => {}

export function useToken(t: string, unauthorized: () => void) {
  token = t
  onUnauthorized = unauthorized
}

export async function api<T>(method: string, path: string, body?: unknown, bearer = token): Promise<T> {
  let res: Response
  try {
    res = await fetch(BASE + path, {
      method,
      headers: {
        Accept: 'application/json',
        ...(bearer ? { Authorization: `Bearer ${bearer}` } : {}),
        ...(body !== undefined ? { 'Content-Type': 'application/json' } : {}),
      },
      body: body !== undefined ? JSON.stringify(body) : undefined,
    })
  } catch {
    throw new ApiError(0, 'The API cannot be reached.')
  }
  if (res.status === 204) return undefined as T
  const data = await res.json().catch(() => ({}))
  if (!res.ok) {
    if (res.status === 401 && bearer && bearer === token) onUnauthorized()
    if (res.status === 429) throw new ApiError(429, 'Too many tries. Wait a minute, then try again.')
    const message = typeof data.error === 'string' ? data.error : res.statusText
    throw new ApiError(res.status, message.charAt(0).toUpperCase() + message.slice(1), data.field)
  }
  return data as T
}

export type Sanction = {
  id: number
  target_kind: 'member' | 'guild'
  target_id: number
  kind: 'suspension' | 'ban'
  reason: string
  until: string | null
  by_operator_id: number
  lifted_at: string | null
  created_at: string
  active: boolean
}

export type Report = {
  id: number
  by_member_id: number
  target_kind: 'member' | 'guild'
  target_id: number
  reason: string
  status: 'open' | 'dismissed' | 'actioned'
  handled_by?: number
  handled_at: string | null
  sanction_id?: number
  created_at: string
}

export type AuditEntry = {
  id: number
  actor_kind: 'member' | 'operator'
  actor_id: number
  action: string
  target_kind: string
  target_id: number
  reason: string
  meta: Record<string, unknown> | null
  ip: string
  at: string
}

// Names maps ids (as strings) to display names and guild names.
export type Names = { members: Record<string, string>; guilds: Record<string, string> }

export type MemberCard = { id: number; email: string; display_name: string; joined_at: string }
export type GuildCard = { id: number; name: string; archived: boolean; founded_at: string; member_count: number }
