# identity

- Subdomain: generic
- Hosted in: `services/api` (module `contexts/identity`)

## Purpose

Knows who a person is: Member accounts, their credentials, and the tokens that
prove a request comes from a signed-in Member. It is **not** responsible for
which guilds a Member belongs to or what they may do there; that is guilds.

## Language

| Term | Meaning |
| ---- | ------- |
| Member | A person with an account: email, display name, password. |
| Credentials | A Member's email and hashed password. |
| Token | The JWT a Member receives on register or login. |
| Operator | The platform owner (and appointees) using the moderation console. Not a Member account; not modelled yet. |

## Model

### Aggregates

| Aggregate | Invariants |
| --------- | ---------- |
| Member (root) | Email is unique and stored lowercased. The password is only ever stored hashed. Display name is 1–60 characters. |

### Commands

- **Register** — create a Member from email, display name and password; returns a token.
- **Log in** — check credentials; returns a token. A failure never says whether the email exists.
- **Current member** — the Member behind a token.

### Domain events

None yet.

## Integration

- **Publishes:** `identity.MemberID(ctx)`, the authenticated member id put on
  the request by the `RequireMember` middleware; a lookup of a member id by
  email for guilds.
- **Consumes:** nothing.

## Why it's shaped this way

- **Generic, so kept small.** Goravel's JWT guard and hashing do the work;
  the context only adds the Member invariants on top. Goravel's `users` table
  may stay as storage, but nothing outside `infra/` calls it a user.
- **Other contexts get an id, not a Member.** Guilds and boards only need to
  know *who* is asking, so they receive a plain member id and never import
  identity's types. That keeps identity replaceable (e.g. by an external
  identity provider) without touching the core.
