import { api } from './api'

export type PersonalToken = { id: number; name: string; created_at: string; last_used_at: string | null }

export const tokens = {
  list: () => api<{ tokens: PersonalToken[] }>('GET', '/api/tokens').then((r) => r.tokens),
  // The secret comes back only here, once.
  create: (name: string) => api<{ token: string; personal_token: PersonalToken }>('POST', '/api/tokens', { name }),
  revoke: (id: number) => api<void>('DELETE', `/api/tokens/${id}`),
}

// In development VITE_API_URL is empty (Vite proxies /api), but Claude talks
// to the API directly.
const API_URL = (import.meta.env.VITE_API_URL || 'http://127.0.0.1:4810').replace(/\/$/, '')

export function mcpAddCommand(secret: string): string {
  return `claude mcp add --transport http bakery ${API_URL}/mcp --header "Authorization: Bearer ${secret}"`
}
