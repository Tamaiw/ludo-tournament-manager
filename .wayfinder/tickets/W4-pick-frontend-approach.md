# W4: Pick frontend approach

**Type:** research
**State:** open
**Assignee:** (unclaimed)
**Blocked by:** (none)
**Blocks:** W11, W22

## Question

How is the frontend built? The candidate set:

- **htmx + Alpine.js + Go html/template** — server-rendered, minimal JS, no node toolchain needed
- **Svelte (plain JS, no TS)** — single-file components, great for bracket viz
- **Vue (plain JS, no TS)** — components, larger ecosystem than Svelte
- **React (plain JS, no TS)** — biggest ecosystem, more setup
- **HTMX-only, no interactivity layer** — pure server-rendered with hypermedia

## Constraints

- No TypeScript anywhere (user excluded it)
- Real-time updates must flow into the UI (bracket state changes push to clients)
- Cross-platform dev: same toolchain on Windows + Linux
- Bracket visualization is the hardest UI piece — render in the chosen approach must be plausible without rebuilding the world
- Setup guide must be doable for a beginner on a fresh machine
- Backend will be Go (decided in earlier round)

## What "good" looks like

A clear single recommendation with reasoning. If a JS framework is chosen, justify it over the server-rendered path (and vice versa). Pay attention to: how real-time is wired in (EventSource / WebSocket / polling), how forms are submitted, how the bracket is rendered. Suggested filename `research/W4-frontend-approach.md`.