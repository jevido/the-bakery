// The operator's sign-in. The token lives in memory and in sessionStorage
// (so a reload keeps it, a closed tab does not), never in localStorage or a
// cookie. Any 401 signs the operator out.

import { api, useToken } from './api'

const KEY = 'bakery-console-token'

export type Operator = { id: number; email: string }

class Session {
  token = $state('')
  operator = $state<Operator | null>(null)
  checked = $state(false)

  async load() {
    const saved = readSaved()
    if (saved) {
      this.use(saved)
      try {
        const { operator } = await api<{ operator: Operator }>('GET', '/api/console/me')
        this.operator = operator
      } catch {
        this.signOut()
      }
    }
    this.checked = true
  }

  use(token: string) {
    this.token = token
    useToken(token, () => this.signOut())
    try {
      sessionStorage.setItem(KEY, token)
    } catch {
      // Private windows may refuse storage; memory still holds it.
    }
  }

  async signIn(token: string) {
    this.use(token)
    const { operator } = await api<{ operator: Operator }>('GET', '/api/console/me')
    this.operator = operator
  }

  signOut() {
    this.token = ''
    this.operator = null
    useToken('', () => {})
    try {
      sessionStorage.removeItem(KEY)
    } catch {
      // Nothing to remove.
    }
  }
}

function readSaved(): string {
  try {
    return sessionStorage.getItem(KEY) ?? ''
  } catch {
    return ''
  }
}

export const session = new Session()
