# W3: Pick database engine

**Type:** research
**State:** open
**Assignee:** (unclaimed)
**Blocked by:** (none)
**Blocks:** W8, W9, W11, W19, W21

## Question

Which database engine for the MVP — Postgres, SQLite, or another — and why?

## Constraints

- Deployment target: local Docker on a laptop, plus a single Hetzner VPS / homelab box
- ~100 players per tournament, a handful of concurrent tournaments
- Real-time bracket updates will produce a modest write load (one row update per result)
- Single-machine deployment: no separate DB host for v1
- Easy backup / restore (single file or single command)
- Cross-platform dev (Linux + Windows)
- Migration story: schema changes during v1 need a clean path

## What "good" looks like

A clear single recommendation with reasoning. Cover: how the deployment target shapes the answer (SQLite's "single file" shines on a single box; Postgres shines on any shared host), how backup works for the chosen engine, and what migration tool will be used (the tool choice is a separate ticket once the engine is locked). Suggested filename `research/W3-database-engine.md`.