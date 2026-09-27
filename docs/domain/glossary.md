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
| Operator | identity | The platform owner, and anyone they appoint, who uses the moderation console. Not a guild role and not a Member account. | Guild member, admin |
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
| Activity | boards | What happened to a task, in order: created, edited, moved, commented, subtask added, subtask done. Written by the system from domain events, never by a member. | Comment, audit log (the Operator's, in the console) |
| Release | platform | A tagged desktop build, `desktop-vX.Y.Z` (semantic versioning: X breaking, Y features, Z fixes), published as a GitHub release with signed artifacts and an update manifest. | Deploy (that is the API and website going out) |
| Environment | platform | Where The Bakery runs: `dev` (local), `next` (pre-production) or `prod`. | Stage, instance |
| MCP tool | api (published language) | One operation Claude can call on `/mcp`, such as `list_guilds` or `move_task`: a thin wrapper over an existing use case, with the same rules as the REST API. | Endpoint (REST), agent |
