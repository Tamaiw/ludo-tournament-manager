# W9: Go data-access layer — research and recommendation

**Type:** research
**Date:** 2026-10-07
**Status:** recommendation

## TL;DR

Use **sqlc** with the SQLite engine, pointed at the migrations folder. Write all queries as raw SQL; let sqlc generate type-safe Go bindings. Pair with a hand-written SQL migration tool (the tool itself is picked in a separate ticket — W9 only constrains that the data-access choice plays nicely with one).

If sqlc's SQLite support becomes a blocker in practice, fall back to **sqlx** + hand-written SQL without re-architecting — the hexagonal port boundaries mean swapping the outbound adapter is a localized change.

## Constraints (from W9)

- Driver: `modernc.org/sqlite` (W3) — pure Go, no cgo, identical Linux/Windows binaries.
- Framework: Chi (W1) — stdlib-shaped middleware, no framework-specific ORM coupling.
- ~100 players per tournament, a handful of concurrent tournaments; write load is modest (one UPDATE per recorded match result).
- Schema changes during v1 need a clean migration story.
- Type safety welcome; codegen that becomes its own maintenance burden is not.
- Pattern should map to "raw queries + small helper" or "annotated entities + repository", not a novel paradigm. User comes from Java / .NET.

## Candidates evaluated

### 1. `sqlc` (SQL-first, generates type-safe Go)

- **Author/maintainer:** sqlc-dev (active in 2026).
- **SQLite status:** documented as **beta** for the Go target (`/sqlc-dev/sqlc` docs, `language-support.md`).
- **Workflow:** write `.sql` files (schema + queries), point `sqlc.yaml` at them, run `sqlc generate`. Output is idiomatic Go with one function per query, typed parameters, and typed scan results.
- **Migration story:** clean. `sqlc.yaml` accepts a *directory* of migrations as the schema source (see `docs/howto/ddl.md`), so the same migration files that the migration tool runs are also what sqlc parses. After every migration, regenerate.
- **Bracket queries:** recursive CTEs for downstream-match walks are first-class SQL — they fit the simple `sqlc` query syntax (`-- name: BracketDownstream :many`) without any DSL gymnastics. SQLite's `json_each` for `players_advancing_per_round` (the JSON map column on `tournaments`) is also plain SQL.
- **Type safety:** compile-time guarantee that query output ↔ generated struct is consistent. Mismatched columns fail at generation, not in production.
- **Pros:** SQL-first matches how the bracket queries actually want to be written. Generated code is boring, in a good way. No runtime reflection. No DSL to learn. Plays well with migrations.
- **Cons:** SQLite is beta, so schema parsing edge cases may surface (especially with SQLite-specific functions like `json_each`, `json_extract`). Codegen step is part of the build pipeline.

### 2. `sqlx` (thin layer over `database/sql`)

- **Author/maintainer:** jmoiron (active).
- **Workflow:** hand-write queries in Go code; use `sqlx.DB.Queryx`, `sqlx.NamedExec`, `StructScan`, etc.
- **Migration story:** completely decoupled from the data-access choice. Hand-written SQL migrations in any tool.
- **Bracket queries:** full — write the CTE in Go, scan into a struct with a `db:"..."` tag per field.
- **Type safety:** struct-scan-time, not compile-time. A column rename will compile but fail at scan. Less robust than sqlc but more forgiving than a raw `database/sql` boilerplate.
- **Pros:** zero codegen, zero DSL, familiar to anyone who's used JdbcTemplate (Java) or Dapper (.NET). Stable across all drivers. No surprises.
- **Cons:** manual struct definition per result set. Column-name drift caught at runtime, not compile time. The helper saves boilerplate but you still write the boilerplate.

### 3. `GORM` (full ORM)

