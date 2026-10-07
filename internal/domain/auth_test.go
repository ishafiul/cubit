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

func TestPermissionMatching(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		required string
		expected bool
	}{
		// Superuser wildcard
		{"Superuser * matches apps:read", "*", "apps:read", true},
		{"Superuser * matches apps:deploy", "*", "apps:deploy", true},
		{"Superuser * matches nodes:write", "*", "nodes:write", true},
		{"Superuser * matches services:kv:read", "*", "services:kv:read", true},
		{"Superuser * matches arbitrary string", "*", "any:custom:perm", true},

		// Prefix wildcard
		{"apps:* matches apps:read", "apps:*", "apps:read", true},
		{"apps:* matches apps:deploy", "apps:*", "apps:deploy", true},
		{"apps:* matches apps:delete", "apps:*", "apps:delete", true},
		{"apps:* does not match services:read", "apps:*", "services:read", false},
		{"apps:* does not match nodes:write", "apps:*", "nodes:write", false},
		{"services:* matches services:read", "services:*", "services:read", true},
		{"services:* matches services:kv:read", "services:*", "services:kv:read", true},
		{"services:kv:* matches services:kv:read", "services:kv:*", "services:kv:read", true},
		{"services:kv:* does not match services:d1:query", "services:kv:*", "services:d1:query", false},

		// Suffix wildcard
		{"*:read matches apps:read", "*:read", "apps:read", true},
		{"*:read matches nodes:read", "*:read", "nodes:read", true},
		{"*:read matches services:read", "*:read", "services:read", true},
		{"*:read matches services:kv:read", "*:read", "services:kv:read", true},
		{"*:read does not match apps:deploy", "*:read", "apps:deploy", false},
		{"*:read does not match nodes:write", "*:read", "nodes:write", false},

		// Exact matches
		{"Exact match apps:deploy", "apps:deploy", "apps:deploy", true},
		{"Exact match mismatch", "apps:deploy", "apps:read", false},
		{"Exact match case-sensitive mismatch", "Apps:Deploy", "apps:deploy", false},

		// Empty edge cases
		{"Empty pattern", "", "apps:read", false},
		{"Empty required", "apps:read", "", false},
		{"Both empty", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := domain.MatchPermission(tt.pattern, tt.required)
			if got != tt.expected {
				t.Errorf("MatchPermission(%q, %q) = %v; expected %v", tt.pattern, tt.required, got, tt.expected)
			}
		})
	}
}

func TestHasPermission(t *testing.T) {
	t.Run("Given an Admin user with superuser wildcard", func(t *testing.T) {
		adminPerms := []string{"*"}

		if !domain.HasPermission(adminPerms, "apps:deploy") {
			t.Errorf("admin should satisfy apps:deploy")
		}
		if !domain.HasPermission(adminPerms, "nodes:manage") {
			t.Errorf("admin should satisfy nodes:manage")
		}
		if !domain.HasPermission(adminPerms, "users:manage") {
			t.Errorf("admin should satisfy users:manage")
		}
	})

	t.Run("Given a Developer user", func(t *testing.T) {
		devPerms := []string{"apps:*", "services:*", "deployments:*", "domains:*", "nodes:read"}

		if !domain.HasPermission(devPerms, "apps:deploy") {
			t.Errorf("developer should satisfy apps:deploy")
		}
		if !domain.HasPermission(devPerms, "services:kv:read") {
			t.Errorf("developer should satisfy services:kv:read")
		}
		if !domain.HasPermission(devPerms, "nodes:read") {
			t.Errorf("developer should satisfy nodes:read")
		}
		if domain.HasPermission(devPerms, "nodes:write") {
			t.Errorf("developer should NOT satisfy nodes:write")
		}
		if domain.HasPermission(devPerms, "users:manage") {
			t.Errorf("developer should NOT satisfy users:manage")
		}
	})

	t.Run("Given a Viewer user", func(t *testing.T) {
		viewerPerms := []string{"*:read", "apps:read", "services:read", "nodes:read", "domains:read"}

		if !domain.HasPermission(viewerPerms, "apps:read") {
			t.Errorf("viewer should satisfy apps:read")
		}
		if !domain.HasPermission(viewerPerms, "nodes:read") {
			t.Errorf("viewer should satisfy nodes:read")
		}
		if domain.HasPermission(viewerPerms, "apps:deploy") {
			t.Errorf("viewer should NOT satisfy apps:deploy")
		}
		if domain.HasPermission(viewerPerms, "services:kv:write") {
			t.Errorf("viewer should NOT satisfy services:kv:write")
		}
	})

	t.Run("Given empty permissions or empty required", func(t *testing.T) {
		if domain.HasPermission(nil, "apps:read") {
			t.Errorf("nil permissions should return false")
		}
		if domain.HasPermission([]string{"apps:*"}, "") {
			t.Errorf("empty required permission should return false")
		}
	})
}

func TestDefaultSystemRoles(t *testing.T) {
	roles := domain.DefaultSystemRoles()

	if len(roles) != 3 {
		t.Fatalf("expected 3 system roles, got %d", len(roles))
	}

	roleMap := make(map[string]*domain.Role)
	for _, r := range roles {
		roleMap[r.ID] = r
		if !r.IsSystem {
			t.Errorf("expected role %s to have IsSystem = true", r.ID)
		}
	}

	admin, ok := roleMap[domain.SystemRoleAdminID]
	if !ok || admin.Name != domain.SystemRoleAdminName || len(admin.Permissions) != 1 || admin.Permissions[0] != "*" {
		t.Errorf("admin role misconfigured: %+v", admin)
	}

	dev, ok := roleMap[domain.SystemRoleDeveloperID]
	if !ok || dev.Name != domain.SystemRoleDeveloperName || len(dev.Permissions) == 0 {
		t.Errorf("developer role misconfigured: %+v", dev)
	}

	viewer, ok := roleMap[domain.SystemRoleViewerID]
	if !ok || viewer.Name != domain.SystemRoleViewerName || len(viewer.Permissions) == 0 {
		t.Errorf("viewer role misconfigured: %+v", viewer)
	}
}
