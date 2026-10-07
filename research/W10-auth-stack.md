# W10: Session / auth middleware stack — research and recommendation

**Type:** research
**Date:** 2026-10-07
**Status:** recommendation

## TL;DR

Five coherent choices that together cover sessions, CSRF, password hashing, invite/reset tokens, and authorization:

| Concern | Choice |
|---|---|
| **Sessions** | [`alexedwards/scs/v2`](https://github.com/alexedwards/scs) — server-side store, stdlib-shaped middleware, `RenewToken` for fixation defence. Cookie name `__Host-id`. Store: SQLite (`modernc.org/sqlite`) via a small adapter (no `scs/sqlite3store` — it depends on `mattn/go-sqlite3` with cgo). |
| **CSRF** | [`gorilla/csrf`](https://github.com/gorilla/csrf) v1.7.3 — masked-token pattern, signed-cookie trust anchor, Chi-compatible `func(http.Handler) http.Handler`. Token rendered into `html/template` via `csrf.TemplateField(r)`. |
| **Password hashing** | [`golang.org/x/crypto/argon2`](https://pkg.go.dev/golang.org/x/crypto/argon2) with Argon2id — `m=19456` (19 MiB), `t=2`, `p=1`, 16-byte salt, 32-byte key; stored in the PHC encoding `$argon2id$v=19$m=...,t=...,p=...$<salt>$<hash>`. |
| **Invite / password-reset tokens** | One opaque random token per request, **256 bits from `crypto/rand`**, base64url-encoded; store SHA-256 of the token + expiry + `kind` (`invite` / `password_reset`) in a single `auth_tokens` table; on accept/redeem, mark `used_at`. Same machinery for both invite and password-reset. |
| **Authorization (per-tournament)** | Chi `r.With(...)` route groups; a thin `RequireUser` middleware reads the user ID from the session, a `RequireTournamentRole(tournamentID, role)` middleware joins the session user against `tournament_manager` / `tournament_player` for the tournament in the URL. The matrix from [`docs/permissions.md`](../../docs/permissions.md) drives which group each route lives in. |

All five choices slot into the locked hexagonal layout (W12) at the existing `internal/adapters/outbound/{bcrypt,session}` and `internal/adapters/inbound/http/middleware` seams.

## Constraints (from W10)

- Backend is Chi v5 (W1) — stdlib-shaped middleware preferred.
- Auth model: email + password, invite-only user creation, per-tournament role (manager / player).
- Must work over HTTPS in prod (Hetzner) and HTTP in dev (Docker); no TLS that breaks local.
- Session store in SQLite (W3) — a `sessions` table or similar.
- No third-party SaaS for auth — user data stays local.
- Stack already locked: Go + Chi + SQLite (modernc) + htmx + Alpine + Go `html/template` + Resend + Mailpit.

## Stack details

### 1. Sessions — `alexedwards/scs/v2`

**Why SCS over JWT / signed cookies / `gorilla/sessions`:**

- **Server-side sessions** keep state (and invalidation power) in the SQLite DB. A signed cookie / JWT is fine for "did you pay?" but bad for "we need to force-logout this user on incident" — invalidating a stateless cookie is impossible without a revocation list, which is just a worse session store. The CSRF middleware (gorilla/csrf, below) needs only its own short-lived signed cookie and doesn't depend on scs.
- **stdlib-shaped middleware.** `sessionManager.LoadAndSave(next)` is `func(http.Handler) http.Handler` — drops into Chi's middleware chain with no wrapping. Same as gorilla/csrf below. This matches the "stdlib-shaped middleware preferred" constraint.
- **Active maintenance in 2026.** Latest is `v2.9.0` (released 2025-07-08) with `Partitioned` cookie attribute, `Hijacker`/`Flusher` support, `HashTokenInStore` option. `gorilla/sessions` exists but the gorilla project has been de facto dead since 2024 (same story as `gorilla/websocket` in W2).
- **Native fix for session fixation.** `sessionManager.RenewToken(ctx)` generates a new session ID *before* a privilege change (login, logout, role elevation). The OWASP Session Management cheat sheet calls this out as required; the library's README has the exact pattern.

**Configuration (final values):**

```go
sessionManager := scs.New()
sessionManager.Lifetime    = 7 * 24 * time.Hour   // absolute timeout
sessionManager.IdleTimeout = 24 * time.Hour       // idle timeout
sessionManager.Cookie.Name = "__Host-id"         // see "Cookie attributes" below
sessionManager.Cookie.HttpOnly = true
sessionManager.Cookie.SameSite = http.SameSiteStrictMode
sessionManager.Cookie.Secure   = true             // see "Dev / prod parity" below
sessionManager.Cookie.Path     = "/"
sessionManager.Store = scsStore.NewStore(db)     // SQLite adapter — see below
```

`Lifetime` is the absolute timeout (total session age); `IdleTimeout` is the inactivity timeout. Both are enforced server-side; scs resets the expiry on every request and the background cleanup goroutine deletes expired rows.

**SQLite adapter (we write it ourselves).**

The bundled `scs/sqlite3store` only takes `*sql.DB` from `github.com/mattn/go-sqlite3` (cgo) — it works with *any* `database/sql` driver in practice (it only uses `?`-style placeholders and a single `expiry REAL` column). W3 picked `modernc.org/sqlite` for cgo-free Windows builds, so we have two options:

1. **Add `mattn/go-sqlite3` for sessions only** — runs the existing `scs/sqlite3store` against the same connection. Cleaner code, but adds cgo to the build, breaks the W3 "cgo-free" decision for a single feature. Rejected.
2. **Write a ~40-line `scs.Store` adapter on top of `modernc.org/sqlite`.** The scs Store interface is four methods (`Delete`, `Find`, `Commit`, optional `All`); the table shape is `token TEXT PRIMARY KEY, data BLOB, expiry REAL` with an `expiry_idx` index. The hexagonal port `SessionStore` already declared in [`docs/architecture/hexagonal.md`](../../docs/architecture/hexagonal.md) sits on top of this adapter.

We pick option 2. The adapter lives in `internal/adapters/outbound/sqlite/session_store.go`; the port interface (`internal/core/ports.SessionStore`) is implemented by an `sqlite.SessionStore` shim that calls into scs's `Store` methods. This keeps the W3 decision load-bearing and adds maybe 40 lines of code.

**Schema:**

```sql
CREATE TABLE sessions (
    token   TEXT PRIMARY KEY,         -- the cookie value (scs sets; we store the raw token)
    data    BLOB NOT NULL,            -- gob-encoded session contents (userID, flash, etc.)
    expiry  REAL NOT NULL,           -- unix seconds; scs renews on every request
    CONSTRAINT expiry_idx_below_max CHECK (expiry < 4102444800) -- sanity, year 2100
);
CREATE INDEX sessions_expiry_idx ON sessions(expiry);
```

The scs v2.8.0+ option `HashTokenInStore = true` stores the SHA-256 hash of the token in the `token` column (so the raw cookie value never sits verbatim in the DB), matching OWASP's "store a one-way verifier" guidance. We enable this.

**Rejection of alternatives:**

- **JWT.** Stateless, but we can't revoke a stolen JWT before its `exp` without a server-side deny-list — which is just a worse session store. JWT also pulls in key-management complexity (rotation, key-id header, public-key distribution if we ever split services). For a single-server v1 with SQLite, sessions are simpler and more secure. *Future note:* if we ever federate auth across hosts (v2+), the same scs store can be backed by Postgres/Redis with no API change.
- **Signed cookies only (no server-side state).** Same revocation problem as JWT, plus the cookie body has to fit the entire session payload (cookie size limits, ~4 KB). Rejected.
- **`gorilla/sessions`.** Works, but the gorilla project is effectively dead in 2026 (last meaningful work before the August 2024 handover; no releases since). Adopting it means inheriting an unmaintained dependency.
- **Custom in-house middleware.** Doable — a `LoadAndSave` is ~100 lines plus a `Store` interface and a cleanup loop. But the security-relevant details (cookie attribute defaults, BREACH-safe generation, idle-vs-absolute semantics, concurrent goroutine-safety of `LoadAndSave`, partition/codec XOR masking) are exactly the bits that get subtly wrong in hand-rolled implementations. Using the battle-tested library is the right call for a security primitive.

**Dev / prod parity.** SCS cookie attributes must differ between dev (HTTP) and prod (HTTPS):

- `Secure: true` blocks the cookie over plain HTTP. In dev (Docker compose, `http://localhost:8080`), setting `Secure: true` means login would set the cookie but the next request wouldn't carry it back, breaking sign-in.
- Fix: the `Secure` flag is driven by config. Default `false` (dev), flip to `true` when `APP_ENV=production` or `COOKIE_SECURE=true`. The code path is unchanged; only the config bit flips. *This is the same pattern as `Cookie.HttpOnly` being always `true` regardless of environment.*

### 3. Password hashing — Argon2id (not bcrypt)

**Why Argon2id over `golang.org/x/crypto/bcrypt`:**

The OWASP Password Storage Cheat Sheet ([`cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html`](https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html)) recommends, in order:

> Use Argon2id with a minimum configuration of 19 MiB of memory, an iteration count of 2, and 1 degree of parallelism. If Argon2id is not available, use scrypt… For legacy systems using bcrypt, use a work factor of 10 or more.

bcrypt is now framed by OWASP as "for legacy systems where Argon2 and scrypt are not available" — which is the exact opposite of our situation: this is a brand-new project in 2026 with a Go stdlib-adjacent dependency (`golang.org/x/crypto/argon2`). The cost of choosing Argon2id is essentially zero; the cost of choosing bcrypt and having to migrate later is a real one.

**Why not just bcrypt anyway (the ticket's "obvious default"):**

- bcrypt is CPU-bound; Argon2id is memory-hard. GPU/ASIC attackers pay real money to brute-force Argon2id at the memory levels we're picking (~19 MiB per attempt × thousands of parallel attempts = gigabytes of VRAM/GDDRL6).
- The "72-byte input limit" trap from bcrypt (any longer password is silently truncated) goes away with Argon2id (no input limit; Unicode-clean).
- The PHC string format encodes the algorithm + parameters in the hash itself (`$argon2id$v=19$m=19456,t=2,p=1$<salt>$<hash>`), making future parameter upgrades transparent: on verify, parse the stored string, recompute with the new params, write back. bcrypt has the same property via its `$2b$12$…` prefix — both support re-hash-on-login, but Argon2id starts on a stronger foot.

**Configuration (final values):**

```go
// internal/adapters/outbound/argon2/hasher.go
const (
    argonTime    = 3          // OWASP recommends t=2; we use t=3 for slightly more headroom
    argonMemory = 19456       // 19 MiB; one of OWASP's recommended minimums
    argonPar    = 1
    saltLen     = 16          // 128-bit salt
    keyLen      = 32          // 256-bit derived key
)
```

The PHC-encoded hash is what's stored in `users.password_hash`. The `PasswordHasher` port (`internal/core/ports/auth.go`) has two methods: `Hash(password string) (string, error)` returning the PHC string, and `Verify(hash, password string) (bool, error)`. The verify impl parses the PHC string, recomputes with the same params, and constant-time compares.

**Re-hash on parameter upgrade.** When OWASP bumps the minimums (or our hardware changes), `Verify` can detect a too-old `$argon2id$v=19$m=...,t=2,...` and re-hash with the current params. The service layer (`internal/core/services/sign_in.go`) checks the returned `bool` plus an optional `needsRehash bool` flag and persists the new hash on successful login. The cost is one extra `argon2id` call per login for the duration of the transition.

**Backwards compatibility note.** No backwards compatibility needed. This is v1; no users exist yet; no legacy hashes to support. If we ever did have bcrypt hashes, the `Verify` impl would parse the `$2b$12$…` prefix and dispatch to a bcrypt path — but YAGNI for v1.

### 4. Invite / password-reset tokens — opaque random + DB-backed hash

**Why an opaque random token over a JWT / HMAC-signed token:**

- **One-shot semantics.** Both invite and password-reset tokens must be single-use (`used_at` column). JWTs don't support this without a deny-list or a server-side state lookup, which defeats the purpose of being stateless. An HMAC-signed opaque token has the same problem (you can't mark it used without state).
- **Easy revocation.** A manager that issues an invite and wants to revoke it before it's accepted just sets `revoked_at`. A JWT doesn't.
- **Secret-key management.** HMAC-signed tokens require a server-side secret. The opaque-token approach doesn't sign — the DB row *is* the authority. There's no secret to rotate, leak, or otherwise manage.

**Design (final):**

```sql
CREATE TABLE auth_tokens (
    id         TEXT PRIMARY KEY,              -- UUID; used for revocation UX (manager UI lists by id, not token)
    kind       TEXT NOT NULL CHECK (kind IN ('invite', 'password_reset')),
    user_id    TEXT,                           -- nullable; set after invite acceptance
    email      TEXT NOT NULL,                  -- the email being invited / the reset target
    token_hash  TEXT NOT NULL UNIQUE,           -- sha256 hex of the raw token; the raw token never leaves the email
    expires_at INTEGER NOT NULL,               -- unix seconds; 7 days for invite, 1 hour for password reset
    used_at    INTEGER,                        -- NULL until redeemed
    created_by TEXT,                           -- manager user_id; NULL for self-served password resets (v2)
    created_at INTEGER NOT NULL
);
CREATE INDEX auth_tokens_email_idx ON auth_tokens(email);
CREATE INDEX auth_tokens_expiry_idx ON auth_tokens(expires_at);
```

**Token generation:**

```go
// internal/core/services/auth_token.go
rawToken := make([]byte, 32)            // 256 bits
if _, err := rand.Read(rawToken); err != nil { return err }
tokenStr := base64.RawURLEncoding.EncodeToString(rawToken)
hash := sha256.Sum256([]byte(tokenStr))
// persist hash (hex), expose tokenStr (raw) only via the email + the manager's "copy URL" UI
```

The cookie `__Host-` prefix is *not* relevant here because they're URL query params, not cookies. The `__Host-` prefix is reserved for the cookie-residing tokens (scs session, gorilla/csrf's masked token).

**Why two paths (invite + password reset) reuse one table:**

- Same machinery (generate, store hash, send link, redeem, mark used, expire cleanup).
- Same security properties (CSPRNG, single-use, expiry, hash storage).
- One `AuthTokenRepository` instead of two; one set of tests.
- One `cleanup` cron / goroutine that deletes expired-and-unused rows (separate from session cleanup, which is per-request).

`kind` differentiates the redemption flow: `invite` redemption creates the user (sets `password_hash`, fills `user_id`); `password_reset` redemption requires the user to already exist and just updates `password_hash`. The redemption handler dispatches on `kind`.

**Dev / prod parity for the link URL.** Token is delivered as a query param (`?token=…`) in the invite / reset URL. Works identically over HTTP (dev) and HTTPS (prod). No cookie attributes apply.

**Out of scope (v1, by W7 / global decisions):**

- "Password reset requested by user" self-serve flow (user clicks "forgot password" without being a manager) — *out of scope*. v1 is manager-only issuance; manager re-invites a user whose password needs resetting, or creates a fresh invite. The `password_reset` path is wired but only triggered by manager action.
- E-mail enumeration via timing — out of scope. v1 has no public sign-up; the invite endpoint is manager-only and behind auth.

### 5. Authorization — Chi `r.With(...)` middleware

**Per-tournament role check.** The permission matrix from [`docs/permissions.md`](../../docs/permissions.md) is role × action; for "edit this tournament" the answer is "are you a manager of *this* tournament?". The middleware reads the session user ID, looks up the tournament ID from the URL (`chi.URLParam(r, "tournamentID")`), joins against `tournament_manager` or `tournament_player`, and either proceeds or 403s.

**Pattern (final):**

```go
// internal/adapters/inbound/http/middleware/require_role.go
type RequireTournamentRole struct {
    Session   *scs.SessionManager
    UserRepo  ports.UserRepository
    ManagerRepo ports.TournamentManagerRepository   // does this user manage this tournament?
    PlayerRepo  ports.TournamentPlayerRepository    // does this user play in this tournament?
    Clock     ports.Clock
}

func (m RequireTournamentRole) For(role domain.Role) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            userID := m.Session.GetString(r.Context(), "user_id")
            if userID == "" { http.Error(w, "unauthorized", http.StatusUnauthorized); return }
            tournamentID := chi.URLParam(r, "tournamentID")
            allowed, err := m.checkRoleAllowed(r.Context(), tournamentID, userID, role)
            if err != nil { /* log; 500 */ }
            if !allowed { http.Error(w, "forbidden", http.StatusForbidden); return }
            next.ServeHTTP(w, r)
        })
    }
}

// in the route table:
r.Route("/tournaments/{tournamentID}", func(r chi.Router) {
    r.With(middleware.RequireUser).Get("/", ...)                          // anyone signed in
    r.With(middleware.RequireTournamentRole.For(domain.RoleManager)).Post("/edit", ...)  // managers
    r.With(middleware.RequireTournamentRole.For(domain.RolePlayer)).Post("/matches/{matchID}/record", ...)  // players
})
```

This is the same shape as Spring Security's `@PreAuthorize` / Struts interceptors — a familiar mental model from a Java/.NET background. The middleware is the *single source* of role enforcement; handlers don't second-guess it.

**Why a custom middleware and not an existing framework:**

- Chi doesn't ship one (per W1: "stdlib-shaped middleware preferred").
- `casbin` / `oso` are over-engineered for a 5-role, ~20-action permission table that's hand-written in markdown. The cost of integrating a policy engine exceeds the cost of writing 100 lines of middleware.
- The matrix is hand-curated in `docs/permissions.md`; making the policy live in *one* file (the markdown) is the right call. Code reflects the rules; tests can pin the matrix.

### 6. CSRF — `gorilla/csrf`

**Why gorilla/csrf over double-submit-cookie-only / `justinas/nofair`:**

- **Masked-token pattern**, not raw double-submit. The cookie holds a "masked" token; the form/header holds the "real" token. Even if the cookie is leaked (XSS, log), the attacker still can't forge a request because they only see the masked form.
- **Signed-cookie trust anchor** via `gorilla/securecookie`. The CSRF cookie is signed (not encrypted) — so the server can verify it wasn't tampered with, but the token itself isn't sensitive enough to need encryption.
- **Origin / Referer validation** built in. The middleware rejects state-changing requests whose `Origin` (preferred) or `Referer` doesn't match an allow-list.
- **Chi-compatible.** `csrf.Protect(authKey, opts...)` returns `func(http.Handler) http.Handler` — same shape as scs's `LoadAndSave`. Drops straight into `r.Use(...)`.
- **BREACH-safe by design** — the token is masked with random XOR per request, so HTTP-compression-based BREACH attacks can't recover it from response bodies.
- **Maintenance.** Latest is `v1.7.3` (2025-04-14), which fixed CVE-2025-24358. Still slow but not dead — recent PRs land within weeks-months. (Contrast with `gorilla/sessions` above.)

**Configuration (final):**

```go
// internal/adapters/inbound/http/middleware/csrf.go
CSRF := csrf.Protect(
    []byte(cfg.CSRFAuthKey),                   // 32-byte secret from config
    csrf.Secure(cfg.CookieSecure),               // same flip as scs (false in dev, true in prod)
    csrf.HttpOnly(true),
    csrf.Path("/"),
    csrf.SameSite(csrf.SameSiteStrictMode),
    csrf.MaxAge(1 * time.Hour),                 // short-lived; the token rotates on every request anyway
    csrf.ErrorHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // render a 403 template, log the rejection
    })),
)
```

The CSRF middleware does NOT depend on scs — it has its own signed cookie (`__Host-csrf-token` by default, configurable). This means CSRF protection works for forms whether or not the user has a session, which is what we want (e.g. the sign-in form must be CSRF-protected too).

**Template integration.** `csrf.TemplateField(r)` returns a `template.HTML` (the hidden `<input type="hidden" name="gorilla.csrf.Token" value="...">`), exposed to templates via the existing data-context pattern. Every `<form method="POST">` in `internal/adapters/inbound/http/templates/` includes `{{ .CSRFField }}`. Field name is left as the default `gorilla.csrf.Token`; renaming it buys nothing.

**Rejection of alternatives:**

- **`justinas/nofair`.** Last meaningful commit 2020. Effectively abandoned. Rejected.
- **Hand-rolled double-submit cookie.** Doable, but the masking pattern, Origin/Referer check, and BREACH mitigation are security primitives — exactly the bits you don't want to be subtle-wrong. `gorilla/csrf` is the smallest production-ready option.
- **`SameSite=Strict` only, no CSRF token.** `SameSite=Strict` is OWASP's "defense in depth, not a replacement." A strict same-site cookie blocks cross-site form submissions but doesn't cover edge cases (top-level navigation that *is* cross-site like a phishing link clicked by a logged-in user, subdomain takeover, etc.). Tokens are required.

**Order in the middleware chain:**

```
Chi middleware chain (top → bottom):
1. middleware.RequestID              // for log correlation
2. middleware.RealIP                 // for rate-limit / audit (proxy-aware)
3. middleware.Logger                 // logs first so it captures everything
4. middleware.Recoverer              // panic recovery (after logger so the log is written)
6. sessionManager.LoadAndSave        // session loading — must come BEFORE auth middleware reads the session
7. csrf.Protect                      // CSRF — must come BEFORE handlers, after session (form rendering needs the session too)
8. (route-group) middleware.RequireUser
9. (route-group) middleware.RequireTournamentRole.For(role)
```

CSRF *after* session because (a) session load is cheaper and shouldn't be gated on CSRF for read-only GETs, and (b) the CSRF middleware needs to set its own cookie *only* on safe methods (GET/HEAD/OPTIONS) — leaving POST/PUT/PATCH/DELETE validation to the OUTPOST path.

## The bundle as a whole — why these five together

```
                                   ┌──────────────────────────┐
                                   │ golang.org/x/crypto/      │
                                   │ argon2  (PasswordHasher)  │
                                   └─────────────┬────────────┘
                                                 │ users.password_hash
                                                 │ (PHC: $argon2id$v=19$...)
                                                 ▼
┌────────────┐  session cookie   ┌──────────────────────────┐
│ Browser    │ ◄───────────────► │ alexedwards/scs/v2        │
│            │   __Host-id=...   │   server-side sessions   │
│            │                  │         │    │             │
│            │   csrf cookie    │         ▼    ▼             │
│            │ ◄───────────────► │   sessions table          │
│            │   __Host-csrf-   │   (modernc.org/sqlite)    │
│            │     token=...    └──────────────────────────┘
│            │
│            │   invite / reset link
│            │ ◄───────────────► ┌──────────────────────────┐
└────────────┘   ?token=...     │  in-house auth_tokens     │
                                 │  (opaque random + hash)  │
                                 └──────────────────────────┘
                                                 ▲
                                                 │ rendered into form via
                                                 │ csrf.TemplateField(r)
                                                 │
                                 ┌──────────────────────────┐
                                 │ gorilla/csrf             │
                                 │   masked-token pattern  │
                                 └──────────────────────────┘
```

- **Sessions** store "what user is this request from?" (signed-cookie token → DB row → user ID).
- **CSRF** runs *in parallel* with sessions: it doesn't know or care who's signed in — its own signed cookie + masked token defends any state-changing request, including the sign-in form itself (where the user isn't signed in yet).
- **Argon2id** stores passwords; defended against a leaked DB.
- **Auth tokens** are URL-borne, not cookie-borne, and are one-shot (invite / password reset).
- **Authorization middleware** composes session + role lookup against `tournament_manager` / `tournament_player`.

A future "spectator with account" feature slots in by:
1. Adding a `spectator` role to `Role` enum.
2. Extending the `auth_tokens` table with a third `kind = 'spectator_signup'` value (if we want to keep the same machinery).
3. Adding a `RequireTournamentRole.For(domain.RoleSpectator)` middleware variant.
4. Adding the spectator role to the permission matrix in `docs/permissions.md`.

No session-stack change. No CSRF change. No password-hash change.

## Files in the hexagonal layout this affects

Locked in W12; these are the seams the auth stack slots into:

```
internal/
├── core/
│   ├── domain/
│   │   └── user.go                         — User, Role enum
│   │   └── auth_token.go                    — AuthToken, AuthTokenKind enum
│   └── ports/
│       └── auth.go                         — PasswordHasher, SessionStore, InviteSigner (renamed: AuthTokenExists)
└── adapters/
    ├── inbound/
    │   └── http/
    │       ├── middleware/
    │       │   ├── session.go               — wraps scs.LoadAndSave
    │       │   ├── csrf.go                  — wraps csrf.Protect
    │       │   ├── require_user.go          — reads session, sets request context
    │       │   └── require_tournament_role.go
    │       └── templates/
    │           └── base.html                — every form gets {{ .CSRFField }}
    └── outbound/
        ├── sqlite/
        │   ├── session_store.go             — ~40-line scs.Store adapter
        │   └── auth_token_repo.go           — CRUD for auth_tokens
        └── argon2/
            └── hasher.go                    — Argon2id Hash/Verify + PHC encode/decode
```

`cmd/server/main.go` (composition root, the only file that imports everything) wires:
- `db := sqlite.Open(cfg.DBPath)`
- `sessionStore := sqlite.NewSessionStore(db)`
- `sessionManager.Store = sessionStore`
- `passwordHasher := argon2.Hasher{}`
- `r.Use(sessionManager.LoadAndSave, csrf.Protect(authKey, ...))`

## Cookie attributes — locked

| Cookie | Name | Purpose | Secure | HttpOnly | SameSite | Path | Notes |
|---|---|---|---|---|---|---|---|
| Session | `__Host-id` | scs session ID | `true` in prod, `false` in dev | `true` | `Strict` | `/` | `__Host-` prefix enforces Secure + Path=/ + no Domain. The `Secure` flip is the only dev-vs-prod configuration delta. |
| CSRF | `_gorilla_csrf` (configurable) | masked CSRF token | `true` in prod, `false` in dev | `true` (gorilla/csrf default) | `Strict` | `/` | Default gorilla/csrf cookie name is `_gorilla_csrf`. Can be changed via `csrf.CookieName(...)`. The `__Host-` prefix isn't mandatory here because gorilla/csrf signs the cookie (and the trust anchor is the cookie signature, not the cookie name). |

`SameSite=Strict` is OWASP's recommended setting for session cookies. It excludes all cross-site requests, including top-level GET navigation. The trade-off: links from email (password reset, invite accept) won't carry the session, but those flows are *unauthenticated* by definition — the user isn't logged in when they click "accept invite" — so the loss of session-cookie-on-cross-site is irrelevant for our flow.

If we ever embed third-party widgets that need the session (we don't, currently), we'd revisit to `Lax`. For v1, `Strict`.

## How sessions are invalidated on logout

```go
// internal/core/services/sign_out.go
func (s SignOut) Handle(ctx context.Context) error {
    // OWASP requires the session ID to change on privilege change;
    // RenewToken issues a new ID while preserving session data.
    if err := s.Sessions.RenewToken(ctx); err != nil { return err }
    // Then destroy the session entirely.
    return s.Sessions.Destroy(ctx)
}
```

Two operations because the OWASP Session Management cheat sheet calls out: renew first (prevent session fixation of the *next* request), then destroy (no usable state remains). In practice `Destroy` is sufficient — it both invalidates the DB row and clears the cookie — but the `RenewToken` call makes the fix explicit and matches the library's own README pattern.

The cookie that scs writes to clear the session uses the same `Secure`/`HttpOnly`/`SameSite`/`Path` attributes as the original. No special logout-only cookie config.

## How password-reset reuses invite-token machinery

One `auth_tokens` table; `kind` differentiates the redemption flow:

| `kind` | Issued by | Lifetime | On redeem |
|---|---|---|---|
| `invite` | A manager (`create_invite` service) | 7 days | Creates the user record; user sets password; `auth_tokens.user_id` filled; audit-logged |
| `password_reset` | A manager (`create_password_reset` service) for now | 1 hour | Requires the user to already exist; updates `users.password_hash`; audit-logged |

The redemption handler dispatches on `kind` and validates the email matches the user record (for `password_reset`) or creates a new user (for `invite`).

The "self-serve forgot-password" flow (no manager involved) is out of scope for v1 (per W7 / global decisions: "no third-party SaaS, no self-service"). The `password_reset` row exists so v2 can add a self-serve flow without a schema migration — just a new `kind` value and a new endpoint that requires no manager.

## Spectator with account (v2+ sketch, not v1)

Future-proofed but not implemented:

1. New `kind = 'spectator_signup'` value (no other change to `auth_tokens`).
2. New `RequireTournamentRole.For(domain.RoleSpectator)` middleware variant.
3. Permission matrix in `docs/permissions.md` extended with `Spectator (with account)` row.
5. Public "request spectator signup" endpoint, manager-approved.

No change to sessions, CSRF, or password hashing. The hexagonal layout means the new role is a new enum value + new middleware variant + new service, not a stack change.

## Dev / prod parity summary

| Concern | Dev (Docker, HTTP) | Prod (Hetzner, HTTPS) |
|---|---|---|
| Session cookie `Secure` | `false` | `true` |
| CSRF cookie `Secure` | `false` | `true` |
| `SameSite=Strict` | `true` | `true` |
| CSRF `Secure(false)` opt-out for tests | `true` | n/a |
| Argon2 params | `m=19456 t=3 p=1` (same) | same |
| Auth token delivery | URL `?token=…` (same) | same |
| Cleanup cron | scs background goroutine (same) | same |

Two env-var-driven flips (`COOKIE_SECURE`, plus `CSRF_COOKIE_SECURE` derived from the same flag if we want to unify). Everything else is config-identical. Same Docker image, same `cmd/server/main.go`, different `.env`.

## What this means for ticket scope

- **W10** (this ticket) — picks the five components and the dev/prod parity model. ✅
- **W11** (deployment shape) — the `COOKIE_SECURE` flip is a config concern, not a deploy concern; nothing extra needed.
- **Future tickets (when implementation starts):**
  - `add_session_table_migration` — `sessions` table per W10 §1 schema.
  - `add_users_table_migration` — `users` + `auth_tokens` tables.
  - `add_auth_services` — sign-in / sign-out / invite / accept-invite / create-password-reset / redeem-password-reset.
  - `add_auth_middleware` — scs `LoadAndSave` wrapper, csrf.Protect, RequireUser, RequireTournamentRole.
  - `add_auth_token_cleanup` — daily cron / background goroutine that deletes expired-and-unused `auth_tokens` rows (separate from scs's session cleanup).

All of the above are scoped at the level of the hexagonal layout; no big-bang stack migration.

## Sources

Primary (fetched 2026-10-07):

- [alexedwards/scs README](https://github.com/alexedwards/scs/blob/master/README.md) — SessionManager config, Store interface, `LoadAndSave`, `RenewToken`, `HashTokenInStore` (v2.8.0+).
- [alexedwards/scs SQLite3 store README](https://github.com/alexedwards/scs/blob/master/sqlite3store/README.md) — schema, cleanup interval option.
- [alexedwards/scs releases](https://github.com/alexedwards/scs/releases) — confirmed v2.9.0 is current (2025-07-08).
- [alexedwards/scs llms.txt](https://context7.com/alexedwards/scs/llms.txt) — Store interface, custom-store recipe, RenewToken usage in login handler.
- [gorilla/csrf Protect reference](https://github.com/gorilla/csrf/blob/main/_autodocs/api-reference/Protect.md) — `func(http.Handler) http.Handler` signature; options list.
- [gorilla/csrf TemplateField reference](https://github.com/gorilla/csrf/blob/main/_autodocs/api-reference/TemplateField.md) — template integration, field-name customization.
- [gorilla/csrf architecture docs](https://github.com/gorilla/csrf/blob/main/_autodocs/architecture.md) — masked-token pattern, double-submit comparison, request flow.
- [gorilla/csrf releases](https://github.com/gorilla/csrf/releases) — confirmed v1.7.3 (2025-04-14, CVE-2025-24358 fix).
- [OWASP Password Storage Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html) — Argon2id m=19456,t=2,p=1 minimum; bcrypt framed as legacy-only.
- [OWASP Session Management Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html) — session ID entropy (≥64 bits), `__Host-` prefix recommendation, `SameSite=Strict`, server-side storage with one-way verifier, RenewToken after privilege change.
- [OWASP Cross-Site Request Forgery Prevention Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Cross-Site_Request_Forgery_Prevention_Cheat_Sheet.html) — `SameSite` as defense-in-depth, not replacement.
- [go-chi Chi docs quickstart](https://github.com/go-chi/docs/blob/master/quickstart.md) — RequestID, RealIP, Logger, Recoverer, CleanPath, Timeout middleware order.
- [go-chi Chi docs routing](https://github.com/go-chi/docs/blob/master/pages/routing.md) — `r.Group`, `r.With(...)` for route-scoped middleware.
- [RFC 9106 — Argon2](https://www.rfc-editor.org/rfc/rfc9106.html#section-3.1) — parameter definitions, parameter choices, security analysis.
- [PHC string format spec](https://github.com/P-H-C/phc-string-format/blob/master/phc-sf-spec.md) — `$argon2id$v=19$m=…,t=…,p=…$<salt>$<hash>`.

Repo-internal:

- [`WAYFINDER.md`](../../WAYFINDER.md) — locked stack (Chi, SQLite, hexagonal).
- [`docs/architecture/hexagonal.md`](../../docs/architecture/hexagonal.md) — port / adapter boundaries; `internal/core/ports/auth.go` (`PasswordHasher`, `SessionStore`, `InviteSigner`).
- [`docs/domain-model.md`](../../docs/domain-model.md) — entity shape (User is out of scope here, deferred to this ticket).
- [`docs/permissions.md`](../../docs/permissions.md) — role × action matrix that `RequireTournamentRole` enforces.
- [`.wayfinder/tickets/W1-pick-go-web-framework.md`](../tickets/W1-pick-go-web-framework.md) — Chi v5 lock; stdlib-shaped middleware preference.
- [`.wayfinder/tickets/W3-pick-database-engine.md`](../tickets/W3-pick-database-engine.md) — SQLite (modernc) lock; no cgo on Windows.
- [`.wayfinder/tickets/W9-pick-go-data-access-layer.md`](../tickets/W9-pick-go-data-access-layer.md) — sqlc; the `session_store.go` adapter sits in the outbound-sqlite adapter, not the data-access layer per se.

## Decisions pending after W10

None — W10 is the auth-stack decision ticket and is fully resolved by this artifact.

Future decisions downstream of implementation:

- **Migration tool** for the `sessions` / `users` / `auth_tokens` tables — picks numbered `.sql` files. Tool itself TBD in a separate ticket; the schema here is the constraint.
- **Cookie / CSRF / Argon2 secret management** — env var for v1, a secrets vault for v2. Picked alongside W11 (deployment shape).