Agent release binaries served at `/releases/bakery-linux-<arch>`
(referenced by `install.sh` when invoked with `--url` pointing at a
running instance) land here at container build time — the Containerfile's
`agent-build` stage cross-compiles fresh ones into `static/releases/` on
every real build (the build-worker's own pipeline now, not CI; see
Phase 08 tasks 12/13), so a running instance always serves a binary that
matches its own exact deployed version.

Files placed directly in this directory are gitignored for local dev, since
committing compiled binaries to the app repo isn't the plan; only what the
build produces at image-build time is what a running instance actually
serves.

A truly fresh box (no `--url`, nothing running yet) doesn't hit this route
at all — `install.sh` builds the agent from source itself instead
(`git clone` + `go build`, Phase 08 task 11). There is no GitHub Releases
fallback anymore.
