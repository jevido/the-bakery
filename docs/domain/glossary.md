# Glossary

The ubiquitous language. These are the words used in conversation, in issues,
in the docs **and in the code**: class, type, function, table and event names
follow this list. If a term is missing or wrong, fix it here first, then in the
code.

A term that means different things in different contexts gets one row per
context. Terms used only inside one context can also live in that context's
document; list them here when people outside the context use them too.

| Term | Context | Meaning | Not to be confused with |
| ---- | ------- | ------- | ----------------------- |
| Member | identity | A person with an account: email, display name and password. Stored in `members`; domain code never says "user". | "User" (not used in domain code); Operator |
| Member | guilds | A member of a particular guild, i.e. a Member (identity) who holds a Membership in it. | |
| Token | identity | The JWT (valid 30 days) a Member receives on register or login and sends as `Authorization: Bearer ...` to prove who they are. | API key |
| Operator | moderation | The platform owner, and anyone they appoint, who uses the console: an account of its own with email, password and TOTP. Not a guild role and not a Member account; member credentials never work in the console. | Guild member, admin, owner |
| Sanction | moderation | A suspension or a ban on one member or one guild, with a reason, by an operator. One active per target. | Punishment, block |
| Suspension | moderation | A sanction until a date; it ends by itself. The member (or the guild's members) can use nothing it covers until then. | Ban |
| Ban | moderation | A sanction without an end; an operator lifts it. | Suspension, deletion |
| Report | moderation | A member telling the operators about a member or a guild, with a reason; open until an operator dismisses it or acts on it. | Flag, complaint |
| Audit log | moderation | Every sensitive action on the platform, append-only: who, what, to whom, why, when, from where. | Activity (a task's history) |
| Signal | moderation | One sign-up, guild founded or rate-limit refusal, with who and from which IP; counted in volume to find abuse. | Event, metric |
| Rate limit | moderation | How many requests a client may make in a while (per IP, member or personal token); past it the API answers 429 with when to try again. | Throttle, quota |
| Web session | identity | A sign-in from the website, carried in an httpOnly cookie that page scripts cannot read. The desktop app keeps its bearer token instead. | Token (the desktop's) |
| Personal token | identity | A long-lived secret a member creates for a tool (Claude, a script): named, shown once, stored only as a hash, revocable one at a time. Looks like `bky_…`. | Token (the desktop's session JWT), API key |
| Handoff code | identity | A one-time code the desktop app gets for its signed-in member and opens the website with, so the website signs them in too. Valid for 60 seconds, usable once; the website trades it for a web session. | Token (never put in a URL) |
| Clock out | identity | What the desktop app calls logging out: the token is forgotten on this machine. | |
| Guild | guilds | The top-level group people work in together. A Member can belong to many guilds. | Workspace, org, organisation, team (never used) |
| Founder | guilds | The Member who founded a guild; they become its first member. No extra rights beyond that. | Owner, admin |
| Invite | guilds | A standing offer to join one guild, made by one of its members. It has a code, an optional expiry and an optional use limit, and can be revoked. Not an e-mail. | Invitation e-mail |
| Invite link | guilds | The URL `https://<website>/join/<code>` that carries an invite. | |
| Archived guild | guilds | A guild nobody works in any more: hidden from lists and read-only. Any member can restore it. | Deleted guild |
| Leaving | guilds | A member ending their own membership. The last member cannot leave; they archive the guild instead. | Being removed (another member ends it) |
| Membership | guilds | That a Member belongs to a guild. Everyone in a guild can do everything; there are no roles yet. | Role, permission |
| Board | boards | A kanban board inside one guild. A guild has many boards. | Project |
| Task | boards | A unit of work on a board: title, description, column and position. | Ticket, issue, card (a card is only how a task is drawn) |
| Column | boards | A named, ordered lane that belongs to one board; every task on the board stands in exactly one. A new board starts with Backlog, To do, Doing and Done; members add, rename, reorder and delete columns. A column can be deleted only when it holds no tasks, and a board keeps at least one. | Status, lane |
| Position | boards | A sortable key that orders tasks within one column of a board. Moving a task changes only its own position. | Index, rank number |
| Subtask | boards | A task whose parent is another task on the same board. One level deep: a subtask cannot have subtasks. It has a `done` flag and is ordered under its parent; it is not placed in a column. Expanding a task adds several subtasks at once. | Checklist item, child ticket |
| Comment | boards | Text a member writes on a task. Only its author can edit or delete it. | Description, note |
| Board event | boards (published language) | A change on a board announced to everyone watching it, over the board's event stream: `task.created`, `task.updated`, `task.moved`, `task.deleted`, `column.created`, `column.updated`, `column.moved`, `column.deleted`, `presence`. | Domain event (inside the context), activity (a task's history) |
| Presence | boards | Which members have a board open right now. Short-lived: kept while their stream is open and gone a minute after it goes quiet; never history. | Online status, activity |
| Work type | boards | A kind of work a task needs (Coding, Research, …), one list per guild, each identified by a key (`coding`, `research`, …). A task has at most one. A guild starts with Coding, Research, Writing, Testing, Design, Review and Ops. | Label, tag, category |
| Agent | agents | A configured Claude CLI worker owned by one member: name, title, backstory, traits, model, permission mode, allowed tools, skillset and work priorities. The desktop app may use RimWorld wording in its labels; the model says Agent. | Colonist, bot, assistant, AI |
| Roster | agents | All agents of one member. | Team, crew |
| Skillset | agents | An agent's skills: the files of its `.claude/skills` directory. | Plugin, toolkit |
| Skill | agents | One directory of a skillset with a `SKILL.md` (name and description in its front matter) and any supporting files. | Command, prompt |
| Trait | agents | A fixed personality or working-style option of an agent; each adds one line to its instructions, and some traits conflict. | Setting, flag |
| Work priority | agents | How much an agent wants a kind of work: 1 (first) to 4 (last), or off, per work type key. | Skill level, weight |
| Share | agents | An owner makes an agent available to one guild they are in, for its members to recruit. | Publish, transfer |
| Recruit | agents | Copy an agent shared with a guild into your own roster. The copy is yours; the original stays its owner's. | Clone, fork, hire |
| Revision | agents | A number that goes up by one with every change to an agent; a change based on an older revision is refused as a conflict. | Version (that is a release) |
| Claim | boards | A time-limited hold a member's agent has on a task, so no other machine starts it. One active claim per task; it lasts 2 minutes and heartbeats extend it; it expires without them, and is released when the run ends or the task is deleted. | Lock, assignment |
| Prioritize | boards | Force one agent onto one task: only that agent may claim it, and it takes it ahead of its work priorities. Cleared by a person. | Pin, assign (a person's own Assign) |
| Forbid | boards | Mark a task as never taken by agents; people still work it. | Block, archive |
| Scheduler | workshop | The loop on one machine that hands the next task to an idle agent enabled on the board. Machines coordinate only through Claims. | Queue, dispatcher |
| Ready column | workshop | The column of a board agents take work from, set per machine in the board config. | Backlog (a column name) |
| Time controls | workshop | Per board on this machine: `paused` (no new runs start), `normal`, `fast` (a higher run limit), like RimWorld's speed buttons. | Pause (only one of the three) |
| Letter | workshop | A notice from a run that needs a person's answer: a permission prompt or a question. The run waits until it is answered or times out. | Alert, event letter (neither blocks anything), notification |
| Portrait | identity, agents | A generated pixel image of a member or an agent, fixed by its portrait seed and re-rolled by a new seed. Presentation only. | Avatar, photo |
| Needs | workshop | An agent's four bars, 0 to 100, computed on this machine from real run data: Budget (money left today), Focus (context left in its run), Morale (recent runs' outcomes), Rest (time without a break). Never stored, always computed. | Stats, health |
| Mood | workshop | One word summarising an agent's needs (`content`, `okay`, `stressed`, `breaking`). Changes how the agent looks and is described, never what it does or which task it takes. | Status, state (the run's) |
| Alert | workshop | A standing condition that needs a person's attention (an idle agent, a failed run, work nobody covers, a question waiting), listed at the right edge until the condition clears. | Notification, letter |
| Event letter | workshop | A one-off, storyteller-style notice about a moment on a board (a raid of bugs, a streak of finished work, a stale column, a newcomer). Never blocks a run, unlike a Letter. | Letter (that one asks), alert |
| Quiet colony | workshop | The setting that turns every flavor element off (mood, needs in the colony view, event letters, flavor lines, sounds, idle animations); portraits, alerts and letters stay. | Plain mode, focus mode |
| Run | boards | One attempt by an Agent to work a Task on a member's machine: `running`, then once `succeeded`, `failed` or `stopped` (read as `lost` when still running after 24 hours), with cost, turns, summary and diff stats. | Job, session (Claude's own) |
| Board config | workshop | The per-machine folder `~/.config/the-bakery/boards/<board-id>/` saying how this machine works a board (linked repo, base branch, worktree root, run limit, budget, finish column, `CLAUDE.md`, `skills/`, `mcp.json`). Never synced. | Board settings on the server (there are none) |
| Worktree | workshop | The git worktree a Run happens in, on branch `bakery/<task-id>-<slug>`. | Checkout, clone |
| Workshop | workshop | The desktop-local part that prepares and runs Claude CLI for a Run and reports it back. | Runner service, executor |
| Run spec | workshop | The exact `claude` arguments, working directory, prompt and files for one Run. | Command |
| Runner token | workshop | A personal token the desktop app makes for itself so Claude can use the Bakery MCP server as the member during a Run. | Token (the desktop's session) |
| Activity | boards | What happened to a task, in order: created, edited, moved, commented, subtask added, subtask done. Written by the system from domain events, never by a member. | Comment, audit log (the Operator's, in the console) |
| Release | platform | A tagged desktop build, `desktop-vX.Y.Z` (semantic versioning: X breaking, Y features, Z fixes), published as a GitHub release with signed artifacts and an update manifest. | Deploy (that is the API and website going out) |
| Environment | platform | Where The Bakery runs: `dev` (local), `next` (pre-production) or `prod`. | Stage, instance |
| MCP tool | api (published language) | One operation Claude can call on `/mcp`, such as `list_guilds` or `move_task`: a thin wrapper over an existing use case, with the same rules as the REST API. | Endpoint (REST), agent |
