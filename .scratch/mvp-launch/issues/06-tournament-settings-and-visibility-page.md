# 06: Tournament settings, visibility, registration mode, show page

**What to build:** A tournament's show page at `GET /tournaments/{id}` renders the tournament header (name, description rendered as markdown, status pill, visibility pill, registration mode pill, scheduled start, advance map). The page is accessible to the manager, the registered players, and (subject to visibility) the public. The page also has a manager-only settings section with forms to edit: name, description, scheduled start, advance map, visibility (radio: public/unlisted/private), and registration mode (radio: invite_only/self_register). Each form is its own POST endpoint (`POST /tournaments/{id}/edit`, `POST /tournaments/{id}/visibility`, `POST /tournaments/{id}/registration-mode`). Each change writes an audit log row with the before/after snapshot. The `RequireTournamentRole.For(RoleManager)` middleware gates the manager actions. Anonymous users hitting a `public` tournament page see it; `unlisted` is reachable by URL only (not listed); `private` returns 403 without a spectator token (T09). The public index now shows `public` tournaments.

**Blocked by:** 05 (Tournament CRUD + public index)

**Status:** ready-for-agent

- [ ] `internal/core/services/edit_tournament.go` with `Handle(ctx, EditTournamentCmd) (Tournament, error)`: validates inputs (name length, advance map shape — round numbers ≥ 1, advance count per round ≤ max players), updates fields, writes audit log (`tournament_edited` action; not in the spec's enum list — add it; the spec's audit enum is a minimum)
- [ ] `internal/core/services/change_visibility.go` with `Handle(ctx, ChangeVisibilityCmd)`: validates the new tier, updates `tournaments.visibility`, writes audit log (`visibility_changed` with before/after)
- [ ] `internal/core/services/change_registration_mode.go` with `Handle(ctx, ChangeRegistrationModeCmd)`: validates the new mode, updates the column, writes audit log (`registration_mode_changed`)
- [ ] `internal/adapters/inbound/http/middleware/require_tournament_role.go` joins the session user against `tournament_manager` (or `tournament_player`, configurable) for the URL's tournament ID
- [ ] `internal/adapters/inbound/http/handlers/tournament.go` adds `ShowTournament`, `EditTournamentSubmit`, `ChangeVisibilitySubmit`, `ChangeRegistrationModeSubmit`
- [ ] `internal/adapters/inbound/http/handlers/index.go` has the public tournament list handler (anonymous-accessible, filters by `visibility = 'public'`)
- [ ] `internal/adapters/inbound/http/templates/pages/tournament_show.html` extends the layout, renders the header, status/visibility/registration-mode pills, and a manager-only settings section with the three forms
- [ ] `internal/adapters/inbound/http/templates/fragments/tournament_status_pill.html` (re-fetched by htmx in T16, but defined now)
- [ ] `internal/adapters/inbound/http/server.go` registers the new routes, gated by `RequireTournamentRole.For(RoleManager)` for the settings POSTs and accessible to anyone (with visibility checks) for the GET
- [ ] Tests at the service layer: `edit_tournament_test.go` (validation, audit log); `change_visibility_test.go` (any-state-allowed, audit log); `change_registration_mode_test.go` (only-draft-or-registration-open enforced)
- [ ] Tests at the HTTP handler layer: `GET /tournaments/{id}` as anonymous on a `public` tournament returns 200; on a `private` tournament returns 403; as the manager returns 200 with the settings section; `POST /tournaments/{id}/visibility` as the manager updates the field and writes the audit log; as a non-manager returns 403; as an anonymous user returns 401
- [ ] Manual smoke test: create a tournament, view it (private, 403 anonymous), change visibility to public, view it as anonymous (200), change to unlisted, see it disappear from the public index but the URL still works
