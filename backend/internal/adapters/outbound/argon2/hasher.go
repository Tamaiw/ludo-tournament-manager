// Package argon2 implements Argon2id password hashing in PHC format.
package argon2

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Parameters per W10 / OWASP Password Storage Cheat Sheet (2025 edition):
// m=19456 (19 MiB), t=3, p=1; 16-byte salt; 32-byte key.
const (
	MemoryKiB = 19456
	Time      = 3
	Threads   = 1
	SaltLen   = 16
	KeyLen    = 32
)

// Hasher implements ports.PasswordHasher.
type Hasher struct{}

func New() *Hasher { return &Hasher{} }

// Hash returns a PHC-encoded Argon2id hash.
func (h *Hasher) Hash(plaintext string) (string, error) {
	if plaintext == "" {
		return "", errors.New("password must not be empty")
	}
	salt := make([]byte, SaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := argon2.IDKey([]byte(plaintext), salt, Time, MemoryKiB, Threads, KeyLen)
	enc := fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		MemoryKiB, Time, Threads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	)
	return enc, nil
}

// Verify checks the plaintext against the hash. Returns ok=true if it matches.
// needsRehash is true if the parameters used to create the hash differ from
// the current Hasher parameters, prompting the caller to re-hash on successful login.
func (h *Hasher) Verify(hash, plaintext string) (bool, bool, error) {
	t, salt, key, err := decode(hash)
	if err != nil {
		return false, false, err
	}
	cmp := argon2.IDKey([]byte(plaintext), salt, t.Time, t.Memory, t.Threads, t.KeyLen)
	if subtle.ConstantTimeCompare(cmp, key) != 1 {
		return false, false, nil
	}
	needsRehash := t.Memory != MemoryKiB || t.Time != Time || t.Threads != Threads
	return true, needsRehash, nil
}

type params struct {
	Memory  uint32
	Time    uint32
	Threads uint8
	KeyLen  uint32
}

func decode(hash string) (params, []byte, []byte, error) {
	parts := strings.Split(hash, "$")
	if len(parts) != 6 {
		return params{}, nil, nil, errors.New("invalid PHC format")
	}
	if parts[1] != "argon2id" {
		return params{}, nil, nil, errors.New("not argon2id")
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return params{}, nil, nil, errors.New("unsupported argon2 version")
	}
	var p params
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &p.Memory, &p.Time, &p.Threads); err != nil {
		return params{}, nil, nil, fmt.Errorf("invalid params: %w", err)
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return params{}, nil, nil, fmt.Errorf("invalid salt: %w", err)
	}
	key, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return params{}, nil, nil, fmt.Errorf("invalid key: %w", err)
	}
	p.KeyLen = uint32(len(key))
	return p, salt, key, nil
}
