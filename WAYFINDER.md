# Ludo Tournament Manager — Wayfinder Map

## Destination

A working MVP for a Ludo tournament manager (Go backend + web frontend), containerized for local development and Hetzner/homelab deployment, with real-time bracket updates, multi-player elimination format (2-4 players per match, 4-player final), invite-only user onboarding, and support for tournaments of up to ~200 players.

## Notes

- **Domain**: tournament management
- **Skills every session should consult**: `grilling`, `domain-modeling` (default for any design work)
- **Standing preferences**:
  - Cross-platform: must work identically on Windows + Linux dev machines
  - Container-first: deployment runs in containers (docker compose)
  - No TypeScript anywhere (user excluded it explicitly)
  - Backend language: Go
  - **Backend architecture: hexagonal (ports & adapters)** — domain at centre, adapters as the only outward contact; directory layout + port boundaries locked by W12
  - **Future-split guardrail**: kept as a single binary, but structured so a future split into separate services (Go JSON adapter + JS frontend) is tractable; do not let business logic leak into adapters
  - Multi-player elimination brackets only (2-4 players per match, manager-tunable advance count per round, 4-player final); data model flexible enough to add other formats later
  - Open spectator view (browse without account); login only for actions
  - Invite-based user onboarding: manager enters email → token-link invite → user sets password
  - First manager bootstrapped via `docker compose run seed`
  - Local-only auth (email + password, no OAuth for v1)
- **Locked stack** (see Decisions so far):
  - Web framework: Chi v5
  - Real-time: `coder/websocket` (formerly nhooyr.io)
  - Database: SQLite (WAL mode, driver `modernc.org/sqlite`)
  - Frontend: htmx 2 + Alpine.js 3 + Go `html/template`
  - Transactional email: Resend (prod) + Mailpit (dev)

## Frontier

Open, unblocked tickets, ready for a session to claim. Each is the body of `.wayfinder/tickets/<id>-*.md`.

(none — the wayfinder map is clear of planning work; the destination is in reach and implementation can begin)

## Blocked

Open tickets, waiting on the frontier. Discovered by the dependency graph; close after resolve.

(none currently)

## Recently closed

- [W6: Sketch core domain model](./.wayfinder/tickets/W6-sketch-core-domain-model.md) — domain model in [`docs/domain-model.md`](./docs/domain-model.md); bracket representation decision in [ADR 0002](./docs/adr/0002-bracket-representation.md); glossary updated in [`CONTEXT.md`](./CONTEXT.md)
- [W7: User roles + permissions matrix](./.wayfinder/tickets/W7-user-roles-permissions-matrix.md) — full role × action matrix in [`docs/permissions.md`](./docs/permissions.md); three-tier visibility decision in [ADR 0003](./docs/adr/0003-tournament-visibility-tiers.md); new glossary terms (Tournament Visibility, Registration Mode, Spectator Token) in [`CONTEXT.md`](./CONTEXT.md); schema additions (visibility enum, registration_mode enum, tournament_spectator_tokens entity) in [`docs/domain-model.md`](./docs/domain-model.md)
- [W10: Pick session / auth middleware stack](./.wayfinder/tickets/W10-pick-auth-stack.md) — `alexedwards/scs/v2` for sessions + `gorilla/csrf` for CSRF + `argon2id` (golang.org/x/crypto/argon2) for passwords + opaque random + DB-hash for invite/reset tokens (one `auth_tokens` table) + Chi `r.With(...)` per-tournament role middleware; full evaluation in [`research/W10-auth-stack.md`](./research/W10-auth-stack.md)
- [W11: Decide deployment shape](./.wayfinder/tickets/W11-decide-deployment-shape.md) — **compose-everywhere with split dev runtime**: docker compose for state in dev (Go on host via `go run ./cmd/server`); full `app + caddy` stack in compose in prod; single static binary in `alpine`; Caddyfile for HTTPS termination; SQLite bind-mounted at `./data:/data`; `cmd/seed/main.go` invoked via `docker compose run --rm seed --email ... --password ...` for first-time bootstrap; `git pull && docker compose up -d --build` for updates; shape in [`deploy/`](./deploy/) (to be created during implementation), ADR [`0004`](./docs/adr/0004-deployment-shape.md); tightens W19 to host-side backup tooling
- [W19: Backup strategy for SQLite](./.wayfinder/tickets/W19-backup-strategy.md) — **Package A: simple local nightly** — `sqlite3 .backup` against `./data/ludo.db` via host cron at 03:00, one file per day at `./backups/ludo-YYYY-MM-DD.db`, `PRAGMA integrity_check` post-backup, 30-day retention; restore is `cp` the chosen file back + restart the app; ~25 lines of bash in [`deploy/scripts/backup.sh`](./deploy/scripts/backup.sh) (to be added with implementation); documented restore in `docs/deploy.md`; trivially upgradable to restic + B2 or Litestream without touching application code
- [W20: Decide real-time scope](./.wayfinder/tickets/W20-real-time-scope.md) — **state-only WebSocket push** — closed event set (`match_status_changed`, `result_recorded`, `bracket_rebuilt`, `tournament_lifecycle_changed`); every event carries `schema_version` for forward compatibility; per-tournament WS rooms; htmx client treats every message as "re-fetch the bracket fragment"; Broadcaster port in `core/ports/realtime.go`, hub in `adapters/inbound/ws/hub.go` (per W12 hex layout); no presence/chat/reactions in v1; mobile-friendly (events are JSON + typed; JSON read API can be added later as a new inbound adapter without rewriting domain logic)

