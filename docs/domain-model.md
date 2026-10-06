# Domain Model

This document describes the core entities of the Ludo Tournament Manager and how they relate. It is the source of truth for the data model; the schema in `internal/adapters/outbound/sqlite/` should match this. Glossary-level definitions live in [`../CONTEXT.md`](../CONTEXT.md); this document covers the *fields*, *relations*, and *algorithms* that the glossary can't capture.

## Scope

The domain model covers Tournament, Bracket, Match, and the recording workflow. User, Manager, Player, Spectator, and Invite are not detailed here — they are covered in the auth/onboarding tickets (W7, W10).

---

## Entities

### Tournament

A manager-organised Ludo event with a fixed set of registered players, a start time, and a final winner.

| Field | Type | Notes |
|------|------|-------|
| `id` | UUID | Primary key |
| `name` | string | Required, 1-120 chars |
| `description` | markdown text | Nullable; rendered as markdown if present |
| `format` | enum | `multi_player_elimination` only for v1; column extensible |
| `status` | enum | `draft / registration_open / in_progress / completed / cancelled` |
| `max_players_per_match` | int | Default 4; bracket is sized from this |
| `min_players_per_match` | int | Default 2; matches are never smaller than this |
| `players_advancing_per_round` | json | Map of round number → advance count; e.g. `{"1": 2, "2": 2, "3": 2}` |
| `scheduled_start_at` | timestamp (UTC) | Nullable; informational only |
| `is_public` | bool | Default true; controls spectator discovery |
| `created_by` | FK → User | Audit only; does not grant extra permissions |
| `created_at` | timestamp | |
| `updated_at` | timestamp | |
| `started_at` | timestamp | Set when status transitions to `in_progress` |
| `completed_at` | timestamp | Set when status transitions to `completed` |

**Lifecycle transitions** (see [ADR 0001](../adr/0001-hexagonal-backend.md) for the architecture the lifecycle service lives in):

```
draft ──(manager)──→ registration_open ──(manager)──→ in_progress ──(auto)──→ completed
                       │                                  │
                       └──(manager)──→ draft              └──(manager)──→ completed (manual close)
                                                          │
draft ──(manager)──→ cancelled       (only valid pre-`in_progress`)
```

- `in_progress` is triggered by the manager clicking "Start tournament", which generates the bracket (locks player list, generates matches, assigns seeds).
- `completed` is automatic when the final match's `winner_player_id` is recorded. The manager can also force-close if a tournament stalls.
- `cancelled` is manager-only and only valid before `in_progress`. Once matches have been played, the tournament can be force-closed into `completed` but not cancelled.

### Tournament Manager (join)

A tournament can have multiple managers, each with full control. The creator is recorded in `tournaments.created_by` but does not have elevated permissions.

| Field | Type | Notes |
|------|------|-------|
| `tournament_id` | FK | Composite PK |
| `manager_id` | FK | Composite PK |
| `added_at` | timestamp | |

All managers are equal. There is no role differentiation in v1.

### Tournament Player (registration)

