// A small history-API router: the current path as state, and navigation
// without reloads. Routes are matched in App.svelte.

class Router {
  path = $state(window.location.pathname)

  constructor() {
    window.addEventListener('popstate', () => (this.path = window.location.pathname))
  }

  navigate(to: string) {
    if (to === this.path) return
    window.history.pushState({}, '', to)
    this.path = new URL(to, window.location.origin).pathname
    window.scrollTo(0, 0)
  }

  // onclick handles clicks on same-origin links anywhere on the page, so
  // plain <a href="/desktop"> works without a Link component.
  onclick = (event: MouseEvent) => {
    if (event.defaultPrevented || event.button !== 0) return
    if (event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) return
    const a = (event.target as Element | null)?.closest('a')
    if (!a || a.target || a.hasAttribute('download')) return
    const url = new URL(a.href, window.location.href)
    if (url.origin !== window.location.origin) return
    event.preventDefault()
    this.navigate(url.pathname + url.search + url.hash)
  }
}

export const router = new Router()
