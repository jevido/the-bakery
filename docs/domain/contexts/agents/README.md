# agents

- Subdomain: supporting
- Hosted in: `services/api` (module `contexts/agents`). The desktop app keeps
  a mirror of the signed-in member's agents as folders
  (`apps/desktop/internal/agents`); it is a copy the app syncs, not a second
  model.

## Purpose

A member's roster of configured Claude CLI agents: who each agent is (name,
title, backstory, traits), how it runs (model, permission mode, allowed
tools), its skillset (the files of its `.claude/skills`), and its work
priorities per kind of work. Agents can be shared with a guild, and other
members of that guild recruit their own copy.

It is **not** responsible for running agents (the Claude runner, later) or for
choosing which task an agent works on next (the scheduler, later), and it does
not own work types: those belong to boards.

## Language

| Term | Meaning |
| ---- | ------- |
| Agent | A configured Claude CLI worker owned by one member. |
| Roster | All agents of one member. |
| Skillset | An agent's skills: the files that make up its `.claude/skills` directory. |
| Skill | One directory of a skillset, with a `SKILL.md` at its top. |
| Trait | A fixed personality or working-style option; each adds one line to the agent's instructions. Some traits conflict. |
| Work priority | 1 (first) to 4 (last), or off, per agent per work type key. |
| Share | An owner makes an agent available to one guild they are in. |
| Recruit | A member of that guild copies a shared agent into their own roster. |
| Origin | The agent a recruited copy was made from, and its revision at the time. |
| Revision | A number that goes up by one with every change to an agent. |

## Model

### Aggregates

| Aggregate | Invariants |
| --------- | ---------- |
| Agent (root) | Owned by one member (by id). Slug unique among the owner's agents: lowercase letters, digits, hyphens, 1–40. Name 1–40 characters, title 0–60, backstory 0–2 000. Traits from the fixed list, no duplicates, no conflicting pair (e.g. `careful` + `fast-worker`). Model `sonnet`, `opus`, `haiku` or a full `claude-*` id. Permission mode one of `manual`, `acceptEdits`, `plan`, `dontAsk`, `auto` — never `bypassPermissions`. At most 100 allowed tools. Work priorities map a work type key to 1–4; a missing key is off. Revision starts at 1; every change names the revision it was based on and bumps it by one, and a change based on an older revision is refused. |
| Skillset (inside Agent) | Every path is `<skill>/<file…>`: relative, no `..`, no hidden directories. Skill names are lowercase letters, digits and hyphens, at most 64. Every skill has a `SKILL.md` whose YAML front matter has a `name` equal to the directory name and a non-empty `description`. UTF-8 text only; at most 50 skills, 300 files, 256 KB per file, 2 MB per agent. |
| Share | An agent shared with a guild: one pair of (agent, guild). Only the owner shares or unshares, and only with a guild they are a member of. |

A recruited copy is an ordinary Agent owned by the recruiter, with its origin
agent id and revision kept for pulling updates later.

### Commands

- **Create** an agent (revision 1); **revise** it (the whole agent: fields and
  skillset, based on a revision); **delete** it (kept as a tombstone so other
  devices learn about it); **list** my agents, optionally only those changed
  since a time; **get** one with its files.
- **Share** an agent with a guild; **unshare** it; **list** the agents shared
  with a guild.
- **Recruit** a shared agent into my roster; **check** whether my copy's origin
  has a newer revision; **pull** the origin's fields and skillset into my copy,
  keeping my own work priorities.

Only an agent's owner can see or change it; to anyone else it does not exist.
Sharing, listing a guild's agents and recruiting require membership of that
guild.

### Domain events

- `AgentCreated` — agent id, owner id, revision 1.
- `AgentRevised` — agent id, owner id, new revision.
- `AgentDeleted` — agent id, owner id.
- `AgentShared` / `AgentUnshared` — agent id, guild id.
- `AgentRecruited` — new agent id, recruiter id, origin agent id and revision, guild id.

## Integration

- **Publishes:** the agents REST endpoints (the desktop's sync talks to them)
  and, for the MCP server, read-only functions to list and get agents (without
  file contents).
- **Consumes:** identity's authenticated member id and display names by id;
  guilds' `Memberships.IsMember` for sharing, listing and recruiting; boards'
  work type **keys** as published language (`coding`, `research`, …), taken as
  plain strings and never checked against boards' tables.

## Why it's shaped this way

- **Agents belong to a member, not a guild.** The same agent works across
  guilds and devices, the way a colonist is yours wherever you take them.
  Sharing copies instead of linking, so a recruiter's edits never change the
  owner's agent and the owner deleting theirs never breaks a copy. The cost: a
  copy drifts from its origin until its member pulls the update.
- **Work priorities are keyed by the work type key, not an id.** A member's
  agent works in many guilds, each with its own list of work types. A key a
  guild doesn't have simply counts as off there, and a priority set for one
  guild's list survives another guild's list.
- **The whole agent is written in one request, guarded by its revision.** A
  skillset is many files that only make sense together; writing them one by
  one would let a sync leave half an agent behind. The revision turns two
  devices editing at once into a visible conflict instead of a silent last
  write wins. The cost: every save sends the whole agent (limited to 2 MB).
- **Tombstones instead of hard deletes.** A device that was offline has to
  learn that an agent was deleted, or it would upload it again.
