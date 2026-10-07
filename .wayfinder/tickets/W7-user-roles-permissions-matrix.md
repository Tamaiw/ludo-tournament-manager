# W7: User roles + permissions matrix

**Type:** grilling
**State:** closed
**Assignee:** session-2026-10-07
**Blocked by:** W6
**Blocks:** (none)

## Question

Given the destination ("one user, role per tournament"), what is the precise permission matrix — who can do what?

## Specific things to settle

- A user can be a **Manager** of multiple tournaments and a **Player** in others. Can they be both for the *same* tournament (organize + play in their own)? Or is being a Manager exclusive?
- **Permissions**:
  - Create tournament (anyone? only invited users? a Manager flag?)
  - Edit tournament (Manager only? co-Managers?)
  - Cancel tournament (Manager only? any player can withdraw their own participation but cancel-tournament is Manager)
  - Register a player (self-registration? manager adds players?)
  - Record match outcome (Manager only? both players confirm? any user?)
  - Invite new users (any Manager? tournament-specific invite?)
- **Spectator permissions** (decided: open, no account needed) — pin down what they can and can't do.

## What "good" looks like

A written matrix (action × role → yes/no) with reasoning for each "yes". Lives as `docs/permissions.md` or an ADR if hard to reverse. Use the `grilling` and `domain-modeling` skills before writing.

## Resolution (2026-10-07)

Decided:

- **Manager-as-Player in same tournament**: allowed. A Manager may also register as a Player; uniqueness is enforced per table (separate `tournament_manager` and `tournament_player` rows). Audit log flags manager-recorded-self-match.
- **Visibility**: three-tier enum `public / unlisted / private`. Replaces the original `is_public bool`. Private tournaments use per-tournament, revocable **Spectator Tokens**.
- **Registration mode**: per-tournament enum `invite_only / self_register`, manager-toggled. Editable only while `draft` or `registration_open`.
- **Create tournament**: any authenticated user.
- **Match-outcome recording**: single-recorder (any participant or manager), no confirmation window. Audit captures who recorded.
- **Spectator scope**: read-only at every tier. Private tier uses Spectator Tokens for non-account holders.
- **Withdrawal / forfeit**: pre-bracket self-withdrawal is its own action; post-bracket forfeits are recorded through the match outcome (no separate forfeit flow).
- **Invite new users**: manager-only platform invite; tournament-context auto-add via `registration_mode`.
- **Manager self-removal**: only via cancel. A manager cannot drop themselves from the manager list except by cancelling the tournament (only valid pre-`in_progress`). Co-manager removal by another manager is allowed.
- **Visibility transitions**: manager can change anytime; effect on currently-reachable spectators is documented in the permissions matrix.
- **Audit log access**: managers of the tournament only. Players never see the audit log.

Outputs:
- [`../../docs/permissions.md`](../../docs/permissions.md) — full role × action matrix with conditions, plus a decisions log linking each row to the grilling Q that settled it
- [`../../docs/adr/0003-tournament-visibility-tiers.md`](../../docs/adr/0003-tournament-visibility-tiers.md) — three-tier visibility decision and trade-offs
- [`../../CONTEXT.md`](../../CONTEXT.md) — glossary updated with **Tournament Visibility**, **Spectator Token**, **Registration Mode**
- [`../../docs/domain-model.md`](../../docs/domain-model.md) — `Tournament.visibility` enum (replacing `is_public`), `Tournament.registration_mode` enum, new `TournamentSpectatorToken` entity, new audit-log actions