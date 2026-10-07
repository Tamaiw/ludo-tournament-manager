# 15: Force-close and cancel lifecycle transitions

**What to build:** A manager can force-close a tournament (transition `in_progress` → `completed`) via `POST /tournaments/{id}/force-close`. Use case: a tournament stalls (e.g. a player no-shows permanently, a manager wants to finalise the bracket with the current results). The service validates the actor is a manager and the current status is `in_progress`, transitions to `completed`, sets `completed_at=now`, writes an audit log row (`tournament_completed` with `before.status=in_progress`, `after.status=completed, completed_at`). The bracket is preserved as-is. The show page now shows the `completed` status pill and a "Tournament ended" banner. A manager can also cancel a tournament (transition `draft` or `registration_open` → `cancelled`) via `POST /tournaments/{id}/cancel`. The service validates status (`draft` or `registration_open` only — cancellation is not allowed once matches have been played), transitions to `cancelled`, writes an audit log row (`tournament_cancelled` with `before.status` and `after.status=cancelled`). The dashboard no longer shows the tournament in active lists but a "Cancelled" filter (or just a "view anyway" link) keeps it visible.

**Blocked by:** 12 (Start tournament)

**Status:** ready-for-agent

- [ ] `internal/core/services/force_close_tournament.go` with `Handle(ctx, ForceCloseTournamentCmd)`: status check (`in_progress` only), update, audit log
- [ ] `internal/core/services/cancel_tournament.go` with `Handle(ctx, CancelTournamentCmd)`: status check (`draft` or `registration_open` only), update, audit log
- [ ] `internal/core/domain/tournament.go` adds `ForceClose(now)` and `Cancel(now)` methods on `Tournament` (returning typed errors)
- [ ] `internal/adapters/inbound/http/handlers/tournament.go` adds `ForceCloseSubmit`, `CancelSubmit`
- [ ] `internal/adapters/inbound/http/templates/pages/tournament_show.html` adds manager-only "Force close" and "Cancel tournament" buttons (visible only when the status allows)
- [ ] `internal/adapters/inbound/http/server.go` registers the routes
- [ ] Tests at the service layer: each service test covers happy path, wrong-status errors, audit log assertions, completed_at / cancelled_at timestamps
- [ ] Tests at the HTTP handler layer: `POST /tournaments/{id}/force-close` as manager returns 302; as non-manager returns 403; on a `draft` tournament returns 409; `POST /tournaments/{id}/cancel` as manager on `draft` returns 302; on `in_progress` returns 409 (the show page shouldn't even render the button in that state, but the API enforces it)
- [ ] Manual smoke test: cancel a `draft` tournament (verify it disappears from active lists); start a tournament, force-close it after round 1 (verify the bracket is preserved and the status pill shows `completed`); attempt to cancel an `in_progress` tournament (button is hidden, direct POST returns 409)
