// The alerts column: standing conditions that need a person, from
// WorkshopService (sent again whenever they change). An alert goes away
// only when its condition clears.

import { Events } from '@wailsio/runtime'
import { WorkshopService, type Alert } from './bindings'

const COLLAPSED = 'bakery.alerts.collapsed'

export class Alerts {
  list = $state.raw<Alert[]>([])
  collapsed = $state(readCollapsed())

  listen() {
    WorkshopService.Alerts()
      .then((l) => (this.list = l ?? []))
      .catch(() => {})
    return Events.On('alerts:changed', (e) => (this.list = (e.data as Alert[]) ?? []))
  }

  toggle() {
    this.collapsed = !this.collapsed
    try {
      localStorage.setItem(COLLAPSED, this.collapsed ? '1' : '')
    } catch {
      // Storage refused: the column stays as it is for this session.
    }
  }
}

function readCollapsed(): boolean {
  try {
    return localStorage.getItem(COLLAPSED) === '1'
  } catch {
    return false
  }
}
