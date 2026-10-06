package domain_test

import (
	"strings"
	"testing"

	"github.com/ishaf/cubit/internal/domain"
)

func TestPasswordHashing(t *testing.T) {
	t.Run("Given a valid plaintext password", func(t *testing.T) {
		password := "SuperSecret123!"

		t.Run("When hashing the password then it generates a valid bcrypt hash", func(t *testing.T) {
			hash, err := domain.HashPassword(password)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if hash == "" || hash == password {
				t.Fatalf("expected non-empty different hash, got %q", hash)
			}
			if !strings.HasPrefix(hash, "$2a$") && !strings.HasPrefix(hash, "$2b$") {
				t.Errorf("expected bcrypt prefix $2a$ or $2b$, got %s", hash)
			}

			// Constant-time check verification
			if !domain.CheckPassword(password, hash) {
				t.Errorf("expected CheckPassword to return true for matching password")
			}
			if domain.CheckPassword("WrongPassword!", hash) {
				t.Errorf("expected CheckPassword to return false for incorrect password")
			}
		})
	})

	t.Run("Given an empty password", func(t *testing.T) {
		t.Run("When hashing then it returns an error", func(t *testing.T) {
			_, err := domain.HashPassword("")
			if err == nil {
				t.Fatal("expected error for empty password, got nil")
			}
		})
	})

	t.Run("Given an invalid or malformed hash", func(t *testing.T) {
		t.Run("When verifying password then it returns false without panicking", func(t *testing.T) {
			if domain.CheckPassword("anyPassword", "not-a-valid-bcrypt-hash") {
				t.Errorf("expected CheckPassword to return false for invalid hash")
			}
			if domain.CheckPassword("anyPassword", "") {
				t.Errorf("expected CheckPassword to return false for empty hash")
			}
		})
	})
}

func TestUserDomainEntity(t *testing.T) {
	t.Run("Given valid user attributes", func(t *testing.T) {
		id := "usr-1"
		name := "Admin User"
		email := "admin@example.com"
		hash := "$2a$12$somevalidhash"
		roleID := "role-admin"

		t.Run("When creating a user then it initializes properly", func(t *testing.T) {
			user, err := domain.NewUser(id, name, email, hash, roleID)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if user.ID != id || user.Email != email || user.RoleID != roleID {
				t.Errorf("user fields mismatch: %+v", user)
			}
			if !user.IsActive {
				t.Errorf("expected user to be active by default")
			}
		})
	})

	t.Run("Given invalid email or missing fields", func(t *testing.T) {
		t.Run("When creating user with empty email then it returns an error", func(t *testing.T) {
			_, err := domain.NewUser("usr-1", "Admin", "", "hash", "role-1")
			if err == nil {
				t.Fatal("expected error for empty email, got nil")
			}
		})

		t.Run("When creating user with missing ID then it returns an error", func(t *testing.T) {
			_, err := domain.NewUser("", "Admin", "admin@example.com", "hash", "role-1")
			if err == nil {
				t.Fatal("expected error for empty ID, got nil")
			}
		})
	})
}
