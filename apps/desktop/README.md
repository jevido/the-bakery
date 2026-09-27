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
- `boardsservice.go` — `BoardsService`: guilds, boards and tasks, each call
  made with the member's token. A refused token ends the session.
- `internal/api/` — typed client for the API. `internal/session/` — the
  signed-in member and token storage.
- `frontend/src/lib/bindings.ts` — the one import point for generated bindings.
- `frontend/src/lib/colony.svelte.ts` — the open guild and board; moves are
  shown right away and undone by reloading the board if the API refuses.
- `frontend/src/screens/` — full screens (`Login`, `Colony`).
- `frontend/src/components/BoardView.svelte` — the four columns and HTML5
  drag and drop; `TaskCard.svelte` — rename (double-click) and delete.

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
builds on Linux, Windows and macOS with the version and the prod API URL
(`https://api.bakery.jevido.app`) set through ldflags, and publishes a GitHub
release with:

| File | For |
| ---- | --- |
| `the-bakery-X.Y.Z-linux-x86_64.AppImage` | Linux install and updates |
| `the-bakery_X.Y.Z_amd64.deb` | Linux install via apt/dpkg |
| `the-bakery-X.Y.Z-windows-amd64-installer.exe` | Windows install (NSIS) |
| `the-bakery-X.Y.Z-windows-amd64.zip` | Windows updates (the exe) |
| `the-bakery-X.Y.Z-darwin-universal.zip` | macOS install and updates (unsigned `.app`) |
| `manifest.json` | the signed Wails update manifest |

Update files are signed with an Ed25519 key. The private key is the
`BAKERY_UPDATE_KEY` Actions secret (the maintainer keeps a copy outside the
repo); the public key is `build/updater.pub`, which the app checks updates
against. A local package: `task desktop:package` (Linux: AppImage, deb, rpm,
Arch package in `bin/`).
