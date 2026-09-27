# next

The pre-production environment. Changes land here before `prod`, running the
same images with their own configuration.

| | |
| --- | --- |
| Coolify | https://coolify.jevido.app, project "the bakery", environment `next` |
| Deploys from | branch `next` |
| Web | https://next.bakery.jevido.app |
| API | https://api.next.bakery.jevido.app, health check `/api/health` |
| Database | Postgres 18, its own Coolify resource, daily backups at 03:00 |

## Resources

| Resource | Coolify name | Build | Port | Health check |
| -------- | ------------ | ----- | ---- | ------------ |
| API | `bakery-api-next` | `infra/images/api/Containerfile` | 4810 | `/api/health` |
| Website | `bakery-web-next` | `infra/images/web/Containerfile`, build arg `VITE_API_URL` | 8080 | `/healthz` |
| Database | `bakery-db-next` | `postgres:18-alpine`, internal only | 5432 | Coolify's own |

Both apps build from the GitHub repo through the GitHub App, base directory
`/`, and deploy automatically on a push to `next` when a file under their
watch paths changes (API: `services/api/**`, `infra/images/api/**`; website:
`apps/web/**`, `packages/ui/**`, `infra/images/web/**`, `package.json`,
`bun.lock`). Environment variable names are in [`env.example`](env.example).

## Deploying

1. Push to `next`. Coolify builds the image.
2. The new API container migrates the database before it listens; a failed
   migration stops it.
3. Coolify waits for the health check (every 2 s, 2 retries, 10 s start
   grace), then swaps traffic to the new container and stops the old one
   (rolling update). An unhealthy new container is dropped and the old one
   keeps serving.
4. The old container drains before it stops: its health check turns 503 so
   Traefik takes it out of rotation, requests that still reach it are served,
   and it exits about 8 s later (`entrypoint.sh` in each image). Both HTTPS
   routers also carry a Traefik retry middleware (`bakery-retry`, 3 attempts)
   in the app's custom labels.

**Roll back:** in Coolify, open the app → Deployments, and redeploy the
previous commit. Migrations only move forward, so a rollback past a
migration needs a fix-forward instead.

**Redeploy by hand:** Coolify → the app → Redeploy, or
`POST /api/v1/deploy?uuid=<app uuid>` with the Coolify API token.
