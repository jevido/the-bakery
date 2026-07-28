# Agent builder stage: cross-compiles the bakery binaries served at
# /releases/bakery-linux-<arch> (src/lib/server/agent/install.sh's
# download URL, when invoked with --url pointing at a running instance).
# Renamed from bakery-agent-linux-<arch> now that the binary itself is a
# `bakery` CLI with subcommands, not a flat bakery-agent (Phase 08 task 01).
# Building it here means every image always carries a fresh binary matching
# the agent source, with nothing committed to git.
FROM docker.io/library/golang:1.25 AS agent-build
WORKDIR /agent
COPY agent/go.mod ./
RUN go mod download
COPY agent/ ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/bakery-linux-amd64 . \
	&& CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o /out/bakery-linux-arm64 .

# Build stage: installs with bun (this project's package manager) and runs
# the SvelteKit adapter-bun build. Built and run with Podman, matching the
# rest of Bakery's own infra (agent Quadlet units, build-worker) — no Docker
# anywhere on the production box.
FROM docker.io/oven/bun:1.3.14-alpine AS build
WORKDIR /app

# better-sqlite3 (an optional peer dep pulled in transitively by better-auth/
# drizzle-orm's multi-dialect support, even though this app only ever uses
# Postgres) has a native addon that needs compiling on install. It ships
# linux-musl prebuilds, so this toolchain may never actually be invoked --
# kept anyway as a fallback for whenever a prebuild isn't available for the
# exact version/arch (build-base and linux-headers are the two pieces most
# commonly missing for node-gyp on alpine specifically, beyond python3/make/g++).
RUN apk add --no-cache python3 make g++ build-base linux-headers

COPY package.json bun.lock ./
COPY patches ./patches
RUN bun install --frozen-lockfile

COPY . .

# Freshly-built agent binaries land in static/releases/ (gitignored locally,
# per static/releases/README.md) so `bun run build` bundles them into
# build/client/releases/ alongside the rest of the static assets.
COPY --from=agent-build /out/bakery-linux-amd64 /out/bakery-linux-arm64 ./static/releases/

# `bun run build` eagerly validates every var declared in `src/env.ts`'s
# `defineEnvVars` (SvelteKit's explicitEnvironmentVariables) and aborts if
# any are missing — even though the *real* values must only ever be supplied
# at container run time (`env_file: .env` in compose.prod.yaml), never baked
# into the image. These placeholders exist only to satisfy that build-time
# check; they are not present in the runtime stage below and are discarded
# along with the rest of this build stage.
ENV DATABASE_URL="postgres://placeholder/placeholder" \
	ORIGIN="http://placeholder.invalid" \
	BETTER_AUTH_SECRET="placeholder-build-time-value-only" \
	GITHUB_CLIENT_ID="placeholder" \
	GITHUB_CLIENT_SECRET="placeholder" \
	GITHUB_APP_ID="placeholder" \
	GITHUB_APP_SLUG="placeholder" \
	GITHUB_APP_PRIVATE_KEY="placeholder" \
	GITHUB_WEBHOOK_SECRET="placeholder"
RUN bun run build

# Runtime stage. adapter-bun's output (`build/`) is not a self-contained
# bundle — its own build output externalizes bare Node-builtin imports
# (`fs`, `net`, `tls`, `crypto`, etc., pulled in via the `postgres` package)
# that are resolved from node_modules at request time, not inlined, the same
# way adapter-node's output did. Nearly everything this app needs at runtime
# (drizzle-orm, better-auth, postgres, svelte, @sveltejs/kit itself) is
# listed under package.json's devDependencies, not dependencies, so a
# `--production`/`--omit=dev` install here would silently strip out packages
# the running server actually requires. Copying the full node_modules the
# build stage already produced (identical to what `bun run dev` runs against
# locally) is the correct, safe choice — not an oversight.
#
# Same base as the build stage (not a separate Debian runtime image like the
# old adapter-node setup used) — both stages sharing one libc family means
# any native addon compiled during `bun install` above is guaranteed ABI-
# compatible here, with no cross-libc concern to manage at all.
FROM docker.io/oven/bun:1.3.14-alpine AS runtime
WORKDIR /app

ENV NODE_ENV=production
ENV HOST=0.0.0.0
ENV PORT=3000

# Baked in at build time (`--build-arg BAKERY_SELF_IMAGE=...`) so a running
# instance always knows its own exact image reference, down to the commit
# sha — read by the bootstrap endpoint (Phase 08 task 06) to create the
# control plane's own `app`/`build` rows without hardcoding any
# operator/fork-specific registry path. Set by `bakery bootstrap`'s own
# local build for the very first image (Phase 08 task 12), and by the real
# build-worker for every build after that (Phase 08 task 13) — never by CI.
# Empty by default (e.g. local `bun run dev`, where bootstrap doesn't
# apply) rather than a required env var, since it's meaningless there.
ARG BAKERY_SELF_IMAGE=""
ENV BAKERY_SELF_IMAGE=$BAKERY_SELF_IMAGE

COPY --from=build /app/build ./build
COPY --from=build /app/node_modules ./node_modules
COPY --from=build /app/package.json ./package.json

# drizzle.config.ts + the migrations themselves — not needed to serve
# requests, but included so `db:migrate` (drizzle-kit, already present in
# node_modules per the comment above) can run from this same published
# image via a one-off container/exec (Phase 08 task 05), with no separate
# repo checkout needed and zero risk of drift from what actually shipped.
COPY --from=build /app/drizzle.config.ts ./drizzle.config.ts
COPY --from=build /app/drizzle ./drizzle
COPY --from=build /app/src/lib/server/db ./src/lib/server/db

EXPOSE 3000
CMD ["bun", "build/index.js"]
