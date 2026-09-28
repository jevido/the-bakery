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
| Member | A person with an account: email, display name, password, and a portrait seed. |
| Portrait seed | Draws the member's generated pixel portrait; a re-roll replaces it. Presentation only. |
| Credentials | A Member's email and hashed password. |
| Token | The JWT a Member receives on register or login. |
| Web session | A sign-in from the website: the same JWT, carried in an httpOnly cookie instead of a bearer header. |
| Personal token | A long-lived secret for a tool, `bky_…`; named, shown once, stored as SHA-256, revocable. Not the desktop's session JWT. |
| Handoff code | A one-time code (60 s, single use, stored only as a hash) that turns the desktop's sign-in into a web session. |
| Operator | The platform owner (and appointees) using the console. Not a Member account: operators are the moderation context's own accounts. |

## Model

### Aggregates

| Aggregate | Invariants |
| --------- | ---------- |
| PersonalToken (root) | Belongs to one member. Name is 1–60 characters. The secret is `bky_` + 32 URL-safe random characters and is only ever stored as its SHA-256. A revoked token never authenticates. Only a session (desktop JWT or web cookie) can create or list tokens, so a leaked token cannot mint more. |
| Member (root) | Email is a bare address, unique, stored trimmed and lowercased. The password is at least 8 characters and only ever stored hashed. Display name is 1–60 characters (trimmed). The portrait seed is set at registration and replaced by a re-roll; no rule depends on it. |

### Commands

- **Register** — create a Member from email, display name and password; returns a token.
- **Log in** — check credentials; returns a token. A failure never says whether the email exists.
- **Current member** — the Member behind a token.
- **Create**, **list** and **revoke personal tokens** — for the signed-in
  member; revoking someone else's token looks like it does not exist.
- **Hand off to the website** — for a signed-in member, a handoff code;
  **redeem** it on the website for a web session.

### Domain events

- `PersonalTokenCreated` — token id, member id, name.
- `PersonalTokenRevoked` — token id, member id.

## Integration

- **Publishes** (package `contexts/identity`, the only one other code may
  import): `identity.RequireMember`, the middleware that refuses requests
  without a valid token (401); `identity.MemberID(ctx)`, the id of the member
  it let through (bearer JWT, web session or personal token);
  `identity.MemberIDByEmail(ctx, email)` for guilds; and
  `identity.VerifyPersonalToken(ctx, token)` for the MCP server.
- **Consumes:** nothing.

## Why it's shaped this way

- **Generic, so kept small.** Goravel's JWT guard and hashing do the work;
  the context only adds the Member invariants on top. Members are stored in a
  `members` table; the token carries only the member id.
- **Handoff codes instead of tokens in URLs.** The desktop opens the
  website signed in by passing a code in the URL, not its token: URLs end up
  in browser history and logs. A code works once, within 60 seconds, and only
  its hash is stored.
- **Tokens last 30 days.** The desktop app keeps the token in the OS keyring
  so a member stays signed in between launches. There is no refresh or
  revocation list yet; changing `JWT_SECRET` signs everyone out.
- **Other contexts get an id, not a Member.** Guilds and boards only need to
  know *who* is asking, so they receive a plain member id and never import
  identity's types. That keeps identity replaceable (e.g. by an external
  identity provider) without touching the core.
