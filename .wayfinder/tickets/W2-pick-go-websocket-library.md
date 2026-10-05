# W2: Pick Go WebSocket library

**Type:** research
**State:** open
**Assignee:** (unclaimed)
**Blocked by:** (none)
**Blocks:** W11, W22

## Question

Which WebSocket library fits the Go backend — `gorilla/websocket`, `coder/websocket` (formerly nhooyr.io), `melody`, or another — and why?

## Constraints

- Works with the framework chosen in W1 (assume stdlib-shaped; no exotic framework coupling)
- Supports ~100 concurrent connections per tournament easily
- Reconnection / heartbeat handling is sane out of the box (or trivially added)
- Cross-platform: builds + behaves identically on Linux + Windows
- Mature, actively maintained
- Plays well with the chosen framework's middleware (auth on upgrade, etc.)

## What "good" looks like

A clear single recommendation with reasoning. Use Context7 for current docs on each candidate; check maintenance status (last release, open issues) before committing. Suggested filename `research/W2-go-websocket-library.md`.