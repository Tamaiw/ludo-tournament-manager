# Ludo Tournament Manager — MVP launch

## Problem Statement

Tournament organisers who run Ludo nights currently manage brackets by hand — paper sheets, shared spreadsheets, messenger threads where results are copied around. Once a tournament has more than about 16 players, hand-management breaks: byes get miscounted, advancing positions are mis-typed, results get recorded twice, and spectators who want to follow along either can't see the bracket at all or get a stale screenshot. There is no existing off-the-shelf product that fits the multi-player elimination format (2-4 players per match, 4-player final) and the small-club, single-machine deployment that the user runs. The first manager needs to be able to install the system, create a tournament, register players, generate a bracket, and have spectators follow the bracket live from any browser — all on one VPS or a homelab box, without depending on any third-party SaaS for the data plane.

## Solution

A single Go binary serving an htmx + Alpine web frontend, backed by SQLite, deployed via `docker compose` on a single VPS. Managers create tournaments through the web UI; players are onboarded via manager-issued email invites; brackets are generated automatically when the tournament starts and update in real time on every connected browser via WebSocket. Spectators browse without an account; private tournaments are gated by per-tournament spectator tokens. Everything — code, data, backups, deployment — lives in one repo, deployable by `git clone && docker compose up -d --build` plus a one-shot seed command for the first manager.

The MVP is the **first working end-to-end slice** of the destination. It includes enough features to run a real tournament from invite through final result, and it explicitly defers notifications, public player profiles, and additional bracket formats.

## User Stories

1. As the first manager, I want to bootstrap my account with a one-shot CLI command, so that I can sign in and start running tournaments without a web-based setup wizard.
2. As a manager, I want to sign in with my email and password, so that I can access management actions on my tournaments.
3. As a manager, I want to sign out, so that my session ends on a shared computer.
4. As a manager, I want to invite a new user by email, so that they receive a link to set their password and join the platform.
5. As an invited user, I want to click the link in my invite email, set a password, and land signed in, so that I can participate in tournaments without a separate sign-up step.
6. As a manager, I want to create a new tournament with a name, description, format, advance map, and scheduled start time, so that I can start organising an event.
7. As a manager, I want to set a tournament's visibility (public, unlisted, or private) at creation and edit it later, so that I can control who can discover and view the bracket.
8. As a manager, I want to set a tournament's registration mode (invite-only or self-register) at creation, so that I can control how players join the roster.
9. As a manager, I want to add a player to an invite-only tournament by user id, so that the roster reflects my intended players.
10. As a manager, I want to send a platform invite that auto-adds a new user to the tournament roster when accepted, so that I can fill an invite-only tournament with people who don't yet have an account.
11. As an authenticated user, I want to self-register for a self-register tournament that is open for registration and below its cap, so that I can join without a manager action.
12. As a player, I want to withdraw from a tournament before the bracket is generated, so that I can back out if I change my mind.
13. As a manager, I want to remove a player from a tournament before the bracket is generated, so that the roster reflects reality.
14. As a manager, I want to set or edit a player's seed before the tournament starts, so that I can place stronger players where I want them.
15. As a manager, I want to see a preview of the bracket shape (rounds, matches, advance counts) before I start the tournament, so that I can adjust the advance map if it doesn't look right.
16. As a manager, I want to open registration, so that players can self-register (in self-register mode) or so I can finalise the roster.
17. As a manager, I want to start the tournament, which generates the bracket, locks the player list, and assigns seeds, so that matches can begin.
18. As a manager, I want to add a co-manager to a tournament, so that we can co-run it.
19. As a manager-as-player, I want to register myself as a player in a tournament I also manage, so that I can compete in my own events.
20. As a spectator, I want to view a public tournament from the public index, so that I can discover and follow ongoing events.
21. As a spectator, I want to view an unlisted tournament if I have the URL, so that I can share a link without listing the event publicly.
22. As a spectator, I want to view a private tournament if I hold a spectator token, so that I can be granted access without an account.
23. As a manager, I want to issue a spectator token for a private tournament, label it, and copy the URL, so that I can give read access to specific people.
24. As a manager, I want to revoke a spectator token, so that I can cut off access that I no longer want to grant.
25. As a player or manager, I want to see the bracket of a tournament, with each match's status, participants, and the path to the final, so that I can follow the event.
26. As a player, I want to see which of my matches are ready to play (upstream matches completed), so that I know when to show up.
27. As a player or manager, I want to mark a match as in-progress, so that the bracket reflects that the game is being played now.
28. As a player in a match, I want to record the match outcome (finishing positions of all participants), so that the bracket advances and the next match becomes ready.
29. As a manager, I want to record or correct a match outcome, so that I can fix errors or record on behalf of absent players.
30. As a manager, I want to edit a completed match outcome, but only if no downstream match has been played, so that I can fix a mistake without corrupting the bracket.
31. As a player, I want to be unable to edit a completed match, so that no one retroactively changes results.
32. As a spectator, I want to see the bracket update in my browser without refreshing, so that I can follow the event live.
33. As a spectator, I want to see the tournament status pill change (e.g. "in progress" → "completed") in real time.
34. As a player, I want to see my own match status change to "ready" the moment the upstream match is recorded, so that I know to show up.
35. As a manager, I want to force-close a tournament (transition to completed) if it stalls, so that the audit log captures a final state.
36. As a manager, I want to cancel a tournament before it has started, so that I can abort an event whose roster fell through.
37. As a manager, I want to view the audit log for a tournament, so that I can see who did what and when.
38. As a player, I want to be unable to view the audit log, so that manager-side decisions are not exposed.
39. As a manager, I want to download the bracket as CSV or an image, so that I can share it offline or print it.
40. As a spectator, I want to download the bracket as CSV or an image, so that I can keep a copy.
41. As an authenticated user, I want to change my own password, so that I can rotate credentials.
42. As a manager, I want to reset another user's password via an email link, so that I can recover a user who is locked out (or whose account needs to be re-secured).
43. As a manager, I want to change the visibility of a tournament at any time, with the effect on currently-reachable spectators recorded in the audit log.
44. As a manager, I want the system to be deployable with a single `git clone` and a `docker compose up -d --build`, so that a fresh VPS can be brought up without bespoke ceremony.
45. As an operator, I want nightly SQLite backups with integrity checks and 30-day retention, so that I can recover from data loss without bespoke tooling.
46. As a developer, I want a clear seam between business logic and I/O (hexagonal architecture), so that I can write fast unit tests against fake ports and substitute a JSON API in v2 without rewriting the domain.
47. As a developer, I want the same Go binary to run on the host in dev (with Mailpit in a container) and in a container in prod (with Caddy reverse-proxying TLS), so that the dev/prod split is a config flip, not a code fork.

