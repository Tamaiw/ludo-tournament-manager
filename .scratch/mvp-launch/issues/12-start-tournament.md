# 12: Start tournament (commits bracket, locks player list, transitions to in_progress)

**What to build:** A manager transitions a tournament from `registration_open` to `in_progress` via `POST /tournaments/{id}/start`. The service: (1) validates status is `registration_open`; (2) loads the registered player list and the tournament's settings; (3) calls the bracket generator with the current `players_advancing_per_round` map; (4) generates a tournament-specific PRNG seed and writes it to the audit log; (5) for each player with null seed, assigns a seed via the PRNG; (6) persists the `matches` and `match_participants` rows per the generated shape (with `is_bye` for byes); (7) updates the tournament: `status=in_progress`, `started_at=now`, locks the player list (a flag or simply the status check in subsequent services); (8) writes audit log rows: one `tournament_started` with `after={random_seed, shape, started_at}`, plus one `player_added` (or `seed_changed`) row per auto-seeded player. The show page now displays the bracket (T13 will render it; for now, a placeholder). The open-registration transition (`POST /tournaments/{id}/open-registration`, draft → registration_open) and close-registration transition (`POST /tournaments/{id}/close-registration`, registration_open → draft) are also wired here.

**Blocked by:** 11 (Bracket preview)

**Status:** ready-for-agent

- [ ] `internal/core/services/start_tournament.go` with `Handle(ctx, StartTournamentCmd) (Tournament, error)`: orchestrates the steps above. Uses a single DB transaction (the SQLite adapter uses `BEGIN` / `COMMIT`).
- [ ] `internal/core/services/open_registration.go` with `Handle(ctx, OpenRegistrationCmd)`: status check (`draft` only), update, audit log
- [ ] `internal/core/services/close_registration.go` with `Handle(ctx, CloseRegistrationCmd)`: status check (`registration_open` only), update, audit log
- [ ] `internal/core/ports/repository.go` adds `MatchRepository.SaveBatch` (or a transaction-aware Save) and `MatchParticipantRepository.SaveBatch`
- [ ] `internal/adapters/outbound/sqlite/match_repo.go` and `match_participant_repo.go` implement the above with sqlc-generated batch inserts
- [ ] `internal/adapters/inbound/http/handlers/tournament.go` adds `OpenRegistrationSubmit`, `CloseRegistrationSubmit`, `StartTournamentSubmit`
- [ ] `internal/adapters/inbound/http/templates/pages/tournament_show.html` adds the "Start tournament" button (visible to managers, only when `status=registration_open`) and the "Open registration" / "Close registration" buttons (visible to managers, only when applicable)
- [ ] `internal/adapters/inbound/http/server.go` registers the routes
- [ ] Tests at the service layer: `start_tournament_test.go` covers happy path (status transitions, matches and participants persisted, audit log rows, PRNG seed recorded), errors (wrong status, no valid bracket shape, no players), idempotency (second start attempt fails with conflict); `open_registration_test.go` and `close_registration_test.go` cover status checks and audit logging
- [ ] Tests at the HTTP handler layer: `POST /tournaments/{id}/start` as manager with valid tournament returns 200/302; as non-manager returns 403; on an `in_progress` tournament returns 409; the resulting show page shows the `in_progress` status pill
- [ ] Manual smoke test: create a tournament, open registration, add 8 players, start; verify the show page now shows the bracket; query the DB to confirm 7 matches (8 players, no byes), 8 `match_participants` rows in round 1, audit log has the `tournament_started` row with `random_seed`
