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
- `internal/api/` — typed client for the API. `internal/session/` — the
  signed-in member and token storage.
- `frontend/src/lib/bindings.ts` — the one import point for generated bindings.
- `frontend/src/screens/` — full screens (`Login`).
- `frontend/src/theme.css` — the base theme as CSS custom properties. Use its
  tokens, not raw colours.
- `frontend/src/components/` — `Panel` (framed box with a title bar),
  `Button` (`plain`, `confirm`, `danger`), `TextField`.
- `build/` — Wails' generated build and packaging tasks per platform.
