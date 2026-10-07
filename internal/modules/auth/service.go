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
	ErrInvalidRefreshToken = errors.New("invalid or expired refresh token")
	ErrTokenRevoked        = errors.New("refresh token has been revoked")
	ErrUserInactive        = errors.New("user account is inactive")
	ErrInvalidCredentials  = errors.New("invalid email or password")
)

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
	IssueTokenPair(ctx context.Context, user *domain.User) (*TokenPair, error)
	RotateRefreshToken(ctx context.Context, refreshTokenPlain string) (*TokenPair, *domain.User, error)
	RevokeToken(ctx context.Context, refreshTokenPlain string) error
	ValidateAccessToken(tokenStr string) (*domain.JWTClaims, error)
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
