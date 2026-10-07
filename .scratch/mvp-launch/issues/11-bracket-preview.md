# 11: Bracket preview (pure generator, preview endpoint, audit-friendly dry-run)

**What to build:** A manager of a tournament in `draft` or `registration_open` status can request a bracket preview via `POST /tournaments/{id}/preview-bracket` (no state change; the tournament's actual `players_advancing_per_round` is used). The endpoint returns a JSON or HTML fragment describing the shape: per-round `{round, matches, players_per_match_distribution, advance_count_distribution, advancing_to_next_round_total}`. The pure generator in `internal/core/domain/bracket_gen.go` (the algorithm from the spec: greedy + backtracking, objective = minimum variance in advance counts within a round, deterministic, "4 in the final" hard constraint) takes the registered player list, the tournament's `min_players_per_match`, `max_players_per_match`, and the `players_advancing_per_round` map, and returns either a `Bracket` value or a `BracketGenerationError` (no valid shape — the manager must adjust the advance map). The endpoint renders the preview as an htmx-swappable fragment in the tournament show page (or a dedicated preview section). Previewing does NOT change state, does NOT lock the player list, and does NOT log an audit row.

**Blocked by:** 06 (Tournament settings, visibility, registration mode, show page), 10 (Seeding)

**Status:** ready-for-agent

- [ ] `internal/core/domain/bracket_gen.go` implements the pure generator: `GenerateBracket(players, minPerMatch, maxPerMatch, advanceMap, prngSeed) (Bracket, error)`. Returns `BracketGenerationError` if no valid shape exists for the inputs.
- [ ] `internal/core/domain/bracket.go` has the `Bracket`, `Round`, `MatchShape` (a non-persisted value type that describes the structure but not the data) value types
- [ ] `internal/core/services/preview_bracket.go` with `Handle(ctx, PreviewBracketCmd) (Bracket, error)`: loads the tournament and the registered player list, calls the generator, returns the shape
- [ ] `internal/adapters/inbound/http/handlers/tournament.go` adds `PreviewBracketSubmit` (returns an HTML fragment with the shape description)
- [ ] `internal/adapters/inbound/http/templates/fragments/bracket_preview.html` renders the per-round summary in human-readable form ("Round 1: 25 games (19 advance 2, 6 advance 3) → 56 players. Round 2: 14 games … Final: 1 game of 4, advance 1.")
- [ ] `internal/adapters/inbound/http/server.go` registers the preview route
- [ ] Tests at the domain layer: `bracket_gen_test.go` is a table-driven test covering: 4 players (final only, advance 1); 8 players (single-elim, no byes); 16 players (no byes); 17 players (1 bye); 25 players (mixed advance); 100 players (4 in final, multi-round); 200 players (stress); the unhappy path "no valid shape" returns `BracketGenerationError`; determinism: same inputs twice → identical output
- [ ] Tests at the service layer: `preview_bracket_test.go` covers: the service returns the same shape as calling the generator directly; the service is read-only (no audit log row written, no state change)
- [ ] Tests at the HTTP handler layer: `POST /tournaments/{id}/preview-bracket` as manager returns 200 with the fragment; as a non-manager returns 403; for a tournament in `in_progress` or `completed` returns 409 (bracket already exists)
- [ ] Manual smoke test: add 17 players, preview; see "Round 1: 5 games (4×4 + 1×1, 3 byes) → 12 players. Round 2: 4 games ..." ; adjust the advance map (T06 already supports this); re-preview; see the shape change
