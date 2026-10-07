package domain_test

import (
	"strings"
	"testing"

	"github.com/ishaf/cubit/internal/domain"
)

func TestSecureTokenGenerationAndHashing(t *testing.T) {
	t.Run("Given request to generate a secure refresh token", func(t *testing.T) {
		t.Run("When generating with prefix then it produces a high-entropy string", func(t *testing.T) {
			token1, err := domain.GenerateSecureToken("cbt_ref_")
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if !strings.HasPrefix(token1, "cbt_ref_") {
				t.Errorf("expected prefix cbt_ref_, got %s", token1)
			}
			if len(token1) < 40 {
				t.Errorf("expected token length >= 40, got %d", len(token1))
			}

			token2, err := domain.GenerateSecureToken("cbt_ref_")
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if token1 == token2 {
				t.Errorf("expected distinct random tokens, but token1 == token2")
			}
		})
	})

	t.Run("Given a plaintext token", func(t *testing.T) {
		token := "cbt_ref_sample_token_for_sha256"

		t.Run("When hashing the token then it produces a deterministic 64-char hex SHA-256 hash", func(t *testing.T) {
			hash1 := domain.HashToken(token)
			hash2 := domain.HashToken(token)

			if len(hash1) != 64 {
				t.Errorf("expected 64 char hex hash, got %d", len(hash1))
			}
			if hash1 != hash2 {
				t.Errorf("expected deterministic hash output, got %s != %s", hash1, hash2)
			}
			if domain.HashToken("") != "" {
				t.Errorf("expected empty string hash for empty token, got %s", domain.HashToken(""))
			}
		})
	})
}
