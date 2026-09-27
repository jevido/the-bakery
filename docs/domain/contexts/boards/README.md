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
| Position | A sortable key ordering tasks within a column, or subtasks under their parent. |
| Subtask | A task whose parent is another task on the same board; one level deep, with a `done` flag, not in a column. |
| Comment | Text a member writes on a task; only its author edits or deletes it. |
| Activity | What happened to a task, in order, built from domain events. |

## Model

### Aggregates

| Aggregate | Invariants |
| --------- | ---------- |
| Board (root) | Belongs to exactly one guild (by id). Name is 1–60 characters. |
| Task (root) | Belongs to one board (by id). Title is 1–200 characters. Column is one of the four. Position is unique per (board, column) among top-level tasks. |
| Task as subtask | Its parent is on the same board and has no parent itself. A task with subtasks cannot become a subtask. Position is unique per parent. Expanding adds 1–50 subtasks in one transaction. |
| Comment (root) | Belongs to one task (by id) and one author (a member id). Body is 1–10 000 characters. Only the author edits or deletes it. |

### Commands

- **Create a board** in a guild; **list boards** of a guild; **get a board** with its tasks by column and position.
- **Create a task** — lands at the bottom of `backlog`.
- **Edit a task** — title, description.
- **Move a task** — to a column, right after one task and/or right before
  another; with neither, to the bottom of the column.
- **Delete a task** (its subtasks, comments and activity go with it).
- **Add a subtask**, **expand** a task into several subtasks, **tick** or
  untick a subtask, and **reorder** subtasks under their parent.
- **Comment** on a task; **edit** or **delete** your own comment.
- **List a task's activity**, newest first.

Every command is refused unless the calling member is a member of the board's
guild, and every change is refused while the guild is archived. A board or task
that does not exist is reported as not found.

### Domain events

- `TaskCreated` — task id, board id, column, position.
- `TaskMoved` — task id, board id, from column, to column, new position.
- `TaskEdited` — task id, board id, which of title and description changed.
- `TaskCommented` — task id, board id, comment id.
- `SubtaskAdded` — subtask id, parent task id, board id.
- `SubtaskCompleted` — subtask id, parent task id, board id.

Each event carries the acting member's id.

## Integration

- **Publishes:** Go functions in the root package (`ListBoards`, `GetBoard`,
  `CreateBoard`, `CreateTask`, `UpdateTask`, `MoveTask`, `DeleteTask`) with their
  own types, for the MCP server.
- **Consumes:** identity's authenticated member id; guilds'
  `Memberships.IsMember` and `Memberships.IsArchived`. A guild is only an id
  here; boards never read the guild tables.

## Why it's shaped this way

- **Task is its own aggregate, not part of Board.** A board can hold many
  tasks, and several members move tasks at the same time. If tasks lived
  inside the Board aggregate, every move would load and lock the whole board.
  As separate aggregates, a move changes one task in one transaction. The cost:
  the board cannot enforce rules across all its tasks at once (e.g. a limit per
  column); such rules would need a different design.
- **Positions are fractional keys.** A task's position is a lexicographically
  sortable string, so a new key can always be made between two neighbours.
  Moving a task updates that one row, never its neighbours. Keys compare by
  byte, so the database column uses the `C` collation. If two moves pick the
  same key at once, the unique (board, column, position) index rejects one and
  it is recomputed and retried.
- **Subtasks are tasks, but not in columns.** A subtask is a row in the same
  table as its parent (`parent_id`), so it has a title, a description and a
  position like any task, and the Task aggregate guards it. It is not placed in
  a column and the board lists only top-level tasks, so the board stays one
  level deep and a card only shows its subtasks' progress. One level is a hard
  rule: deeper trees would turn the board into an outline.
- **Activity is a projection, not an aggregate.** No member writes activity.
  Use cases publish domain events after a change succeeds, and a projector
  turns them into activity rows. A refused change publishes nothing, so it
  leaves no activity.
