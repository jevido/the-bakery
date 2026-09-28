// Who is signed in on the website. The session itself is an httpOnly cookie
// the API sets; this only knows the member it belongs to.

import { api, ApiError, type Member } from './api'

class Session {
  // undefined until the first check has answered.
  member = $state<Member | null | undefined>(undefined)
  // Why this account may not use The Bakery right now (a suspension or a
  // ban), shown instead of a sign-in that would fail the same way.
  refusal = $state('')

  // Bumped on every sign-in or sign-out, so a /api/me answer that was already
  // under way cannot undo it.
  #version = 0

  async load() {
    const version = this.#version
    let member: Member | null
    try {
      member = (await api<{ member: Member }>('GET', '/api/me')).member
    } catch (err) {
      if (err instanceof ApiError && err.code?.startsWith('account_')) this.refusal = err.message
      else if (!(err instanceof ApiError) || err.status !== 401) console.warn(err)
      member = null
    }
    if (version === this.#version) this.member = member
  }

  // adopt records a sign-in that happened elsewhere (the desktop handoff).
  adopt(member: Member | null) {
    this.#version++
    this.member = member
    if (member) this.refusal = ''
  }

  async signIn(email: string, password: string) {
    try {
      const res = await api<{ member: Member }>('POST', '/api/web/login', { email, password })
      this.adopt(res.member)
    } catch (err) {
      if (err instanceof ApiError && err.code?.startsWith('account_')) this.refusal = err.message
      throw err
    }
  }

  async signUp(email: string, displayName: string, password: string) {
    const res = await api<{ member: Member }>('POST', '/api/web/register', {
      email,
      display_name: displayName,
      password,
    })
    this.adopt(res.member)
  }

  async signOut() {
    await api('POST', '/api/web/logout')
    this.adopt(null)
  }
}

export const session = new Session()
