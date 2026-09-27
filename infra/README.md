# infra

Container, environment and deploy configuration. No application code.

| Directory            | What goes there                                                     |
| -------------------- | ------------------------------------------------------------------- |
| [`dev/`](dev)        | Local development: compose stack for backing services, helper scripts. |
| [`next/`](next)      | The pre-production environment: config that differs from prod.      |
| [`prod/`](prod)      | Production: config, domains, runbook notes.                          |
| [`images/`](images)  | Containerfiles, one directory per deployed resource, shared by next and prod. |

**One image, many environments.** An image is built once from
`infra/images/<resource>/Containerfile` and runs unchanged in `next` and then
`prod`. Only configuration (environment variables, domains, replicas) differs
between environments, and that lives in `infra/next/` and `infra/prod/`. If
something only works in one environment, it belongs in config, not in the
image.

## Environments

| Environment | Branch | Web | API | Database |
| ----------- | ------ | --- | --- | -------- |
| `dev` | any, local | http://127.0.0.1:4840 | http://127.0.0.1:4810 | Postgres 18 in `infra/dev/compose.yml` on `127.0.0.1:4820` |
| `next` | `next` | https://next.bakery.jevido.app | https://api.next.bakery.jevido.app | Postgres 18 on Coolify, its own resource |
| `prod` | `main` | https://bakery.jevido.app | https://api.bakery.jevido.app | Postgres 18 on Coolify, its own resource, daily backups |

All of it runs on Coolify at https://coolify.jevido.app, in the Coolify
project "the bakery": environment `production` for prod, `next` for next.
Every domain above resolves to that server. A push to `next` deploys next; a
merge to `main` deploys prod.

Desktop releases are GitHub releases on `jevido/the-bakery` tagged
`desktop-vX.Y.Z`. Release builds talk to the prod API by default;
`BAKERY_API_URL` overrides it.
