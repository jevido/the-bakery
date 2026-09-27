# web

The Bakery's website, bakery.jevido.app: a Svelte 5 + Vite + TypeScript
single-page app (no SvelteKit) using [`@bakery/ui`](../../packages/ui). Today
it has the landing page `/` and the download page `/desktop`; sign-in and the
guild admin come later on the same app. A client of `services/api`; it hosts
no bounded context.

## Run

```sh
task web:dev     # from the repo root: http://127.0.0.1:4840, /api proxied to :4810
task web:check   # svelte-check, tsc, bun test, production build
```

## Layout

- `src/lib/router.svelte.ts` — routing. A small history-API router of our
  own: the current path as state, and a document click handler so plain
  `<a href="/desktop">` links navigate without a reload. Routes are a table in
  `App.svelte`; anything else is the 404 page. Chosen over `svelte-spa-router`,
  which routes on the URL hash by default and would give `/#/desktop` URLs.
- `src/lib/releases.ts` — reads the GitHub releases of `jevido/the-bakery`,
  picks the newest published `desktop-v*` tag and sorts its files by OS.
  Tested in `releases.test.ts` against a fixture.
- `src/pages/` — one component per route. Each sets its own `<title>` and
  description with `<svelte:head>`.
- `src/components/` — site pieces (`Layout`, `DownloadButton`).

The site is served as static files with a fallback to `index.html` for every
path, so deep links like `/desktop` work.
