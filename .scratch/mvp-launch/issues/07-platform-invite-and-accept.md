# 07: Platform invite, invite-accept, password set

**What to build:** A manager can send a platform invite to an email address via `POST /tournaments/{id}/roster/invite` (or, later, a global invite — for now it's tournament-scoped because that's what the spec describes). The system generates a 256-bit opaque random token, stores the SHA-256 hash in `auth_tokens` with `kind=invite`, `email=<email>`, `expires_at=now+7d`, `created_by=<manager>`, and emails a link of the form `https://<host>/invites/{token}` (in dev, the link is printed to stdout AND shown in the Mailpit UI; in prod, it's emailed via Resend). The invitee clicks the link, lands on `GET /invites/{token}` (a set-password form), submits a new password via `POST /invites/{token}`, the system creates the `users` row (Argon2id-hashed), fills in `auth_tokens.user_id`, sets `auth_tokens.used_at`, signs the user in (`RenewToken` + session put), and redirects to `/dashboard`. Re-using the token returns 410 Gone. An unknown token returns 404. An expired token returns 410 with a "this invite has expired; ask {manager name} for a new one" message. The `EmailSender` port is implemented by the SMTP adapter (Resend in prod, Mailpit in dev).

**Blocked by:** 04 (Dashboard, RequireUser middleware, base layout)

**Status:** ready-for-agent

- [ ] `internal/core/domain/auth_token.go` has the `AuthToken` struct, `AuthTokenKind` enum (`invite` / `password_reset`), typed errors (`ErrTokenNotFound`, `ErrTokenExpired`, `ErrTokenUsed`)
- [ ] `internal/core/services/invite_user.go` with `Handle(ctx, InviteUserCmd)`: generates token, stores hash, calls `EmailSender.SendInvite(ctx, email, inviteURL, managerName)`. Does not create the user yet — that's done on redemption.
- [ ] `internal/core/services/accept_invite.go` with `Handle(ctx, AcceptInviteCmd)`: looks up by token hash, validates (not used, not expired), creates user via `create_user` service logic, fills `auth_tokens.user_id`, sets `used_at`, writes audit log (`invite_accepted`).
- [ ] `internal/core/ports/email.go` declares `EmailSender` with `SendInvite(ctx, toEmail, inviteURL, inviterName) error` (and `SendPasswordReset` as a stub for T19)
- [ ] `internal/adapters/outbound/smtp/email_sender.go` implements `EmailSender` using `net/smtp`; reads `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD`, `SMTP_FROM` from config; sends to Mailpit in dev and Resend in prod (Resend is just an SMTP relay with the same envelope; no SDK needed)
- [ ] `internal/adapters/inbound/http/handlers/invite.go` has `AcceptInvitePage`, `AcceptInviteSubmit`
- [ ] `internal/adapters/inbound/http/templates/pages/invite_accept.html` renders the set-password form with `{{ .CSRFField }}`
- [ ] `internal/adapters/inbound/http/server.go` registers `GET /invites/{token}`, `POST /invites/{token}` (no auth required — the token IS the auth)
- [ ] `cmd/server/main.go` wires the email sender (and in dev, prints the invite URL to stdout in addition to emailing Mailpit, for developer ergonomics)
- [ ] Tests at the service layer: `invite_user_test.go` (token generated, hash stored, email sent, expiry = now+7d); `accept_invite_test.go` (happy path creates user and signs them in, used token returns error, expired token returns error, unknown token returns error)
- [ ] Tests at the HTTP handler layer: `GET /invites/{token}` renders form for a valid token, 404 for unknown, 410 for used/expired; `POST /invites/{token}` with valid data + new password creates user and signs them in (cookie set, redirect to /dashboard)
- [ ] Manual smoke test: as manager, invite `newuser@example.com`; check Mailpit for the email; click the link (or copy from stdout in dev); set a password; land on the dashboard signed in as the new user
