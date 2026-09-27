# workshop

- Subdomain: supporting
- Hosted in: `apps/desktop` (module `internal/workshop`), on each member's
  machine. It has no server side.

## Purpose

Turn an agent, a task and this machine's board config into a Claude CLI
process working in its own git worktree, stream what it does to the member,
and report the run back to the API.

It is **not** responsible for storing runs (boards is), for agents (agents
is; the workshop reads the member's agent folders) or for choosing which task
an agent works on next (the scheduler, later).

## Language

| Term | Meaning |
| ---- | ------- |
| Board config | The per-machine folder `~/.config/the-bakery/boards/<board-id>/` saying how this machine works a board: the linked repo, base branch, worktree root, run limit, budget, finish column, board instructions, board skills and extra MCP servers. Never synced. |
| Linked | A board whose board config names a valid git repository on this machine. Only a linked board can run agents here. |
| Worktree | The git worktree a run happens in, on branch `bakery/<task-id>-<slug>`, under the board config's worktree root. |
| Run spec | Everything one run needs: the `claude` arguments, the working directory, the prompt, and the files to write (the run's MCP config). Built by a pure function. |
| Runner token | A personal token the desktop app makes for itself (`desktop runner (<host>)`), so Claude can use the Bakery MCP server as the member during a run. |

## Model

### Aggregates

The workshop keeps no aggregate of its own on the server. On the machine:

| Aggregate | Invariants |
| --------- | ---------- |
| Board config | One per board id per machine. `repo` is a git repository; `base_branch` exists in it; `max_concurrent_runs` 1–8; `max_budget_usd` > 0. Loading a missing config gives the defaults, not linked. |
| Local run | One per started run: at most `max_concurrent_runs` running per board on this machine; one worktree per task at a time; ends once, and reports that end to the API. |

### Commands

- **Get** and **save** a board's config; **open** its folder.
- **Start a run**: prepare the worktree and branch, place the agent's and the
  board's skills in it, build the run spec, register the run with the API,
  start Claude CLI.
- **Stop a run**; **list** a run's events; **finish** a run (diff stats,
  summary comment, move to the finish column); **remove** a run's worktree,
  keeping its branch.

### Domain events

Local only, sent to the frontend as Wails events: a run's init (model,
permission mode, skills, MCP servers), its text, tool calls and tool results,
and its result (status, cost, turns, duration).

## Integration

- **Consumes:** boards through the REST API (the task with its subtasks and
  comments, the board's columns, starting and finishing runs, commenting and
  moving tasks), agents through the member's agent folders (the agents
  context's synced copy) and the traits list from the API, identity's personal
  tokens for the runner token. It translates all of these into its own
  `RunSpec`, and never shares their types.
- **Publishes:** nothing to other contexts; runs reach the guild through
  boards.

## Why it's shaped this way

- **Board configs are kept per API.** Board ids are the API's, so board 1 on
  a dev API and board 1 on bakery.jevido.app are different boards: the release
  API's configs live in `boards/`, any other API's in `boards-<host>/`.
- **Board config is per machine and never synced.** Where a repo is checked
  out, where worktrees go, how much a run may cost and which MCP servers are
  at hand differ from one machine to the next, even for one member. Syncing
  them would put one machine's paths on another. The cost: a member links the
  board again on each machine.
- **Runs are reported to the server.** The process runs here, but the guild
  must see that a task is being worked and what came of it; boards keeps that
  record. Everything bulky or private (the stream log, the worktree, the MCP
  config holding the runner token) stays on the machine.
- **The run spec is a pure function.** Building the command line from its
  inputs without I/O keeps it testable with golden files; only the runner
  executes it.