A registered player for a specific tournament. Created when a player (or a manager on a player's behalf) registers, and persists across the tournament lifecycle.

| Field | Type | Notes |
|------|------|-------|
| `tournament_id` | FK | Composite PK |
| `player_id` | FK | Composite PK |
| `registered_at` | timestamp | Used as tiebreaker when seeds are not assigned |
| `seed` | int | Nullable; set by manager or auto-generated at bracket creation |

A player can register for many tournaments; a tournament has many registered players.

### Match

A single Ludo game within a tournament, with 2 to 4 participants. The game is played on a physical board; the app records the outcome.

| Field | Type | Notes |
|------|------|-------|
| `id` | UUID | Primary key |
| `tournament_id` | FK | |
| `round` | int | 1-indexed; round 1 is the first round played |
| `position_in_round` | int | 1-indexed; unique within `(tournament_id, round)` |
| `status` | enum | `pending / ready / in_progress / completed` |
| `scheduled_at` | timestamp (UTC) | Nullable; per-match (not per-round) since some tournaments play out of order |
| `next_match_id` | FK → Match | Nullable; null for the final match |
| `slot_in_next_match` | enum | `home / away / third / fourth` or null. Where the winner of this match feeds into the next. |
| `winner_player_id` | FK → User | Nullable; set when status = `completed` and `players_advancing_per_round[round] == 1` |
| `created_at` | timestamp | |
| `updated_at` | timestamp | |

**Status transitions:**

```
pending ──(upstream matches complete)──→ ready ──(player or manager)──→ in_progress ──(player or manager)──→ completed
```

- `pending`: upstream matches not yet completed; participants unknown.
- `ready`: all upstream matches completed; participants determined; game not yet started.
- `in_progress`: a participant or manager has marked "we are playing this game now." This is informational — the app does not run the game.
- `completed`: a result has been recorded (all advancing positions set, or explicitly marked complete with optional positions left null).

**Editing a completed match:**

A `completed` match can be edited by a manager only if no downstream match (any match that this match feeds into, transitively) is `in_progress` or `completed`. This prevents retroactive changes from corrupting the bracket.

A player can record a result for a match they are a participant in, transitioning `in_progress → completed`. They cannot edit an already-`completed` match — only managers can do that, subject to the downstream lock.

### Match Participant

A player's entry in a match, recording their slot and finishing position.

| Field | Type | Notes |
|------|------|-------|
| `match_id` | FK | Composite PK |
| `slot` | enum | `home / away / third / fourth`; composite PK |
| `player_id` | FK | Nullable for byes (see below) |
| `advancing_position` | int | 1-indexed; 1 = winner, 2 = second, etc. Required for players who advance; nullable for eliminated players whose position is unrecorded |
| `is_bye` | bool | Default false; if true, this slot was filled by an auto-advance (no game played) |

A match has between 2 and 4 participants. The `slot` enum matches the `slot_in_next_match` enum on the match record, so participants from a parent match can be slotted into the next match by joining on `(match_id, slot)`.

A bye participant has `player_id = NULL` and `is_bye = true`. They occupy a slot but never play; they automatically count as an advancer.

### Bracket

The bracket is **not a separate entity**; it is the set of all `Match` records for a tournament, plus the adjacency relationship encoded by `next_match_id` and `slot_in_next_match`. Querying "the bracket" means querying all matches where `tournament_id = X`.

The bracket is generated when the tournament transitions from `registration_open` to `in_progress`. It is **immutable** from that point forward (modulo manager edits to completed matches under the downstream lock).

### Audit Log

A record of state-changing actions on a tournament. Used both for manager audit (e.g. "who changed this seed") and as the data source for future player profile pages ("show me this player's tournament history").

| Field | Type | Notes |
|------|------|-------|
| `id` | UUID | PK |
| `tournament_id` | FK | |
| `actor_id` | FK → User | Who performed the action |
| `action` | enum | `seed_changed / manager_added / manager_removed / registration_opened / registration_closed / tournament_started / match_result_recorded / match_result_corrected / tournament_cancelled / tournament_completed` |
| `subject_id` | FK | Polymorphic; the match / player / manager being acted on |
| `before` | json | Snapshot of the subject before the action (nullable) |
| `after` | json | Snapshot of the subject after the action (nullable) |
| `recorded_at` | timestamp | |

The audit log is append-only. Spectators cannot read it; managers of the tournament can.

---

## Algorithms

### Bracket Generation

Triggered when a manager starts a tournament (`registration_open → in_progress`).

**Inputs:** `tournament_id`, registered player list, `max_players_per_match`, `min_players_per_match`, `players_advancing_per_round` map.

**Constraints:**
1. Final round has exactly 4 participants.
2. Every non-final round has matches with 2 to `max_players_per_match` participants.
3. The number advancing from each match in round N is between 1 and `min(players_in_match, players_advancing_per_round[N])`.
4. The number advancing from round N equals the number of participants in round N+1.
5. The total players in round 1 equals the number of registered players (or, if the registration is uneven, the round 1 is padded with byes).
6. Within a round, the variance in advance counts across matches is minimised ("most even rounds").
7. The layout is deterministic — given the same inputs, the same bracket is produced.

**Algorithm sketch (greedy + backtracking):**

```
function generate(N, round=1, advance_map):
    if N <= 4:
        # Final round, must be exactly 4
        return [{round, matches: ceil(N/min_players_per_match), per_match: N/ceil(...), advance: 1}]
    
    for G in [ceil(N / max_per_match) .. floor(N / min_players_per_match)]:
        # Try G matches this round
        distribution = partition(N, G, min=min_per_match, max=max_per_match)
        if not distribution: continue
        
        # For each candidate next-round player count
        for next_N in valid_next_counts(distribution, advance_map, target_final=4):
            # Find an advance_mix that sums to next_N with minimum variance
            advance_mix = find_min_variance_advance(distribution, target=next_N)
            if advance_mix:
                return [{round, distribution, advance: advance_mix}, 
                        *generate(next_N, round+1, advance_map)]
    
    # No valid shape found — return error
    raise BracketGenerationError
```

**Performance:** For N ≤ 200, the search space is at most a few hundred states per round, and the depth is at most log₂(N) ≈ 8 rounds. The whole search completes in milliseconds.

**Determinism:** The algorithm tries smaller `G` values first, then smaller variance first. Given the same inputs, it always picks the same shape.

**Preview:** Before the manager clicks "Start tournament", the system runs `generate()` and shows a preview: "Round 1: 25 games (19 advance 2, 6 advance 3) → 56 players. Round 2: 14 games (12 advance 2, 2 advance 3) → 30 players. Round 3: 8 games ... Final: 1 game of 4, advance 1." The manager confirms or adjusts `players_advancing_per_round` and previews again.

### Seeding

Triggered as part of bracket generation, after the layout is computed.

**Inputs:** registered player list, with optional manager-assigned `seed` values.

**Algorithm:**

1. Players with a non-null `seed` are placed in seed order.
2. Players with a null `seed` are sorted by `random()` (using a tournament-specific seed for reproducibility).
3. The combined sorted list is sliced into the round-1 matches per the bracket layout.

The `random()` order is recorded in the audit log so the bracket is reproducible from the audit log alone.

**Manager override:** Managers can edit a player's `seed` after registration but before the tournament starts. Each edit is recorded in the audit log (`action = seed_changed`, `before` and `after` snapshots). Once the tournament is `in_progress`, seeds are locked.

---

## Relationships (entity-relationship summary)

```
User ─┬─< TournamentPlayer >── Tournament ─< Match ─< MatchParticipant >── User
      │                          │             │
      │                          │             └─ (next_match_id, slot_in_next_match) → Match
      │                          │
      ├─< TournamentManager >─── │ (N:M, all managers equal)
      │                          │
      │                          └─< TournamentAuditLog
      │
      └─ (winner_player_id) ─→ Match
```

---

## Open questions (deferred)

These are intentionally not settled in this document; they will be picked up in later tickets.

- **Spectator discovery UX**: how tournaments are listed, filtered, searched (W4 follow-on).
- **Notifications**: email triggers for "your match is ready", "tournament starting soon", etc. (Resend is in place; trigger surface is open).
- **Time zones**: how `scheduled_start_at` is rendered to users in different zones. (Working assumption: store UTC, render in user TZ.)
- **Public match history for player profile pages**: deferred to v2. The data is captured by the audit log; the projection/view is not.
- **Tournament re-bracketing**: not supported in v1. Once a tournament starts, the bracket is fixed.
- **Scores per game**: deferred to v2 / future format. v1 only records `advancing_position`.

---

## Why this design (rationale pointers)

- **Adjacency list for bracket representation** (vs. position indices or closure table): chosen for query clarity and natural fit to "find all matches downstream of this one." See [ADR 0002](../adr/0002-bracket-representation.md).
- **Variable advance within a round**: chosen to satisfy the hard requirement of 4 in the final without requiring a manager to manually compute the per-round math.
- **`players_advancing_per_round` as a JSON map**: chosen because the advance count is per-round, not per-tournament. JSON keeps it extensible (the manager can add/remove rounds from the configuration as they preview).
- **Audit log for all state changes**: chosen to support both manager audit and future player history views. Cheap to add now; hard to retrofit.
- **Match status `pending / ready / in_progress / completed`**: chosen to model the fact that the game is played physically and the app records outcomes. `in_progress` is informational, not enforced.
