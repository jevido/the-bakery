# next

The pre-production environment. Changes land here before `prod`, running the
same images with their own configuration.

| | |
| --- | --- |
| Coolify | https://coolify.jevido.app, project "the bakery", environment `next` |
| Deploys from | branch `next` |
| Web | https://next.bakery.jevido.app |
| API | https://api.next.bakery.jevido.app, health check `/api/health` |
| Database | Postgres 18, its own Coolify resource |

Per resource (port, environment variable names, deploy steps) is filled in
when the resources exist. Real values live in Coolify, never in the repo.
