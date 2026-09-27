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

`@wailsio/runtime` is pinned to the same version as the Go module
(`github.com/wailsapp/wails/v3`); upgrade both together.

## Layout

- `main.go` — the application and its window.
- `frontend/src/theme.css` — the base theme as CSS custom properties. Use its
  tokens, not raw colours.
- `frontend/src/components/` — `Panel` (framed box with a title bar),
  `Button` (`plain`, `confirm`, `danger`), `TextField`.
- `build/` — Wails' generated build and packaging tasks per platform.
