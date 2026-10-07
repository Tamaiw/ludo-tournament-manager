# 10: Seeding (manager edit, default random at start)

**What to build:** While a tournament is in `draft` or `registration_open` status, a manager can edit a registered player's `seed` via `POST /tournaments/{id}/roster/{playerID}/seed` (form field: integer, must be unique within the tournament's roster, must be in `[1, len(roster)]`). The roster page shows the current seed (or "—" if unset) per player, with an inline edit form (htmx-swap, so no full page reload). Each edit writes an audit log row (`seed_changed` with `before` and `after` values). When the tournament is later started (T12), any player with a null seed is assigned a random seed based on a tournament-specific PRNG; the PRNG seed is recorded in the audit log (`tournament_started` row's `after` includes `random_seed: <int>`) so the bracket is reproducible from the audit log alone. Players with manager-assigned seeds are placed first in seed order; the rest fill in by PRNG order.

**Blocked by:** 06 (Tournament settings, visibility, registration mode, show page), 08 (Roster management)

**Status:** ready-for-agent

- [ ] `internal/core/services/edit_seed.go` with `Handle(ctx, EditSeedCmd)`: validates the tournament status (`draft` or `registration_open`); validates the seed is in range and unique within the tournament; updates `tournament_player.seed`; writes audit log (`seed_changed` with `before` and `after`).
- [ ] `internal/core/domain/seed.go` (or fold into tournament.go) has a `GenerateSeeds(players, prngSeed int) (seeds, error)` pure function: players with non-null seeds stay, players with null seeds get assigned by PRNG order; the function returns the full ordered list and the PRNG seed used (so the caller can record it).
- [ ] `internal/core/ports/clock.go` is unchanged from T01 (already declared)
- [ ] `internal/adapters/inbound/http/handlers/player.go` adds `EditSeedSubmit` (htmx-friendly: returns the updated row fragment)
- [ ] `internal/adapters/inbound/http/templates/fragments/roster_table.html` and `pages/tournament_roster.html` show the seed column with an inline edit form per row
- [ ] `internal/adapters/inbound/http/server.go` registers the seed-edit route
- [ ] Tests at the service layer: `edit_seed_test.go` covers happy path, out-of-range seed, duplicate seed, status check, audit log. `seed_generator_test.go` covers determinism (same PRNG seed → same output), manager-assigned seeds preserved, null seeds filled in, audit log captures PRNG seed.
- [ ] Tests at the HTTP handler layer: `POST /tournaments/{id}/roster/{playerID}/seed` as manager updates and returns the new row fragment; as a non-manager returns 403; on an `in_progress` tournament returns 409 (locked)
- [ ] Manual smoke test: add 4 players to a tournament; edit seeds to 2, 4, 1, 3; refresh roster; verify seeds persisted; start the tournament (T12 will trigger); verify the bracket ordering reflects the seeds
