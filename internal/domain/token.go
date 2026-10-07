package domain

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
)

// GenerateSecureToken generates a cryptographically secure random token with the given prefix.
func GenerateSecureToken(prefix string) (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to read secure random bytes: %w", err)
	}
	encoded := base64.RawURLEncoding.EncodeToString(bytes)
	return prefix + encoded, nil
}

// HashToken computes the deterministic hex-encoded SHA-256 digest of a token string.
func HashToken(token string) string {
	if token == "" {
		return ""
	}
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
