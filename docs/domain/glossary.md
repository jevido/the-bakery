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
| Column | boards | Where a task stands on its board: one of `backlog`, `todo`, `doing`, `done` (shown as Backlog, To do, Doing, Done). Fixed for now. | Status, lane |
| Position | boards | A sortable key that orders tasks within one column of a board. Moving a task changes only its own position. | Index, rank number |
| Release | platform | A tagged desktop build, `desktop-vX.Y.Z`, published as a GitHub release with signed artifacts and an update manifest. | Deploy (that is the API and website going out) |
| Environment | platform | Where The Bakery runs: `dev` (local), `next` (pre-production) or `prod`. | Stage, instance |
