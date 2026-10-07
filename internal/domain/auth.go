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
	SystemRoleAdminID   = "admin"
	SystemRoleAdminName = "Admin"
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
