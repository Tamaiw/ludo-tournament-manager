package services_test

import "context"

// stubHasher is a tiny fake PasswordHasher that pretends to hash & verify
// without doing real Argon2 work. Real Argon2 is slow; tests don't need it.
type stubHasher struct{}

func (stubHasher) Hash(plaintext string) (string, error) {
	return "v1$" + plaintext, nil
}

func (stubHasher) Verify(hash, plaintext string) (bool, bool, error) {
	return hash == "v1$"+plaintext, false, nil
}

// ensure we don't have an unused import warning
var _ = context.Background