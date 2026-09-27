# guilds

- Subdomain: supporting
- Hosted in: `services/api` (module `contexts/guilds`)

## Purpose

Knows who works together: guilds and their memberships. It answers one
question for the rest of the system: is this member in this guild? It is
**not** responsible for accounts (identity) or for the work inside a guild
(boards).

## Language

| Term | Meaning |
| ---- | ------- |
| Guild | The top-level group people work in. Never workspace, org or team. |
| Founder | The member who founded the guild and became its first member. |
| Membership | That a member belongs to a guild. No roles yet: every member can do everything. |

## Model

### Aggregates

| Aggregate | Invariants |
| --------- | ---------- |
| Guild (root, holds its memberships) | Name is 1–60 characters. The founder becomes the first member. A member cannot be added twice. |

### Commands

- **Found a guild** — by a member, with a name; the founder becomes its first member.
- **List guilds of a member.**
- **Add a member** — by an existing member of the guild, using the new member's email.

### Domain events

- `GuildFounded` — guild id, name, founder's member id.
- `MemberJoined` — guild id, member id.

## Integration

- **Publishes:** the `guilds.Memberships` interface,
  `IsMember(ctx, guildID, memberID) (bool, error)`. This is the only thing
  other contexts may import from guilds.
- **Consumes:** identity's authenticated member id and its member lookup by
  email, translated into a plain member id here.

## Why it's shaped this way

- **Memberships live inside the Guild aggregate.** "A member cannot be added
  twice" is a rule about the whole guild, so the guild guards it. Guilds are
  small enough that this does not create contention.
- **No roles yet.** Everyone in a guild can do everything. Roles are easy to
  add inside this context later without changing `IsMember` callers.
