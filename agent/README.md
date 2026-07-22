# bakery

Lightweight Go binary that runs on a customer's host, checks in with the
Bakery control plane over outbound HTTP, and (from Phase 04 onward)
executes container lifecycle commands. It holds no local decision-making
authority — the control plane decides, the agent reports and executes.

This is a standalone Go module, independent of the SvelteKit app's
`package.json`/tooling. It's a single binary (`bakery`) with subcommands;
`daemon` is the only one implemented so far — `setup`, `join`, and
`bootstrap` are added across the rest of Phase 08.

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
