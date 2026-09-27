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
| Web session | A sign-in from the website: the same JWT, carried in an httpOnly cookie instead of a bearer header. |
| Operator | The platform owner (and appointees) using the moderation console. Not a Member account; not modelled yet. |

## Model

### Aggregates

| Aggregate | Invariants |
| --------- | ---------- |
| Member (root) | Email is a bare address, unique, stored trimmed and lowercased. The password is at least 8 characters and only ever stored hashed. Display name is 1–60 characters (trimmed). |

### Commands

- **Register** — create a Member from email, display name and password; returns a token.
- **Log in** — check credentials; returns a token. A failure never says whether the email exists.
- **Current member** — the Member behind a token.

### Domain events

None yet.

## Integration

- **Publishes** (package `contexts/identity`, the only one other code may
  import): `identity.RequireMember`, the middleware that refuses requests
  without a valid token (401); `identity.MemberID(ctx)`, the id of the member
  it let through; `identity.MemberIDByEmail(ctx, email)` for guilds.
- **Consumes:** nothing.

## Why it's shaped this way

- **Generic, so kept small.** Goravel's JWT guard and hashing do the work;
  the context only adds the Member invariants on top. Members are stored in a
  `members` table; the token carries only the member id.
- **Tokens last 30 days.** The desktop app keeps the token in the OS keyring
  so a member stays signed in between launches. There is no refresh or
  revocation list yet; changing `JWT_SECRET` signs everyone out.
- **Other contexts get an id, not a Member.** Guilds and boards only need to
  know *who* is asking, so they receive a plain member id and never import
  identity's types. That keeps identity replaceable (e.g. by an external
  identity provider) without touching the core.
