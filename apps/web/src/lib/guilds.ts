// Guild admin calls. Every one needs a signed-in member.

import { api } from './api'

export type Guild = { id: number; name: string; archived: boolean }
export type GuildMember = { id: number; display_name: string; joined_at: string }

export const guilds = {
  list: (archived: boolean) =>
    api<{ guilds: Guild[] }>('GET', `/api/guilds${archived ? '?archived=1' : ''}`).then((r) => r.guilds),
  get: (id: number) => api<{ guild: Guild }>('GET', `/api/guilds/${id}`).then((r) => r.guild),
  found: (name: string) => api<{ guild: Guild }>('POST', '/api/guilds', { name }).then((r) => r.guild),
  rename: (id: number, name: string) => api<{ guild: Guild }>('PATCH', `/api/guilds/${id}`, { name }).then((r) => r.guild),
  archive: (id: number) => api<{ guild: Guild }>('POST', `/api/guilds/${id}/archive`).then((r) => r.guild),
  restore: (id: number) => api<{ guild: Guild }>('POST', `/api/guilds/${id}/restore`).then((r) => r.guild),
  members: (id: number) => api<{ members: GuildMember[] }>('GET', `/api/guilds/${id}/members`).then((r) => r.members),
  remove: (id: number, memberId: number) => api<void>('DELETE', `/api/guilds/${id}/members/${memberId}`),
  leave: (id: number) => api<void>('POST', `/api/guilds/${id}/leave`),
}

export function joinedOn(iso: string): string {
  return new Date(iso).toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric' })
}
