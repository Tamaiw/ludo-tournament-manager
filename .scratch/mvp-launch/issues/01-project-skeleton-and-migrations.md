# 01: Project skeleton, Go module, composition root, migrations runner, first migration

**What to build:** The repo has a working Go module, a stub `cmd/server/main.go` composition root that opens a SQLite DB at `./data/ludo.db`, applies all unapplied migrations from `backend/migrations/` via a tiny in-house runner, sets `PRAGMA journal_mode = WAL; foreign_keys = ON; busy_timeout = 5000;`, and serves a `GET /healthz` returning 200. The first migration (`0001_init.sql`) creates the 11 tables (users, sessions, auth_tokens, tournaments, tournament_manager, tournament_player, tournament_spectator_tokens, matches, match_participants, tournament_audit_log, schema_migrations) with all columns, foreign keys, and indexes per the spec. Running the binary on an empty directory creates the DB file and the tables; running it again is a no-op.

**Blocked by:** None (can start immediately)

**Status:** ready-for-agent

- [ ] `go.mod` exists with the locked stack dependencies (chi, coder/websocket, modernc.org/sqlite, scs/v2, gorilla/csrf, golang.org/x/crypto/argon2, sqlc, Resend go client if used)
- [ ] `cmd/server/main.go` reads `APP_ENV` and `DB_PATH` from env, opens SQLite via the outbound adapter, applies migrations, serves `GET /healthz`, prints "ready" to stdout
- [ ] `internal/adapters/outbound/sqlite/open.go` opens the DB, sets pragmas, runs migrations
- [ ] `internal/adapters/outbound/sqlite/migrate.go` (or similar) is the migration runner: reads `migrations/*.sql` in lexical order, applies unapplied ones, records in `schema_migrations`
- [ ] `backend/migrations/0001_init.sql` creates all 11 tables with the columns, FKs, and indexes from the spec
- [ ] `sqlc.yaml` configured with `migrations/` as the schema source and `internal/adapters/outbound/sqlite/gen/` as the output; running `sqlc generate` produces empty-but-valid generated code
- [ ] `internal/core/ports/repository.go` exists with placeholder interfaces (TournamentRepository, MatchRepository, UserRepository, TournamentManagerRepository, TournamentPlayerRepository, TournamentSpectatorTokenRepository, MatchParticipantRepository, AuditLogRepository, AuthTokenRepository, SessionStore) — not implemented yet, just declared
- [ ] `internal/core/ports/{auth,realtime,email,clock}.go` exist with placeholder interfaces
- [ ] `deploy/docker-compose.dev.yml` runs Mailpit only (W8); `deploy/scripts/backup.sh` is a stub `exit 0`
- [ ] `make test` (or `go test ./...`) runs and passes
- [ ] README section in `docs/deploy.md` (or `docs/setup.md` if no deploy doc yet) explains the dev-loop: `docker compose -f deploy/docker-compose.dev.yml up -d && cd backend && go run ./cmd/server`
