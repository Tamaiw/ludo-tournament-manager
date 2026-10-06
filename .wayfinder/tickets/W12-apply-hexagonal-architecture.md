# W12: Apply hexagonal architecture to backend

**Type:** grilling
**State:** in-progress
**Assignee:** opencode
**Blocked by:** (none)
**Blocks:** W6, W9, W10, W11

## Question

Apply hexagonal (ports & adapters) architecture to the Go backend: lock down the layer boundaries, the directory layout under `backend/`, and which concerns are interfaces (ports) vs in-process helpers.

## Specific things to settle

- **Directory layout** under `backend/`:
  - `internal/domain/` — entities (`Tournament`, `Match`, `BracketNode`, `User`, `Invite`) + value types; pure Go, no DB tags, no JSON tags
  - `internal/ports/` — interfaces: `TournamentRepository`, `MatchRepository`, `UserRepository`, `PasswordHasher`, `EmailSender`, `SessionStore`, `BracketGenerator`, `InviteSigner`, real-time `Broadcaster`
  - `internal/app/` — use case services that orchestrate domain + ports: `CreateTournament`, `RegisterPlayer`, `RecordMatchResult`, `InviteUser`, `AcceptInvite`, `SignIn`, etc.
  - `internal/adapters/` — concrete adapters grouped by direction
    - inbound (driving): `http_handler/` (Chi), `ws_handler/` (`coder/websocket`)
    - outbound (driven): `sqlite_repo/`, `smtp_sender/`, `bcrypt_hasher/`, `cookie_sessions/`
    - in-process: `bracket_gen/` (pure algorithm; lives in `internal/app/` or `internal/domain/` — decide)
  - `cmd/server/main.go` — composition root: read config, instantiate adapters, wire them into use cases, hand the wired use cases to the HTTP/WS handlers, start the server
- **Port vs in-process helper**: e.g. is the bracket generator a port with adapter, or a pure domain service? Decide one rule and apply it.
- **Testability story**: mock adapters live alongside real ones (e.g. `internal/adapters/sqlite_repo/mem_repo.go` or `internal/adapters/mem/`) so unit tests for use cases run without DB or SMTP.
- **Dependency direction** (the rule that makes hexagonal work):
  - `domain` → depends on nothing else in the project
  - `app` → depends on `domain` + `ports`
  - `adapters` → depends on `app` + `ports`
  - `cmd` → depends on everything (composition root only)
  - No upward dependencies; no adapter-to-adapter imports.
- **`embed.FS` for templates**: where in the tree does `templates/*.html` live so the binary ships them self-contained? Convention is `internal/adapters/http_handler/templates/` and `embed.FS` declared at the adapter.

## Constraints

- Backend is Go; framework is Chi (resolved by W1); DB is SQLite via `modernc.org/sqlite` (resolved by W3); WebSocket is `coder/websocket` (resolved by W2)
- This is a Go-idiomatic hexagonal layout — flat packages, no class-heavy inheritance; avoid pretending Go is Java
- Frontend is server-rendered htmx + Alpine (resolved by W4); the inbound boundary is HTTP requests and WS upgrades only
- Email via Resend (prod) / Mailpit (dev) (resolved by W8); wired behind an `EmailSender` port

## What "good" looks like

A written layout sketch as a resolution comment, plus a one-pager (`docs/architecture/hexagonal.md`) or ADR (`docs/adr/0001-hexagonal-backend.md`) naming each package, what goes in it, and what it depends on. This is the convention W6, W9, W10, W11 build on.