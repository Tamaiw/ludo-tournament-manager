# W20: Decide real-time scope

**Type:** grilling
**State:** open
**Assignee:** (unclaimed)
**Blocked by:** (none)
**Blocks:** (none)

## Question

What exactly does the WebSocket push to clients? Two scopes are possible:

- **State-only**: push when bracket state changes (a result is recorded, a match moves from `pending` → `in_progress` → `completed`, a new round unlocks). Spectators and players see the bracket update live; that's it.
- **State + presence**: above, plus "who else is currently viewing this tournament" and (optionally) per-match "this match has been claimed by player X".

## Constraints

- Real-time delivery is wired via `coder/websocket` (resolved by W2) behind Chi (W1); both are framework-agnostic on the wire
- Bracket state changes are the obvious must-have; presence is the question
- v1 scope is ~100 concurrent connections per tournament — presence is cheap to add or omit
- No chat, no notifications-in-app, no spectator reactions in scope for v1

## What "good" looks like

A short written scope statement naming what gets pushed and what doesn't, with reasoning. The answer is a resolution comment on this ticket.