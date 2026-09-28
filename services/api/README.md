# api

The Bakery's JSON API: the source of truth for members, guilds, boards,
tasks and agents (contexts identity, guilds, boards and agents). A Goravel v1.18 app on Postgres 18, listening on `127.0.0.1:4810`.

## Run

```sh
cp .env.example .env                 # once; then:
go run . artisan key:generate        # once
go run . artisan jwt:secret          # once
task db:up                           # from the repo root: dev Postgres on :4820
task api:dev                         # go run .
curl -s 127.0.0.1:4810/api/health    # {"ok":true}
```

Other tasks: `task api:check` (gofmt, go vet, go test), `task api:migrate`,
`task api:seed` (members ada, bram and cas `@bakery.test`, password
`password`, in the guild First Colony; safe to run again).
The feature tests in `tests/feature` run against the dev database and skip
when it is not reachable; run `task db:up` and `task api:migrate` first.
Artisan runs as `go run . artisan ...`.

## Two ways to sign in

The desktop app sends `Authorization: Bearer <token>`. The website signs in
through `/api/web/login` and `/api/web/register`, which put the same token in
an httpOnly cookie `bakery_session` (host-only, SameSite=Lax, Secure outside
`APP_ENV=local`). `RequireMember` accepts either. A cookie request that
changes something must carry `X-Bakery-Web: 1`, which cross-site forms cannot
send; CORS (`config/cors.go`) lets only `WEB_ORIGIN` call with credentials.

## Operators (the console)

`/api/console/*` is for operators only: accounts of the moderation context,
apart from members, with their own JWT guards (`operator`, and
`operator_challenge` between the password and the code; set
`OPERATOR_JWT_SECRET` to sign them apart from members' tokens). Signing in
takes two steps, `POST /api/console/login` (email and password → a
five-minute challenge) and `POST /api/console/totp/verify` (challenge and a
TOTP code → a two-hour token), each limited to five tries a minute per
client. There is no sign-up:

```sh
go run . artisan operator:create owner@example.com   # asks for a password; prints an otpauth:// URL
go run . artisan operator:confirm owner@example.com 123456
```

`operator:create` reads the password from stdin when stdin is not a
terminal. An operator signs in only after confirming a first code.

Sensitive actions go to the platform's **audit log** (`audit_entries`,
append-only): operator sign-ins, personal tokens made and revoked, guilds
archived and restored, members removed from a guild, and (later) sanctions
and reports. Identity and guilds record through a one-method port that
moderation's wiring sets (`SetAuditRecorder`); `moderation.RememberIP` puts
the client IP in every request's context. Operators read it at
`GET /api/console/audit` (filters `actor_kind`, `actor_id`, `action`,
`target_kind`, `target_id`, `from`, `to`; paging with `before` and `limit`).

## MCP server

`/mcp` serves MCP over streamable HTTP (stateless, JSON replies) with the
official Go SDK (`github.com/modelcontextprotocol/go-sdk`). Clients sign in
with a personal token (`Authorization: Bearer bky_…`). The server lives in
`mcp/`; it has no domain of its own: every tool calls a function that the
guilds or boards context publishes from its root package, so it follows the
same membership and archive rules as REST.

- Read: `list_guilds`, `list_boards`, `get_board`, `get_task` (subtasks and
  the last 20 comments), `list_work_types`, `list_agents` (yours, or those
  shared with a guild), `get_agent`.
- Write: `create_board`, `create_task`, `update_task`, `move_task`,
  `expand_task`, `set_subtask_done`, `add_comment`, `delete_task` (marked
  destructive, so clients ask first).

A refused call (not a member, archived guild, bad column) comes back as a
tool error with the same message REST gives. Connect Claude Code with:

```sh
claude mcp add --transport http bakery-dev http://127.0.0.1:4810/mcp --header "Authorization: Bearer bky_…"
```

## API description

`openapi.yaml` describes every endpoint as built. It is also the body of the
jevidocs page `api/reference`; update both when an endpoint changes.

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