## Implementation Decisions

### Architecture

- **Hexagonal (ports & adapters)** as locked in [ADR 0001](../docs/adr/0001-hexagonal-backend.md). All business rules live in `internal/core/{domain,ports,services}`; HTTP, WebSocket, SQLite, SMTP, sessions, password hashing, and the clock are all adapters. Domain has no project imports. Adapters depend inward. `cmd/server/main.go` is the composition root.
- **Service layer is the single source of business rules.** No domain logic in handlers, no domain logic in SQL queries, no domain logic in templates. Handlers translate HTTP requests to service commands; services call ports; ports are implemented by adapters.
- **Compile-time port assertions.** Every adapter that satisfies a port declares `var _ ports.X = (*Y)(nil)`. If a port's signature changes, every adapter that needs updating fails to build.
- **Future-split guardrail.** Business logic must not leak into adapters. The structure is a single binary today, but the architecture keeps a future split into "Go JSON API + JS frontend" tractable.

### Stack (locked)

- **Web framework:** Chi v5 (W1) — stdlib-shaped handlers and middleware, no framework magic.
- **Real-time:** `coder/websocket` (W2) — only candidate with active maintenance in 2026; gorilla stalled.
- **Database:** SQLite in WAL mode, driver `modernc.org/sqlite` (W3) — single-file, single-machine, cgo-free Windows builds.
- **Data access:** `sqlc` (W9) — raw SQL queries, type-safe generated Go. Schema source = migrations directory. Pair with hand-written SQL migrations.
- **Frontend:** htmx 2 + Alpine 3 + Go `html/template` (W4) — server-rendered, no JS toolchain, brackets fit CSS-Grid + Alpine.
- **Email:** Resend (prod) + Mailpit (dev) via `net/smtp` (W8) — same code path, config flip.
- **Auth:** `alexedwards/scs/v2` (sessions) + `gorilla/csrf` (CSRF) + Argon2id (passwords, `golang.org/x/crypto/argon2`, PHC format, m=19456, t=3, p=1) + opaque random + DB-hash for invite/reset tokens (one `auth_tokens` table, `kind` enum) + Chi `r.With(...)` per-tournament role middleware (W10).
- **Deployment:** docker compose for state in dev; `app + caddy` stack in compose in prod; single static binary in `alpine`; Caddyfile for HTTPS termination; SQLite bind-mounted at `./data:/data`; first manager bootstrapped via `docker compose run seed` (W11, W19).
- **Real-time shape:** state-only WebSocket push, per-tournament rooms, closed enum of events with `schema_version: 1` on every event; htmx client treats every message as "re-fetch the bracket fragment" (W20).

### Backend layout

