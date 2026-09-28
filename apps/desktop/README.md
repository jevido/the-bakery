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
- `agentsservice.go` — `AgentsService`: runs the agents sync on its own (at
  start, when a folder changes, when the window gets focus, every 5 minutes;
  one at a time, retrying with backoff while offline), reports it with the
  Wails events `agents:status` and `agents:changed`, and resolves conflicts
  (`local`, `server` or `both`).
- `workshopservice.go` — `WorkshopService`: this machine's board configs
  (`internal/workshop`): read, check and save `board.toml` in
  `~/.config/the-bakery/boards/<board-id>/` (`boards-<host>/` for an API other
  than the release one), open the folder, pick the repository. Nothing about
  a board config reaches the API. `components/BoardSettings.svelte` is its
  panel, opened from the gear in the board's header; the header says
  **Not linked** until a valid repository is saved.
- `internal/workshop/` — the workshop: board configs (`boardconfig.go`), a
  task's worktree on branch `bakery/<task-id>-<slug>` (`worktree.go`; a task
  keeps its first branch after a rename), and the run's skills placed in the
  worktree as `.claude/skills/bakery-<agent>-<name>` and `bakery-board-<name>`
  (`skills.go`), kept out of git by one line in the repo's common
  `info/exclude`.
  `runspec.go` builds one run's `claude` command line, prompt and MCP config
  from the task, the agent and the board config without any I/O (golden
  files in `testdata/`; `go test ./internal/workshop -update` rewrites them).
  `WorkshopService.PrintRunCommand` prepares a run without starting it and
  returns the command to paste into a terminal, for checking a board's setup.
  `WorkshopService.StartRun` puts an agent to work: it checks the board's
  run limit, prepares the run, records it with the API (`POST
  /api/tasks/{task}/runs`) and starts `claude` (`BAKERY_CLAUDE` overrides the
  binary) in its own process group (`process.go`). Every output line goes to
  `runs/<id>/stream.jsonl`, stderr to `stderr.log`; `stream.go` decodes the
  lines into run events, sent to the frontend as the Wails event
  `run:<id>`, with the list of runs as `workshop:runs`. `StopRun` interrupts
  the run and kills it after 5 seconds; the run's end is recorded with
  `PATCH /api/runs/{run}`. Runs still going when the app quits are stopped.
  When a run ends (`finish.go`), the app counts the diff it left (committed
  on the branch since the base, and anything uncommitted), records the run's
  outcome, comments on the task as the member (agent, branch, status, +/−,
  cost, Claude's summary), and moves a task that succeeded to the board's
  finish column. Each run's `run.json` lets the next launch finish a run the
  app never saw end: it ends a leftover claude and records the run as failed
  ("Desktop closed during the run."). The run panel's footer opens the
  worktree, copies the branch name, and removes the worktree (the branch
  stays).
- `colony.go` — the colony's scheduler, in `WorkshopService`: every 10
  seconds, when a run ends and on the open board's events, each board on
  this machine that is not paused and has agents enabled (`speed`,
  `agents`, `ready_column` in `board.toml`) hands free tasks of its ready
  column to idle agents. `workshop.PickNext` decides (a task prioritized for
  an agent first, then work priority 1 to 4, then board order, within the
  run limit of the board's speed). Every run, the scheduler's or a person's,
  first claims its task (`POST /api/tasks/{task}/claim`, with this machine's
  id from `~/.config/the-bakery/machine`), heartbeats it every 30 seconds and
  releases it when the run ends, so two machines never work the same task.
- Time controls and drafting: the board header's ⏸ ▶ ▶▶ (keys Space, 1, 2
  while not typing) set the board's `speed` on this machine
  (`WorkshopService.SetSpeed`). Right-click a card to prioritize it for one
  of the agents enabled on the board, clear that, or forbid it for agents
  (`BoardsService.DraftTask`, `PATCH /api/tasks/{task}`). A card shows the
  agent holding it (portrait and a moving stripe), a flag when prioritized
  and ⊘ when forbidden. Board settings pick the ready column, the agents
  enabled here and the run limit at fast speed.
- `internal/workshop/desk.go` — the desk: an MCP server inside the app on a
  random loopback port, behind a secret made at launch. Every run gets it in
  its MCP config as `bakery_desk` (at `/mcp/<run id>`) and as Claude's
  `--permission-prompt-tool`, so a tool call the agent may not make on its
  own, or a question it asks, becomes a **letter** (Wails events
  `letter:new`, `letter:closed`) and waits up to 30 minutes for
  `WorkshopService.AnswerLetter`: allow, allow always (adds the rule, such as
  `Bash(curl:*)`, to `extra_allowed_tools` in the board's `board.toml`),
  deny with a reason, or the answers to a question. Stopping a run denies
  its open letters. Runs get `MCP_TOOL_TIMEOUT` a little over those 30
  minutes; Claude otherwise gives up on the desk after one.
  `components/Letters.svelte` stacks the open letters as envelopes on the
  right edge (yellow for a permission, blue for a question) and
  `LetterDialog.svelte` answers one. A letter that arrives while the window
  is not focused also comes as a desktop notification (Wails' notifications
  service).
- `internal/session/runner.go` — the runner token: a personal token the app
  makes for itself (`desktop runner (<host>)`) so Claude can use the Bakery
  MCP server as the member during a run. Kept in its own keyring entry,
  remade when the API refuses it, revoked on clock out.
- `liveservice.go` — `LiveService`: keeps the open board's event stream
  (`internal/api/events.go`) open from Go and passes each event to the
  frontend as the Wails event `board:event`, with `board:status` for the
  indicator and `board:resync` after a reconnect (backoff 1 s up to 30 s).
- `internal/api/` — typed client for the API. `internal/session/` — the
  signed-in member and token storage.
- `internal/agents/` — the member's agents as folders under
  `~/.config/the-bakery/agents/<slug>/` (`agent.toml`, `skills/`, `.sync.json`)
  and the sync engine that keeps them in step with the API: pushes local
  edits, pulls the server's, creates new folders as agents, trashes deleted
  ones, and turns edits on both sides into a conflict (the server's copy
  under `.conflicts/<slug>/`) instead of overwriting either. The folder belongs
  to one member of one API; another's moves aside to `agents-<host>-<id>`.
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
- `frontend/src/components/AgentsScreen.svelte` — the Agents screen (the
  sidebar switches between Boards, Agents and Work): the roster, the open agent's
  `CharacterCard.svelte` (name, title, backstory, traits, model, permission
  mode, allowed tools, skills with import from `~/.claude/skills` or a folder,
  sharing per guild, delete), and a Recruit tab for the guild's shared agents.
  `Portrait.svelte` draws a stand-in portrait from the agent's portrait seed;
  `lib/roster.svelte.ts` holds the screen's state.
- `frontend/src/components/WorkTab.svelte` — the Work screen: agents against
  the open guild's work types, each cell a work priority (click raises it,
  right-click lowers it, saved to `agent.toml` a second after the last click),
  and the work type editor (add, rename, drag to reorder, delete). Task cards
  show their work type as a chip (`lib/worktype.ts` picks its colour); the
  task panel sets it.
- `frontend/src/components/AgentPicker.svelte` and `RunPanel.svelte` — the
  task panel's **Assign** (off until the board is linked on this machine)
  lists the member's agents by their work priority for the task's work
  type; the run panel, in the task panel's place, streams what the agent
  says and does (`run:<id>`), with Stop. The task panel's **Runs** tab lists
  every member's runs of the task; a card shows **Working** while a run is
  going (`lib/workshop.svelte.ts`, from `workshop:runs` and the board's
  `run.started` / `run.finished` events).
- `frontend/src/components/ColonyView.svelte` — the colony from above (the
  board header's **Colony** switch): a workbench per work type, each agent
  with a run going at the bench of its task's work type, agents enabled on
  the board and idle at the table. Pixel art is drawn from character maps
  in `lib/sprites.ts`. What an agent does comes from Go as the Wails event
  `agent:state` (thinking, editing, running a command, waiting for a letter,
  done, failed). It animates at most 30 frames a second, only while an
  agent is busy, and not at all with reduced motion; hover for the task and
  cost, click to open the run, or the letter of an agent that waits.
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
