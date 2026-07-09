Agent release binaries served at `/releases/bakery-agent-linux-<arch>`
(referenced by `/install.sh`) are manually uploaded here for now — no
build/release pipeline exists yet.

Files placed directly in this directory are gitignored, since committing
compiled binaries to the app repo isn't the long-term plan. Task 12
(agent self-update) replaces this with a real `agentRelease` table and
versioned download endpoint.
