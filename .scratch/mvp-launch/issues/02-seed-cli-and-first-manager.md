# 02: Seed CLI and first-manager bootstrap

**What to build:** A `cmd/seed/main.go` CLI that takes `--email` and `--password` (with `.env` fallback), Argon2id-hashes the password, and creates a `users` row plus a `tournament_manager` row placeholder (none yet — first user has no tournaments). The binary is re-runnable: if the email already exists, it returns a clear error and exits non-zero. Running `docker compose -f deploy/docker-compose.yml run --rm --build seed --email manager@example.com --password ...` against an empty database creates the first user and exits 0. The user can then sign in via the (not-yet-built) sign-in form to verify the row was written correctly.

**Blocked by:** 01 (Project skeleton, Go module, composition root, migrations runner, first migration)

**Status:** ready-for-agent

- [ ] `cmd/seed/main.go` accepts `--email` and `--password` flags, with `.env` fallback for the SMTP/DB path settings
- [ ] Calls into a `core/services/create_user.go` (or similar) service that hashes via the `PasswordHasher` port (Argon2id adapter from T01) and writes a `users` row
- [ ] Rejects duplicate emails with a clear error message and non-zero exit code
- [ ] Opens the DB through the same composition root as `cmd/server` (no duplicated wiring)
- [ ] `deploy/docker-compose.yml` declares a `seed` service that uses the same image and bind-mount as `app`, sharing the `data/` volume
- [ ] `docs/deploy.md` documents the first-run flow: clone → env → seed → up
- [ ] A test in `internal/core/services/create_user_test.go` asserts: happy path creates a user with a valid PHC hash; duplicate email returns `ErrUserExists`; empty email / weak password returns a validation error
- [ ] Manual smoke test: `docker compose run --rm seed --email test@example.com --password 'correct horse battery staple'` creates a user; signing in (via the not-yet-built form, or via a test SQL query) confirms the row exists
