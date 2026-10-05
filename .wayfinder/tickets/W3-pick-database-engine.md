# W3: Pick database engine

**Type:** research
**State:** resolved
**Assignee:** opencode-research
**Blocked by:** (none)
**Blocks:** W9, W11, W19, W21

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

## Resolution

**Engine: SQLite** (WAL mode, default Go driver `modernc.org/sqlite`).

**Why:** The deployment target is a single machine both on a dev laptop and on the Hetzner VPS — SQLite's "one file on a bind-mount" maps to that 1:1 and removes the second process, the auth surface, and the shared-memory tuning that Postgres would force for no v1 benefit. The write load (one UPDATE per recorded match result) sits comfortably in SQLite's single-writer / many-readers WAL profile; backup is `sqlite3 .backup` or `VACUUM INTO`, restore is swapping the file back in; and `modernc.org/sqlite` keeps the dev loop cgo-free on Windows.

Detailed comparison, configuration to lock in (WAL, `foreign_keys`, `busy_timeout`), rejected alternatives and source links: `research/W3-database-engine.md` on branch `research/W3`.