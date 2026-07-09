# bakery-agent

Lightweight Go binary that runs on a customer's host, checks in with the
Bakery control plane over outbound HTTP, and (from Phase 04 onward)
executes container lifecycle commands. It holds no local decision-making
authority — the control plane decides, the agent reports and executes.

This is a standalone Go module, independent of the SvelteKit app's
`package.json`/tooling.

## Build

```sh
cd agent
go build -o bakery-agent .
```

## Run

```sh
BAKERY_TOKEN=bkry_host_... BAKERY_URL=https://your-bakery.example.com ./bakery-agent
```

Or via flags:

```sh
./bakery-agent -token bkry_host_... -url https://your-bakery.example.com
```

| Flag        | Env             | Default | Description                          |
| ----------- | --------------- | ------- | ------------------------------------ |
| `-token`    | `BAKERY_TOKEN`  | —       | Per-host bearer token (required)     |
| `-url`      | `BAKERY_URL`    | —       | Control-plane base URL (required)    |
| `-interval` | —               | `45s`   | Check-in interval                    |

Both the token and URL must be supplied (flag or env); the agent exits
immediately at startup if either is missing.

## Behavior

- Checks in immediately on startup, then every `-interval`.
- POSTs to `<BAKERY_URL>/api/v1/agent/checkin` with `Authorization: Bearer
  <token>`.
- Network errors and non-200 responses (e.g. an invalid/revoked token) are
  logged and retried on the next interval — the process never crashes on a
  failed check-in.
- Exits cleanly on `SIGINT`/`SIGTERM`.

## Metrics

Real CPU/mem/disk/podman metrics collection is implemented in a later task
(09); this scaffold reports zero values / placeholders to prove the loop
and HTTP contract work end-to-end.