```
backend/
├── go.mod, go.sum
├── cmd/
│   ├── server/main.go                    (composition root)
│   └── seed/main.go                      (first-manager bootstrap CLI)
├── migrations/                           (numbered .sql files; schema source for sqlc)
│   ├── 0001_init.sql                     (users, sessions, auth_tokens, tournaments, tournament_manager, tournament_player, tournament_spectator_tokens, matches, match_participants, tournament_audit_log)
│   └── …
├── queries/                              (sqlc input)
│   ├── tournament.sql
│   ├── match.sql
│   ├── user.sql
│   └── …
├── internal/
│   ├── core/
│   │   ├── domain/                       (entities, value types, pure logic, bracket_gen.go)
│   │   │   ├── tournament.go
│   │   │   ├── match.go
│   │   │   ├── bracket.go
│   │   │   ├── bracket_gen.go            (pure generator, multi-player elimination with 4-player final)
│   │   │   ├── user.go
│   │   │   ├── auth_token.go
│   │   │   └── visibility.go             (enum: public / unlisted / private)
│   │   ├── ports/
│   │   │   ├── repository.go             (TournamentRepo, MatchRepo, UserRepo, TournamentManagerRepo, TournamentPlayerRepo, TournamentSpectatorTokenRepo, MatchParticipantRepo, AuditLogRepo, AuthTokenRepo, SessionRepo)
│   │   │   ├── auth.go                   (PasswordHasher, SessionStore)
│   │   │   ├── realtime.go               (Broadcaster with closed enum of events)
│   │   │   ├── email.go                  (EmailSender)
│   │   │   └── clock.go                  (Clock — for testability)
│   │   └── services/                     (one file per use case)
│   │       ├── create_tournament.go
│   │       ├── edit_tournament.go
│   │       ├── change_visibility.go
│   │       ├── change_registration_mode.go
│   │       ├── issue_spectator_token.go
│   │       ├── revoke_spectator_token.go
│   │       ├── open_registration.go
│   │       ├── start_tournament.go       (generates bracket)
│   │       ├── preview_bracket.go        (calls generator, returns shape without persisting)
│   │       ├── cancel_tournament.go
│   │       ├── force_close_tournament.go
│   │       ├── register_player.go        (self-register path)
│   │       ├── add_player_by_user.go     (manager path: invite-only, existing user)
│   │       ├── invite_user.go            (sends platform invite; auto-adds to roster on accept)
│   │       ├── accept_invite.go
│   │       ├── withdraw_player.go
│   │       ├── remove_player.go          (manager, pre-bracket)
│   │       ├── edit_seed.go
│   │       ├── mark_match_in_progress.go
│   │       ├── record_match_result.go
│   │       ├── correct_match_result.go   (manager, downstream-lock check)
│   │       ├── sign_in.go
│   │       ├── sign_out.go
│   │       ├── change_password.go
│   │       ├── request_password_reset.go
│   │       ├── redeem_password_reset.go
│   │       └── redeem_invite.go
│   └── adapters/
│       ├── inbound/
│       │   ├── http/
│       │   │   ├── server.go             (Chi setup, route registration, middleware chain)
│       │   │   ├── handlers/             (per-resource, one file per concern)
│       │   │   │   ├── auth.go           (sign-in, sign-out, change password)
│       │   │   │   ├── invite.go         (accept-invite, password-reset, GET/POST)
│       │   │   │   ├── tournament.go     (CRUD, lifecycle transitions, visibility, registration mode, spectator tokens)
│       │   │   │   ├── player.go         (self-register, withdraw, list roster)
│       │   │   │   ├── match.go          (mark in-progress, record result, manager correct)
│       │   │   │   ├── bracket.go        (HTML fragment endpoint for htmx re-fetch)
│       │   │   │   ├── index.go          (public tournament index, dashboard)
│       │   │   │   └── download.go       (CSV / image)
│       │   │   ├── middleware/
│       │   │   │   ├── session.go        (wraps scs.LoadAndSave)
│       │   │   │   ├── csrf.go           (wraps csrf.Protect; renders {{ .CSRFField }})
│       │   │   │   ├── require_user.go
│       │   │   │   ├── require_tournament_role.go
│       │   │   │   ├── require_spectator_token.go  (private-tournament access)
│       │   │   │   ├── request_id.go
│       │   │   │   ├── real_ip.go
│       │   │   │   ├── logger.go
│       │   │   │   └── recoverer.go
│       │   │   └── templates/            (embed.FS; one .html per page; fragments inline)
│       │   └── ws/
│       │       ├── hub.go                (per-tournament rooms, implements Broadcaster)
│       │       └── client.go             (one WS connection; reads heartbeat; writes JSON events)
│       └── outbound/
│           ├── sqlite/
│           │   ├── open.go               (opens DB, applies migrations, sets WAL)
│           │   ├── session_store.go      (scs.Store adapter, ~40 lines)
│           │   ├── tournament_repo.go
│           │   ├── match_repo.go
│           │   ├── user_repo.go
│           │   ├── tournament_manager_repo.go
│           │   ├── tournament_player_repo.go
│           │   ├── tournament_spectator_token_repo.go
│           │   ├── match_participant_repo.go
│           │   ├── audit_log_repo.go
│           │   ├── auth_token_repo.go
│           │   └── gen/                  (sqlc-generated; not edited by hand)
│           ├── smtp/
│           │   └── email_sender.go       (Resend in prod, Mailpit in dev; same code path)
│           ├── argon2/
│           │   └── hasher.go             (Argon2id Hash/Verify + PHC encode/decode)
│           └── migrations/               (numbers matching migrations/; run on app start)
```

### Frontend layout

```
backend/internal/adapters/inbound/http/templates/
├── layouts/
│   └── base.html                        (header, nav, flash messages, CSRF field)
├── pages/
│   ├── index.html                       (public tournament index)
│   ├── dashboard.html                   (signed-in user dashboard)
│   ├── tournament_new.html
│   ├── tournament_show.html             (tournament header, status pill, bracket container)
│   ├── tournament_edit.html
│   ├── tournament_roster.html
│   ├── tournament_audit_log.html
│   ├── tournament_spectator_tokens.html
│   ├── match_show.html
│   ├── match_record.html                (form: advancing positions per slot)
│   ├── sign_in.html
│   ├── invite_accept.html               (set-password form for invite link)
│   ├── password_reset.html
│   ├── change_password.html
│   └── 404.html, 403.html, 500.html
├── fragments/
│   ├── bracket.html                     (the re-fetch target for htmx-on-ws)
│   ├── match_card.html
│   ├── roster_table.html
│   ├── audit_log_table.html
│   ├── tournament_status_pill.html
│   └── flash.html
└── components/                          (Alpine x-data blocks reused across pages)
```

### Domain entities (per `docs/domain-model.md`)

