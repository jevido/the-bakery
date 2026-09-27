// Who is signed in on the website. The session itself is an httpOnly cookie
// the API sets; this only knows the member it belongs to.

import { api, ApiError, type Member } from './api'

class Session {
  // undefined until the first check has answered.
  member = $state<Member | null | undefined>(undefined)

  async load() {
    try {
      const res = await api<{ member: Member }>('GET', '/api/me')
      this.member = res.member
    } catch (err) {
      if (!(err instanceof ApiError) || err.status !== 401) console.warn(err)
      this.member = null
    }
  }

  async signIn(email: string, password: string) {
    const res = await api<{ member: Member }>('POST', '/api/web/login', { email, password })
    this.member = res.member
  }

  async signUp(email: string, displayName: string, password: string) {
    const res = await api<{ member: Member }>('POST', '/api/web/register', {
      email,
      display_name: displayName,
      password,
    })
    this.member = res.member
  }

  async signOut() {
    await api('POST', '/api/web/logout')
    this.member = null
  }
}

export const session = new Session()
