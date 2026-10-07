package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const (
	// BcryptDefaultCost represents the work factor for password hashing in Cubit.
	BcryptDefaultCost = 12

	// CubitVersion defines the current control plane release version.
	CubitVersion = "1.3.0"

	// Built-in system role IDs and names.
	SystemRoleAdminID       = "admin"
	SystemRoleAdminName     = "Admin"
	SystemRoleDeveloperID   = "developer"
	SystemRoleDeveloperName = "Developer"
	SystemRoleViewerID      = "viewer"
	SystemRoleViewerName    = "Viewer"

	// Standard permissions catalog vocabulary.
	PermissionSuperuser = "*"

	// Applications
	PermissionAppsWildcard = "apps:*"
	PermissionAppsRead     = "apps:read"
	PermissionAppsCreate   = "apps:create"
	PermissionAppsUpdate   = "apps:update"
	PermissionAppsDelete   = "apps:delete"
	PermissionAppsDeploy   = "apps:deploy"

	// Services
	PermissionServicesWildcard  = "services:*"
	PermissionServicesKV        = "services:kv:*"
	PermissionServicesD1        = "services:d1:*"
	PermissionServicesR2        = "services:r2:*"
	PermissionServicesCron      = "services:cron:*"
	PermissionServicesQueues    = "services:queues:*"
	PermissionServicesWorkflows = "services:workflows:*"
	PermissionServicesDO        = "services:do:*"

	// Deployments
	PermissionDeploymentsWildcard = "deployments:*"
	PermissionDeploymentsRead     = "deployments:read"
	PermissionDeploymentsRollback = "deployments:rollback"

	// Domains
	PermissionDomainsWildcard = "domains:*"
	PermissionDomainsRead     = "domains:read"
	PermissionDomainsWrite    = "domains:write"

	// Nodes
	PermissionNodesWildcard = "nodes:*"
	PermissionNodesRead     = "nodes:read"
	PermissionNodesWrite    = "nodes:write"

	// Users, Roles, Tokens
	PermissionUsersWildcard  = "users:*"
	PermissionUsersRead      = "users:read"
	PermissionUsersManage    = "users:manage"
	PermissionRolesWildcard  = "roles:*"
	PermissionRolesRead      = "roles:read"
	PermissionRolesManage    = "roles:manage"
	PermissionTokensWildcard = "tokens:*"
	PermissionTokensRead     = "tokens:read"
	PermissionTokensManage   = "tokens:manage"
)

var (
	ErrEmptyPassword   = errors.New("password cannot be empty")
	ErrPasswordTooLong = errors.New("password exceeds maximum allowed length of 72 bytes")
	ErrInvalidUserID   = errors.New("user id cannot be empty")
	ErrInvalidEmail    = errors.New("user email cannot be empty")
	ErrInvalidRoleID   = errors.New("role id cannot be empty")
)

// User represents an authenticated identity within the Cubit control plane.
type User struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	RoleID       string    `json:"roleId"`
	IsActive     bool      `json:"isActive"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// Role represents a collection of permissions granted to users or API tokens.
type Role struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	IsSystem    bool      `json:"isSystem"`
	Permissions []string  `json:"permissions"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// RefreshToken represents a long-lived rotation token for maintaining user sessions.
