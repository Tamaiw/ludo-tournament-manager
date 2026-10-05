# W7: User roles + permissions matrix

**Type:** grilling
**State:** open
**Assignee:** (unclaimed)
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