# console

The operator console: a Svelte 5 + Vite + TypeScript single-page app using
[`@bakery/ui`](../../packages/ui), for the people who run The Bakery. An
operator signs in with a password and a code from an authenticator app, then
finds members and guilds, suspends or bans them, works the reports queue and
reads the audit log. A client of the moderation context's `/api/console/*`
routes in `services/api`; it hosts no bounded context and is never linked from
the public website.

Operators are made on the server: `go run . artisan operator:create <email>`
in `services/api`, then `operator:confirm`.

## Run

```sh
task console:dev     # from the repo root: http://127.0.0.1:4850, /api proxied to :4810
task console:check   # svelte-check, tsc, bun test, production build
```

## Layout

- `src/lib/session.svelte.ts` — the operator's token, kept in memory and
  `sessionStorage` only; any 401 signs out.
- `src/lib/api.ts` — the API client (bearer token, no cookies) and the
  response types.
- `src/pages/` — Login, Members, Member, Guilds, Guild, Reports, Audit.
- `src/components/` — the sanction dialog, sanction list, reports and audit
  tables, and the layout.

Served as static files with a fallback to `index.html` and an
`X-Robots-Tag: noindex` header (see `infra/images/console`).