- **Tournament** — `id`, `name`, `description`, `format` (enum, `multi_player_elimination` for v1), `status` (enum, `draft / registration_open / in_progress / completed / cancelled`), `min_players_per_match` (default 2), `max_players_per_match` (default 4), `players_advancing_per_round` (JSON map), `scheduled_start_at` (UTC, nullable), `visibility` (enum, replaces `is_public`; default `private`), `registration_mode` (enum, `invite_only / self_register`; default `invite_only`), `created_by`, `created_at`, `updated_at`, `started_at`, `completed_at`.
- **Tournament Manager (join)** — `tournament_id`, `manager_id`, `added_at`. All managers equal; no creator-elevated role.
- **Tournament Spectator Token** — `id`, `tournament_id`, `token_hash` (sha256 of raw token), `label`, `issued_at`, `issued_by`, `revoked_at`, `revoked_by`. Raw token shown once at issue.
- **Tournament Player (registration)** — `tournament_id`, `player_id`, `registered_at`, `seed` (nullable; auto-generated at bracket-creation if unset).
- **Match** — `id`, `tournament_id`, `round` (1-indexed), `position_in_round` (1-indexed), `status` (`pending / ready / in_progress / completed`), `scheduled_at` (UTC, nullable), `next_match_id`, `slot_in_next_match` (`home / away / third / fourth`), `created_at`, `updated_at`. Note: `winner_player_id` is **derived**, not stored — when all `match_participants.advancing_position`s are recorded, the lowest-position player is the "winner" by the bracket's rules. The participant with the lowest advancing position in the **final** match is the tournament winner.
- **Match Participant** — `match_id`, `slot` (enum, same as `slot_in_next_match`), `player_id` (nullable for byes), `advancing_position` (1-indexed; required for advancers, nullable for eliminated players whose position was not recorded), `is_bye` (default false).
- **Audit Log** — `id`, `tournament_id`, `actor_id`, `action` (enum, see below), `subject_id` (polymorphic), `before` (JSON), `after` (JSON), `recorded_at`.
- **User** — `id`, `email` (unique), `name`, `password_hash` (PHC), `created_at`, `updated_at`.
- **Auth Token** — `id`, `kind` (`invite / password_reset`), `user_id` (nullable; set after invite acceptance), `email`, `token_hash` (sha256 hex), `expires_at`, `used_at`, `created_by`, `created_at`.
- **Session** — `token_hash` (sha256 of cookie; per scs v2.8.0+ `HashTokenInStore`), `data` (BLOB, gob), `expiry` (REAL, unix seconds).

### Bracket generation (per `docs/domain-model.md`)

- **Trigger:** `registration_open → in_progress` transition.
- **Inputs:** `tournament_id`, registered player list, `min_players_per_match`, `max_players_per_match`, `players_advancing_per_round` map.
- **Hard constraint:** final round has exactly 4 participants.
- **Algorithm:** greedy + backtracking search; objective = minimum variance in advance counts within a round ("most even rounds"). Deterministic — same inputs produce the same bracket.
- **Performance:** N ≤ 200, search space ~hundreds of states per round, depth ≤ 8, completes in milliseconds.
- **Preview:** before commit, the manager sees "Round 1: 25 games (19 advance 2, 6 advance 3) → 56 players. Round 2: …". The manager confirms or adjusts the advance map and re-previews.
- **Seeding:** players with a non-null seed placed first; players with null seed sorted by a tournament-specific PRNG seed; combined list sliced into round-1 matches per layout. The PRNG seed is recorded in the audit log so the bracket is reproducible from the log alone.
- **Byes:** `is_bye = true, player_id = NULL` rows fill slots to reach the configured `min_players_per_match`. They auto-advance; the slot in the next match is filled in at generation time.
- **Service location:** `internal/core/services/start_tournament.go` (the commit) and `internal/core/services/preview_bracket.go` (the dry-run). The pure generator function lives in `internal/core/domain/bracket_gen.go`.

### Lifecycle transitions

- `draft → registration_open` (manager)
- `registration_open → in_progress` (manager; triggers bracket generation; locks player list and seeds)
- `registration_open → draft` (manager; closes registration and returns to editing)
- `in_progress → completed` (auto, when the final match is recorded; also manual, manager force-close)
- `draft | registration_open → cancelled` (manager; only valid pre-`in_progress`)
- Visibility and registration mode editable per the permissions matrix.

### Match status transitions

- `pending → ready` (auto, when all upstream matches are completed)
- `ready → in_progress` (player of the match or manager)
- `in_progress → completed` (player of the match or manager, with all advancing positions recorded)
- `completed → completed` (manager only, downstream-lock check: no downstream match is `in_progress` or `completed`)

### Permissions (per `docs/permissions.md`)

- **Anonymous** — view public tournaments; view unlisted/private if URL or spectator token grants access; no create, no sign-up.
- **Authenticated user** — all anonymous capabilities; sign in; change own password; create tournaments; self-register in `self_register` mode.
- **Manager (per-tournament)** — all tournament configuration; lifecycle transitions; add/remove players (pre-bracket); edit seeds (pre-bracket); mark matches in progress; record/correct match results; issue/revoke spectator tokens; view audit log; cancel; force-close.
- **Player (per-tournament)** — view tournament; withdraw self (pre-bracket); mark own matches in progress; record own match results.
- **Spectator** — view tournament (subject to visibility + spectator token); download bracket.
- **Manager-as-Player** allowed; audit log records `manager_self_record: true` for self-recorded results.

### Visibility matrix (effect on currently-reachable spectators)

| Change | Effect |
|--------|--------|
| `public → unlisted` | Index entries removed; URL viewers unaffected |
| `public → private` | Index removed; URL viewers without a token get 403 |
| `unlisted → public` | Tournament now listed; existing URL viewers unaffected |
| `unlisted → private` | URL viewers without a token get 403 |
| `private → public` | All token holders retain access; tokens become redundant (left or revoked) |
| `private → unlisted` | Token holders retain access; URL alone also sufficient |

### Registration mode matrix

- `invite_only` — only managers can add players (by user id, or via platform invite that auto-adds on accept)
- `self_register` — any authenticated user can self-register while `registration_open` and below cap; managers retain remove power
- Editable while `draft` or `registration_open`

