# bakery

Lightweight Go binary that runs on a customer's host, checks in with the
Bakery control plane over outbound HTTP, and (from Phase 04 onward)
executes container lifecycle commands. It holds no local decision-making
authority — the control plane decides, the agent reports and executes.

This is a standalone Go module, independent of the SvelteKit app's
`package.json`/tooling. It's a single binary (`bakery`) with subcommands:

| Subcommand                          | What it does                                                                                                                                                       |
| ----------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `bakery daemon`                     | The check-in loop (below) — what actually runs long-term, as a systemd `--user` unit.                                                                              |
| `bakery setup`                      | Idempotently installs rootless Podman + prerequisites, subuid/subgid ranges, the unprivileged-port sysctl, and firewall rules.                                     |
| `bakery join --token=... --url=...` | Enrolls this host against an already-running instance: writes the daemon's config, installs its systemd unit, sets up the per-host Caddy Quadlet.                  |
| `bakery bootstrap --domain=...`     | Stands up a brand-new, self-hosted instance from nothing — Postgres, the control plane itself, the first admin/guild/host/app — then hands off into `bakery join`. |

`setup`/`join`/`bootstrap` all run as (or re-exec into) a dedicated
`bakery` system user, created by `install.sh` (`src/lib/server/agent/install.sh`,
served at `<instance>/install.sh` when enrolling against a running instance;
built from source on the box itself, via `git clone` + `go build`, when
bootstrapping a box with nothing running yet — no GitHub Release involved
either way).

## Build

```sh
cd agent
go build -o bakery .
```

## Run

```sh
BAKERY_TOKEN=bkry_host_... BAKERY_URL=https://your-bakery.example.com ./bakery daemon
```

Or via flags:

```sh
./bakery daemon -token bkry_host_... -url https://your-bakery.example.com
```

| Flag        | Env            | Default | Description                       |
| ----------- | -------------- | ------- | --------------------------------- |
| `-token`    | `BAKERY_TOKEN` | —       | Per-host bearer token (required)  |
| `-url`      | `BAKERY_URL`   | —       | Control-plane base URL (required) |
| `-interval` | —              | `45s`   | Check-in interval                 |

Both the token and URL must be supplied (flag or env); the agent exits
immediately at startup if either is missing.

For backward compatibility with hosts installed before subcommands
existed, running with no subcommand (or a first argument that looks like a
flag, e.g. `./bakery -token=...`) defaults to `daemon`.

## Behavior

- Checks in immediately on startup, then every `-interval`.
- POSTs to `<BAKERY_URL>/api/v1/agent/checkin` with `Authorization: Bearer
<token>`.
- Network errors and non-200 responses (e.g. an invalid/revoked token) are
  logged and retried on the next interval — the process never crashes on a
  failed check-in.
- Exits cleanly on `SIGINT`/`SIGTERM`.

## Metrics

Each check-in reports real host CPU/mem/disk (`cpu.go`/`mem.go`/`disk.go`)
and Podman version/running-container-count (`podman.go`), plus per-container
CPU/mem and log tail for every Quadlet-managed unit, and on-host size for
every Bakery-managed volume (`volumes.go`).

## Commands

Pending `deploy`/`stop`/`restart`/`configureProxy` commands come back in the
check-in response and are executed in order (`commands.go`), each gated by
`checkRootless` — the agent refuses to touch containers at all if Podman
isn't running rootless.
