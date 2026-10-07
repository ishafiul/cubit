package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ishaf/cubit/internal/domain"
)

const (
	DefaultAccessTokenTTL  = 15 * time.Minute
	DefaultRefreshTokenTTL = 7 * 24 * time.Hour
	RefreshTokenPrefix     = "cbt_ref_"
)

var (
	ErrInvalidRefreshToken   = errors.New("invalid or expired refresh token")
	ErrTokenRevoked          = errors.New("refresh token has been revoked")
	ErrUserInactive          = errors.New("user account is inactive")
	ErrInvalidCredentials    = errors.New("invalid email or password")
	ErrSetupAlreadyCompleted = errors.New("setup has already been completed")
	ErrRoleAlreadyExists     = errors.New("role with this name already exists")
	ErrCannotModifySystemRole = errors.New("cannot modify system roles")
	ErrCannotDeleteSystemRole = errors.New("cannot delete system roles")
	ErrInvalidRoleName       = errors.New("role name cannot be empty")
)

// StatusResponse contains the cluster initialization state and current release version.
type StatusResponse struct {
	Initialized bool   `json:"initialized"`
	Version     string `json:"version"`
}

// TokenPair encapsulates the dual token response returned upon authentication or rotation.
type TokenPair struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    int64  `json:"expiresIn"`
}

// Service defines business logic operations for the authentication & session engine.
type Service interface {
	Login(ctx context.Context, email, password string) (*TokenPair, *domain.User, error)
	Refresh(ctx context.Context, refreshTokenPlain string) (*TokenPair, *domain.User, error)
	Logout(ctx context.Context, refreshTokenPlain string) error
	GetStatus(ctx context.Context) (*StatusResponse, error)
	Setup(ctx context.Context, name, email, password string) (*TokenPair, *domain.User, error)
	AutoProvisionAdmin(ctx context.Context, email, password string) (*domain.User, error)
	IssueTokenPair(ctx context.Context, user *domain.User) (*TokenPair, error)
	RotateRefreshToken(ctx context.Context, refreshTokenPlain string) (*TokenPair, *domain.User, error)
	RevokeToken(ctx context.Context, refreshTokenPlain string) error
	ValidateAccessToken(tokenStr string) (*domain.JWTClaims, error)

	ListPermissions() []string
	ListRoles(ctx context.Context) ([]*domain.Role, error)
	GetRole(ctx context.Context, id string) (*domain.Role, error)
	CreateRole(ctx context.Context, name, description string, permissions []string) (*domain.Role, error)
	UpdateRole(ctx context.Context, id, name, description string, permissions []string) (*domain.Role, error)
	DeleteRole(ctx context.Context, id string) error
}

type authService struct {
	repo       Repository
	jwtSecret  string
	accessTTL  time.Duration
	refreshTTL time.Duration
}

// NewService creates a new auth service instance.
func NewService(repo Repository, jwtSecret string) Service {
	return &authService{
		repo:       repo,
		jwtSecret:  jwtSecret,
		accessTTL:  DefaultAccessTokenTTL,
		refreshTTL: DefaultRefreshTokenTTL,
	}
}

// IssueTokenPair generates a fresh JWT access token and a rotating refresh token for the user.
func (s *authService) IssueTokenPair(ctx context.Context, user *domain.User) (*TokenPair, error) {
	if !user.IsActive {
		return nil, ErrUserInactive
	}

	role, err := s.repo.GetRoleByID(ctx, user.RoleID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve user role %s: %w", user.RoleID, err)
	}

	claims := domain.JWTClaims{
		UserID:      user.ID,
		Email:       user.Email,
		RoleID:      user.RoleID,
		Permissions: role.Permissions,
	}

	accessToken, err := domain.GenerateJWT(claims, s.jwtSecret, s.accessTTL)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshTokenPlain, err := domain.GenerateSecureToken(RefreshTokenPrefix)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	tokenHash := domain.HashToken(refreshTokenPlain)
	now := time.Now().UTC()
	rt := &domain.RefreshToken{
		ID:        uuid.New().String(),
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: now.Add(s.refreshTTL),
		CreatedAt: now,
	}

	if err := s.repo.CreateRefreshToken(ctx, rt); err != nil {
		return nil, fmt.Errorf("failed to persist refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshTokenPlain,
		ExpiresIn:    int64(s.accessTTL.Seconds()),
	}, nil
}

// RotateRefreshToken validates the refresh token, revokes it, and issues a fresh token pair.
func (s *authService) RotateRefreshToken(ctx context.Context, refreshTokenPlain string) (*TokenPair, *domain.User, error) {
	if refreshTokenPlain == "" {
		return nil, nil, ErrInvalidRefreshToken
	}

	tokenHash := domain.HashToken(refreshTokenPlain)
	rt, err := s.repo.GetRefreshTokenByHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, ErrRefreshTokenNotFound) {
			return nil, nil, ErrInvalidRefreshToken
		}
		return nil, nil, fmt.Errorf("failed to query refresh token: %w", err)
	}

	// Verify not revoked
	if rt.RevokedAt != nil {
		return nil, nil, ErrTokenRevoked
	}

	// Verify not expired
	if time.Now().UTC().After(rt.ExpiresAt) {
		return nil, nil, ErrInvalidRefreshToken
	}

	// Immediately revoke old token to prevent replay attacks (Rotation)
	if err := s.repo.RevokeRefreshToken(ctx, rt.ID); err != nil {
		return nil, nil, fmt.Errorf("failed to revoke consumed refresh token: %w", err)
	}

	user, err := s.repo.GetUserByID(ctx, rt.UserID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to retrieve user for token: %w", err)
	}

	if !user.IsActive {
		return nil, nil, ErrUserInactive
	}

	pair, err := s.IssueTokenPair(ctx, user)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to issue rotated token pair: %w", err)
	}

	return pair, user, nil
}

