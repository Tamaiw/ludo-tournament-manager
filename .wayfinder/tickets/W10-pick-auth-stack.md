# W10: Pick session / auth middleware stack

**Type:** research
**State:** closed
**Assignee:** opencode-research
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

## Resolution (2026-10-07)

Locked five pieces that together cover the auth story:

| Concern | Choice |
|---|---|
| **Sessions** | `alexedwards/scs/v2` (v2.9.0, active in 2026); server-side store backed by SQLite (`modernc.org/sqlite`) via a ~40-line `scs.Store` adapter (`scs/sqlite3store` uses `mattn/go-sqlite3` cgo, rejected). `Lifetime = 7d`, `IdleTimeout = 24h`, `Cookie.Name = "__Host-id"`, `SameSite=Strict`, `HttpOnly=true`, `Secure=true` (prod) / `false` (dev — config flip). `HashTokenInStore = true` (v2.8.0+) so raw token never sits in the DB. |
| **CSRF** | `gorilla/csrf` v1.7.3 (active enough; CVE-2025-24358 patched 2025-04-14). Masked-token pattern, signed cookie trust anchor, Origin/Referer check, BREACH-safe. `__Host-`-style config, `SameSite=Strict`, `HttpOnly=true`. Token rendered into templates via `csrf.TemplateField(r)`. CSRF middleware is independent of scs — it has its own signed cookie, so it protects the sign-in form too. |
| **Password hashing** | `golang.org/x/crypto/argon2` Argon2id — `m=19456` (19 MiB), `t=3`, `p=1`, 16-byte salt, 32-byte key. Stored in PHC format `$argon2id$v=19$m=…,t=…,p=…$<salt>$<hash>`. OWASP 2026 primary; bcrypt explicitly framed as "legacy only" by OWASP. Re-hash-on-verify supported. |
| **Invite / password-reset tokens** | One `auth_tokens` table; opaque random token (256 bits from `crypto/rand`, base64url), SHA-256 hash stored; `kind` enum (`invite` / `password_reset`); `expires_at` + `used_at` columns. Same machinery for both. URL-borne (not cookie-borne). No HMAC signing needed — DB row is the authority. |
| **Authorization** | Chi `r.With(...)` middleware: `RequireUser` reads session user ID; `RequireTournamentRole.For(role)` against `tournament_manager` / `tournament_player` for the tournament in the URL. The matrix in `docs/permissions.md` drives which group each route lives in. |

**Why not JWT / signed cookies for sessions:** stateless tokens can't be revoked without a deny-list, which is just a worse session store. Server-side sessions keep invalidation power in our hands.

**Why Argon2id over the ticket's "obvious default" bcrypt:** OWASP 2026 ranks bcrypt as legacy-only; Argon2id is memory-hard (GPU-unfriendly), has no 72-byte password truncation issue, is in `golang.org/x/crypto` (zero extra deps), and supports PHC strings for future re-hash. No backward-compat constraint exists (v1, no users yet).

**Why gorilla/csrf over hand-rolled double-submit:** masking pattern + Origin/Referer check + BREACH mitigation are security primitives that get subtly wrong in hand-rolled impls. gorilla/csrf is the smallest production-ready choice; v1.7.3 still gets security patches.

**Logout:** `RenewToken` then `Destroy` (per OWASP "renew after privilege change"). scs handles both; the cookie attributes used to clear the session are the same as the originals.

**Password-reset reuse:** `auth_tokens.kind = 'password_reset'` reuses the same generate/store/redeem flow as invite. Self-serve forgot-password is out of scope for v1 (manager-issued only); the row exists so v2 can add it without a schema migration.

**Spectator with account (v2+):** no stack change required. New `kind = 'spectator_signup'`, new `RequireTournamentRole.For(RoleSpectator)` middleware variant, new endpoint that requires no manager. Sessions / CSRF / password hashing unchanged.

**Dev / prod parity:** two `Secure` cookie flags flipped by env (`COOKIE_SECURE`, single source of truth drives both scs + csrf). All other attributes identical. Same Docker image, same `cmd/server/main.go`, different `.env`.

**Files added to the locked hexagonal layout** (W12):
- `internal/adapters/outbound/sqlite/session_store.go` — scs.Store adapter
- `internal/adapters/outbound/sqlite/auth_token_repo.go` — auth_tokens CRUD
- `internal/adapters/outbound/argon2/hasher.go` — Argon2id Hash/Verify + PHC encode/decode
- `internal/adapters/inbound/http/middleware/session.go` — scs.LoadAndSave wrapper
- `internal/adapters/inbound/http/middleware/csrf.go` — csrf.Protect wrapper
- `internal/adapters/inbound/http/middleware/require_user.go` — session → request context
- `internal/adapters/inbound/http/middleware/require_tournament_role.go`
- `internal/core/domain/user.go` and `auth_token.go` — User entity + AuthTokenKind enum (currently "out of scope" in `docs/domain-model.md`; resolved by this ticket)

Detailed evaluation, rejected alternatives, and source links: [`../research/W10-auth-stack.md`](../research/W10-auth-stack.md).