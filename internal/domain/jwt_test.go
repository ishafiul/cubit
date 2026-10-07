package domain_test

import (
	"strings"
	"testing"
	"time"

	"github.com/ishaf/cubit/internal/domain"
)

func TestJWTGenerationAndValidation(t *testing.T) {
	secret := "test-cluster-hmac-secret-key-32b!"
	claims := domain.JWTClaims{
		UserID:      "usr-123",
		Email:       "operator@cubit.local",
		RoleID:      "role-admin",
		Permissions: []string{"*"},
	}

	t.Run("Given valid claims and secret", func(t *testing.T) {
		t.Run("When generating a JWT with 15min TTL then it returns a valid 3-part token", func(t *testing.T) {
			token, err := domain.GenerateJWT(claims, secret, 15*time.Minute)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			parts := strings.Split(token, ".")
			if len(parts) != 3 {
				t.Fatalf("expected 3 parts in JWT, got %d", len(parts))
			}

			// Validate
			parsed, err := domain.ValidateJWT(token, secret)
			if err != nil {
				t.Fatalf("expected token validation to succeed, got %v", err)
			}
			if parsed.UserID != claims.UserID {
				t.Errorf("expected UserID %s, got %s", claims.UserID, parsed.UserID)
			}
			if parsed.Email != claims.Email {
				t.Errorf("expected Email %s, got %s", claims.Email, parsed.Email)
			}
			if parsed.RoleID != claims.RoleID {
				t.Errorf("expected RoleID %s, got %s", claims.RoleID, parsed.RoleID)
			}
			if len(parsed.Permissions) != 1 || parsed.Permissions[0] != "*" {
				t.Errorf("expected permissions [*], got %v", parsed.Permissions)
			}
		})
	})

	t.Run("Given an expired JWT token", func(t *testing.T) {
		t.Run("When validating then it returns ErrTokenExpired", func(t *testing.T) {
			token, err := domain.GenerateJWT(claims, secret, -1*time.Minute)
			if err != nil {
				t.Fatalf("failed to generate expired token: %v", err)
			}
			_, err = domain.ValidateJWT(token, secret)
			if err == nil || !strings.Contains(err.Error(), "expired") {
				t.Fatalf("expected expired token error, got %v", err)
			}
		})
	})

	t.Run("Given a token signed with a different secret", func(t *testing.T) {
		t.Run("When validating with wrong secret then it returns signature mismatch error", func(t *testing.T) {
			token, err := domain.GenerateJWT(claims, "original-secret-123456789012345", 15*time.Minute)
			if err != nil {
				t.Fatalf("failed to generate token: %v", err)
			}
			_, err = domain.ValidateJWT(token, "different-wrong-secret-12345678")
			if err == nil || !strings.Contains(err.Error(), "signature") {
				t.Fatalf("expected signature validation failure, got %v", err)
			}
		})
	})

	t.Run("Given a malformed token string", func(t *testing.T) {
		t.Run("When validating malformed token then it returns error without panicking", func(t *testing.T) {
			for _, bad := range []string{"", "header.payload", "a.b.c.d", "not-a-token", "invalid..sig"} {
				_, err := domain.ValidateJWT(bad, secret)
				if err == nil {
					t.Errorf("expected error for malformed token %q, got nil", bad)
				}
			}
		})
	})
}
