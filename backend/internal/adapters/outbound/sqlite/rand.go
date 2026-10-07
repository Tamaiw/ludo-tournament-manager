package sqlite

import cryptorand "crypto/rand"

// cryptoRandRead wraps crypto/rand.Read so the sqlite package doesn't have to
// import crypto/rand directly (keeping the adapter light).
func cryptoRandRead(p []byte) (int, error) {
	return cryptorand.Read(p)
}