# dev

Local development stack.

- `compose.yml` runs the backing services with
  `podman compose -f infra/dev/compose.yml` (`task db:up` / `task db:down`).
  Today that is one Postgres 18 on `127.0.0.1:4820`, user, password and
  database `bakery`, data in the named volume `postgres18`. `task dev` starts it
  first. Bind every port to
  `127.0.0.1` and pick it from the unit's block in the port table in
  `CLAUDE.md`. Local credentials in it are not secrets.
- `down.sh` stops every local dev server by the ports it listens on;
  `task down` runs it with `DEV_PORTS` from the root `Taskfile.yml`.
