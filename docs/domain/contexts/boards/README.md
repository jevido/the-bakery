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
| Column | A named, ordered lane of one board. A new board starts with Backlog, To do, Doing, Done. |
| Position | A sortable key ordering tasks within a column, or subtasks under their parent. |
| Subtask | A task whose parent is another task on the same board; one level deep, with a `done` flag, not in a column. |
| Comment | Text a member writes on a task; only its author edits or deletes it. |
| Activity | What happened to a task, in order, built from domain events. |
| Board event | A change on a board, announced on the board's event stream (published language, below). |
| Presence | Which members have a board open right now; short-lived, never history. |

## Model

### Aggregates

| Aggregate | Invariants |
| --------- | ---------- |
| Board (root) | Belongs to exactly one guild (by id). Name is 1–60 characters. Owns its columns: at least one, names 1–40 characters and unique on the board (ignoring case), ordered by position. A new board starts with Backlog, To do, Doing, Done. A column is removed only when it holds no tasks. |
| Task (root) | Belongs to one board (by id). Title is 1–200 characters. A task on the board refers to one of that board's columns by id. Position is unique per column among top-level tasks. |
| Task as subtask | Its parent is on the same board and has no parent itself. A task with subtasks cannot become a subtask. Position is unique per parent. Expanding adds 1–50 subtasks in one transaction. |
| Comment (root) | Belongs to one task (by id) and one author (a member id). Body is 1–10 000 characters. Only the author edits or deletes it. |

### Commands

- **Create a board** in a guild; **list boards** of a guild; **get a board** with its columns in order and each column's tasks by position.
- **Add**, **rename**, **move** and **remove** a column (remove only when empty and not the last).
- **Create a task** — lands at the bottom of the board's first column.
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

Each event carries the acting member's id. Use cases publish events once the
change is stored, to an in-process dispatcher; its handlers log them, project
them into task activity, and turn them into board events.

### Published language: the board event stream

`GET /api/boards/{board}/events` is a server-sent event stream for the board's
members. Each event is `event: <type>` with JSON data
`{type, board_id, at, actor_id, data}`. `data` holds ids and the changed fields
only; a client that is unsure refetches the board.

| Type | When |
| ---- | ---- |
| `task.created` | a task was added to the board |
| `task.updated` | a task's title or description changed, or its subtasks or comments did |
| `task.moved` | a task changed column or position |
| `task.deleted` | a task was deleted |
| `column.created` · `column.updated` · `column.moved` · `column.deleted` | a column was added, renamed, reordered or removed |
| `presence` | someone opened or left the board, or the current list (`snapshot`) on connect: `state` (`snapshot`, `joined`, `left`), `conn_id` (one open stream), `member_id`, `display_name`; a snapshot lists `present` streams. A member with two windows open has two streams and counts once. |

Changing a type or removing a field is a breaking change for the desktop app.

## Integration

- **Publishes:** Go functions in the root package (`ListBoards`, `GetBoard`,
  `CreateBoard`, `CreateTask`, `GetTask`, `UpdateTask`, `ExpandTask`,
  `MoveTask`, `DeleteTask`, `ListComments`, `CommentOnTask`) with their own
  types, for the MCP server.
- **Consumes:** identity's authenticated member id and display names by id
  (`identity.DisplayNames`, behind boards' `MemberNames` port); guilds'
  `Memberships.IsMember` and `Memberships.IsArchived`. A guild and a member
  are only ids here; boards never read the guild or member tables.

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
- **Columns belong to the Board aggregate.** The rules about columns (at least
  one, unique names, order) are rules about the board as a whole, so the board
  guards them. Tasks stay separate aggregates and only refer to a column by id;
  "remove only when empty" is checked by the use case, which asks the tasks
  before the board removes the column.
- **Board events fan out through Postgres `LISTEN/NOTIFY`.** Each API process
  listens on one connection and passes events to the streams it holds, so a
  change made through one process reaches members connected to another,
  including while a deploy runs two side by side, without adding Redis or a
  broker. `NOTIFY` payloads are small, so events carry ids and changed fields
  and clients refetch when in doubt. Delivery is best effort: a client that
  reconnects refetches the board instead of replaying missed events.
