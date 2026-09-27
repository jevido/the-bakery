# CLAUDE.md

Guidance for coding agents (and people) working in this repository.

## What this is

The Bakery is a desktop workbench for the Claude CLI where guilds plan and run
their work. Read `README.md` for the overview and `docs/domain/` for the domain
model and why it is shaped the way it is.

- **Core domain: boards** — planning work as tasks on boards.
- **Supporting: guilds** — who works together.
- **Generic: identity** — accounts and authentication.

`services/api` hosts all three contexts. `apps/desktop` is a client of the API
only and holds no domain data of its own.

Tone: the desktop app feels like RimWorld (a colony-sim UI; agents are
configured like RimWorld colonists). The word is **guild**, never
workspace, org or team.

## Where things go

| Directory   | Rule                                                                  |
| ----------- | --------------------------------------------------------------------- |
| `apps/`     | A person opens it and interacts with it.                              |
| `services/` | Runs unattended; no person interacts with it directly.                |
| `packages/` | Code used by **two or more** apps/services. Not before.               |
| `infra/`    | Environments (`dev`, `next`, `prod`) and images. No application code. |
| `docs/`     | The domain model: language, contexts, and why. Never how code works.  |

- Each unit is one directory directly under its category, e.g.
  `services/api`, `apps/web`. Keep a short `README.md` in each unit saying
  what it is and how to run it.
- Each unit owns its own dependency manifest and tooling. Register it with the
  root workspace (if the language has one) and add its tasks to the root
  `Taskfile.yml`.
- Start code inside the unit that needs it. Move it to `packages/` only when a
  second consumer actually appears. `packages/` holds technical code
  (clients, UI kits, tooling), never a context's domain model.

## Stack rules

- **Go 1.27** (via mise), `gofmt` + `go vet`. Load the `go` skill for Go code.
- **`services/api`**: Goravel v1.18, run artisan as `go run . artisan ...`.
  Postgres 17.
- **`apps/desktop`**: Wails v3 (`wails3` CLI, v3.0.0-beta.18). Load the
  `wails` skill and use only its v3 section; v2 APIs do not exist here.
- **Frontend**: Svelte 5 runes + TypeScript, bun as package manager, Vite.
  Load `svelte-core-bestpractices` / `svelte-code-writer` for `.svelte` files.
- **The desktop app talks to the API from Go** (Wails services), never `fetch`
  from the webview, so the token stays in Go and the OS keyring.
- **Podman**, not Docker, in scripts and docs (`podman compose`,
  `Containerfile`).

## Commands

```sh
task                 # list tasks
task dev             # Postgres, migrations, then every unit's dev server
task seed            # seed the dev database (ada@bakery.test / password)
task check           # format, lint, test and type-check every unit
task down            # stop every dev server this repo started (not Postgres)
task db:down         # stop Postgres
```

**Local ports:** range 48xx, each unit its own decade, with strict port
binding so a clash fails loudly instead of silently moving.

| Port | Unit                               |
| ---- | ---------------------------------- |
| 4810 | `services/api`                     |
| 4820 | Postgres (`infra/dev/compose.yml`) |
| 4830 | `apps/desktop` Vite dev server     |
| 4840 | `apps/web`                         |

New units take the next free decade; add them here and, for dev servers, to
`DEV_PORTS` in the root `Taskfile.yml` so `task down` stops them. Containers
(Postgres) stay out of `DEV_PORTS`; `task db:down` stops them.

## Conventions

- **Commits:** conventional commits scoped by unit, e.g.
  `feat(web): ...`, `fix(api): ...`, `chore(infra): ...`, `docs: ...`.
- **`.gitignore`:** anchor patterns (`/bin/`, not `bin`).
- **Secrets:** never in the repo. Commit a `.env.example` when a unit needs
  config.
- **Environments:** `dev` (local), `next` (pre-production), `prod`. Each has
  a directory in `infra/`. Changes reach `next` before `prod`.
- **Images:** each deployed unit gets `infra/images/<resource>/Containerfile`
  (+ `Containerfile.dockerignore`), built from the repo root. One image runs
  unchanged in `next` and `prod`; only config differs, and it lives in
  `infra/next/` and `infra/prod/`. See `infra/README.md`.
- **Comments** explain non-obvious *how* and *why here*. Longer reasoning goes
  in the context's document under `docs/domain/contexts/`.

## Domain-driven design

Every project is built domain-first. The model lives in `docs/domain/` and
the code follows it.

- **Ubiquitous language.** Names in code (types, functions, modules, tables,
  events, endpoints) use the terms in `docs/domain/glossary.md`, in the same
  meaning. No technical synonyms (`Manager`, `Data`, `Info`, `Handler` for a
  domain concept). A new or changed term goes in the glossary first.
- **Bounded contexts are modules, not services.** A unit in `apps/` or
  `services/` hosts one or more contexts, each as its own top-level module
  named after the context. Split a context into its own service only when it
  needs to deploy or scale on its own, and update the context map when you do.
- **Boundaries are hard.** A context never imports another context's
  internals, reads its tables, or shares its domain types. It talks to other
  contexts through their published contract (a module interface, an API, or
  domain events) and translates what it receives into its own language.
- **Dependency rule.** Inside a context, the domain model depends on nothing:
  no framework, database, HTTP, clock or other I/O. Application code (use
  cases) orchestrates the domain; infrastructure (persistence, transport,
  external services) depends inward on it, never the other way. Folder names
  are up to each unit; the direction is not.
- **Aggregates guard invariants.** State changes go through the aggregate
  root, which rejects anything that breaks its rules. One transaction changes
  one aggregate; coordinate across aggregates with domain events.
- **Keep docs and code in step.** When a change adds or alters a context, an
  aggregate, an invariant, an event or a relationship, update
  `docs/domain/` in the same change. When a choice about the model is costly
  to reverse, record the reasoning in that context's "Why it's shaped this
  way" section.

## Documentation (jevidocs)

The Bakery's public and contributor documentation lives in the jevidocs
project **`bakery`** (https://jevidocs.jevido.app/p/bakery), edited through
the `jevidocs` MCP server (`get_page_tree`, `read_page`, `create_page`,
`update_page`). `docs/domain/` is the model for people changing the code;
jevidocs is for people using The Bakery or chipping in.

- **Same change, same time.** When a change alters what a person can do, how
  a unit is built or run, an API endpoint or MCP tool, or a glossary term,
  update the matching jevidocs pages before calling the work done. Say which
  pages you changed.
- **Describe what exists.** Pages describe the running system. Planned work
  goes on the "Not built yet" page only, never mixed into a guide as if it
  works.
- **Read before writing.** `read_page` first, then `update_page` with the
  whole new body; never overwrite a page you have not read in this session.
- **Same language as the code.** Terms follow `docs/domain/glossary.md`; the
  jevidocs glossary pages mirror it in plain words.
- If the MCP call fails (no token, server down), say so and list the pages
  that still need the update, instead of skipping it silently.

## Working style for agents

- Do what the task asks, in the unit and context it concerns.
- Read the context's document in `docs/domain/contexts/` before changing its
  code. If the task does not fit any existing context, say so instead of
  guessing where it goes.
- Verify before claiming done: build, test, and run the thing where possible.
  Say plainly what you could not verify.
- Prefer existing libraries and patterns already in the repo over new ones.
