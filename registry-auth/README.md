The `htpasswd` file the `registry` compose service reads its auth from is
generated, not committed (see `.gitignore`) — it's derived from the
`BAKERY_REGISTRY_{PUSH,PULL}_{USERNAME,PASSWORD}` values in `.env`, which
are secrets.

## Setup

1. Set `BAKERY_REGISTRY_PUSH_USERNAME`/`_PASSWORD` and
   `BAKERY_REGISTRY_PULL_USERNAME`/`_PASSWORD` in `.env`.
2. Run `bun scripts/generate-registry-htpasswd.ts` to write `htpasswd` here
   (bcrypt-hashed via Bun's built-in `Bun.password` — the registry's
   htpasswd auth handler only accepts bcrypt).
3. `docker compose up` (or restart the `registry` service if it's already
   running) to pick up the file.

Both users currently have identical (full push+pull) access — htpasswd auth
has no concept of read-only users, so this is authentication (deny
anonymous), not authorization (scope what an authenticated user can do).
Splitting "push" and "pull" credentials now is about having separate,
rotatable secrets for the build worker vs. host agents, not real isolation.
Real per-host pull scoping (so a compromised host token can't pull unrelated
apps' images) needs a token auth server in front of the registry instead of
htpasswd — that's explicitly deferred to Phase 07.

## Manually verifying an image

```sh
podman login $BAKERY_REGISTRY_HOST -u $BAKERY_REGISTRY_PULL_USERNAME -p $BAKERY_REGISTRY_PULL_PASSWORD
podman pull $BAKERY_REGISTRY_HOST/<organizationId>/<appId>:<commitSha>
```

Pulling without `podman login` first (or with wrong credentials) fails with
an auth error — that's the registry correctly rejecting anonymous/invalid
access.
