package local

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// GeneratePassword generates a cryptographically random hex password of the given length.
func GeneratePassword(length int) (string, error) {
	if length <= 0 {
		return "", fmt.Errorf("password length must be positive, got %d", length)
	}
	b := make([]byte, (length+1)/2)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate random bytes: %w", err)
	}
	s := hex.EncodeToString(b)
	return s[:length], nil
}