## Decisions so far

- [W1: Pick Go web framework](./.wayfinder/tickets/W1-pick-go-web-framework.md): **Chi v5** — stdlib-shaped handlers, smallest dep tree, Java/.NET-friendly mental model
- [W2: Pick Go WebSocket library](./.wayfinder/tickets/W2-pick-go-websocket-library.md): **`coder/websocket`** — only candidate with active maintenance in 2026; gorilla stalled since 2024
- [W3: Pick database engine](./.wayfinder/tickets/W3-pick-database-engine.md): **SQLite** (WAL, `modernc.org/sqlite` driver) — single-machine deployment, one-file backup, cgo-free Windows builds
- [W4: Pick frontend approach](./.wayfinder/tickets/W4-pick-frontend-approach.md): **htmx 2 + Alpine 3 + Go `html/template`** — no JS toolchain, server-rendered, bracket fits CSS-Grid + Alpine for v1 scope
- [W5: Decide monorepo vs polyrepo](./.wayfinder/tickets/W5-decide-monorepo-vs-polyrepo.md): **Monorepo, Go in `/backend/`** — no separate frontend codebase; modular monolith (hexagonal) makes future split tractable if needed
- [W8: Transactional email approach](./.wayfinder/tickets/W8-transactional-email-approach.md): **Resend (prod) + Mailpit (dev)** via `net/smtp` — same code path, config flip; Resend's pre-warmed shared IPs sidestep the Hetzner reputation trap
- [W12: Apply hexagonal architecture to backend](./.wayfinder/tickets/W12-apply-hexagonal-architecture.md): **Hexagonal (ports & adapters)** — `internal/core/{domain,ports,services}` + `internal/adapters/{inbound,outbound}`, composition root in `cmd/server/main.go`; full layout at [`docs/architecture/hexagonal.md`](./docs/architecture/hexagonal.md), ADR [`0001`](./docs/adr/0001-hexagonal-backend.md)
- [W6: Sketch core domain model](./.wayfinder/tickets/W6-sketch-core-domain-model.md): **Multi-player elimination, adjacency-list bracket** — 2-4 player matches, manager-configurable per-round advance count, 4-player final; model in [`docs/domain-model.md`](./docs/domain-model.md), ADR [`0002`](./docs/adr/0002-bracket-representation.md), glossary in [`CONTEXT.md`](./CONTEXT.md)
- [W7: User roles + permissions matrix](./.wayfinder/tickets/W7-user-roles-permissions-matrix.md): **Manager-as-Player allowed; three-tier visibility (public/unlisted/private); per-tournament registration_mode (invite_only/self_register); single-recorder match outcome; manager-only self-removal via cancel** — full matrix in [`docs/permissions.md`](./docs/permissions.md), ADR [`0003`](./docs/adr/0003-tournament-visibility-tiers.md), glossary + schema updates in [`CONTEXT.md`](./CONTEXT.md) and [`docs/domain-model.md`](./docs/domain-model.md)
- [W9: Pick Go data-access layer](./.wayfinder/tickets/W9-pick-go-data-access-layer.md): **sqlc** (SQL-first, type-safe Go from raw SQL; SQLite support beta — risk mitigated by `database/sql` escape hatch inside the adapter) — full evaluation in [`research/W9-data-access-layer.md`](./research/W9-data-access-layer.md)
- [W10: Pick session / auth middleware stack](./.wayfinder/tickets/W10-pick-auth-stack.md): **`alexedwards/scs/v2` (sessions) + `gorilla/csrf` (CSRF) + `argon2id` via `golang.org/x/crypto/argon2` (passwords, PHC format, OWASP m=19456,t=3,p=1) + opaque random + DB-hash invite/reset tokens (one `auth_tokens` table, `kind` enum) + `r.With(...)` per-tournament role middleware** — full evaluation in [`research/W10-auth-stack.md`](./research/W10-auth-stack.md)
- [W21: Set up developer environment](./.wayfinder/tickets/W21-developer-setup.md): **All dev tools installed** (Go 1.26.8, Docker 29.8.2, VS Code + Go extension, gopls, SQLite CLI, Make); guide at [`docs/setup.md`](./docs/setup.md) — gates implementation
- [W11: Decide deployment shape](./.wayfinder/tickets/W11-decide-deployment-shape.md): **Compose-everywhere with split dev runtime** — `docker compose -f deploy/docker-compose.dev.yml up -d` for Mailpit in dev (Go on host via `go run ./cmd/server`); `docker compose -f deploy/docker-compose.yml up -d` for `app + caddy` in prod; single static binary in `alpine` (multi-stage `Dockerfile`); Caddyfile for ACME HTTPS termination; SQLite bind-mounted at `./data:/data`; `cmd/seed/main.go` invoked via `docker compose run --rm seed --email ... --password ...` for first-time bootstrap; `git pull && docker compose up -d --build` for updates; ADR [`0004`](./docs/adr/0004-deployment-shape.md). Tightens W19 to host-side backup tooling (cron / Litestream against `./data/ludo.db`).
- [W19: Backup strategy for SQLite](./.wayfinder/tickets/W19-backup-strategy.md): **Package A — simple local nightly** — `sqlite3 .backup` against `./data/ludo.db` via host cron at 03:00; one file per day at `./backups/ludo-YYYY-MM-DD.db`; `PRAGMA integrity_check` post-backup; 30-day retention via `find -mtime +30 -delete`; restore is `cp` + restart the app. ~25 lines of bash in `deploy/scripts/backup.sh`. Trivially upgradable to restic + Backblaze B2 (off-host + encryption + dedup) or Litestream (continuous WAL replication) by swapping the script + cron line — no application-code change.
- [W20: Decide real-time scope](./.wayfinder/tickets/W20-real-time-scope.md): **State-only WebSocket push** — closed enum of event types (`match_status_changed`, `result_recorded`, `bracket_rebuilt`, `tournament_lifecycle_changed`) with `schema_version: 1` on every event for forward compatibility; per-tournament WS rooms; htmx client re-fetches bracket fragment on `ws:message`; Broadcaster port lives in `core/ports/realtime.go`, hub in `adapters/inbound/ws/hub.go` (per W12 hex layout). No presence/chat/reactions in v1; per-match edit collisions handled at the write layer via audit log + downstream-lock rule (W7). Mobile-friendly via closed enum + schema_version; adding a JSON read API in v2 is a new inbound adapter, not a domain rewrite.

