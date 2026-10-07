package auth_test

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ishaf/cubit/internal/domain"
	"github.com/ishaf/cubit/internal/infrastructure/db"
	authModule "github.com/ishaf/cubit/internal/modules/auth"
)

func setupTestService(t *testing.T) (authModule.Service, authModule.Repository) {
	t.Helper()
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "auth_service_test.db")
	dsn := "file:" + dbPath + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)"

	database, err := db.OpenSQLite(dsn)
	if err != nil {
		t.Fatalf("failed to open test sqlite: %v", err)
	}
	t.Cleanup(func() {
		database.Close()
	})

	repo := authModule.NewRepository(database)
	secret := "cluster-secret-key-32-bytes-long!"
	service := authModule.NewService(repo, secret)
	return service, repo
}

func TestAuthService_TokenEngine(t *testing.T) {
	ctx := context.Background()
	service, repo := setupTestService(t)

	// Seed role and user
	role := &domain.Role{
		ID:          "role-developer",
		Name:        "Developer",
		Description: "Developer role",
		IsSystem:    true,
		Permissions: []string{"apps:*", "services:*"},
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	if err := repo.CreateRole(ctx, role); err != nil {
		t.Fatalf("failed to seed role: %v", err)
	}

	user, err := domain.NewUser("usr-dev", "Alice Dev", "alice@cubit.local", "$2a$12$hash", role.ID)
	if err != nil {
		t.Fatalf("failed to create domain user: %v", err)
	}
	if err := repo.CreateUser(ctx, user); err != nil {
		t.Fatalf("failed to persist user: %v", err)
	}

	t.Run("Given an active user", func(t *testing.T) {
		t.Run("When issuing token pair then it returns valid JWT and refresh token", func(t *testing.T) {
			pair, err := service.IssueTokenPair(ctx, user)
			if err != nil {
				t.Fatalf("expected no error issuing tokens, got %v", err)
			}
			if pair.AccessToken == "" || pair.RefreshToken == "" {
				t.Fatalf("expected non-empty tokens: %+v", pair)
			}
			if !strings.HasPrefix(pair.RefreshToken, "cbt_ref_") {
				t.Errorf("expected refresh token prefix cbt_ref_, got %s", pair.RefreshToken)
			}

			// Validate JWT
			claims, err := service.ValidateAccessToken(pair.AccessToken)
			if err != nil {
				t.Fatalf("failed to validate access token: %v", err)
			}
			if claims.UserID != user.ID || claims.Email != user.Email {
				t.Errorf("claims mismatch: %+v", claims)
			}
			if len(claims.Permissions) != 2 || claims.Permissions[0] != "apps:*" {
				t.Errorf("permissions mismatch: %v", claims.Permissions)
			}
		})
	})

	t.Run("Given an issued refresh token", func(t *testing.T) {
		pair, err := service.IssueTokenPair(ctx, user)
		if err != nil {
			t.Fatalf("failed issuing token pair: %v", err)
		}

		t.Run("When rotating the refresh token then it issues a new pair and revokes the old one", func(t *testing.T) {
			newPair, returnedUser, err := service.RotateRefreshToken(ctx, pair.RefreshToken)
			if err != nil {
				t.Fatalf("expected successful rotation, got %v", err)
			}
			if returnedUser.ID != user.ID {
				t.Errorf("expected user ID %s, got %s", user.ID, returnedUser.ID)
			}
			if newPair.RefreshToken == pair.RefreshToken {
				t.Errorf("expected rotated refresh token to be different from old one")
			}

			// Replay attack / reusing old refresh token must fail
			_, _, err = service.RotateRefreshToken(ctx, pair.RefreshToken)
			if err == nil {
				t.Fatal("expected reuse of old refresh token to fail, got nil error")
			}
		})
	})

	t.Run("Given a refresh token to revoke explicitly", func(t *testing.T) {
		pair, err := service.IssueTokenPair(ctx, user)
		if err != nil {
			t.Fatalf("failed issuing token: %v", err)
		}

		t.Run("When revoking token then subsequent rotation fails", func(t *testing.T) {
			if err := service.RevokeToken(ctx, pair.RefreshToken); err != nil {
				t.Fatalf("expected no error revoking token, got %v", err)
			}

			_, _, err = service.RotateRefreshToken(ctx, pair.RefreshToken)
			if err == nil {
				t.Fatal("expected rotation of revoked token to fail, got nil")
			}
		})
	})

	t.Run("Given an invalid or malformed refresh token", func(t *testing.T) {
		t.Run("When rotating then it returns an error", func(t *testing.T) {
			_, _, err := service.RotateRefreshToken(ctx, "cbt_ref_non_existent_token")
			if err == nil {
				t.Fatal("expected error for non-existent token, got nil")
			}
		})
	})
}

func TestAuthService_StatusProbe(t *testing.T) {
	ctx := context.Background()
	service, repo := setupTestService(t)

	t.Run("Given an empty database with no users", func(t *testing.T) {
		t.Run("When GetStatus is called then it returns initialized: false and current version", func(t *testing.T) {
			status, err := service.GetStatus(ctx)
			if err != nil {
				t.Fatalf("expected no error from GetStatus, got %v", err)
			}
			if status.Initialized {
				t.Errorf("expected initialized to be false on empty database, got true")
			}
			if status.Version != domain.CubitVersion {
				t.Errorf("expected version %s, got %s", domain.CubitVersion, status.Version)
			}
		})
	})

	t.Run("Given a database with an existing user", func(t *testing.T) {
		role := &domain.Role{
			ID:          "role-test",
			Name:        "TestRole",
			IsSystem:    false,
			Permissions: []string{"apps:read"},
			CreatedAt:   time.Now().UTC(),
			UpdatedAt:   time.Now().UTC(),
		}
		if err := repo.CreateRole(ctx, role); err != nil {
			t.Fatalf("failed to create test role: %v", err)
		}
		user, _ := domain.NewUser("usr-1", "Test", "test@cubit.local", "hash", role.ID)
		if err := repo.CreateUser(ctx, user); err != nil {
			t.Fatalf("failed to create test user: %v", err)
		}

		t.Run("When GetStatus is called then it returns initialized: true", func(t *testing.T) {
			status, err := service.GetStatus(ctx)
			if err != nil {
				t.Fatalf("expected no error from GetStatus, got %v", err)
			}
			if !status.Initialized {
				t.Errorf("expected initialized to be true, got false")
			}
			if status.Version != domain.CubitVersion {
				t.Errorf("expected version %s, got %s", domain.CubitVersion, status.Version)
			}
		})
	})
}

func TestAuthService_SetupAndLockout(t *testing.T) {
	ctx := context.Background()
	service, _ := setupTestService(t)

	t.Run("Given a clean cluster with 0 users", func(t *testing.T) {
		t.Run("When Setup is called with valid admin credentials then it creates root admin and returns active tokens", func(t *testing.T) {
			pair, user, err := service.Setup(ctx, "Root Admin", "admin@cubit.local", "SuperSecurePassword123!")
			if err != nil {
				t.Fatalf("expected successful setup, got %v", err)
			}
			if user == nil || user.Email != "admin@cubit.local" {
				t.Fatalf("expected admin user, got %+v", user)
			}
			if user.RoleID != domain.SystemRoleAdminID {
				t.Errorf("expected role ID %s, got %s", domain.SystemRoleAdminID, user.RoleID)
			}
			if pair.AccessToken == "" || pair.RefreshToken == "" {
				t.Fatalf("expected non-empty tokens in pair: %+v", pair)
			}

			// Verify JWT contains admin wildcard permissions
			claims, err := service.ValidateAccessToken(pair.AccessToken)
			if err != nil {
				t.Fatalf("failed to validate setup access token: %v", err)
			}
			if len(claims.Permissions) != 1 || claims.Permissions[0] != "*" {
				t.Errorf("expected superuser wildcard permissions ['*'], got %v", claims.Permissions)
			}

			// Verify cluster status is now initialized
			status, err := service.GetStatus(ctx)
			if err != nil {
				t.Fatalf("failed to get status: %v", err)
			}
			if !status.Initialized {
				t.Errorf("expected cluster to be initialized after setup")
			}
		})

		t.Run("When Setup is called again then it locks out with ErrSetupAlreadyCompleted", func(t *testing.T) {
			_, _, err := service.Setup(ctx, "Attacker", "attacker@evil.local", "AttackerPass123!")
			if err == nil {
				t.Fatal("expected second setup call to fail, got nil")
			}
			if !errors.Is(err, authModule.ErrSetupAlreadyCompleted) {
				t.Fatalf("expected ErrSetupAlreadyCompleted, got %v", err)
			}
		})
	})

	t.Run("Given invalid setup parameters", func(t *testing.T) {
		freshService, _ := setupTestService(t)

		t.Run("When email is empty then it returns validation error", func(t *testing.T) {
			_, _, err := freshService.Setup(ctx, "Admin", "", "Password123!")
			if err == nil {
				t.Fatal("expected error with empty email, got nil")
			}
		})

		t.Run("When password is empty then it returns validation error", func(t *testing.T) {
			_, _, err := freshService.Setup(ctx, "Admin", "admin@cubit.local", "")
			if err == nil {
				t.Fatal("expected error with empty password, got nil")
			}
		})
	})
}

