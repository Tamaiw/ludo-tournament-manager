# 04: Dashboard, RequireUser middleware, base layout

**What to build:** A signed-in user lands on `/dashboard` (renamed from index for signed-in users) which shows "Welcome, {name}" and a placeholder for "tournaments you manage" (empty list for now, populated in T05). The `RequireUser` middleware reads the user ID from the session, 401s if missing, and puts the user into the request context for downstream handlers. The base layout (`layouts/base.html`) renders the header (site name, nav, signed-in user name + sign-out form, CSRF field on the sign-out form), the flash message region, and a `{{ block "content" . }}` slot for pages. The sign-in form from T03 uses this layout. The CSRF field is rendered as `{{ .CSRFField }}` via a template function set up in the server.

**Blocked by:** 03 (Sign-in / sign-out)

**Status:** ready-for-agent

- [ ] `internal/adapters/inbound/http/middleware/require_user.go` reads the user ID from the session, looks up the user via `UserRepository.Find`, 401s if either is missing, puts the user in the request context
- [ ] `internal/core/domain/user.go` has the `User` struct with the fields from the spec
- [ ] `internal/core/services/get_user.go` (or fold into `UserRepository` directly) returns a `User` by ID
- [ ] `internal/adapters/inbound/http/templates/layouts/base.html` renders header, nav, signed-in user, sign-out form with `{{ .CSRFField }}`, flash region, and a `{{ block "content" . }}` slot
- [ ] `internal/adapters/inbound/http/templates/pages/dashboard.html` extends the layout and renders the welcome line + empty tournament list placeholder
- [ ] `internal/adapters/inbound/http/templates/pages/sign_in.html` updated to extend the layout (or stay as a minimal un-laid-out page if that's the convention)
- [ ] `internal/adapters/inbound/http/server.go` registers the routes: `GET /sign-in`, `POST /sign-in`, `POST /sign-out`, `GET /dashboard` (RequireUser)
- [ ] `cmd/server/main.go` wires everything (composition root is now non-trivial but still the only file that imports every layer)
- [ ] Tests at the service layer for `get_user` / user lookup (happy path, not found)
- [ ] Tests at the HTTP handler layer: `GET /dashboard` without a session returns 401, with a session returns 200 and the user's name; `POST /sign-out` from dashboard clears the cookie and redirects to `/sign-in`
- [ ] Manual smoke test: sign in, see "Welcome, {name}", sign out, redirected to sign-in