// RevokeToken revokes an active refresh token.
func (s *authService) RevokeToken(ctx context.Context, refreshTokenPlain string) error {
	if refreshTokenPlain == "" {
		return nil
	}
	tokenHash := domain.HashToken(refreshTokenPlain)
	rt, err := s.repo.GetRefreshTokenByHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, ErrRefreshTokenNotFound) {
			return nil
		}
		return fmt.Errorf("failed to find token for revocation: %w", err)
	}

	return s.repo.RevokeRefreshToken(ctx, rt.ID)
}

// ValidateAccessToken validates the HMAC-SHA256 signature and expiration of an access token.
func (s *authService) ValidateAccessToken(tokenStr string) (*domain.JWTClaims, error) {
	return domain.ValidateJWT(tokenStr, s.jwtSecret)
}

// Login verifies user credentials and issues an active token pair.
func (s *authService) Login(ctx context.Context, email, password string) (*TokenPair, *domain.User, error) {
	cleanEmail := strings.ToLower(strings.TrimSpace(email))
	if cleanEmail == "" || password == "" {
		return nil, nil, ErrInvalidCredentials
	}

	user, err := s.repo.GetUserByEmail(ctx, cleanEmail)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, nil, ErrInvalidCredentials
		}
		return nil, nil, fmt.Errorf("failed to query user for login: %w", err)
	}

	if !user.IsActive {
		return nil, nil, ErrUserInactive
	}

	if !domain.CheckPassword(password, user.PasswordHash) {
		return nil, nil, ErrInvalidCredentials
	}

	pair, err := s.IssueTokenPair(ctx, user)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to issue tokens: %w", err)
	}

	return pair, user, nil
}

// Refresh rotates the refresh token and returns a fresh token pair.
func (s *authService) Refresh(ctx context.Context, refreshTokenPlain string) (*TokenPair, *domain.User, error) {
	return s.RotateRefreshToken(ctx, refreshTokenPlain)
}

// Logout revokes the provided refresh token session.
func (s *authService) Logout(ctx context.Context, refreshTokenPlain string) error {
	return s.RevokeToken(ctx, refreshTokenPlain)
}

// GetStatus checks whether the cluster has been initialized.
func (s *authService) GetStatus(ctx context.Context) (*StatusResponse, error) {
	count, err := s.repo.CountUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count users for status probe: %w", err)
	}
	return &StatusResponse{
		Initialized: count > 0,
		Version:     domain.CubitVersion,
	}, nil
}

// Setup performs one-time initialization of the root administrator account.
func (s *authService) Setup(ctx context.Context, name, email, password string) (*TokenPair, *domain.User, error) {
	cleanName := strings.TrimSpace(name)
	cleanEmail := strings.ToLower(strings.TrimSpace(email))
	if cleanName == "" || cleanEmail == "" || password == "" {
		return nil, nil, errors.New("name, email, and password cannot be empty")
	}

	count, err := s.repo.CountUsers(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to check existing user count: %w", err)
	}
	if count > 0 {
		return nil, nil, ErrSetupAlreadyCompleted
	}

	// Ensure built-in admin role exists
	adminRole, err := s.repo.GetRoleByID(ctx, domain.SystemRoleAdminID)
	if err != nil {
		if errors.Is(err, ErrRoleNotFound) {
			now := time.Now().UTC()
			adminRole = &domain.Role{
				ID:          domain.SystemRoleAdminID,
				Name:        domain.SystemRoleAdminName,
				Description: "Root administrator with unrestricted permissions",
				IsSystem:    true,
				Permissions: []string{"*"},
				CreatedAt:   now,
				UpdatedAt:   now,
			}
			if err := s.repo.CreateRole(ctx, adminRole); err != nil {
				return nil, nil, fmt.Errorf("failed to create admin role: %w", err)
			}
		} else {
			return nil, nil, fmt.Errorf("failed to query admin role: %w", err)
		}
	}

	passHash, err := domain.HashPassword(password)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to hash password: %w", err)
	}

	userID := uuid.New().String()
	user, err := domain.NewUser(userID, cleanName, cleanEmail, passHash, adminRole.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize admin user: %w", err)
	}

	if err := s.repo.CreateUser(ctx, user); err != nil {
		return nil, nil, fmt.Errorf("failed to persist admin user: %w", err)
	}

	pair, err := s.IssueTokenPair(ctx, user)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to issue tokens for setup: %w", err)
	}

	return pair, user, nil
}

