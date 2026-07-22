Agent release binaries served at `/releases/bakery-linux-<arch>`
(referenced by `/install.sh` when invoked with `--url` pointing at a
running instance) are manually uploaded here for now — no build/release
pipeline exists yet for this route specifically (every CI-built container
image bakes fresh ones in via the Containerfile's `agent-build` stage).

Files placed directly in this directory are gitignored, since committing
compiled binaries to the app repo isn't the long-term plan. Task 12
(agent self-update) replaces this with a real `agentRelease` table and
versioned download endpoint.

The separate, fixed-location copies published to GitHub Releases (used by
the generalized `install.sh` shim for bootstrapping a brand-new instance,
before any Bakery control plane exists to serve this route) are built by
`.github/workflows/ci.yml`'s `release-agent` job, not by anything in this
directory.
