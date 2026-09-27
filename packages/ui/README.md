# ui

`@bakery/ui`: The Bakery's look, shared by the desktop app
(`apps/desktop/frontend`) and the website. A bun workspace package with no
build step: consumers compile the Svelte source with their own Vite config.

```ts
import '@bakery/ui/theme.css' // once, at the app's entry
import { Panel, Button, TextField } from '@bakery/ui'
```

**Only presentational components.** No API calls, no bindings, no domain
types. A component that knows about tasks or guilds stays in its app.

## Components

- `Panel` — framed box with an optional title bar (`title`, `actions` snippet).
- `Button` — `variant` `plain` (default), `confirm` or `danger`; takes every
  `<button>` attribute.
- `TextField` — labelled input with `bind:value`; takes every `<input>` attribute.

## Tokens (`src/theme.css`)

| Group | Tokens |
| ----- | ------ |
| Ground | `--bg`, `--bg-deep` |
| Panels | `--panel`, `--panel-raised`, `--panel-title`, `--panel-inset`, `--frame`, `--frame-dim`, `--shadow-inner`, `--shadow-outer` |
| Text | `--text`, `--text-dim`, `--text-faint` |
| Accents | `--olive`, `--olive-bright`, `--rust`, `--rust-bright`, `--steel`, `--steel-bright` |
| Type | `--font-display` (Pixelify Sans), `--font-body` (Inter) |
| Shape | `--radius`, `--gap` |

Fonts are bundled (Pixelify Sans from `@fontsource`, Inter in `src/fonts/`),
so apps work offline.

`task ui:check` runs svelte-check.
