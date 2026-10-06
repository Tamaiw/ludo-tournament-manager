# W10: Pick session / auth middleware stack

**Type:** research
**State:** open
**Assignee:** (unclaimed)
**Blocked by:** W1 ✓, W12 ✓
**Blocks:** W21

## Question

What is the session + auth middleware stack for the Go backend?

Pieces to settle together:
- **Session management**: cookie-backed sessions with server-side store, JWT, signed cookies only
- **CSRF protection**: per-request token, double-submit cookie, framework middleware
- **Password hashing**: `golang.org/x/crypto/bcrypt` is the obvious default — confirm or argue for `argon2id` (memory-hard, OWASP-recommended)
- **Invite-token flow**: how the manager-issued invite token is generated, signed, single-use, expired
- **Authorization**: role/permission checks on routes (per-tournament manager vs player vs spectator)

## Constraints

- Backend is Chi (resolved by W1): stdlib-shaped middleware preferred
- Auth model already settled: email + password, invite-only user creation, per-tournament role (manager or player)
- Must work over HTTPS in prod (Hetzner) and HTTP in dev (Docker); no TLS assumptions that break local
- Session storage in SQLite (resolved by W3): a `sessions` table or similar
- No third-party SaaS for auth (no Auth0, no Clerk) — user data stays local

## What "good" looks like

A coherent stack recommendation that names each library or each in-house pattern, with reasoning for the bundle as a whole. Cover: how sessions are invalidated on logout, how password reset / re-invite flow reuses the invite-token machinery, how a future "spectator with account" (currently none) would slot in. Suggested filename `research/W10-auth-stack.md`.