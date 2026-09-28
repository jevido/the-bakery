// The website's API client. It sends the web session cookie and the
// X-Bakery-Web header the API requires on cookie requests that change
// something. In development VITE_API_URL is empty and Vite proxies /api.

const BASE = (import.meta.env.VITE_API_URL ?? '').replace(/\/$/, '')

export class ApiError extends Error {
  constructor(
    readonly status: number,
    message: string,
    readonly field?: string,
    // code is set on a refusal because of a suspension or a ban
    // (account_suspended, account_banned, guild_suspended, guild_banned).
    readonly code?: string,
  ) {
    super(message)
  }
}

export type Member = { id: number; email: string; display_name: string }

export async function api<T>(method: string, path: string, body?: unknown): Promise<T> {
  let res: Response
  try {
    res = await fetch(BASE + path, {
      method,
      credentials: 'include',
      headers: {
        Accept: 'application/json',
        'X-Bakery-Web': '1',
        ...(body !== undefined ? { 'Content-Type': 'application/json' } : {}),
      },
      body: body !== undefined ? JSON.stringify(body) : undefined,
    })
  } catch {
    throw new ApiError(0, 'The Bakery cannot be reached right now. Try again in a moment.')
  }
  if (res.status === 204) return undefined as T
  const data = await res.json().catch(() => ({}))
  if (!res.ok) {
    const message = typeof data.error === 'string' ? data.error : res.statusText
    throw new ApiError(res.status, message.charAt(0).toUpperCase() + message.slice(1), data.field, data.code)
  }
  return data as T
}
