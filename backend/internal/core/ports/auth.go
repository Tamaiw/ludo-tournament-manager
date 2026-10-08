package ports

import "context"

// PasswordHasher hashes and verifies passwords using Argon2id (PHC format).
type PasswordHasher interface {
	Hash(plaintext string) (hash string, err error)
	Verify(hash, plaintext string) (ok bool, needsRehash bool, err error)
}

// SessionStore abstracts the backing store for scs sessions. The concrete
// implementation is the scs Store adapter in the sqlite adapter package.
type SessionStoreAdapter interface {
	Delete(ctx context.Context, tokenHash string) error
	Find(ctx context.Context, tokenHash string) ([]byte, bool, error)
	Commit(ctx context.Context, tokenHash string, data []byte, expiry int64) error
	All(ctx context.Context) (map[string]int64, error)
	DeleteExpired(ctx context.Context) error
}

// PasswordHasherAdapter is the interface exposed to services that need to
// hash or verify a password without depending on the concrete adapter.
type PasswordHasherAdapter interface {
	Hash(plaintext string) (string, error)
	Verify(hash, plaintext string) (bool, bool, error)
}
