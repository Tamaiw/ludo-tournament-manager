# 17: Audit log page (manager-only)

**What to build:** A tournament's audit log page at `GET /tournaments/{id}/audit` lists every row from `tournament_audit_log` for that tournament, paginated (50 rows per page, newest first), with columns: timestamp, actor (name + email), action, subject (a human-readable description; for `seed_changed` it's the player name; for `match_result_recorded` it's "Match N"; for `manager_added`/`manager_removed` it's the manager's name; for `spectator_token_issued`/`revoked` it's the token's label), and a "view diff" toggle that expands the row to show the `before` and `after` JSON side-by-side (Alpine for the toggle). Manager-only — `RequireTournamentRole.For(RoleManager)`. Players and spectators get 403. The page is a regular full-page render (not a fragment) because it's a deep page; the diff toggle is the only client-side behaviour.

**Blocked by:** 13 (Match page, record and mark in-progress)

**Status:** ready-for-agent

- [ ] `internal/core/services/list_audit_log.go` with `Handle(ctx, ListAuditLogCmd) ([]AuditLogEntry, error)`: queries the audit log for a tournament with pagination, joins against `users` for the actor name, and resolves the subject to a human-readable description
- [ ] `internal/core/domain/audit_log.go` has the `AuditLogEntry` value type (includes the resolved subject description)
- [ ] `internal/adapters/outbound/sqlite/audit_log_repo.go` adds `ListByTournament(tournamentID, limit, offset) ([]AuditLogEntry, error)` with a join against `users` for the actor name
- [ ] `internal/adapters/inbound/http/handlers/tournament.go` adds `AuditLogPage` (and a `?page=N` query param for pagination)
- [ ] `internal/adapters/inbound/http/templates/pages/tournament_audit_log.html` renders the table with the per-row diff toggle (Alpine `x-data="{ open: false }"`)
- [ ] `internal/adapters/inbound/http/server.go` registers the route with `RequireTournamentRole.For(RoleManager)`
- [ ] Tests at the service layer: `list_audit_log_test.go` covers: rows returned in newest-first order, pagination works, subject descriptions resolved correctly per action type
- [ ] Tests at the HTTP handler layer: `GET /tournaments/{id}/audit` as manager returns 200 with rows; as a non-manager returns 403; as an anonymous user returns 401
- [ ] Manual smoke test: do a series of actions on a tournament (open registration, add players, edit seeds, start, record a result, correct it); view the audit log; see the rows in order; toggle the diff on a `match_result_recorded` row to see the before/after