### Audit log actions (enum)

`seed_changed / manager_added / manager_removed / registration_opened / registration_closed / registration_mode_changed / visibility_changed / spectator_token_issued / spectator_token_revoked / tournament_started / match_result_recorded / match_result_corrected / tournament_cancelled / tournament_completed / player_added / player_removed / player_withdrew / player_invited / invite_accepted / password_reset_issued / password_changed`

The audit log is append-only, manager-only readable, and is the data source for future player-profile match-history views.

### Real-time (W20)

- **Event set (closed enum):** `match_status_changed`, `result_recorded`, `bracket_rebuilt`, `tournament_lifecycle_changed`.
- **Wire format:** every event carries `schema_version: 1`; JSON, snake_case; fields `{ schema_version, type, tournament_id, ... }`.
- **Per-tournament rooms:** `ws://host/ws/tournaments/{id}`. Hub is a `map[TournamentID]map[*Client]struct{}`.
- **Broadcaster port:** `core/ports/realtime.go` — `Broadcast(ctx, Event) error`. Service layer never blocks on broadcast; the handler fires `go broadcaster.Broadcast(...)` after the write commits, with a panic-recover boundary.
- **Client pattern (htmx):** `<div hx-ext="ws" ws-connect="/ws/tournaments/T123" hx-trigger="ws:message from:body" hx-get="/fragments/tournaments/T123/bracket" hx-swap="outerHTML">` — every WS message triggers a re-fetch of the bracket fragment.
- **No presence, no chat, no reactions, no per-match claim, no viewer count** in v1. Per-match edit collisions handled at the write layer by audit log + downstream-lock rule.

### Authentication (W10)

- **Sessions:** `alexedwards/scs/v2` with a hand-written SQLite store adapter (~40 lines, sits on top of `modernc.org/sqlite` to avoid cgo). Cookie name `__Host-id`. `Lifetime: 7d`, `IdleTimeout: 24h`, `HttpOnly: true`, `SameSite: Strict`, `Secure: true in prod, false in dev`. `HashTokenInStore: true` stores the SHA-256 hash of the cookie token.
- **CSRF:** `gorilla/csrf` v1.7.3. `__Host-csrf-token` cookie. `Secure: true in prod, false in dev`. `SameSite: Strict`. `MaxAge: 1h`. `csrf.TemplateField(r)` rendered as `{{ .CSRFField }}` in every form.
- **Passwords:** Argon2id, m=19456, t=3, p=1, 16-byte salt, 32-byte key; PHC-encoded. `PasswordHasher` port has `Hash(plaintext) (hash, error)` and `Verify(hash, plaintext) (ok, needsRehash, error)`. On successful login, the service layer re-hashes if `needsRehash` is true.
- **Auth tokens:** one `auth_tokens` table, `kind` enum (`invite / password_reset`). 256-bit opaque random, base64url-encoded; SHA-256 hash stored. `invite` lifetime 7d, `password_reset` lifetime 1h. Single-use (`used_at`).
- **Authorization:** `RequireUser` middleware reads user ID from session. `RequireTournamentRole.For(role)` middleware joins session user against `tournament_manager` / `tournament_player` for the URL's tournament ID. `RequireSpectatorToken` middleware validates a token in cookie or query for `private` tournament access.
- **Middleware chain order:** RequestID → RealIP → Logger → Recoverer → `sessionManager.LoadAndSave` → `csrf.Protect` → route-group `RequireUser` / `RequireTournamentRole` / `RequireSpectatorToken`.

### Frontend behaviour (W4)

- **htmx + Alpine + Go `html/template`.** No JS toolchain. Server renders HTML; htmx swaps fragments in-place; Alpine handles small client-only state (form toggles, dropdowns, clipboard copy).
- **Bracket rendering:** CSS-Grid with rounds as columns; matches as cards; arrows via CSS `::after` or SVG. Alpine used for "show 4-player / show 2-player" toggle if useful.
- **Forms:** every `<form method="POST">` includes `{{ .CSRFField }}`. Field name default `gorilla.csrf.Token`.
- **Realtime updates:** every page that shows a bracket subscribes via `hx-ext="ws"`; the WS handler upgrades the connection and pumps events. On every event, the bracket fragment is re-fetched (not the whole page).

### Database schema and migrations

- **Tool:** hand-written SQL migrations in `backend/migrations/` as numbered `.sql` files (`0001_init.sql`, `0002_…`, …). A tiny in-house runner reads them, applies the unapplied ones, and records applied versions in a `schema_migrations` table. The tool is picked alongside W3 and is a small piece of code, not a heavyweight framework.
- **Initial migration** (`0001_init.sql`) creates: `users`, `sessions`, `auth_tokens`, `tournaments`, `tournament_manager`, `tournament_player`, `tournament_spectator_tokens`, `matches`, `match_participants`, `tournament_audit_log`, `schema_migrations`. All IDs are TEXT (UUIDv7, sortable). Foreign keys enforced. Indexes on every FK and on `(tournament_id, round, position_in_round)`.
- **WAL mode** is set at open: `PRAGMA journal_mode = WAL;` `PRAGMA foreign_keys = ON;` `PRAGMA busy_timeout = 5000;`.
- **sqlc schema source** is the `migrations/` directory. After every migration, run `sqlc generate` to refresh `internal/adapters/outbound/sqlite/gen/`.
- **Schemas are forward-only.** No down-migrations in v1. A botched migration means `git revert` + a new migration; backups are the safety net (W19).

### Deployment (W11)

