package sqlite

import (
	"context"
	"database/sql"
	"time"
)

func nowUnix() int64 { return time.Now().Unix() }

// SessionStore is the SQLite-backed implementation of scs.Store, used via the
// scs adapter in the http inbound layer. Tokens are stored as their SHA-256
// hex digest (HashTokenInStore).
type SessionStore struct{ db *sql.DB }

// Delete removes the session by token hash.
func (s *SessionStore) Delete(ctx context.Context, tokenHash string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE token_hash = ?`, tokenHash)
	return err
}

// Find returns the gob-encoded data blob for the session, and whether the session is present.
func (s *SessionStore) Find(ctx context.Context, tokenHash string) ([]byte, bool, error) {
	var data []byte
	err := s.db.QueryRowContext(ctx, `SELECT data FROM sessions WHERE token_hash = ?`, tokenHash).Scan(&data)
	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return data, true, nil
}

// Commit stores the session data blob with a unix-seconds expiry.
func (s *SessionStore) Commit(ctx context.Context, tokenHash string, data []byte, expiry int64) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO sessions(token_hash, data, expiry)
		VALUES (?, ?, ?)
		ON CONFLICT(token_hash) DO UPDATE SET data = excluded.data, expiry = excluded.expiry`,
		tokenHash, data, expiry)
	return err
}

// All returns every active session's expiry, keyed by token hash.
func (s *SessionStore) All(ctx context.Context) (map[string]int64, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT token_hash, expiry FROM sessions`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int64{}
	for rows.Next() {
		var hash string
		var exp int64
		if err := rows.Scan(&hash, &exp); err != nil {
			return nil, err
		}
		out[hash] = exp
	}
	return out, rows.Err()
}

// DeleteExpired is called periodically by scs's background cleanup loop.
func (s *SessionStore) DeleteExpired(ctx context.Context) error {
	now := nowUnix()
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE expiry <= ?`, now)
	return err
}