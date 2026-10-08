package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	_ "modernc.org/sqlite"
)

// Open opens (or creates) the SQLite file at path, applies the bundled migrations,
// sets WAL + busy_timeout + foreign_keys, and returns *sql.DB ready to use.
func Open(ctx context.Context, path string, migrations []Migration) (*sql.DB, error) {
	if path != ":memory:" {
		if dir := filepath.Dir(path); dir != "" && dir != "." {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return nil, fmt.Errorf("create data dir: %w", err)
			}
		}
	}
	dsn := path
	if path != ":memory:" {
		dsn = path + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(on)"
	}
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("sqlite open: %w", err)
	}
	// modernc.org/sqlite is single-writer; serialize writes via single conn.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("sqlite ping: %w", err)
	}
	if path == ":memory:" {
		if _, err := db.ExecContext(ctx, `PRAGMA foreign_keys=ON; PRAGMA busy_timeout=5000;`); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("sqlite pragmas: %w", err)
		}
	}
	if err := applyMigrations(ctx, db, migrations); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

// applyMigrations reads schema_migrations and applies any unapplied migrations.
// The schema_migrations table is created on the fly if it doesn't exist; the
// migration files themselves must NOT create schema_migrations — that's the
// runner's job.
func applyMigrations(ctx context.Context, db *sql.DB, migrations []Migration) error {
	applied, err := loadAppliedMigrations(ctx, db)
	if err != nil {
		// Table doesn't exist yet — create it.
		if _, err2 := db.ExecContext(ctx, `
			CREATE TABLE schema_migrations (
				version INTEGER PRIMARY KEY,
				applied_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
			)`); err2 != nil {
			return fmt.Errorf("create schema_migrations: %w", err2)
		}
		applied = map[int]bool{}
	}
	if err != nil && applied == nil {
		return err
	}
	sort.Slice(migrations, func(i, j int) bool { return migrations[i].Version < migrations[j].Version })
	for _, m := range migrations {
		if applied[m.Version] {
			continue
		}
		if err := applyMigration(ctx, db, m); err != nil {
			return fmt.Errorf("apply migration %d: %w", m.Version, err)
		}
	}
	return nil
}

func loadAppliedMigrations(ctx context.Context, db *sql.DB) (map[int]bool, error) {
	rows, err := db.QueryContext(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, fmt.Errorf("read schema_migrations: %w", err)
	}
	defer rows.Close()
	applied := map[int]bool{}
	for rows.Next() {
		var v int
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		applied[v] = true
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return applied, nil
}

func applyMigration(ctx context.Context, db *sql.DB, m Migration) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, stmt := range splitStatements(m.SQL) {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("statement %q: %w", firstLine(stmt), err)
		}
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations(version) VALUES (?)`, m.Version); err != nil {
		return err
	}
	return tx.Commit()
}

// splitStatements splits SQL into individual statements, stripping -- line
// comments first so that semicolons inside comments don't confuse the splitter.
func splitStatements(sql string) []string {
	var lines []string
	for _, line := range strings.Split(sql, "\n") {
		if i := strings.Index(line, "--"); i >= 0 {
			line = line[:i]
		}
		lines = append(lines, line)
	}
	cleaned := strings.Join(lines, "\n")
	return strings.Split(cleaned, ";")
}

func firstLine(s string) string {
	if i := strings.Index(s, "\n"); i >= 0 {
		return s[:i]
	}
	return s
}

// Migration is one numbered SQL migration.
type Migration struct {
	Version int
	Name    string
	SQL     string
}
