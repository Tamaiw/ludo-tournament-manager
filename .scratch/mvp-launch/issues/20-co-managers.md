# 20: Co-managers (add, remove, last-manager guard)

**What to build:** A manager can add a co-manager to a tournament via `POST /tournaments/{id}/managers` (form: user id or email of the user to add; the system looks up the user by id, or by email if the user already exists, or sends a platform invite if they don't — for now, the simpler path: add by user id, with the user-existing check). The service inserts into `tournament_manager` (composite PK on `tournament_id, manager_id`), writes an audit log row (`manager_added` with `actor_id=<actor>`, `subject_id=<new manager>`, `before=null`, `after={manager_id: <new>}`). The new co-manager now has full manager permissions on the tournament. A manager can remove a co-manager via `POST /tournaments/{id}/managers/{managerID}/remove`. The service validates the actor is a manager of the tournament, deletes the `tournament_manager` row, writes an audit log row (`manager_removed`). The matrix blocks **self-removal** unless the tournament is `cancelled` — the handler returns 409 with "Use cancel to leave this tournament" if the actor attempts to remove themselves and the tournament is not `cancelled`. There is no last-manager guard (any manager can remove any other manager, even if it leaves just one — the design decision in W7 was that this is intentional friction, not a bug). The tournament show page's manager section has a "Co-managers" subsection listing current managers and the add/remove forms.

**Blocked by:** 05 (Tournament CRUD + public index)

**Status:** ready-for-agent

- [ ] `internal/core/services/add_manager.go` with `Handle(ctx, AddManagerCmd)`: validates the actor is a manager; validates the target user exists; inserts into `tournament_manager`; writes audit log
- [ ] `internal/core/services/remove_manager.go` with `Handle(ctx, RemoveManagerCmd)`: validates the actor is a manager; validates the actor is not removing themselves unless the tournament is `cancelled`; deletes the row; writes audit log
- [ ] `internal/core/ports/repository.go` adds `TournamentManagerRepository.Add`, `Remove`, `ListByTournament`, `IsManager` (already partially there from T05; extend as needed)
- [ ] `internal/adapters/inbound/http/handlers/tournament.go` adds `AddManagerSubmit`, `RemoveManagerSubmit`
- [ ] `internal/adapters/inbound/http/templates/pages/tournament_show.html` adds the manager section (or `fragments/managers_section.html`) listing current managers and the add/remove forms
- [ ] `internal/adapters/inbound/http/server.go` registers the routes with `RequireTournamentRole.For(RoleManager)`
- [ ] Tests at the service layer: `add_manager_test.go` (happy path, double-add is a no-op or error, non-manager actor, audit log); `remove_manager_test.go` (happy path, self-removal blocked unless cancelled, audit log)
- [ ] Tests at the HTTP handler layer: the full flows; the self-removal block returns 409 with the right message
- [ ] Manual smoke test: as manager A, add manager B; sign in as B, see the tournament on B's dashboard with full manager actions; as A, remove B; sign in as B, the tournament is no longer on B's dashboard
