# 05: Tournament CRUD + public index

**What to build:** An authenticated user can create a tournament via `GET /tournaments/new` and `POST /tournaments`. The form takes name (required, 1-120 chars), description (optional markdown), `min_players_per_match` (default 2), `max_players_per_match` (default 4), `players_advancing_per_round` (JSON map; defaults to `{1: 1, 2: 1}` for now), and `scheduled_start_at` (optional datetime). The tournament is created in `draft` status with `visibility=private` and `registration_mode=invite_only` defaults (T06 lets the manager change these). The creator is added to `tournament_manager`. The dashboard now lists tournaments the user manages (empty for now beyond what they just created). The public index at `GET /` lists `public` tournaments sorted by `started_at DESC NULLS LAST, scheduled_start_at DESC NULLS LAST` — for now this is empty because all new tournaments default to `private`. Anonymous users hit `/` and see the (empty) public index. Signed-in users hitting `/` get the dashboard.

**Blocked by:** 04 (Dashboard, RequireUser middleware, base layout)

**Status:** ready-for-agent

- [ ] `internal/core/domain/tournament.go` has the `Tournament` struct, `TournamentStatus` enum, `TournamentFormat` enum, lifecycle methods (`OpenRegistration`, `Start`, `Cancel`, `ForceClose`) returning typed errors
- [ ] `internal/core/services/create_tournament.go` with `Handle(ctx, CreateTournamentCmd) (Tournament, error)`: validates inputs, calls `domain.NewTournament`, persists via `TournamentRepository.Save`, adds creator to `tournament_manager`, writes an audit log row (`tournament_created` is not in the audit enum; the addition of the creator to `tournament_manager` is the action that gets logged as `manager_added` with `actor_id = creator`). Returns the new tournament.
- [ ] `internal/core/services/list_tournaments.go` with two methods: `ListForUser(ctx, userID)` and `ListPublic(ctx)` — different queries, different visibility filtering
- [ ] `internal/core/ports/repository.go` adds concrete `TournamentRepository` interface methods (`Save`, `Find`, `ListPublic`, `ListForUser`, `ListManagedBy`, `ListPlayingIn`)
- [ ] `internal/adapters/outbound/sqlite/tournament_repo.go` implements `TournamentRepository` (sqlc-generated or hand-written with sqlx; whichever the W9 decision lands on)
- [ ] `internal/adapters/outbound/sqlite/tournament_manager_repo.go` implements `TournamentManagerRepository`
- [ ] `internal/adapters/outbound/sqlite/audit_log_repo.go` implements `AuditLogRepository` with `Insert` and `ListByTournament`
- [ ] `internal/adapters/inbound/http/handlers/tournament.go` has `NewTournamentPage`, `CreateTournamentSubmit`, `ListMyTournaments` (dashboard data), `ListPublicTournaments` (index)
- [ ] `internal/adapters/inbound/http/templates/pages/{tournament_new,index,dashboard}.html` render the forms and lists
- [ ] `internal/adapters/inbound/http/server.go` registers the new routes
- [ ] Tests at the service layer: `create_tournament_test.go` covers validation (empty name → error, too-long name → error, defaults applied), persistence, audit log row written with correct `actor_id` / `action=manager_added` / `before=null` / `after={manager_id: ...}`; `list_tournaments_test.go` covers the user / public split (private tournaments don't show in public list)
- [ ] Tests at the HTTP handler layer: `GET /tournaments/new` requires auth, renders form with CSRF; `POST /tournaments` with valid data creates and redirects to `/tournaments/{id}` (T06 will build the show page; for now redirect to a placeholder or the dashboard with a flash); `GET /` for an anonymous user shows the empty public index
- [ ] Manual smoke test: sign in as the seeded manager, create a tournament, see it on the dashboard, sign out, see the (still empty) public index at `/`
