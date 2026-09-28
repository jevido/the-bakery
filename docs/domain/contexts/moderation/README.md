# moderation

- Subdomain: supporting
- Hosted in: `services/api` (module `contexts/moderation`). Its only client
  is the operator console (`apps/console`); members meet it as reports they
  file and as the refusals a sanction causes.

## Purpose

Keeping bad actors out of the platform: the **operators** who run it, the
**sanctions** they put on a member or a guild, the **reports** members file,
and an **audit log** of every sensitive action on the platform.

It is **not** responsible for who is in a guild (guilds is), for accounts
and sign-in of members (identity is), or for enforcing a sanction inside
each use case: identity and guilds ask moderation and refuse themselves.

## Language

| Term | Meaning |
| ---- | ------- |
| Operator | An account of the platform owner (or someone they appoint) for the console: email, password and TOTP. Not a member; member credentials never work here, nor operator ones on member routes. |
| Sanction | A measure on one member or one guild: a suspension or a ban, with a reason, by an operator. One active per target. |
| Suspension | A sanction until a date; it ends by itself. |
| Ban | A sanction with no end; an operator lifts it. |
| Report | A member telling the operators about a member or a guild: a reason, then open until an operator dismisses it or acts on it. |
| Audit log | Every sensitive action on the platform, append-only: who (member or operator), what, to whom, why, when, from where. |

## Model

### Aggregates

| Aggregate | Invariants |
| --------- | ---------- |
| Operator (root) | Email unique; a password hash; a TOTP secret, confirmed with a first code before the operator can sign in. |
| Sanction (root) | Targets one member or one guild (by id). Kind `suspension` (needs an `until` in the future) or `ban` (none). A reason of 1–1000 characters. By one operator. At most one active sanction per target; lifting ends it once. |
| Report (root) | By one member, about one member or guild (by id), with a reason of 1–1000 characters. Open, then once `dismissed` or `actioned` (with the sanction it led to) by an operator. At most 10 per member per day. |
| Audit entry (root) | Append-only: actor kind (`member`, `operator`) and id, action, target kind and id, reason, details, IP, time. Never changed or deleted. |

### Commands

- **Create** an operator (artisan only); **sign in** with password, then
  TOTP.
- **Sanction** a member or a guild; **lift** a sanction; **list** sanctions
  of a target.
- **File** a report (members); **list** open reports; **dismiss** one or
  mark it **actioned** with a sanction.
- **Record** an audit entry (any context, through a small port); **list**
  the log with filters.
- **List** members and guilds for the console, through lookups the owning
  contexts publish.

### Domain events

- `MemberSuspended` / `MemberBanned` — member id, sanction id, until.
- `GuildBanned` — guild id, sanction id (a guild suspension is announced
  the same way with its until).
- `SanctionLifted` — sanction id, target.
- `ReportFiled` — report id, target.

## Integration

- **Publishes:** `ActiveFor` (the active sanction of a member or a guild,
  cached for a few seconds), set by moderation's wiring into identity (member
  checks on REST, personal tokens, MCP and sign-in) and guilds (guild checks
  in every guild-scoped use case, which boards reaches through
  `guilds.Memberships`). An `Audit` recorder that other contexts reach
  through a one-method port of their own.
- **Consumes:** identity's member lookups (by id, by email, a search for the
  console) and guilds' guild lookups (by id, a search with member counts),
  through interfaces those contexts publish; boards' `CloseStreamsOf` to end
  the live board streams of a sanctioned member or guild. Moderation never
  reads another context's tables.

## Why it's shaped this way

- **Operators are not members.** A separate table, sign-in and token guard
  mean a stolen member password or token never reaches the console, and an
  operator acting on a guild is never mistaken for one of its members. TOTP
  is required because the console can shut anyone out.
- **Enforcement is asked, not imposed.** Identity and guilds keep their own
  refusals and ask moderation whether a sanction is active; moderation sets
  that check into them when it is wired, so no context imports moderation's
  internals and there is no import cycle. The cost: a sanction takes effect
  within the cache's few seconds, not instantly.
- **The audit log is append-only and platform-wide.** It is how anyone can
  later tell who did what to whom and why, including operators; nothing in
  the code updates or deletes an entry.