- **Local dev:** `docker compose -f deploy/docker-compose.dev.yml up -d` starts Mailpit. `cd backend && go run ./cmd/server` runs the Go app on the host. SQLite at `./data/ludo.db`. No Caddy. HTTP only.
- **Production:** `deploy/docker-compose.yml` runs `app` (Go binary) + `caddy` (HTTPS). Both share image, env, bind-mounts. Single multi-stage `Dockerfile` produces a CGO-disabled static binary; runtime is `alpine` (~10 MB) for `docker exec` debugging.
- **Caddyfile:** bind-mounted into caddy container. ACME auto-cert. HTTP→HTTPS redirect on. Dev runs HTTP only.
- **SQLite bind-mount:** `./data:/data` in both dev and prod. Host-side tooling (cron, `sqlite3 .backup`, Litestream) reads `./data/ludo.db` directly.
- **First-time bootstrap:** `docker compose -f deploy/docker-compose.yml run --rm --build seed --email <email> --password <password>`. The seed binary is re-runnable for disaster recovery.
- **Update flow:** `git pull && docker compose -f deploy/docker-compose.yml up -d --build`. Two commands, no helper script.
- **Env vars (`.env`):** `APP_ENV`, `COOKIE_SECURE`, `CSRF_COOKIE_SECURE` (derived from `COOKIE_SECURE` if unified), `SESSION_KEY` (32 bytes, used as CSRF auth key too in v1), `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD`, `SMTP_FROM`, `PUBLIC_URL`, `DB_PATH` (default `/data/ludo.db`).

### Backups (W19)

- **Package A — simple local nightly.** `deploy/scripts/backup.sh` runs `sqlite3 .backup` against `./data/ludo.db` via host cron at 03:00; one file per day at `./backups/ludo-YYYY-MM-DD.db`; `PRAGMA integrity_check` post-backup; 30-day retention via `find -mtime +30 -delete`. ~25 lines of bash.
- **Restore:** `cp` the chosen file back to `./data/ludo.db` and restart the app. Documented in `docs/deploy.md`.
- **Upgradable** to restic + B2 or Litestream by swapping the script and the cron line — no application-code change.

### API contract (HTML, server-rendered)

- **All read endpoints return HTML pages or fragments** for htmx.
- **All write endpoints are POST forms** (htmx-friendly) with CSRF. A future v2 JSON API adds new inbound adapter handlers; no domain change.
- **Route table (Chi):**
  - `GET /` — public tournament index (htmx-targeted, no auth)
  - `GET /tournaments/{id}` — tournament page (auth depends on visibility; bracket fragment is inner)
  - `GET /fragments/tournaments/{id}/bracket` — htmx target; returns the bracket fragment
  - `GET /fragments/tournaments/{id}/status` — htmx target; returns the status pill
  - `GET /tournaments/{id}/roster` — roster table
  - `GET /tournaments/{id}/audit` — audit log (manager only)
  - `GET /tournaments/{id}/spectator-tokens` — token management (manager only)
  - `GET /tournaments/{id}/download.csv` — bracket CSV
  - `GET /tournaments/{id}/download.png` — bracket PNG (server-rendered or SVG-to-PNG)
  - `POST /tournaments` — create
  - `POST /tournaments/{id}/edit` — edit settings
  - `POST /tournaments/{id}/visibility` — change visibility
  - `POST /tournaments/{id}/registration-mode` — change registration mode
  - `POST /tournaments/{id}/open-registration` — `draft → registration_open`
  - `POST /tournaments/{id}/close-registration` — `registration_open → draft`
  - `POST /tournaments/{id}/preview-bracket` — return bracket shape preview (HTML fragment)
  - `POST /tournaments/{id}/start` — `registration_open → in_progress` (generates bracket)
  - `POST /tournaments/{id}/cancel` — cancel
  - `POST /tournaments/{id}/force-close` — `in_progress → completed`
  - `POST /tournaments/{id}/roster/add` — add player by user id (manager)
  - `POST /tournaments/{id}/roster/remove` — remove player (manager)
  - `POST /tournaments/{id}/roster/invite` — send platform invite (manager)
  - `POST /tournaments/{id}/roster/{playerID}/withdraw` — withdraw (self)
  - `POST /tournaments/{id}/roster/{playerID}/seed` — edit seed (manager, pre-bracket)
  - `POST /tournaments/{id}/spectator-tokens` — issue
  - `POST /tournaments/{id}/spectator-tokens/{tokenID}/revoke` — revoke
  - `GET /tournaments/{id}/register` — self-register form (auth required, `self_register` only)
  - `POST /tournaments/{id}/register` — self-register (auth required, `self_register` only)
  - `GET /matches/{id}` — match page
  - `POST /matches/{id}/start` — `ready → in_progress`
  - `POST /matches/{id}/record` — `in_progress → completed` (form fields: per-slot advancing positions)
  - `POST /matches/{id}/correct` — manager correction (downstream-lock check)
  - `GET /sign-in` / `POST /sign-in` — sign in
  - `POST /sign-out` — sign out
  - `GET /change-password` / `POST /change-password`
  - `GET /invites/{token}` / `POST /invites/{token}` — accept invite
  - `GET /password-resets/{token}` / `POST /password-resets/{token}` — redeem password reset
  - `GET /tournaments/new` / `POST /tournaments` — create tournament
  - `GET /ws/tournaments/{id}` — WebSocket upgrade (per-tournament room; auth via session or spectator token)

### Cookie and session details (W10, locked)

| Cookie | Name | Purpose | Secure | HttpOnly | SameSite | Path |
|---|---|---|---|---|---|---|
| Session | `__Host-id` | scs session ID | prod true, dev false | true | Strict | `/` |
| CSRF | `_gorilla_csrf` | masked CSRF token | prod true, dev false | true | Strict | `/` |

