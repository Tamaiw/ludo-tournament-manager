# W9: Pick Go data-access layer

**Type:** research
**State:** closed
**Assignee:** session-2026-10-07
**Blocked by:** W1 ✓, W3 ✓, W12 ✓
**Blocks:** W21 (already closed — no blocking effect)

## Question

How does the Go backend talk to SQLite — `database/sql` + `sqlx`, `sqlc` (SQL-first, generates type-safe Go), `GORM` (heavy ORM), `ent` (Facebook's graph ORM), or hand-written `database/sql` with a tiny query helper — and why?

## Constraints

- Driver is `modernc.org/sqlite` (resolved by W3): pure Go, no cgo, identical Linux/Windows binaries
- Backend uses Chi (resolved by W1): stdlib-shaped handlers, no framework-specific ORM coupling
- ~100 players per tournament, a handful of concurrent tournaments — write load is modest (one UPDATE per recorded match result)
- Schema changes during v1 need a clean migration story (tool chosen separately, but the data-access choice must play nicely with migrations)
- Type safety is welcome but should not require a code-generation step that becomes its own maintenance burden
- Comes from a Java / .NET background — pattern should map to either "raw queries are fine, give me a small helper" or "annotated entities + repository pattern", not a novel approach

## What "good" looks like

A clear single recommendation with reasoning. Cover: how the choice affects schema migration (e.g. `sqlc` regenerates on each migration, `GORM` has AutoMigrate with caveats), how queries are written, how the bracket / tournament queries (which are recursive-ish in single-elim) fit the chosen pattern, and how testing is approached. Suggested filename `research/W9-data-access-layer.md`.

## Resolution (2026-10-07)

Decided:

- **Pick: sqlc** (SQL-first code generation, SQLite engine, `database/sql` SQL package, schema source pointed at the migrations directory).
- **Why**: the bracket queries are SQL-shaped — recursive CTEs for downstream-match walks, JSON1 (`json_each`, `json_extract`) for `players_advancing_per_round` — and they're easier to write as raw SQL than as any ORM DSL. sqlc keeps them in SQL files and gives back idiomatic typed Go.
- **Why not GORM**: AutoMigrate is info-schema-queries-based; not the clean migration tool this project wants. We'd use it only to escape raw `db.Raw(...)` for bracket queries anyway — two systems when one does both.
- **Why not ent**: the schema is relational, not graph. ent's graph traversal is wasted complexity here; its code-gen is heavier than sqlc's (regenerate on every schema-struct change vs only on `.sql` change).
- **Why not sqlx** (the runner-up): type safety at the SQL boundary is valuable and sqlc's codegen is minimal (`sqlc generate` — not a maintenance burden). The bracket queries' recursive-CTE / JSON1 surface benefits more from sqlc's typed result structs than from sqlx's scan-time check.
- **Risk**: sqlc SQLite is documented as **beta** for the Go target. Mitigation: keep `database/sql` escape hatch inside the outbound adapter; if a specific query blocks on a sqlc SQLite edge case, move just that one query to hand-written SQL with sqlx-style scanning, locally inside `internal/adapters/outbound/sqlite/`.
- **Layout**: `internal/adapters/outbound/sqlite/{migrations,queries,gen,repo}/`, with `sqlc.yaml` at the adapter root. Domain code calls into port interfaces in `internal/core/ports/`; the concrete repo calls into sqlc-generated functions.
- **Build**: `sqlc generate` runs via `make generate`. Generated code is committed (so non-sqlc users can build); CI runs `sqlc generate` and diffs to detect drift.
- **Testing**: integration tests against in-memory SQLite (`:memory:` with WAL via modernc.org/sqlite), with migrations applied fresh per test. Bracket-generation algorithm is unit-tested pure against fixture rosters.

Outputs:

- [`../../research/W9-data-access-layer.md`](../../research/W9-data-access-layer.md) — full evaluation of all five candidates with citations, the `sqlc.yaml` layout, and the concrete bracket-query sketches