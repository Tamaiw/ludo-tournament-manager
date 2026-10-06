# Bracket representation: adjacency list with variable per-round advance

Tournament brackets are stored as an adjacency list on the `matches` table (`next_match_id` + `slot_in_next_match`), with a separate `match_participants` table recording each participant's slot and finishing position. The bracket is generated when a tournament transitions from `registration_open` to `in_progress`, and is immutable from that point (modulo manager edits to completed matches under the downstream lock). Bracket generation is a greedy + backtracking search that finds a sequence of (matches, advance mix) per round satisfying the constraints, with the objective of minimising variance in advance counts within a round ("most even rounds").

## Context

Ludo is a 2-4 player board game played on a physical board, and the app is for *managing* tournaments, not running the games. Each "match" in the bracket is a single Ludo game; the manager decides how many finishers from each match advance to the next round. The final round must contain exactly 4 players (whose finishing positions in the final game determine 1st/2nd/3rd/4th place). Player counts are uneven in real use, so byes exist (a player auto-advances without playing). The bracket generation problem is constrained but underdetermined: many valid shapes satisfy "4 in the final" for a given N, and the choice of shape is a fairness/UX question.

## Decision

1. **Adjacency list for bracket structure.** Each match has `next_match_id` and `slot_in_next_match` pointing to the match it feeds into. No position-index derivation; the relationship is stored as data. Reasoning: clear to query, easy to walk downstream, no derivation layer between the data and what the UI shows.
2. **`match_participants` is a separate table.** Each participant is a row with `(match_id, slot, player_id, advancing_position, is_bye)`. This makes "all matches for player X" a clean indexed query, supports 2-4 player matches with the same schema, and makes byes a first-class concept (a row with `player_id IS NULL, is_bye = true`).
3. **Bracket generation runs at tournament-start time.** The bracket is generated when the manager transitions the tournament to `in_progress`. It is a deterministic, balanced search (greedy + backtracking). The output is previewed to the manager before commit.
4. **Variable advance within a round is allowed.** A round can have some matches advancing 2 and some advancing 3, as long as the totals add up and the final round has 4. This is required to satisfy "4 in the final" for arbitrary N.
5. **The objective is "most even rounds."** Within a round, the variance in advance counts is minimised. The algorithm prefers "all advance 2" over "half advance 2, half advance 3" when both are valid.
6. **The layout is deterministic.** Same inputs → same output. This means a manager can preview the bracket, walk away, come back, and see the same shape.

## Considered options

- **Position indices** (matches have `round` + `position_in_round`, `next_match` is computed): rejected because the variable-advance rule breaks the simple "next match is `ceil(position/2)` in round+1" formula. The relationship stops being purely positional and becomes a function of the per-round advance counts. An adjacency list is honest about that.
- **Closure table** (`match_ancestors` materialising every ancestor relationship): rejected as overkill. The bracket is at most 7-8 rounds deep; recursive CTEs are fast enough that materialising the closure buys nothing.
- **Fixed advance per round (no within-round variation)**: rejected because it makes "4 in the final" impossible for many player counts. For example, 100 players with all rounds advance 2 gives 50 → 25 → 13 → 7 → 4 → 2 → 1, which is 6 rounds and a 2-player final — fails the hard requirement.
- **Constraint solver / SMT** for bracket generation: rejected as overkill. The search space for N ≤ 200 is at most a few hundred states per round, depth ≤ 8; greedy + backtracking runs in milliseconds and is much easier to implement and reason about.
- **Round-robin or other formats**: explicitly out of scope for v1; the `format` column on `tournaments` is extensible so this can be added later.

## Consequences

- **Bracket generation is a backend service**, not a SQL query. It lives in `internal/core/services/bracket_generator.go` and is unit-testable with a pure function signature. See [`../architecture/hexagonal.md`](../architecture/hexagonal.md) for where this fits in the hexagonal layout.
- **The `tournament_audit_log` is non-optional.** Variable advance within a round, plus the manager's ability to edit seeds after registration, means the bracket is the result of a sequence of decisions. The audit log captures them all and is the source of truth for "why is the bracket shaped this way?" It's also the data source for the future player-profile match-history view.
- **The "4 in the final" rule is a hard constraint, not a soft one.** The bracket generator must return an error if no valid shape exists for the given N and `players_advancing_per_round` map. The UI surfaces this as "with your current settings, the tournament cannot reach 4 in the final; please adjust the advance counts."
- **Reversing the decision** (e.g. moving from adjacency list to position indices, or from "variable advance" to "fixed advance") would require either a data migration or a "tournament re-bracket" feature, neither of which is in v1. Once a tournament has been played, the bracket shape is part of the historical record.
