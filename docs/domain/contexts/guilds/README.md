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
| Invite | A standing offer to join one guild, with a code, an optional expiry and use limit. |
| Invite link | `https://<website>/join/<code>`. |
| Archived guild | Hidden from lists and read-only; any member can restore it. |
| Leaving | A member ending their own membership. |

## Model

### Aggregates

| Aggregate | Invariants |
| --------- | ---------- |
| Guild (root, holds its memberships) | Name is 1–60 characters. The founder becomes the first member. A member cannot be added twice. An archived guild refuses every change except restore. The last membership cannot end: the last member archives instead of leaving, and nobody can remove them. |
| Invite (root) | Belongs to one guild (by id). The code is unique. `expires_at` and `max_uses` are optional; uses are counted; a revoked or expired invite, or one with no uses left, cannot be accepted. |

### Commands

- **Found a guild** — by a member, with a name; the founder becomes its first member.
- **List guilds of a member.**
- **Rename**, **archive** and **restore** a guild — any member.
- **Remove a member** — any member removes another; **leave** — end your own
  membership.
- **Create**, **list** and **revoke invites**; **accept an invite** — a
  signed-in member joins the invite's guild.
- **Add a member** — by an existing member of the guild, using the new
  member's email. To someone outside the guild, a guild that does not exist
  and one they are not in look the same (refused).

### Domain events

- `GuildFounded` — guild id, name, founder's member id.
- `GuildRenamed` — guild id, new name.
- `GuildArchived`, `GuildRestored` — guild id, by whom.
- `MemberJoined` — guild id, member id (added by email or through an invite).
- `MemberLeft` — guild id, member id.
- `MemberRemoved` — guild id, member id, removed by.
- `InviteCreated` — invite id, guild id, expiry, use limit.
- `InviteRevoked` — invite id, guild id.

## Integration

- **Publishes:** the `guilds.Memberships` interface,
  `IsMember(ctx, guildID, memberID) (bool, error)` (`guilds.NewMemberships()`).
  This is the only thing other contexts may import from guilds. The events are
  only logged for now; nothing subscribes to them.
- **Consumes:** identity's authenticated member id and its member lookup by
  email, translated into a plain member id here.

## Why it's shaped this way

- **Memberships live inside the Guild aggregate.** "A member cannot be added
  twice" is a rule about the whole guild, so the guild guards it. Guilds are
  small enough that this does not create contention.
- **Member ids, not members.** A membership holds identity's member id and
  nothing else; there is no foreign key from guilds' tables to identity's.
- **Invite is its own aggregate.** Accepting changes two things: the guild
  gains a member and the invite counts a use. One transaction changes one
  aggregate, so the member is added first and the use counted after; two
  people accepting at the same moment can go one use over the limit. That is
  accepted: a use limit is a courtesy, not a security boundary.
- **Archive, never delete.** A guild holds its members' work; archiving hides
  it and makes it read-only, and any member can bring it back.
- **No roles yet.** Everyone in a guild can do everything. Roles are easy to
  add inside this context later without changing `IsMember` callers.