## Not yet specified

<!-- see "Fog of war": in-scope fog you can't ticket yet; graduates as the frontier advances -->

- **Public tournament discovery**: how are tournaments listed/found by spectators (browse all, search, filter by status, filter by date)? UI shape — covered partially by the W4 stack lock, but the exact feature set is open.
- **Tournament time zones**: how are scheduled times displayed when organizer and players are in different zones? (Store UTC, render in user TZ is the working assumption; this is a "how do we render" decision.)
- **Notifications**: do players/managers get email when a tournament starts, when their match is ready, when the bracket advances? Resend is in place for transactional; notifications would be a separate trigger surface.
- **Styling system**: plain CSS, a small utility framework, or scoped styles in Alpine components. Depends on how big the bracket CSS-Grid pattern turns out to be (a W4 follow-on).
- **Public player profile with match history**: a per-player page showing past tournaments, placements, and opponents. v2. The audit log is the data source; the view is not in v1.

## Out of scope

<!-- see "Out of scope": work ruled beyond the destination; closed, never graduates -->

- Mobile-native apps — web frontend only
- Public cloud deployment (AWS/GCP/Azure) — local + Hetzner only
- Multi-format brackets beyond multi-player elimination — multi-player elimination only for v1
- OAuth / social login — email + password only for v1
- Account self-service creation — invites are manager-only by destination design