// AutoProvisionAdmin automatically creates the root administrator account if the cluster is uninitialized.
// If either email or password is empty, or if the cluster already has users, it safely returns nil without error.
func (s *authService) AutoProvisionAdmin(ctx context.Context, email, password string) (*domain.User, error) {
	cleanEmail := strings.ToLower(strings.TrimSpace(email))
	cleanPassword := strings.TrimSpace(password)
	if cleanEmail == "" || cleanPassword == "" {
		return nil, nil
	}

	count, err := s.repo.CountUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count users for auto-provisioning: %w", err)
	}
	if count > 0 {
		return nil, nil
	}

	_, user, err := s.Setup(ctx, "Administrator", cleanEmail, cleanPassword)
	if err != nil {
		if errors.Is(err, ErrSetupAlreadyCompleted) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to auto-provision root admin: %w", err)
	}

	return user, nil
}

// ListPermissions returns the comprehensive catalog of platform permissions.
func (s *authService) ListPermissions() []string {
	return domain.AllPermissions()
}

// ListRoles retrieves all configured roles (both system and custom).
func (s *authService) ListRoles(ctx context.Context) ([]*domain.Role, error) {
	return s.repo.ListRoles(ctx)
}

// GetRole retrieves a single role by its unique identifier.
func (s *authService) GetRole(ctx context.Context, id string) (*domain.Role, error) {
	if strings.TrimSpace(id) == "" {
		return nil, ErrRoleNotFound
	}
	return s.repo.GetRoleByID(ctx, strings.TrimSpace(id))
}

// CreateRole creates a new custom role.
func (s *authService) CreateRole(ctx context.Context, name, description string, permissions []string) (*domain.Role, error) {
	cleanName := strings.TrimSpace(name)
	if cleanName == "" {
		return nil, ErrInvalidRoleName
	}

	existing, err := s.repo.GetRoleByName(ctx, cleanName)
	if err == nil && existing != nil {
		return nil, ErrRoleAlreadyExists
	}
	if err != nil && !errors.Is(err, ErrRoleNotFound) {
		return nil, fmt.Errorf("failed to check existing role: %w", err)
	}

	if permissions == nil {
		permissions = []string{}
	}

	now := time.Now().UTC()
	role := &domain.Role{
		ID:          uuid.New().String(),
		Name:        cleanName,
		Description: strings.TrimSpace(description),
		IsSystem:    false,
		Permissions: permissions,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.repo.CreateRole(ctx, role); err != nil {
		return nil, fmt.Errorf("failed to persist role: %w", err)
	}

	return role, nil
}

// UpdateRole updates a custom role. Built-in system roles cannot be modified.
func (s *authService) UpdateRole(ctx context.Context, id, name, description string, permissions []string) (*domain.Role, error) {
	cleanID := strings.TrimSpace(id)
	cleanName := strings.TrimSpace(name)
	if cleanID == "" {
		return nil, ErrRoleNotFound
	}
	if cleanName == "" {
		return nil, ErrInvalidRoleName
	}

	role, err := s.repo.GetRoleByID(ctx, cleanID)
	if err != nil {
		return nil, err
	}

	if role.IsSystem {
		return nil, ErrCannotModifySystemRole
	}

	if !strings.EqualFold(role.Name, cleanName) {
		existing, err := s.repo.GetRoleByName(ctx, cleanName)
		if err == nil && existing != nil && existing.ID != cleanID {
			return nil, ErrRoleAlreadyExists
		}
		if err != nil && !errors.Is(err, ErrRoleNotFound) {
			return nil, fmt.Errorf("failed to check existing role: %w", err)
		}
	}

	if permissions == nil {
		permissions = []string{}
	}

	role.Name = cleanName
	role.Description = strings.TrimSpace(description)
	role.Permissions = permissions
	role.UpdatedAt = time.Now().UTC()

	if err := s.repo.UpdateRole(ctx, role); err != nil {
		return nil, fmt.Errorf("failed to update role: %w", err)
	}

	return role, nil
}

// DeleteRole removes a custom role. Built-in system roles cannot be deleted.
func (s *authService) DeleteRole(ctx context.Context, id string) error {
	cleanID := strings.TrimSpace(id)
	if cleanID == "" {
		return ErrRoleNotFound
	}

	role, err := s.repo.GetRoleByID(ctx, cleanID)
	if err != nil {
		return err
	}

	if role.IsSystem {
		return ErrCannotDeleteSystemRole
	}

	return s.repo.DeleteRole(ctx, cleanID)
}

