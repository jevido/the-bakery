import { api } from './api'
import type { Guild } from './guilds'

export type Invite = {
  id: number
  guild_id: number
  code: string
  expires_at: string | null
  max_uses: number | null
  uses: number
  state: 'active' | 'expired' | 'used_up' | 'revoked'
}

export type InviteInfo = { guild_name: string; member_count: number; valid: boolean; problem?: string }

export const invites = {
  list: (guildId: number) => api<{ invites: Invite[] }>('GET', `/api/guilds/${guildId}/invites`).then((r) => r.invites),
  create: (guildId: number, expiresInHours: number | null, maxUses: number | null) =>
    api<{ invite: Invite }>('POST', `/api/guilds/${guildId}/invites`, {
      expires_in_hours: expiresInHours,
      max_uses: maxUses,
    }).then((r) => r.invite),
  revoke: (guildId: number, id: number) => api<void>('DELETE', `/api/guilds/${guildId}/invites/${id}`),
  info: (code: string) => api<{ invite: InviteInfo }>('GET', `/api/invites/${encodeURIComponent(code)}`).then((r) => r.invite),
  accept: (code: string) =>
    api<{ guild: Guild }>('POST', `/api/invites/${encodeURIComponent(code)}/accept`).then((r) => r.guild),
}

export function inviteLink(code: string): string {
  return `${window.location.origin}/join/${code}`
}
