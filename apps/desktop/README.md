# desktop

The Bakery's desktop app: a Wails v3 window with a Svelte 5 + TypeScript
frontend, styled like a colony sim. It is a client of `services/api` and
holds no domain data of its own; the frontend never calls the API directly,
Go services do.

## Run

Requirements: Go 1.27, bun, `wails3` v3.0.0-beta.18, and on Linux the WebKitGTK
dev packages Wails needs (`wails3 doctor` lists what is missing).

```sh
task desktop:dev      # from the repo root: window + Vite on 127.0.0.1:4830 with hot reload
task desktop:check    # bindings, svelte-check, frontend build, gofmt, go vet, go test
```

The app talks to the API at `BAKERY_API_URL` (default
`http://127.0.0.1:4810`). The member's token is kept in the OS keyring
(service `the-bakery`, one entry per API URL), so a relaunch skips the login;
"Clock out" removes it. Without a keyring (Linux with no Secret Service such
as gnome-keyring or KWallet) it falls back to a `0600` file in the user config
directory and logs a warning.

`@wailsio/runtime` is pinned to the same version as the Go module
(`github.com/wailsapp/wails/v3`); upgrade both together.

## Layout

- `main.go` — the application, its services and its window.
- `sessionservice.go` — `SessionService`, the Wails service the frontend signs
  in and out with (`Register`, `Login`, `Logout`, `Me`).
- `updateservice.go` — `UpdateService`: self-update from GitHub releases (see
  below).
- `boardsservice.go` — `BoardsService`: guilds, boards and tasks, each call
  made with the member's token. A refused token ends the session.
- `taskservice.go` — `TaskService`: one task in full for the task panel
  (title, description, subtasks, comments, activity), the same way.
- `liveservice.go` — `LiveService`: keeps the open board's event stream
  (`internal/api/events.go`) open from Go and passes each event to the
  frontend as the Wails event `board:event`, with `board:status` for the
  indicator and `board:resync` after a reconnect (backoff 1 s up to 30 s).
- `internal/api/` — typed client for the API. `internal/session/` — the
  signed-in member and token storage.
- `frontend/src/lib/bindings.ts` — the one import point for generated bindings.
- `frontend/src/lib/colony.svelte.ts` — the open guild and board; moves are
  shown right away and undone by reloading the board if the API refuses.
  Live events are applied in place where they carry enough (moves, deletes,
  renames) and otherwise refetch the board.
- `frontend/src/lib/task.svelte.ts` — the task open in the panel; ticks and
  subtask moves are shown right away and undone if the API refuses.
- `frontend/src/lib/markdown.ts` — markdown for descriptions and comments: raw
  HTML is shown as text, the result is sanitised with DOMPurify, links open in
  the browser.
- `frontend/src/lib/activity.ts` — how activity reads: `activityLine(entry)`
  builds every sentence, `timeAgo` the relative times.
- `frontend/src/screens/` — full screens (`Login`, `Colony`).
- `frontend/src/components/ColumnHeader.svelte` — a column's title bar: drag
  handle, name (double-click to rename), task count, and a menu whose Delete is
  off for a column with tasks or the last one.
- `frontend/src/components/BoardView.svelte` — the board's columns (scrolling
  sideways when they do not fit, "+ Column" at the end, dragged by their header
  with their own drag type), the live indicator, and HTML5
  drag and drop; `TaskCard.svelte` — a card: double-click opens the task
  panel, a badge counts done subtasks, delete after confirming;
  `TaskPanel.svelte` — the side panel: title, markdown description with
  edit and preview, the subtask checklist, and Comments and Activity tabs
  (only your own comments can be edited or deleted).

## Driving the app from an agent

`EXTRA_TAGS=mcp task desktop:dev` compiles in Wails' MCP server on
`127.0.0.1:9099` (`WAILS_MCP_PORT` to change it). Its tools (`js_eval`,
`mouse_drag`, `keyboard_type`, ...) let an agent click, type and drag in the
running window.
- The theme and the base components (`Panel`, `Button`, `TextField`) come
  from [`@bakery/ui`](../../packages/ui); use its tokens, not raw colours.
  The frontend is a bun workspace member: run `bun install` anywhere in the
  repo, the lockfile is `/bun.lock`.
- `build/` — Wails' generated build and packaging tasks per platform.

## Releases

A tag `desktop-vX.Y.Z` runs `.github/workflows/desktop-release.yml`: it
builds on Linux (Windows and macOS builds are left out for now; their Wails
build tasks are still in `build/`) with the version and the prod API URL
(`https://api.bakery.jevido.app`) set through ldflags, and publishes a GitHub
release with:

| File | For |
| ---- | --- |
| `the-bakery-X.Y.Z-linux-x86_64.AppImage` | Linux install and updates; needs the system's GTK 4 and WebKitGTK 6.0 (`webkitgtk-6.0` on Arch and Fedora, `libwebkitgtk-6.0-4` on Debian and Ubuntu) |
| `the-bakery_X.Y.Z_amd64.deb` | Linux install via apt/dpkg |
| `manifest.json` | the signed Wails update manifest |

The update file is signed with an Ed25519 key. The private key is the
`BAKERY_UPDATE_KEY` Actions secret (the maintainer keeps a copy outside the
repo); the public key is `build/updater.pub`, which the app checks updates
against. A local package: `task desktop:package` (Linux: AppImage, deb, rpm,
Arch package in `bin/`).

## Self-update

Release builds check
`https://github.com/jevido/the-bakery/releases/latest/download/manifest.json`
five seconds after start and then every 6 hours. The Wails updater downloads
the file for this platform and verifies its digest and Ed25519 signature
against the embedded `build/updater.pub`; anything that does not verify is
refused and nothing changes. The frontend's `UpdatePanel` listens to the
updater's `wails:updater:*` events and offers "Install now" or "Later".

On Linux the verified AppImage replaces the running `$APPIMAGE` file and the
new version is started (the stock updater would try to write into the
AppImage's read-only mount). On Windows and macOS the updater swaps the
executable or `.app` and restarts. Dev builds (version `dev`) never update.
`BAKERY_UPDATE_MANIFEST_URL` points a build at another manifest, for testing.