type RefreshToken struct {
	ID        string     `json:"id"`
	UserID    string     `json:"userId"`
	TokenHash string     `json:"-"`
	ExpiresAt time.Time  `json:"expiresAt"`
	RevokedAt *time.Time `json:"revokedAt,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
}

// APIToken represents a machine or programmatic bearer token for automated access.
type APIToken struct {
	ID         string     `json:"id"`
	UserID     string     `json:"userId"`
	Name       string     `json:"name"`
	TokenHash  string     `json:"-"`
	RoleID     string     `json:"roleId"`
	ExpiresAt  *time.Time `json:"expiresAt,omitempty"`
	LastUsedAt *time.Time `json:"lastUsedAt,omitempty"`
	CreatedAt  time.Time  `json:"createdAt"`
}

// NewUser creates and validates a new User domain model.
func NewUser(id, name, email, passwordHash, roleID string) (*User, error) {
	if strings.TrimSpace(id) == "" {
		return nil, ErrInvalidUserID
	}
	if strings.TrimSpace(email) == "" {
		return nil, ErrInvalidEmail
	}
	if strings.TrimSpace(roleID) == "" {
		return nil, ErrInvalidRoleID
	}

	now := time.Now().UTC()
	return &User{
		ID:           strings.TrimSpace(id),
		Name:         strings.TrimSpace(name),
		Email:        strings.ToLower(strings.TrimSpace(email)),
		PasswordHash: passwordHash,
		RoleID:       strings.TrimSpace(roleID),
		IsActive:     true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}

// HashPassword hashes a plaintext password using bcrypt with standard work factor 12.
func HashPassword(password string) (string, error) {
	if password == "" {
		return "", ErrEmptyPassword
	}
	if len([]byte(password)) > 72 {
		return "", ErrPasswordTooLong
	}

	bytes, err := bcrypt.GenerateFromPassword([]byte(password), BcryptDefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to generate bcrypt hash: %w", err)
	}
	return string(bytes), nil
}

// CheckPassword verifies a plaintext password against a stored bcrypt hash.
func CheckPassword(password, hash string) bool {
	if password == "" || hash == "" {
		return false
	}
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// MatchPermission checks if a granted permission pattern satisfies a required permission.
//
// Evaluation rules:
//   - "*" satisfies any required permission.
//   - An exact match (e.g. "apps:deploy" == "apps:deploy") satisfies the requirement.
//   - A prefix wildcard ending with ":*" (e.g. "apps:*" or "services:*") satisfies any permission
//     sharing the same prefix (e.g. "apps:read", "services:kv:read").
//   - A suffix wildcard starting with "*:" (e.g. "*:read") satisfies any permission
//     sharing the same suffix (e.g. "apps:read", "nodes:read", "services:kv:read").
func MatchPermission(pattern, required string) bool {
	if pattern == "" || required == "" {
		return false
	}
	if pattern == "*" {
		return true
	}
	if pattern == required {
		return true
	}
	if strings.HasSuffix(pattern, ":*") {
		prefix := pattern[:len(pattern)-2]
		if required == prefix || strings.HasPrefix(required, prefix+":") {
			return true
		}
	}
	if strings.HasPrefix(pattern, "*:") {
		suffix := pattern[2:]
		if required == suffix || strings.HasSuffix(required, ":"+suffix) {
			return true
		}
	}
	return false
}

// HasPermission checks whether any of the user's granted permissions satisfies the required permission.
func HasPermission(userPerms []string, required string) bool {
	if required == "" {
		return false
	}
	for _, perm := range userPerms {
		if MatchPermission(perm, required) {
			return true
		}
	}
	return false
}

// DefaultSystemRoles returns the initial pre-seeded system roles (Admin, Developer, Viewer).
func DefaultSystemRoles() []*Role {
	now := time.Now().UTC()
	return []*Role{
		{
			ID:          SystemRoleAdminID,
			Name:        SystemRoleAdminName,
			Description: "Root administrator with unrestricted permissions",
			IsSystem:    true,
			Permissions: []string{"*"},
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          SystemRoleDeveloperID,
			Name:        SystemRoleDeveloperName,
			Description: "Developer role with application, deployment, service, domain, and node inspection permissions",
			IsSystem:    true,
			Permissions: []string{"apps:*", "services:*", "deployments:*", "domains:*", "nodes:read"},
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          SystemRoleViewerID,
			Name:        SystemRoleViewerName,
			Description: "Read-only viewer with inspection access across platform resources",
			IsSystem:    true,
			Permissions: []string{"*:read", "apps:read", "services:read", "nodes:read", "domains:read"},
			CreatedAt:   now,
			UpdatedAt:   now,
		},
	}
}
