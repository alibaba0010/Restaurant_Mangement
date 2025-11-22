package utils

import "github.com/google/uuid"
func GenerateUUIDv7() (uuid.UUID, error) {
	// uuid.NewV7 requires a source of randomness and the current time.
	// We'll use time.Now() and the default crypto/rand source.
	return uuid.NewV7()
}