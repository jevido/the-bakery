# boards

- Subdomain: core
- Hosted in: `services/api` (module `contexts/boards`)

## Purpose

Planning work as tasks on boards: a guild's boards, the tasks on them, and
where each task stands. This is where The Bakery competes. It is **not**
responsible for who is in a guild (guilds) or who a person is (identity).

## Language

| Term | Meaning |
| ---- | ------- |
| Board | A kanban board inside one guild. |
| Task | A unit of work on a board: title, description, column, position. |
| Column | One of `backlog`, `todo`, `doing`, `done`. Fixed for now. |
| Position | A sortable key ordering tasks within a column. |

## Model

### Aggregates

| Aggregate | Invariants |
| --------- | ---------- |
| Board (root) | Belongs to exactly one guild (by id). Name is 1–60 characters. |
| Task (root) | Belongs to one board (by id). Title is 1–200 characters. Column is one of the four. Position is unique per (board, column). |

### Commands

- **Create a board** in a guild; **list boards** of a guild; **get a board** with its tasks by column and position.
- **Create a task** — lands at the bottom of `backlog`.
- **Edit a task** — title, description.
- **Move a task** — to a column, between two neighbours (or at either end).
- **Delete a task.**

Every command is refused unless the calling member is a member of the board's
guild.

### Domain events

- `TaskCreated` — task id, board id, column, position.
- `TaskMoved` — task id, board id, from column, to column, new position.

## Integration

- **Publishes:** nothing yet.
- **Consumes:** identity's authenticated member id; guilds'
  `Memberships.IsMember`. A guild is only an id here; boards never read the
  guild tables.

## Why it's shaped this way

- **Task is its own aggregate, not part of Board.** A board can hold many
  tasks, and several members move tasks at the same time. If tasks lived
  inside the Board aggregate, every move would load and lock the whole board.
  As separate aggregates, a move changes one task in one transaction. The cost:
  the board cannot enforce rules across all its tasks at once (e.g. a limit per
  column); such rules would need a different design.
- **Positions are fractional keys.** A task's position is a lexicographically
  sortable string, so a new key can always be made between two neighbours.
  Moving a task updates that one row, never its neighbours.
