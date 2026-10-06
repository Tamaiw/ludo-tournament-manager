# Ludo Tournament Manager — Wayfinder Map

## Destination

A working MVP for a Ludo tournament manager (Go backend + web frontend), containerized for local development and Hetzner/homelab deployment, with real-time bracket updates, single-elimination format, invite-only user onboarding, and support for tournaments of up to ~100 players.

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
  - Single-elimination brackets only; data model flexible enough to add other formats later
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

- [W6: Sketch core domain model](./.wayfinder/tickets/W6-sketch-core-domain-model.md) *(grilling)*
- [W9: Pick Go data-access layer](./.wayfinder/tickets/W9-pick-go-data-access-layer.md) *(research)*
- [W10: Pick session / auth middleware stack](./.wayfinder/tickets/W10-pick-auth-stack.md) *(research)*
- [W11: Decide deployment shape](./.wayfinder/tickets/W11-decide-deployment-shape.md) *(grilling)*
- [W19: Backup strategy for SQLite](./.wayfinder/tickets/W19-backup-strategy.md) *(grilling)*
- [W20: Decide real-time scope](./.wayfinder/tickets/W20-real-time-scope.md) *(grilling)*

## Blocked

Open tickets, waiting on the frontier. Discovered by the dependency graph; close after resolve.

- [W7: User roles + permissions matrix](./.wayfinder/tickets/W7-user-roles-permissions-matrix.md) *(blocked by W6)*

## Decisions so far

- [W1: Pick Go web framework](./.wayfinder/tickets/W1-pick-go-web-framework.md): **Chi v5** — stdlib-shaped handlers, smallest dep tree, Java/.NET-friendly mental model
- [W2: Pick Go WebSocket library](./.wayfinder/tickets/W2-pick-go-websocket-library.md): **`coder/websocket`** — only candidate with active maintenance in 2026; gorilla stalled since 2024
- [W3: Pick database engine](./.wayfinder/tickets/W3-pick-database-engine.md): **SQLite** (WAL, `modernc.org/sqlite` driver) — single-machine deployment, one-file backup, cgo-free Windows builds
- [W4: Pick frontend approach](./.wayfinder/tickets/W4-pick-frontend-approach.md): **htmx 2 + Alpine 3 + Go `html/template`** — no JS toolchain, server-rendered, bracket fits CSS-Grid + Alpine for v1 scope
- [W5: Decide monorepo vs polyrepo](./.wayfinder/tickets/W5-decide-monorepo-vs-polyrepo.md): **Monorepo, Go in `/backend/`** — no separate frontend codebase; modular monolith (hexagonal) makes future split tractable if needed
- [W8: Transactional email approach](./.wayfinder/tickets/W8-transactional-email-approach.md): **Resend (prod) + Mailpit (dev)** via `net/smtp` — same code path, config flip; Resend's pre-warmed shared IPs sidestep the Hetzner reputation trap
- [W12: Apply hexagonal architecture to backend](./.wayfinder/tickets/W12-apply-hexagonal-architecture.md): **Hexagonal (ports & adapters)** — `internal/core/{domain,ports,services}` + `internal/adapters/{inbound,outbound}`, composition root in `cmd/server/main.go`; full layout at [`docs/architecture/hexagonal.md`](./docs/architecture/hexagonal.md), ADR [`0001`](./docs/adr/0001-hexagonal-backend.md)
- [W21: Set up developer environment](./.wayfinder/tickets/W21-developer-setup.md): **All dev tools installed** (Go 1.26.8, Docker 29.8.2, VS Code + Go extension, gopls, SQLite CLI, Make); guide at [`docs/setup.md`](./docs/setup.md) — gates implementation

## Not yet specified

<!-- see "Fog of war": in-scope fog you can't ticket yet; graduates as the frontier advances -->

- **Public tournament discovery**: how are tournaments listed/found by spectators (browse all, search, filter by status, filter by date)? UI shape — covered partially by the W4 stack lock, but the exact feature set is open.
- **Tournament audit trail**: is there a record of who did what when on a tournament (manager edits, match results, registrations)? v2 candidate.
- **Tournament time zones**: how are scheduled times displayed when organizer and players are in different zones? (Store UTC, render in user TZ is the working assumption; this is a "how do we render" decision.)
- **Notifications**: do players/managers get email when a tournament starts, when their match is ready, when the bracket advances? Resend is in place for transactional; notifications would be a separate trigger surface.
- **Styling system**: plain CSS, a small utility framework, or scoped styles in Alpine components. Depends on how big the bracket CSS-Grid pattern turns out to be (a W4 follow-on).

## Out of scope

<!-- see "Out of scope": work ruled beyond the destination; closed, never graduates -->

- Mobile-native apps — web frontend only
- Public cloud deployment (AWS/GCP/Azure) — local + Hetzner only
- Multi-format brackets beyond single elimination — single elimination only for v1
- OAuth / social login — email + password only for v1
- Account self-service creation — invites are manager-only by destination design