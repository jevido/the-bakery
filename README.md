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
app's `package.json`) — a single `bakery` binary with subcommands, meant to
be installed on each host you deploy to. See `agent/README.md` for the full
subcommand reference. In short:

- `curl -fsSL <url>/install.sh | bash` — root-level shim: creates the
  dedicated `bakery` system user and installs the binary. Pass
  `--url=<that-instance>` to enroll against an already-running instance (the
  binary is downloaded from that instance's own `/releases/` route, built by
  its own build-worker); omit it entirely on a box with nothing running yet
  and the script builds the binary from source itself (`git clone` + `go
build`, installing Go/git first if missing) — no GitHub Release, no
  pre-built artifact, ever.
- `bakery setup` — idempotently installs rootless Podman and its
  prerequisites, subuid/subgid ranges, the unprivileged-port sysctl, and
  firewall rules.
- `bakery join --token=... --url=...` — enrolls this host against an
  already-running instance (the flow behind the UI's "Add Host" button).
- `bakery bootstrap --domain=...` — stands up a brand-new, self-hosted
  instance from nothing: provisions Postgres, brings up the control plane
  loopback-only, creates the first admin/guild/host/app, then hands off
  into `bakery join` so the control plane ends up managed by its own real
  deployment engine.

## Deploying The Bakery itself

The one-line self-hosting path is: on a fresh Debian 13 box with DNS for
your domain already pointed at it,

```sh
curl -fsSL https://raw.githubusercontent.com/jevido/the-bakery/main/src/lib/server/agent/install.sh | bash
bakery setup
bakery bootstrap --domain=<your-domain>
```

(`raw.githubusercontent.com` is plain source hosting, not CI — the same
distinction that lets `install.sh` itself `git clone` the repo below. No
GitHub Actions job publishes anything either of these commands depends on.)

This ends with `https://<domain>` serving the control plane over a real
certificate, dogfooding itself through its own deployment engine. See
`agent/README.md` and `phases/08-self-hosted-bootstrap` in the planning
repo for the full design.

At this point the running instance's own `bakery` app has no repo
connected yet — bootstrap seeds it with a working, already-built image
(see `phases/08-self-hosted-bootstrap/12-bootstrap-builds-local-image.md`)
but nothing to build from on future pushes. One manual browser step closes
that loop (the only step in this whole flow that can't be scripted, since
it's a real GitHub OAuth-style redirect): sign in, go to **Sources**,
click **Connect GitHub**, and install the app on this repo. Then open the
`bakery` app's **Deployments** tab — with no repo connected yet it shows a
repo picker instead of the usual build history — and connect the repo you
just installed. From that point on, every push to `main` goes through the
exact same webhook → build → registry → deploy pipeline as any other app,
no special-casing.

Deploying to something other than this flow (e.g. a different platform
entirely) still works via `@sveltejs/adapter-node` — see the
[SvelteKit adapter docs](https://svelte.dev/docs/kit/adapters). Either way,
production needs:

- Real Postgres (not the dev `compose.yaml` container)
- `BAKERY_DOMAIN` set, with wildcard DNS pointed at your infrastructure (apps get `<app>.<guild-slug>.<bakery-domain>`)
- A real container registry reachable from both the build worker and every host agent
- `ENCRYPTION_KEY`/`BETTER_AUTH_SECRET` from a real secret store, not `.env` — see the key-rotation notes in `src/lib/server/secrets/crypto.ts` if either ever needs to change

CI runs on every push via `.github/workflows/ci.yml`; the container image is published to GHCR on push to `main` only. (The `bakery` agent binaries and install shim are no longer published there — `install.sh` builds the binary from source itself now; see "The host agent" above.)
