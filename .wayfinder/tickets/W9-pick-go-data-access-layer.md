# W9: Pick Go data-access layer

**Type:** research
**State:** open
**Assignee:** (unclaimed)
**Blocked by:** W1 ✓, W3 ✓, W12
**Blocks:** W21

## Question

How does the Go backend talk to SQLite — `database/sql` + `sqlx`, `sqlc` (SQL-first, generates type-safe Go), `GORM` (heavy ORM), `ent` (Facebook's graph ORM), or hand-written `database/sql` with a tiny query helper — and why?

## Constraints

- Driver is `modernc.org/sqlite` (resolved by W3): pure Go, no cgo, identical Linux/Windows binaries
- Backend uses Chi (resolved by W1): stdlib-shaped middleware, no framework-specific ORM coupling
- ~100 players per tournament, a handful of concurrent tournaments — write load is modest (one UPDATE per recorded match result)
- Schema changes during v1 need a clean migration story (tool chosen separately, but the data-access choice must play nicely with migrations)
- Type safety is welcome but should not require a code-generation step that becomes its own maintenance burden
- Comes from a Java / .NET background — pattern should map to either "raw queries are fine, give me a small helper" or "annotated entities + repository pattern", not a novel approach

## What "good" looks like

A clear single recommendation with reasoning. Cover: how the choice affects schema migration (e.g. `sqlc` regenerates on each migration, `GORM` has AutoMigrate with caveats), how queries are written, how the bracket / tournament queries (which are recursive-ish in single-elim) fit the chosen pattern, and how testing is approached. Suggested filename `research/W9-data-access-layer.md`.