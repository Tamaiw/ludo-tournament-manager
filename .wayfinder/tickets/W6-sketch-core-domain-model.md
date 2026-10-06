# W6: Sketch core domain model

**Type:** grilling
**State:** closed
**Assignee:** session-2026-10-06
**Blocked by:** W12 ✓
**Blocks:** W7 (now unblocked)

## Question

What are the fields, types, and relationships of Tournament, Bracket, and Match? The data model needs to support single elimination cleanly today and slot other formats in later without a rewrite.

## Specific things to settle

- **Tournament lifecycle states**: e.g. draft → registration_open → in_progress → completed → cancelled. Which transitions exist, who triggers them.
- **Tournament fields**: name, description, scheduled start, max players, status, format (enum, even if single-elim is the only value for v1), timestamps.
- **Bracket representation**: adjacency list (match has `next_match_id`), or closure table, or position indices. Single-elim bracket for N players = N-1 matches; the choice of representation affects query patterns.
- **Match fields**: participants, parent match dependencies, scheduled time, status, score, winner. Empty slots when upstream matches haven't been played.
- **Byes**: handling when player count is not a power of two.
- **Seeding**: is there a seed order, or are players placed in registration order?
- **Third-place match**: optional in single-elim; decide.

## What "good" looks like

A written sketch that updates `CONTEXT.md` (new terms) and produces a `docs/domain-model.md` (or `docs/adr/0001-...md` if the decision is hard to reverse) describing entities, fields, and relationships with brief rationale. Use the `grilling` and `domain-modeling` skills before writing.

## Resolution (2026-10-06)

Decided:
- Tournament lifecycle: `draft → registration_open → in_progress → completed` + `cancelled` terminal state
- N:M managers per tournament (no creator-elevated role)
- Bracket representation: adjacency list with `matches.next_match_id` + `slot_in_next_match`
- Match: 2-4 players, separate `match_participants` table, status `pending → ready → in_progress → completed`
- Byes: auto-advance, no phantom participant
- Final: hard requirement of 4 players
- Per-round advance count: manager-editable JSON map, system previews bracket shape, manager confirms
- Bracket generation: greedy + backtracking search, objective = minimum variance in advance counts within a round ("most even rounds"), deterministic output
- Seeding: random by default, manager-overridable, edits recorded in audit log (not public)
- Recording workflow: player can record their own match result; manager can override any match; completed matches locked once a downstream match is `in_progress` or `completed`
- Audit log: `tournament_audit_log` for all state changes; supports manager audit and future player history

Outputs:
- [`../../docs/domain-model.md`](../../docs/domain-model.md) — full entity-relationship model
- [`../../docs/adr/0002-bracket-representation.md`](../../docs/adr/0002-bracket-representation.md) — bracket representation decision
- [`../../CONTEXT.md`](../../CONTEXT.md) — glossary updated with new terms (Bye, Match Participant, Advancing Position, Seed, Tournament Audit Log, Round)