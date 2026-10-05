# W2: Pick Go WebSocket library

**Type:** research
**State:** resolved
**Assignee:** opencode-research
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

## Resolution

**Chosen library:** [`github.com/coder/websocket`](https://github.com/coder/websocket) (formerly `nhooyr.io/websocket`).

**Why:** Only one with active maintenance in 2026. `gorilla/websocket` is de facto stalled (last release v1.5.3 in June 2024, last commit March 2025). `coder/websocket` ships commits every few weeks and is backed by Coder. It gives us first-class `context.Context` support, safe concurrent writes (no per-connection mutex for broadcasting), a built-in ping/pong API, and is pure Go with zero dependencies — so Linux/Windows builds are identical and it slots behind any stdlib-shaped framework.

Detailed comparison, alternatives considered (gorilla, melody, others) and source links: `research/W2-go-websocket-library.md` on branch `research/W2`.