`SameSite=Strict` is OWASP's recommendation. The trade-off (cross-site navigation doesn't carry the session) is irrelevant for v1 — invite-accept and password-reset links are unauthenticated by definition.

### Bracket query patterns (sqlc-tagged)

- `GetTournament(id)` — single tournament
- `ListPublicTournaments()` — index, filtered `visibility = 'public'`, sorted by `started_at DESC NULLS LAST, scheduled_start_at DESC NULLS LAST`
- `ListTournamentsForUser(userID)` — dashboard; tournaments where the user is a manager or a player
- `GetBracket(tournamentID)` — all matches + participants
- `GetDownstreamMatches(matchID)` — recursive CTE on `next_match_id`
- `GetUpstreamMatches(matchID)` — recursive CTE reverse
- `GetReadyMatchesForPlayer(userID, tournamentID)` — `ready` status + user is a participant
- `InsertMatch`, `UpdateMatchStatus`, `UpdateMatchParticipants` — write path
- `InsertAuditLog`, `ListAuditLog(tournamentID)` — audit log

### Configuration

- **`.env` file** at the repo root, loaded by `cmd/server/main.go` via a small in-house env reader (no dotenv library).
- **Required env vars:** `APP_ENV` (`development` or `production`), `COOKIE_SECURE` (bool, derived from `APP_ENV`), `SESSION_KEY` (32 bytes; also used as CSRF auth key in v1), `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD`, `SMTP_FROM`, `PUBLIC_URL`, `DB_PATH` (default `./data/ludo.db` in dev, `/data/ludo.db` in prod).
- **`.env.example`** checked into the repo, `.env` gitignored.
- **First-time setup** documented in `docs/deploy.md`: `git clone` → `cp .env.example .env && $EDITOR .env` → `docker compose -f deploy/docker-compose.yml run --rm --build seed --email ... --password ...` → `docker compose -f deploy/docker-compose.yml up -d --build`.

### Project structure (top-level)

```
ludo-tournament-manager/
├── AGENTS.md
├── WAYFINDER.md
├── GLOSSARY.md
├── CONTEXT.md
├── docs/
│   ├── adr/
│   ├── agents/
│   ├── architecture/hexagonal.md
│   ├── domain-model.md
│   ├── permissions.md
│   ├── setup.md
│   ├── deploy.md
│   └── deploy-script-notes.md
├── deploy/
│   ├── docker-compose.yml              (prod: app + caddy)
│   ├── docker-compose.dev.yml          (dev: mailpit only)
│   ├── Caddyfile
│   ├── Dockerfile                      (multi-stage; static binary in alpine)
│   └── scripts/backup.sh
├── backend/
│   ├── go.mod, go.sum
│   ├── cmd/{server,seed}/main.go
│   ├── migrations/
│   ├── queries/
│   ├── sqlc.yaml
│   └── internal/{core,adapters}/
├── research/
└── .scratch/
    └── mvp-launch/
        ├── spec.md                    (this file)
        └── issues/                    (implementation tickets, numbered)
```

## Testing Decisions

### What makes a good test

- **Test external behaviour, not implementation.** Service tests assert that calling a service with a given command produces the right domain state and the right side effects on its ports. Handler tests assert that the rendered HTML / response status / redirect matches the spec. Don't test private helpers or "the service called the repo" — test "given X, the visible state is Y."
- **No test should depend on another test's state.** Every test owns its DB (an in-memory SQLite) and its fake ports.
- **Pure tests over flaky tests.** No time.Sleep, no real time (use `ports.Clock` fakes), no real network. The single exception is the integration test that opens a real WS against the hub — and that's marked as such.
- **One assertion concept per test.** A test titled "TestStartTournament_LocksPlayerList" should not also assert the bracket shape.

### Test seams

