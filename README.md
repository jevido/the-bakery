# The Bakery

The Bakery is a desktop workbench for the Claude CLI where guilds plan and run
their work. The Goravel API (`services/api`) is the source of truth; the
website (later) is the admin; a separate owner console (later) handles
moderation.

## Layout

| Directory   | What goes there                                                  |
| ----------- | ---------------------------------------------------------------- |
| `apps/`     | Things a person opens and interacts with (sites, apps, CLIs).    |
| `services/` | Things that run unattended (APIs, workers, schedulers).          |
| `packages/` | Code shared by two or more units. Not before.                    |
| `infra/`    | Environments (`dev`, `next`, `prod`) and images. No app code.    |
| `docs/`     | The domain model: glossary, context map, one doc per context.    |

Each unit is one directory under its category (`apps/<name>`,
`services/<name>`) with its own `README.md` and tooling.

## Getting started

Requirements:

- [Go](https://go.dev) 1.27 (e.g. via [mise](https://mise.jdx.dev))
- [bun](https://bun.sh)
- [Wails](https://v3.wails.io) CLI v3.0.0-beta.18:
  `go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.18`
  (`wails3 doctor` lists the system libraries it still needs)
- [Podman](https://podman.io) with `podman compose`
- [Task](https://taskfile.dev)
- On Linux, a Secret Service for the keyring (gnome-keyring or KWallet).
  Without one the desktop app keeps its token in a file and warns.

Then:

```sh
task dev    # Postgres, migrations, the API on :4810, the desktop app and the website on :4840
task seed   # in a second terminal: the "First Colony" guild and its board
```

`task dev` creates `services/api/.env` from `.env.example` with fresh keys
the first time. Sign in to the desktop app as `ada@bakery.test` with password
`password` (bram@ and cas@ work too), open First Colony, and drag tasks on
"Getting settled".

```sh
task            # list tasks
task check      # format, lint, test and type-check every unit
task down       # stop the API and the desktop app (Postgres keeps running)
task db:down    # stop Postgres
```

Want to help? Start with [`CONTRIBUTING.md`](CONTRIBUTING.md). Conventions for
people and coding agents live in [`CLAUDE.md`](CLAUDE.md).
The project is built with domain-driven design; the model lives in
[`docs/domain`](docs/domain).
