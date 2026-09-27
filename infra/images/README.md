# images

Build files, one directory per deployed resource:

```
infra/images/<resource>/Containerfile
infra/images/<resource>/Containerfile.dockerignore
infra/images/shared/            # scripts used by more than one image
```

Images build from the repository root as context, so a Containerfile can copy
shared files. Reproduce a build locally with
`podman build -f infra/images/<resource>/Containerfile .`.

The same image runs in `next` and `prod`. Never bake environment-specific
values (URLs, credentials, feature flags) into an image; read them from the
environment at runtime.

## api

`infra/images/api/Containerfile`: a static Go binary (`CGO_ENABLED=0`) on
Alpine as a non-root user (Coolify's health check needs `/bin/sh` and
`wget` in the image), listening on `0.0.0.0:4810`. Configuration comes
only from the environment (`APP_KEY`, `JWT_SECRET`, `DB_*`, ...); no `.env` is
baked in. With `MIGRATE_ON_START=true` (the image default) the binary runs the
migrations before it serves and exits non-zero if one fails, so a bad release
never takes traffic. `entrypoint.sh` runs it as a child: on SIGTERM it sends
SIGUSR1 (health turns 503, requests are still served), waits
`DRAIN_SECONDS` (8) so the proxy stops routing here, then sends SIGTERM, on
which the API finishes in-flight requests (bounded by the 3 s request
timeout) and exits.
`LOG_PRINT=true` sends log lines to stdout. `task image:api` builds it.

## web

`infra/images/web/Containerfile`: `bun install --frozen-lockfile` at the
workspace root, `bun run build` in `apps/web`, then Caddy serving `dist/` on
port 8080. Every unknown path falls back to `index.html` (so `/desktop` works
as a deep link); `/assets/*` are cached for a year, everything else is
revalidated. `VITE_API_URL` is a build argument, compiled into the site, so
next and prod each build their own. `/healthz` is the container's health
check; `entrypoint.sh` drains it the same way as the API before stopping
Caddy. `task image:web` builds it.
