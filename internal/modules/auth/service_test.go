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

	// Retrieve pre-seeded developer role
	role, err := repo.GetRoleByID(ctx, domain.SystemRoleDeveloperID)
	if err != nil {
		t.Fatalf("failed to retrieve pre-seeded developer role: %v", err)
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
			if len(claims.Permissions) != len(role.Permissions) || claims.Permissions[0] != role.Permissions[0] {
				t.Errorf("permissions mismatch: %v (expected %v)", claims.Permissions, role.Permissions)
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

func TestAuthService_AutoProvisionAdmin(t *testing.T) {
	ctx := context.Background()

	t.Run("Given an empty database and valid headless credentials", func(t *testing.T) {
		service, _ := setupTestService(t)

		t.Run("When AutoProvisionAdmin is invoked then it creates root admin and enables login", func(t *testing.T) {
			admin, err := service.AutoProvisionAdmin(ctx, "headless@cubit.local", "HeadlessAdminSecret123!")
			if err != nil {
				t.Fatalf("expected successful auto-provisioning, got %v", err)
			}
			if admin == nil || admin.Email != "headless@cubit.local" {
				t.Fatalf("expected admin user, got %+v", admin)
			}
			if admin.RoleID != domain.SystemRoleAdminID {
				t.Errorf("expected role ID %s, got %s", domain.SystemRoleAdminID, admin.RoleID)
			}

			// Verify status is now initialized
			status, err := service.GetStatus(ctx)
			if err != nil || !status.Initialized {
				t.Fatalf("expected cluster to be initialized, got %v (status: %+v)", err, status)
			}

			// Verify user can log in with provisioned credentials
			pair, loggedInUser, err := service.Login(ctx, "headless@cubit.local", "HeadlessAdminSecret123!")
			if err != nil {
				t.Fatalf("expected successful login with provisioned credentials, got %v", err)
			}
			if loggedInUser.ID != admin.ID {
				t.Errorf("expected user ID %s, got %s", admin.ID, loggedInUser.ID)
			}
			if pair.AccessToken == "" {
				t.Error("expected non-empty access token")
			}
		})
	})

	t.Run("Given a database with existing users", func(t *testing.T) {
		service, repo := setupTestService(t)

		// Seed existing role and user
		role := &domain.Role{
			ID:          "role-existing",
			Name:        "ExistingRole",
			IsSystem:    false,
			Permissions: []string{"apps:read"},
			CreatedAt:   time.Now().UTC(),
			UpdatedAt:   time.Now().UTC(),
		}
		_ = repo.CreateRole(ctx, role)
		existingUser, _ := domain.NewUser("usr-orig", "Original User", "original@cubit.local", "hash", role.ID)
		_ = repo.CreateUser(ctx, existingUser)

		t.Run("When AutoProvisionAdmin is invoked then it safely returns nil without error or modifications", func(t *testing.T) {
			user, err := service.AutoProvisionAdmin(ctx, "ignored@cubit.local", "IgnoredPass123!")
			if err != nil {
				t.Fatalf("expected no error when users already exist, got %v", err)
			}
			if user != nil {
				t.Errorf("expected nil user returned when users already exist, got %+v", user)
			}

			count, _ := repo.CountUsers(ctx)
			if count != 1 {
				t.Errorf("expected user count to remain 1, got %d", count)
			}
		})
	})

	t.Run("Given empty or blank credentials", func(t *testing.T) {
		service, _ := setupTestService(t)

		t.Run("When email or password is empty then it safely returns nil without error", func(t *testing.T) {
			user, err := service.AutoProvisionAdmin(ctx, "", "SomePass123!")
			if err != nil || user != nil {
				t.Fatalf("expected nil user and nil error for empty email, got %v, %v", user, err)
			}

			user, err = service.AutoProvisionAdmin(ctx, "admin@cubit.local", "")
			if err != nil || user != nil {
				t.Fatalf("expected nil user and nil error for empty password, got %v, %v", user, err)
			}

			user, err = service.AutoProvisionAdmin(ctx, "", "")
			if err != nil || user != nil {
				t.Fatalf("expected nil user and nil error for empty credentials, got %v, %v", user, err)
			}
		})
	})
}

func TestAuthService_RolesAndPermissions(t *testing.T) {
	ctx := context.Background()
	service, _ := setupTestService(t)

	t.Run("Given permissions catalog query", func(t *testing.T) {
		t.Run("When listing permissions then it returns all system permissions", func(t *testing.T) {
			perms := service.ListPermissions()
			if len(perms) == 0 {
				t.Fatal("expected non-empty permissions list")
			}
			foundAdmin := false
			for _, p := range perms {
				if p == "*" {
					foundAdmin = true
					break
				}
			}
			if !foundAdmin {
				t.Error("expected '*' in permissions list")
			}
		})
	})

	t.Run("Given pre-seeded system roles", func(t *testing.T) {
		t.Run("When listing roles then it includes Admin, Developer, Viewer", func(t *testing.T) {
			roles, err := service.ListRoles(ctx)
			if err != nil {
				t.Fatalf("failed to list roles: %v", err)
			}
			if len(roles) < 3 {
				t.Fatalf("expected at least 3 system roles, got %d", len(roles))
			}

			admin, err := service.GetRole(ctx, domain.SystemRoleAdminID)
			if err != nil {
				t.Fatalf("failed to get admin role: %v", err)
			}
			if !admin.IsSystem {
				t.Errorf("admin role must have IsSystem = true")
			}
		})

		t.Run("When attempting to update a system role then it rejects with ErrCannotModifySystemRole", func(t *testing.T) {
			_, err := service.UpdateRole(ctx, domain.SystemRoleAdminID, "SuperAdmin", "Modified", []string{"*"})
			if !errors.Is(err, authModule.ErrCannotModifySystemRole) {
				t.Fatalf("expected ErrCannotModifySystemRole, got %v", err)
			}
		})

		t.Run("When attempting to delete a system role then it rejects with ErrCannotDeleteSystemRole", func(t *testing.T) {
			err := service.DeleteRole(ctx, domain.SystemRoleDeveloperID)
			if !errors.Is(err, authModule.ErrCannotDeleteSystemRole) {
				t.Fatalf("expected ErrCannotDeleteSystemRole, got %v", err)
			}
		})
	})

	t.Run("Given a custom role creation request", func(t *testing.T) {
		t.Run("When name is empty then it returns ErrInvalidRoleName", func(t *testing.T) {
			_, err := service.CreateRole(ctx, "   ", "Desc", []string{"apps:read"})
			if !errors.Is(err, authModule.ErrInvalidRoleName) {
				t.Fatalf("expected ErrInvalidRoleName, got %v", err)
			}
		})

		t.Run("When creating a valid custom role then it succeeds with IsSystem = false", func(t *testing.T) {
			role, err := service.CreateRole(ctx, "QA Tester", "Quality Assurance team", []string{"apps:read", "deployments:read"})
			if err != nil {
				t.Fatalf("expected no error creating role, got %v", err)
			}
			if role.ID == "" || role.Name != "QA Tester" || role.IsSystem {
				t.Errorf("unexpected role fields: %+v", role)
			}

			// Duplicate name check
			_, err = service.CreateRole(ctx, "QA Tester", "Duplicate", nil)
			if !errors.Is(err, authModule.ErrRoleAlreadyExists) {
				t.Fatalf("expected ErrRoleAlreadyExists, got %v", err)
			}

			// Update custom role
			updated, err := service.UpdateRole(ctx, role.ID, "Lead QA", "Updated description", []string{"apps:read", "deployments:*"})
			if err != nil {
				t.Fatalf("failed to update custom role: %v", err)
			}
			if updated.Name != "Lead QA" || updated.Description != "Updated description" {
				t.Errorf("unexpected updated role fields: %+v", updated)
			}

			// Delete custom role
			err = service.DeleteRole(ctx, role.ID)
			if err != nil {
				t.Fatalf("failed to delete custom role: %v", err)
			}

			// Verify gone
			_, err = service.GetRole(ctx, role.ID)
			if !errors.Is(err, authModule.ErrRoleNotFound) {
				t.Fatalf("expected ErrRoleNotFound, got %v", err)
			}
		})
	})
}


