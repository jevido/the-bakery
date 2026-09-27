# prod

Production.

| | |
| --- | --- |
| Coolify | https://coolify.jevido.app, project "the bakery", environment `production` |
| Deploys from | branch `main` |
| Web | https://bakery.jevido.app |
| API | https://api.bakery.jevido.app, health check `/api/health` |
| Database | Postgres 18, its own Coolify resource, daily backups |
| Desktop releases | GitHub releases tagged `desktop-vX.Y.Z`; release builds default to the prod API |

Per resource (port, environment variable names, deploy steps, backups, how to
roll back, redeploying by hand) is filled in when the resources exist. Real
values live in Coolify, never in the repo.
