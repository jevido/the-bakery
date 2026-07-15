# The Bakery

A self-hosted, multi-tenant container deployment platform (think a small
Heroku/Render/Coolify) — connect a GitHub repo, push, and it builds a
Podman/Quadlet image, deploys it to one of your own hosts, and puts it
behind a managed reverse proxy with a real domain, zero-downtime rollovers,
and rollback.

The control plane is a SvelteKit app; each host you deploy to runs a small
Go agent (`agent/`) that checks in over outbound HTTP and executes
container lifecycle commands — the control plane decides, the agent
reports and executes. No inbound access to hosts is required.

## Stack

- **Control plane**: SvelteKit (adapter-node), TypeScript, Tailwind, Drizzle ORM + Postgres
- **Auth**: Better Auth (organizations plugin — an org is a "guild"), email/password + GitHub OAuth
- **Host agent**: standalone Go module (`agent/`), rootless Podman + systemd `--user` Quadlet units
- **Reverse proxy**: Caddy, driven by control-plane-generated Caddyfiles pushed to each host
- **Registry**: a private container registry (`compose.yaml`'s `registry` service) the build worker pushes to and host agents pull from
- **Source/CI**: GitHub App installation per guild, webhook-triggered builds

## Core concepts

- **Guild** — a tenant/organization. All data (hosts, apps, sources, domains, volumes) is strictly scoped to one guild; guild membership and roles are Better Auth's organization plugin plus a few custom roles/permissions (`src/lib/auth/permissions.ts`).
- **Host** — a customer-owned machine running `bakery-agent`, authenticated to the control plane with a per-host bearer token issued once at creation.
- **Source / repo** — a GitHub App installation and the repos it can see; a push to a repo's default branch queues a build.
- **App** — one deployable unit: a repo + build context, deployed to a host as a versioned Podman Quadlet unit, reachable at a default subdomain (and optionally custom domains) via Caddy.
- **Build** — clones the repo, builds the Dockerfile/Containerfile with Podman, pushes to the private registry.
- **Deployment** — a zero-downtime rollover state machine: start the new unit → health-check it → flip proxy traffic → stop the old unit, with rollback to any prior successful build.

## Developing

Requires [Bun](https://bun.sh) and Docker (for local Postgres + registry).

```sh
cp .env.example .env   # fill in the required values, see below
bun install
bun run db:start        # starts Postgres + the local registry (compose.yaml)
bun run db:push         # applies the Drizzle schema
bun run dev              # or: bun run dev -- --open
```

Then in separate terminals as needed:

```sh
bun run build-worker      # polls for queued builds and runs them
bun run retention-cleanup # one-off/cron: prunes old metric/log rows
```

### Environment

See `.env.example` for the full list with explanations. At minimum for local dev you'll need:

- `DATABASE_URL` — matches `compose.yaml`'s `db` service by default
- `BETTER_AUTH_SECRET` — high-entropy random string
- `ENCRYPTION_KEY` — 32 random bytes, base64-encoded (`node -e "console.log(require('crypto').randomBytes(32).toString('base64'))"`); encrypts app env vars at rest
- `GITHUB_APP_ID` / `GITHUB_APP_SLUG` / `GITHUB_APP_PRIVATE_KEY` / `GITHUB_WEBHOOK_SECRET` — a GitHub App for the Sources/build pipeline (see [GitHub Apps settings](https://github.com/settings/apps))
- `BAKERY_REGISTRY_*` — credentials for the local registry; after setting these, run `bun scripts/generate-registry-htpasswd.ts` (see `registry-auth/README.md`)

### Useful scripts

| Command                              | What it does                                                         |
| ------------------------------------ | -------------------------------------------------------------------- |
| `bun run db:studio`                  | Drizzle Studio, browse the local database                            |
| `bun run db:generate` / `db:migrate` | Generate/apply a schema migration                                    |
| `bun run check`                      | Svelte/TypeScript type checking                                      |
| `bun run lint` / `bun run format`    | Prettier + ESLint                                                    |
| `bun run auth:schema`                | Regenerate Better Auth's Drizzle schema after a plugin/config change |

## The host agent

`agent/` is a separate Go module (its own `go.mod`, independent of this
app's `package.json`) meant to be built and installed on each host you
deploy to — see `agent/README.md`. The control plane also serves an install
script at `GET /install.sh` for a one-line rootless setup on a fresh host.

## Deploying The Bakery itself

This repo uses `@sveltejs/adapter-node`; see the [SvelteKit adapter docs](https://svelte.dev/docs/kit/adapters)
for deploy targets. In production you'll also need:

- Real Postgres (not the dev `compose.yaml` container)
- `BAKERY_DOMAIN` set, with wildcard DNS pointed at your infrastructure (apps get `<app>.<guild-slug>.<bakery-domain>`)
- A real container registry reachable from both the build worker and every host agent
- `ENCRYPTION_KEY`/`BETTER_AUTH_SECRET` from a real secret store, not `.env` — see the key-rotation notes in `src/lib/server/secrets/crypto.ts` if either ever needs to change