- **Workflow:** structs-as-tables; `db.Create(&user)`, `db.Where(...).First(&user)`.
- **Migration story:** AutoMigrate (queries `information_schema` to decide what to add/drop). Documented in `/go-gorm/gorm` docs as convenient but produces *silent destructive changes* on column drops or type changes. Not a true migration tool.
- **Bracket queries:** possible via raw `db.Raw(...)` escape hatch (`api-reference/chainable-api.md`), but then you're using the ORM DSL for CRUD and raw SQL for the complex queries — learning two systems.
- **Type safety:** generic API `gorm.G[T]` provides compile-time-friendly generics (`api-reference/generics.md`).
- **Pros:** big community, lots of recipes, the generic API is modern. Common Hibernate / EF Core pattern — familiar to Java / .NET users coming from the ORM side.
- **Cons:** AutoMigrate is not a real migration story; we'd need a separate migration tool anyway. Heavy for six tables. Performance overhead vs raw SQL. The "annotated entities" paradigm is not what the user asked for ("not a novel approach" caveat aside, full ORM is more paradigm than this project needs).

### 4. `ent` (Facebook graph ORM)

- **Author/maintainer:** ent/ent (active; the entity framework for Go).
- **Workflow:** define schema as Go structs; `go generate ./...` produces typed client code; migrations via `client.Schema.Create(ctx)` (ent's auto-migrate) or hand-written migrations in `migrate/`.
- **Bracket queries:** ent's strength is graph traversal (`ent.User.Query().WithFriends().All(ctx)`); for SQL-side bracket walks (recursive CTEs), the escape hatch is `client.ExecContext(ctx, sql, args...)` (`doc/md/features.md`). So ent wins for graph queries and loses for relational queries — wrong paradigm for this project.
- **SQLite support:** present in docs (`tutorial-setup.md` shows SQLite in-memory connection).
- **Type safety:** strong — every query is a typed function on the generated client.
- **Pros:** type-safe, schema-as-code, code generation is integrated with `go generate`.
- **Cons:** code-gen is heavier than sqlc (regenerate on every schema struct change). Graph ORM isn't a natural fit for a tournament bracket, which is relational. Learning curve for the schema-as-Go paradigm.

### 5. Hand-written `database/sql` (no helper)

- **Workflow:** `sql.Open`, `db.Query`, manual `rows.Scan(&a, &b, ...)`.
- **Pros:** zero dependencies, full SQL power, no abstractions.
- **Cons:** every query writes a `Scan` line that lists every column by position. Boilerplate adds up over twenty queries. No struct scanning.

This is the floor; sqlx is essentially "hand-written database/sql with struct scanning" and is the natural pickup if the abstraction overhead of the others is unwanted.

## Recommendation

**sqlc** with SQLite engine, schema source = migrations directory, generation target = `internal/adapters/outbound/sqlite/gen/`.

### Why sqlc over sqlx

The deciding factor is the bracket queries. From `docs/domain-model.md`, the data model has:

- `tournaments` with a `players_advancing_per_round JSON` field.
- `matches` with `next_match_id` adjacency (chosen in [ADR 0002](docs/adr/0002-bracket-representation.md)).
- `match_participants` with `slot` and `advancing_position`.

The natural way to walk a bracket (e.g., "all downstream matches of match X") is a recursive CTE. The natural way to read `players_advancing_per_round` is SQLite's JSON1 (`json_each`, `json_extract`). These are SQL features; expressing them in a Go DSL or ORM is friction. Writing them as plain SQL and getting type-safe Go back is exactly what sqlc is built for.

sqlc's codegen burden is a single `sqlc generate` invocation, hooked into `make generate` or invoked manually when a `.sql` or migration changes. That's not a maintenance burden — it's a normal part of the build.

### Why sqlc over GORM

GORM's AutoMigrate is convenient but not a real migration tool. The ticket's "Schema changes during v1 need a clean migration story" constraint disqualifies AutoMigrate. We'd end up with GORM's DSL for CRUD plus raw `db.Raw(...)` for bracket queries — two systems, when one (sqlc) does both.

### Why sqlc over ent

The schema is relational, not graph. ent's graph traversal is wasted complexity. ent's codegen is heavier than sqlc's (regenerate on every struct change vs only on `.sql` change).

### Risk and mitigation

sqlc's SQLite engine is documented as beta. Edge cases to watch:
- `json_each` / `json_extract` on the `players_advancing_per_round` column — these are SQLite-specific and may not be in sqlc's type-checker.
- Recursive CTEs — these are standard SQL and should work, but worth a smoke test on day one.

**Mitigation if sqlc SQLite blocks us:**
1. Identify the query.
2. Move just that query off sqlc: write it as raw SQL, expose via a `Querier` interface in the same outbound adapter, hand-scan into the struct (sqlx pattern).
3. Keep sqlc for everything else.

The hexagonal port boundaries mean this is a localized change inside `internal/adapters/outbound/sqlite/`, not a project-wide refactor.

## How queries are written

```
internal/
└── adapters/
    └── outbound/
        └── sqlite/
            ├── gen/                    # output of `sqlc generate` — checked in OR gitignored + regenerated in CI
            │   ├── db.go                # Querier interface + sqlc-generated methods
            │   ├── models.go           # sqlc-generated row types
            │   └── *.sql.go            # one per query
            ├── migrations/             # *.sql migration files, ordered by version
            │   ├── 0001_initial.sql
            │   ├── 0002_add_audit_log.sql
            │   └── ...
            ├── queries/
            │   ├── tournament.sql       # -- name: GetTournament :one / ListTournaments :many
            │   ├── match.sql            # -- name: BracketDownstream :many
            │   └── ...
            ├── sqlc.yaml                # engine: sqlite, schema: migrations/, queries: queries/, gen: ...
            └── repo/
                ├── tournament_repo.go   # implements internal/core/ports.TournamentRepository
                ├── match_repo.go        # implements internal/core/ports.MatchRepository
                └── ...
```

Application code calls `tournamentRepo.Get(ctx, id)`, which calls into the sqlc-generated function, which calls into `database/sql`. The domain layer knows only the port interface.

`sqlc.yaml`:

```yaml
version: "2"
sql:
  - engine: "sqlite"
    queries: "queries"
    schema: "migrations"
    gen:
      go:
        package: "gen"
        out: "gen"
        sql_package: "database/sql"   # not sqlx; we hand-wrap if we ever need named params
        emit_json_tags: true
        emit_db_tags: true            # needed for json marshalling if used
```

## Migration story

- **Tool:** separate ticket (W19, or whichever picks migrations).
- **Files:** numbered `.sql` files in `migrations/`. `0001_initial.sql` is `docs/domain-model.md` rendered as CREATE TABLE statements. Subsequent files append, never modify prior files.
- **sqlc integration:** `schema: "migrations"` in `sqlc.yaml` makes sqlc parse the *latest* snapshot of all migrations (the union) as its type-checking source. After every new migration: `sqlc generate`.
- **Production:** migration tool runs the migrations on startup (or via a CLI subcommand).

The advantage of pointing sqlc at the migrations directory rather than a single `schema.sql`: there's no drift between "what the migration tool runs" and "what sqlc thinks the schema is." They're reading the same files.

## Bracket queries — concrete sketch

`queries/match.sql`:

```sql
-- name: BracketDownstream :many
-- Walk all matches downstream of a starting match in the tournament bracket.
WITH RECURSIVE downstream(id, depth) AS (
    SELECT id, 0 FROM matches WHERE id = ?
    UNION ALL
    SELECT m.id, d.depth + 1
    FROM matches m
    JOIN downstream d ON m.id IN (
        SELECT next_match_id FROM matches WHERE id = d.id
    )
    WHERE d.depth < 32
)
SELECT m.* FROM matches m JOIN downstream d ON m.id = d.id ORDER BY d.depth, m.round, m.position_in_round;

-- name: AdvanceMapFor :one
-- Read the per-round advance-count map from a tournament row.
SELECT json_each.key AS round, json_each.value AS advance_count
FROM tournaments, json_each(tournaments.players_advancing_per_round)
WHERE tournaments.id = ?;
```

These are the queries GORM would force us to drop into raw SQL for; sqlc lets them live in the SQL files alongside everything else.

## Testing approach

Three layers, each tied to the hexagonal boundary:

1. **Domain services** (`internal/core/services`) — pure logic, no DB. Unit-tested with hand-written fixtures. (Already specified in `docs/architecture/hexagonal.md`.)
2. **Outbound adapter** (`internal/adapters/outbound/sqlite`) — integration tests against an *in-memory* SQLite database (`modernc.org/sqlite` supports `:memory:` with the same WAL semantics as file-backed). Each repository test creates a fresh in-memory DB, runs the migrations, inserts fixtures, calls the repo, asserts. Fast (no I/O), hermetic.
3. **Inbound adapter** (`internal/adapters/inbound/http`) — handler tests with the repository port mocked; no DB.

The bracket-generation algorithm is in `internal/core/services/bracket_generator.go` and is pure — unit-tested against fixture rosters. The repository layer just persists what the service computes.

`sqlc_test.go` for each generated function uses the same in-memory pattern: spin up SQLite, run migrations, exec the query, compare against fixture. This is where sqlc's "compile-time match" guarantee pays off — if a struct and a query drift apart, the test catches it immediately.

## Sources

Primary (fetched 2026-10-07):

- [sqlc docs: language support](https://github.com/sqlc-dev/sqlc/blob/main/docs/reference/language-support.md) — confirms SQLite is "beta" for the Go target.
- [sqlc docs: DDL / migrations](https://github.com/sqlc-dev/sqlc/blob/main/docs/howto/ddl.md) — confirms `schema:` can point at a migrations directory.
- [sqlc GitHub README](https://github.com/sqlc-dev/sqlc) — confirms "generates type-safe code from SQL" and the four-step workflow (write SQL → run sqlc → call generated code).
- [ent docs: SQLite tutorial](https://github.com/ent/ent/blob/master/doc/md/tutorial-setup.md) — confirms SQLite support and `ent.Open(dialect.SQLite, "...")` pattern.
- [ent docs: features](https://github.com/ent/ent/blob/master/doc/md/features.md) — confirms `ExecContext` / `QueryContext` raw-SQL escape hatch.
- [GORM docs: generics API](https://github.com/go-gorm/gorm/blob/master/_autodocs/api-reference/generics.md) — confirms `gorm.G[T]` generic API.
- [GORM docs: chainable API / raw](https://github.com/go-gorm/gorm/blob/master/_autodocs/api-reference/chainable-api.md) — confirms `db.Raw(...)` escape hatch.
- [GORM docs: migrator interface](https://github.com/go-gorm/gorm/blob/master/_autodocs/api-reference/migrator.md) — confirms `AutoMigrate` is info-schema-driven.

Repo-internal:

- `WAYFINDER.md` — stack locks (Chi, SQLite, hexagonal).
- `docs/domain-model.md` — schema and bracket-representation context.
- `docs/adr/0002-bracket-representation.md` — adjacency-list choice and the `next_match_id` / `slot_in_next_match` join pattern.
- `docs/architecture/hexagonal.md` — port / adapter boundaries; repository port lives in `internal/core/ports/`.

## Decisions pending after W9

- **Migration tool** itself — sqlc reads the migration files but does not run them. The tool that applies migrations at startup / via a CLI subcommand is picked in a later ticket. Candidates (informational, not decided here): `golang-migrate`, `pressly/goose`, `sqlx`-style hand-rolled runner. This ticket only requires that the choice plays nicely with sqlc — i.e., uses numbered `.sql` files in a directory.

## Decisions to surface

- `sqlc.yaml` lives in `internal/adapters/outbound/sqlite/sqlc.yaml`.
- `sqlc generate` is wired into a Makefile target (`make generate`). Not part of the regular build (slow); run on demand and in a CI check.
- The generated code is committed (git) so other developers don't need sqlc installed locally to build — but a CI check runs `sqlc generate --check` (planned feature; fall back to a "diff is empty" comparison if not available).
- The migrations directory is committed; `db.Open` opens a connection and the app runs migrations on startup if a `--migrate` flag is set, or via `cmd/migrate/main.go`. (Decided alongside the migration tool.)