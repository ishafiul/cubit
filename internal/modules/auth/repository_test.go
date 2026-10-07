package auth_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/ishaf/cubit/internal/domain"
	"github.com/ishaf/cubit/internal/infrastructure/db"
	authModule "github.com/ishaf/cubit/internal/modules/auth"
)

func setupTestDB(t *testing.T) authModule.Repository {
	t.Helper()
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "auth_repo_test.db")
	dsn := "file:" + dbPath + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)"

	database, err := db.OpenSQLite(dsn)
	if err != nil {
		t.Fatalf("failed to open test sqlite: %v", err)
	}
	t.Cleanup(func() {
		database.Close()
	})

	return authModule.NewRepository(database)
}

func TestAuthRepository_UsersAndTokens(t *testing.T) {
	ctx := context.Background()
	repo := setupTestDB(t)

	// Pre-seed a test role for foreign key constraints
	role := &domain.Role{
		ID:          "role-test-admin",
		Name:        "Test Admin",
		Description: "Admin role for testing",
		IsSystem:    true,
		Permissions: []string{"*"},
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	if err := repo.CreateRole(ctx, role); err != nil {
		t.Fatalf("failed to create test role: %v", err)
	}

	t.Run("Given a new user", func(t *testing.T) {
		user, err := domain.NewUser("usr-1", "Admin", "admin@cubit.local", "$2a$12$hash123", role.ID)
		if err != nil {
			t.Fatalf("failed to create domain user: %v", err)
		}

		t.Run("When saving the user then it can be retrieved by ID and by email", func(t *testing.T) {
			if err := repo.CreateUser(ctx, user); err != nil {
				t.Fatalf("expected no error saving user, got %v", err)
			}

			count, err := repo.CountUsers(ctx)
			if err != nil {
				t.Fatalf("failed to count users: %v", err)
			}
			if count != 1 {
				t.Errorf("expected count 1, got %d", count)
			}

			byID, err := repo.GetUserByID(ctx, "usr-1")
			if err != nil {
				t.Fatalf("failed to get user by ID: %v", err)
			}
			if byID.Email != "admin@cubit.local" || byID.RoleID != role.ID {
				t.Errorf("user fields mismatch: %+v", byID)
			}

			byEmail, err := repo.GetUserByEmail(ctx, "admin@cubit.local")
			if err != nil {
				t.Fatalf("failed to get user by email: %v", err)
			}
			if byEmail.ID != "usr-1" {
				t.Errorf("expected user ID usr-1, got %s", byEmail.ID)
			}
		})
	})

	t.Run("Given a refresh token", func(t *testing.T) {
		tokenPlain := "cbt_ref_test_random_token_string"
		tokenHash := domain.HashToken(tokenPlain)
		now := time.Now().UTC()

		rt := &domain.RefreshToken{
			ID:        "ref-1",
			UserID:    "usr-1",
			TokenHash: tokenHash,
			ExpiresAt: now.Add(7 * 24 * time.Hour),
			CreatedAt: now,
		}

		t.Run("When creating and querying the refresh token then it returns active status", func(t *testing.T) {
			if err := repo.CreateRefreshToken(ctx, rt); err != nil {
				t.Fatalf("failed to create refresh token: %v", err)
			}

			got, err := repo.GetRefreshTokenByHash(ctx, tokenHash)
			if err != nil {
				t.Fatalf("failed to get refresh token by hash: %v", err)
			}
			if got.ID != "ref-1" || got.UserID != "usr-1" {
				t.Errorf("unexpected token data: %+v", got)
			}
			if got.RevokedAt != nil {
				t.Errorf("expected token not to be revoked")
			}
		})

		t.Run("When revoking the refresh token then it records revoked timestamp", func(t *testing.T) {
			if err := repo.RevokeRefreshToken(ctx, "ref-1"); err != nil {
				t.Fatalf("failed to revoke refresh token: %v", err)
			}

			got, err := repo.GetRefreshTokenByHash(ctx, tokenHash)
			if err != nil {
				t.Fatalf("failed to get revoked token: %v", err)
			}
			if got.RevokedAt == nil {
				t.Fatalf("expected revoked_at to be non-nil")
			}
		})
	})
}
