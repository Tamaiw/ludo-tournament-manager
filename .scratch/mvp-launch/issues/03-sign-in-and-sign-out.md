# 03: Sign-in / sign-out (sessions, Argon2id, CSRF, middleware chain)

**What to build:** A user can sign in at `GET /sign-in` (form) and `POST /sign-in` (handler), and sign out at `POST /sign-out`. Successful sign-in writes a session cookie (`__Host-id`, `HttpOnly`, `SameSite=Strict`, `Secure` driven by `COOKIE_SECURE`) backed by the `sessions` table (via a hand-written scs.Store adapter on top of `modernc.org/sqlite`), and `RenewToken` is called before persisting the user ID. The CSRF middleware (`gorilla/csrf`) is in the chain; every form renders `{{ .CSRFField }}`. The `PasswordHasher` port is implemented by the Argon2id adapter (m=19456, t=3, p=1, PHC format). Wrong email / wrong password returns a generic "invalid credentials" error (no enumeration). Sign-out calls `RenewToken` then `Destroy`. The `Secure` cookie flag is `true` when `COOKIE_SECURE=true` and `false` otherwise, allowing the dev (HTTP) flow to work without disabling the security model.

**Blocked by:** 02 (Seed CLI and first-manager bootstrap)

**Status:** ready-for-agent

- [ ] `internal/core/services/sign_in.go` with `Handle(ctx, SignInCmd) (User, error)`: looks up user by email, verifies password via the `PasswordHasher` port, calls `Sessions.RenewToken` then `Sessions.Put(ctx, "user_id", user.ID)`. Returns `ErrInvalidCredentials` on any failure (no enumeration).
- [ ] `internal/core/services/sign_out.go` with `Handle(ctx) error`: calls `Sessions.RenewToken` then `Sessions.Destroy`
- [ ] `internal/adapters/outbound/argon2/hasher.go` implements `PasswordHasher` (`Hash`, `Verify`) with PHC encoding and `needsRehash` reporting
- [ ] `internal/adapters/outbound/sqlite/session_store.go` implements the scs.Store interface (~40 lines) on top of `modernc.org/sqlite`; enables `HashTokenInStore = true`
- [ ] `internal/adapters/inbound/http/middleware/{session,csrf}.go` wire scs's `LoadAndSave` and gorilla/csrf's `Protect` into the chain (order per W10)
- [ ] `internal/adapters/inbound/http/handlers/auth.go` has `SignInPage`, `SignInSubmit`, `SignOutSubmit`
- [ ] `internal/adapters/inbound/http/templates/{layouts/base.html, pages/sign_in.html}` render the sign-in form with `{{ .CSRFField }}` and flash messages
- [ ] `internal/core/ports/auth.go` declares `PasswordHasher` and `SessionStore`
- [ ] Tests at the service layer: `sign_in_test.go` covers correct credentials, wrong password, unknown email (same error for the latter two), `RenewToken` called on success, `needsRehash=true` path re-hashes
- [ ] Tests at the HTTP handler layer: `auth_handler_test.go` covers GET renders the form with CSRF field, POST with valid creds sets cookie and redirects, POST with bad creds re-renders form with error, POST without CSRF token is rejected, POST /sign-out clears the cookie and redirects to `/`
- [ ] Manual smoke test: seed a user, start the server, sign in via the form, see the `__Host-id` cookie, sign out, see the cookie cleared
