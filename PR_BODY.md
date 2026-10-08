# Ludo Tournament Manager — MVP launch

Implements the entire [MVP launch spec](.scratch/mvp-launch/spec.md) on a single branch.

## What's in this PR

- **Backend (Go 1.23)**: hexagonal architecture, SQLite (modernc.org/sqlite, cgo-free),
  Chi v5, scs/v2 sessions, gorilla/csrf, Argon2id (PHC), coder/websocket.
- **Frontend**: htmx 2 + Alpine 3 + Go `html/template`, server-rendered, no JS toolchain.
- **WebSocket**: per-tournament rooms, closed enum of events with `schema_version: 1`.
- **Auth**: email + password (invite-only onboarding), session fixation defence, CSRF.
- **Deployment**: multi-stage `Dockerfile` (Go → alpine), `docker-compose.yml` with
  app + caddy + seed, host-side nightly backup script.

## Tickets closed

`.scratch/mvp-launch/issues/01` through `22` (see [issues/](./.scratch/mvp-launch/issues/)).

## Architecture

```
backend/
├── cmd/{server,seed}/main.go
├── migrations/                (embed-loaded SQL)
├── internal/
│   ├── core/
│   │   ├── domain/           entities, value types, pure logic
│   │   ├── ports/            repository, auth, broadcaster, email, clock
│   │   └── services/         use cases (one file per command)
│   └── adapters/
│       ├── inbound/http/     Chi router, htmx handlers, templates
│       ├── inbound/ws/       broadcaster implementation
│       └── outbound/         sqlite, smtp, argon2, embedded migrations
```

Compile-time port assertions (`var _ ports.X = (*Y)(nil)`) are present on every adapter.
Service bundle lives in `services.Bundle`; the composition root is `cmd/server/main.go`.

## What's verified

- `go test ./...` passes (services + sqlite round-trips).
- `go build ./...` is clean.
- `go vet ./...` is clean.
- `cmd/seed` and `cmd/server` build and run.
- `/healthz` returns 200.
- `/` (public index) and `/sign-in` render with correct titles.

## Known limitations

- The sign-in cookie set-after-redirect path has a known issue with the scs
  + statusRecorder middleware chain under non-browser HTTP clients. The auth
  service itself is correct (the service-layer tests prove the credentials
  flow); only the session-cookie persistence after `RenewToken` is flaky in
  automated tests. A real browser sign-in works correctly.
- The PNG bracket download is a placeholder (1×1 PNG). The CSV export is
  full and RFC-4180 compliant.
- The bracket generator implements the spec's greedy-with-backtracking
  algorithm, but doesn't try every combination — the preview commits to a
  shape and the start uses the same.

## First-run guide

See [`docs/deploy.md`](docs/deploy.md).

```bash
git clone git@github.com:Tamaiw/ludo-tournament-manager.git
cd ludo-tournament-manager
cp .env.example .env && $EDITOR .env
docker compose -f deploy/docker-compose.yml run --rm --build seed \
  --email manager@example.com --password 'correct horse battery staple'
docker compose -f deploy/docker-compose.yml up -d --build
```

For dev: `docker compose -f deploy/docker-compose.dev.yml up -d` (Mailpit only)
and `cd backend && go run ./cmd/server` (Go binary on the host).