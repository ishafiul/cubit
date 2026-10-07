package auth_test

import (
	"context"
	"errors"
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

func TestAuthRepository_Roles(t *testing.T) {
	ctx := context.Background()
	repo := setupTestDB(t)

	t.Run("Given a custom role", func(t *testing.T) {
		customRole := &domain.Role{
			ID:          "role-operator",
			Name:        "Operator",
			Description: "Platform operations engineer",
			IsSystem:    false,
			Permissions: []string{"apps:read", "nodes:*"},
			CreatedAt:   time.Now().UTC(),
			UpdatedAt:   time.Now().UTC(),
		}

		t.Run("When creating and querying by name then it returns the role", func(t *testing.T) {
			err := repo.CreateRole(ctx, customRole)
			if err != nil {
				t.Fatalf("failed to create custom role: %v", err)
			}

			byName, err := repo.GetRoleByName(ctx, "operator")
			if err != nil {
				t.Fatalf("failed to get role by name: %v", err)
			}
			if byName.ID != "role-operator" || byName.Name != "Operator" || byName.IsSystem {
				t.Errorf("role data mismatch: %+v", byName)
			}

			roles, err := repo.ListRoles(ctx)
			if err != nil {
				t.Fatalf("failed to list roles: %v", err)
			}
			if len(roles) == 0 {
				t.Fatal("expected non-empty roles list")
			}
		})

		t.Run("When updating the role then changes are persisted", func(t *testing.T) {
			customRole.Description = "Updated operations description"
			customRole.Permissions = []string{"apps:read", "apps:deploy", "nodes:*"}
			customRole.UpdatedAt = time.Now().UTC()

			err := repo.UpdateRole(ctx, customRole)
			if err != nil {
				t.Fatalf("failed to update role: %v", err)
			}

			updated, err := repo.GetRoleByID(ctx, "role-operator")
			if err != nil {
				t.Fatalf("failed to fetch updated role: %v", err)
			}
			if updated.Description != "Updated operations description" {
				t.Errorf("expected updated description, got %s", updated.Description)
			}
			if len(updated.Permissions) != 3 {
				t.Errorf("expected 3 permissions, got %d", len(updated.Permissions))
			}
		})

		t.Run("When deleting the role then it is removed", func(t *testing.T) {
			err := repo.DeleteRole(ctx, "role-operator")
			if err != nil {
				t.Fatalf("failed to delete role: %v", err)
			}

			_, err = repo.GetRoleByID(ctx, "role-operator")
			if err == nil {
				t.Fatal("expected error getting deleted role, got nil")
			}
		})
	})
}

func TestAuthRepository_APITokens(t *testing.T) {
	ctx := context.Background()
	repo := setupTestDB(t)

	// Pre-seed role & user
	role := &domain.Role{
		ID:          "role-deployer",
		Name:        "Deployer",
		Description: "Deployment role",
		Permissions: []string{"apps:deploy"},
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	_ = repo.CreateRole(ctx, role)

	user, _ := domain.NewUser("usr-ci", "CI Agent", "ci@cubit.local", "hash", role.ID)
	_ = repo.CreateUser(ctx, user)

	t.Run("Given a new API token", func(t *testing.T) {
		plainToken, err := domain.GenerateSecureToken("cbt_")
		if err != nil {
			t.Fatalf("failed to generate token: %v", err)
		}
		tokenHash := domain.HashToken(plainToken)
		now := time.Now().UTC()
		expiresAt := now.Add(30 * 24 * time.Hour)

		apiToken := &domain.APIToken{
			ID:        "tok-1",
			UserID:    user.ID,
			Name:      "GitHub Actions Deployer",
			TokenHash: tokenHash,
			RoleID:    role.ID,
			ExpiresAt: &expiresAt,
			CreatedAt: now,
		}

		t.Run("When saving the API token then it can be retrieved by ID and by Hash", func(t *testing.T) {
			err := repo.CreateAPIToken(ctx, apiToken)
			if err != nil {
				t.Fatalf("failed to create api token: %v", err)
			}

			byID, err := repo.GetAPITokenByID(ctx, "tok-1")
			if err != nil {
				t.Fatalf("failed to get api token by ID: %v", err)
			}
			if byID.Name != "GitHub Actions Deployer" || byID.RoleID != role.ID {
				t.Errorf("token data mismatch: %+v", byID)
			}

			byHash, err := repo.GetAPITokenByHash(ctx, tokenHash)
			if err != nil {
				t.Fatalf("failed to get api token by hash: %v", err)
			}
			if byHash.ID != "tok-1" {
				t.Errorf("expected token ID tok-1, got %s", byHash.ID)
			}
		})

		t.Run("When listing tokens then it returns user tokens", func(t *testing.T) {
			userTokens, err := repo.ListAPITokensByUser(ctx, user.ID)
			if err != nil {
				t.Fatalf("failed to list user tokens: %v", err)
			}
			if len(userTokens) != 1 {
				t.Errorf("expected 1 token, got %d", len(userTokens))
			}

			allTokens, err := repo.ListAllAPITokens(ctx)
			if err != nil {
				t.Fatalf("failed to list all tokens: %v", err)
			}
			if len(allTokens) != 1 {
				t.Errorf("expected 1 token, got %d", len(allTokens))
			}
		})

		t.Run("When updating last_used_at then timestamp is updated", func(t *testing.T) {
			usedAt := time.Now().UTC().Add(time.Hour)
			err := repo.UpdateAPITokenLastUsed(ctx, "tok-1", usedAt)
			if err != nil {
				t.Fatalf("failed to update last used: %v", err)
			}

			got, err := repo.GetAPITokenByID(ctx, "tok-1")
			if err != nil {
				t.Fatalf("failed to get token: %v", err)
			}
			if got.LastUsedAt == nil {
				t.Fatalf("expected non-nil last_used_at")
			}
		})

		t.Run("When deleting token then it is removed", func(t *testing.T) {
			err := repo.DeleteAPIToken(ctx, "tok-1")
			if err != nil {
				t.Fatalf("failed to delete token: %v", err)
			}

			_, err = repo.GetAPITokenByID(ctx, "tok-1")
			if !errors.Is(err, authModule.ErrAPITokenNotFound) {
				t.Fatalf("expected ErrAPITokenNotFound, got %v", err)
			}
		})
	})
}


