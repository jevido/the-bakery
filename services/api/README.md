# api

The Bakery's JSON API: the source of truth for members, guilds, boards and
tasks. A Goravel v1.18 app on Postgres 17, listening on `127.0.0.1:4810`.

## Run

```sh
cp .env.example .env                 # once; then:
go run . artisan key:generate        # once
go run . artisan jwt:secret          # once
task db:up                           # from the repo root: dev Postgres on :4820
task api:dev                         # go run .
curl -s 127.0.0.1:4810/api/health    # {"ok":true}
```

Other tasks: `task api:check` (gofmt, go vet, go test), `task api:migrate`.
The feature tests in `tests/feature` run against the dev database and skip
when it is not reachable; run `task db:up` and `task api:migrate` first.
Artisan runs as `go run . artisan ...`.

## Layout

Everything Goravel generates (`app/`, `bootstrap/`, `config/`, `database/`,
`routes/`, `tests/`) stays where the framework expects it. The bounded
contexts live in `contexts/<context>/`, each with the same four packages:

| Package   | Holds                                   | May import                     |
| --------- | --------------------------------------- | ------------------------------ |
| `domain/` | Aggregates, invariants, domain events   | The standard library only      |
| `app/`    | Use cases that orchestrate the domain   | `domain/`, interfaces it owns  |
| `infra/`  | ORM repositories, other adapters        | `domain/`, `app/`, Goravel     |
| `http/`   | Controllers and request validation      | `app/`, Goravel                |

The context's root package (`contexts/<context>`) is its published contract
and wiring: routes, middleware and the few functions other contexts may call.
A context never imports another context's internals. It uses only what that
context publishes (see `docs/domain/context-map.md`).