1. **Service layer (`internal/core/services/`, package `services_test`)** — primary seam. Every business rule is covered here. Black-box: only the exported `Handle(ctx, cmd)` API. Fakes for every port (`fakeTournamentRepo`, `fakeClock`, `fakeBroadcaster`, `fakeEmailSender`, `fakeSessionStore`).
2. **HTTP handler layer (`internal/adapters/inbound/http/handlers/`, package `handlers_test`)** — secondary seam. Asserts routing, CSRF, role middleware, form binding, template rendering, redirect-after-POST, and the htmx fragment targets. Uses `httptest.NewServer` with the real services and a real in-memory SQLite (so the WS hub and templates are exercised end-to-end through the HTTP boundary). Spectator-token auth is tested at this seam.
3. **Domain layer (`internal/core/domain/`, package `domain`)** — white-box tests for invariants and the pure bracket generator. The bracket generator is the most important domain test: a table of N values × advance maps → expected shape.
4. **Adapter layer (`internal/adapters/outbound/sqlite/`, package `sqlite`)** — round-trip tests against a real in-memory SQLite. Verifies that the port is correctly implemented. Especially important for `session_store.go` (the scs.Store adapter) and the audit log writes (since they're append-only and not exercised by every service test).
5. **Integration test (single file, `internal/adapters/inbound/ws/hub_test.go`)** — opens two real WS clients against the real hub, broadcasts an event, asserts both clients receive it. The single test that exercises the WS hub end-to-end; everything else mocks `ports.Broadcaster`.

### Prior art in the repo

- The repo is greenfield (no code yet) — there's no in-repo test prior art. The hexagonal.md writeup contains a service-test example using `fakeRepo` and `fakeClock`; that pattern is the model for every service test in this MVP.
- For handler tests, the closest reference is the `httptest` package from Go stdlib, which is well-trodden.
- For bracket-generation tests, the algorithm sketch in `docs/domain-model.md` is the spec; tests should cover: 4 players (final only), 8 players (single-elim with no byes), 16 players (single-elim no byes), 17 players (1 bye), 25 players (mixed advance), 100 players (4 in final), 200 players (stress test), and the unhappy path "no valid shape" → returns an error.
- For audit log tests, every state-changing service should have a test that asserts the audit row was written with the right `actor_id`, `action`, `before`, `after`.

### Coverage targets

- **Service layer: 100% of public methods, 100% of business-rule branches.** A new business rule without a service test is a review blocker.
- **HTTP handlers: every route exercised once with success and once with the most likely failure (403, 404, 422).** No exhaustive table tests; the goal is "every route has a smoke test."
- **Domain bracket generator: every constraint checked explicitly.** Final-4 rule, byes, mixed advance, determinism (same input → same output, twice), preview-and-commit shape match.
- **Outbound SQLite: every repo has a round-trip test.** Insert a row, read it back, assert equality. Plus a constraint test (FK rejection, unique violation).
- **No coverage-of-coverage metric.** Coverage numbers can drift; the rules above are what to aim for.

## Out of Scope

- **Notifications** — no email triggers for "your match is ready," "tournament starting soon," "your manager removed you." Resend is wired for invite / password reset only.
- **Public player profile with match history** — a per-player page showing past tournaments, placements, and opponents. The data is captured in the audit log; the view is v2.
- **Multi-format brackets** — only `multi_player_elimination` is in v1. The `format` column is extensible; the generator is a single-elim-with-variable-advance that satisfies the destination.
- **OAuth / social login** — email + password only. No third-party auth.
- **Account self-service creation** — invites are manager-only by destination design.
- **Self-serve password reset** — manager-issued only. The `password_reset` machinery exists so v2 can add a self-serve flow without a schema migration.
- **Mobile-native apps** — web frontend only. The WS event set is shaped to be mobile-friendly, but no app is in v1.
- **Real-time presence, chat, reactions, viewer count, per-match claim locks** — state-only WS. Per-match edit collisions are handled at the write layer (audit log + downstream-lock rule).
- **Internationalisation** — English-only UI strings. No i18n framework.
- **Email enumeration prevention** — the invite endpoint is manager-only and behind auth, so timing-based enumeration is not a concern.
- **Multi-host / clustered deployment** — single VPS or homelab. No k8s, no ECS.
- **Public cloud (AWS / GCP / Azure) deployment** — local + Hetzner only.
- **A web-based first-run wizard** — the seed CLI is the bootstrap; a wizard adds state to the binary that the CLI doesn't.
- **Ansible / Terraform / helper scripts for the deploy** — two commands (`git pull && docker compose up -d --build`) don't earn an abstraction layer.
- **Bracket re-bracketing** — once a tournament is `in_progress`, the bracket is fixed.
- **Score-per-game (cumulative points, total moves)** — v1 only records `advancing_position` (1=winner, 2=second, …). Per-game score is v2 / a future format.
- **Manager self-removal from a non-cancelled tournament** — only via cancel or via co-manager (per W7 / Q9). The "add a co-manager, ask them to remove me" path is the only one.
- **Forfeit / no-show as a separate action** — recorded through the normal match outcome (absent player's `advancing_position` left null or set to lowest).

## Further Notes

- **The wayfinder map is the source of architectural truth.** This spec synthesises the decisions made there (W1, W2, W3, W4, W5, W6, W7, W8, W9, W10, W11, W12, W19, W20, W21). The locked stack, the hexagonal layout, the domain model, the permissions matrix, the visibility tiers, the registration modes, the bracket representation, the auth stack, the deployment shape, the backup strategy, and the real-time scope are all referenced rather than re-decided here. The spec is a synthesis, not a re-decision.
- **The audit log is non-optional.** Every state-changing action writes an audit row, captured at the service layer. This is the cost of every "manager can change X" decision, and it pays for itself twice: (1) manager audit ("who changed this seed"), and (2) the future player-profile match-history view, where the audit log is the data source.
- **Bracket immutability is a hard rule.** Once a tournament is `in_progress`, the bracket shape is part of the historical record. The only editing path is the downstream-lock-correct rule for completed matches, which is enforced at the service layer.
- **The `format` column is a placeholder for v2.** `multi_player_elimination` is the only value in v1, but the column is `TEXT` (with a CHECK constraint) so adding round-robin, swiss, etc. is a service-layer change, not a schema change.
- **Dev/prod parity is a config flip.** The same Docker image runs in dev (with Mailpit) and prod (with Caddy). The same Go binary runs on the host in dev and in a container in prod. The only env-driven flips are `COOKIE_SECURE`, `CSRF_COOKIE_SECURE`, and `DB_PATH`.
- **Time zones** are stored UTC, rendered in the browser's local time via JavaScript (`Intl.DateTimeFormat`). The server never formats times in a specific zone; the `<time datetime="…">` element carries the ISO-8601 UTC value, and a tiny Alpine snippet formats it. This avoids a server-side time-zone setting and keeps the multi-zone case trivial.
- **Markdown rendering of tournament descriptions** uses a small, safe markdown library (the choice is in the same family as the rest of the stack — minimal, server-side, no JS). The library is picked during implementation; the constraint is that raw HTML is stripped.
- **CSV / image downloads** — CSV is generated server-side from a single sqlc query; the image is a server-side SVG rendered to PNG (the SVG is the same data as the bracket HTML, just shaped for download). The library choices are implementation-time, not architectural.
- **Triage label applied:** `ready-for-agent`. No additional triage.
