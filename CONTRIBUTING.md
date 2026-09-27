# Contributing to The Bakery

Thanks for chipping in. This page gets you from a clone to a running stack
and to your first change.

## Run it

Requirements: Go 1.27, bun, the Wails CLI v3.0.0-beta.18
(`go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.18`),
Podman with `podman compose`, and Task. On Linux you also need WebKitGTK 6.0
and a Secret Service for the keyring (gnome-keyring or KWallet).

```sh
git clone https://github.com/jevido/the-bakery
cd the-bakery
task dev     # Postgres, migrations, API :4810, desktop app, website :4840
task seed    # in a second terminal: First Colony with ada, bram and cas
```

Sign in to the desktop app as `ada@bakery.test` with password `password`.
Before you push, run `task check` (format, lint, tests and type checks for
every unit; the API's feature tests need Postgres up). `task down` stops the
dev servers, `task db:down` stops Postgres.

## How the repository is laid out

`apps/` holds what a person opens (the desktop app, the website),
`services/` what runs unattended (the API), `packages/` code two or more of
those share, `infra/` environments and images, and `docs/domain/` the domain
model. Each unit has its own `README.md`. [`CLAUDE.md`](CLAUDE.md) has the
full rules; they apply to people and coding agents alike.

## How work is planned

Work is planned in phases, each a short list of tasks done in order. What
exists and what comes next is on the docs site:
[Not built yet](https://jevidocs.jevido.app/p/bakery/unfinished). For anything
bigger than a fix, open an issue first so we can agree on where it fits.

## Rules that matter most

- **Domain first.** The words in [`docs/domain/glossary.md`](docs/domain/glossary.md)
  are the words in the code: a guild is a guild, never a workspace, team or
  org. A new term goes into the glossary before it goes into code. A change to
  a context, aggregate, rule or event updates `docs/domain/` in the same change.
- **Contexts stay apart.** The API hosts the contexts identity, guilds and
  boards. One never reads another's tables or imports its internals; it uses
  what the other publishes.
- **Docs move with the code.** When a change alters what someone can do, how
  a unit runs, or an endpoint, update the matching pages on the
  [docs site](https://jevidocs.jevido.app/p/bakery), and `services/api/openapi.yaml`
  for API changes.
- **Commits** are conventional and scoped by unit: `feat(desktop): ...`,
  `fix(api): ...`, `chore(infra): ...`, `docs: ...`.
- **Podman, not Docker**, in scripts and docs. No secrets in the repository.

## Shipping

A push to `next` deploys the pre-production site and API, a merge to `main`
deploys production (see [`infra/README.md`](infra/README.md)). A tag
`desktop-vX.Y.Z` builds and publishes a desktop release, which installed apps
pick up and install themselves.

## Where to ask

Open an issue on [GitHub](https://github.com/jevido/the-bakery/issues).
