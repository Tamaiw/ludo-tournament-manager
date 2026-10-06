# W6: Sketch core domain model

**Type:** grilling
**State:** open
**Assignee:** (unclaimed)
**Blocked by:** W12 ✓
**Blocks:** W7

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