package auth

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/ishaf/cubit/internal/domain"
)

var (
	ErrUserNotFound         = errors.New("user not found")
	ErrRoleNotFound         = errors.New("role not found")
	ErrRefreshTokenNotFound = errors.New("refresh token not found")
)

// Repository defines data access operations for users, roles, and tokens.
type Repository interface {
	CreateUser(ctx context.Context, user *domain.User) error
	GetUserByID(ctx context.Context, id string) (*domain.User, error)
	GetUserByEmail(ctx context.Context, email string) (*domain.User, error)
	CountUsers(ctx context.Context) (int, error)

	CreateRole(ctx context.Context, role *domain.Role) error
	GetRoleByID(ctx context.Context, id string) (*domain.Role, error)

	CreateRefreshToken(ctx context.Context, token *domain.RefreshToken) error
	GetRefreshTokenByHash(ctx context.Context, tokenHash string) (*domain.RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, id string) error
}

type sqliteRepository struct {
	db *sql.DB
}

// NewRepository creates a new SQLite-backed auth repository.
func NewRepository(db *sql.DB) Repository {
	return &sqliteRepository{db: db}
}

func (r *sqliteRepository) CreateUser(ctx context.Context, user *domain.User) error {
	query := `
		INSERT INTO users (id, name, email, password_hash, role_id, is_active, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	isActiveInt := 0
	if user.IsActive {
		isActiveInt = 1
	}

	_, err := r.db.ExecContext(ctx, query,
		user.ID,
		user.Name,
		user.Email,
		user.PasswordHash,
		user.RoleID,
		isActiveInt,
		user.CreatedAt,
		user.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert user: %w", err)
	}
	return nil
}

func (r *sqliteRepository) GetUserByID(ctx context.Context, id string) (*domain.User, error) {
	query := `
		SELECT id, name, email, password_hash, role_id, is_active, created_at, updated_at
		FROM users
		WHERE id = ?
	`
	var user domain.User
	var isActiveInt int

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.PasswordHash,
		&user.RoleID,
		&isActiveInt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query user by id: %w", err)
	}

	user.IsActive = (isActiveInt == 1)
	return &user, nil
}

func (r *sqliteRepository) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `
		SELECT id, name, email, password_hash, role_id, is_active, created_at, updated_at
		FROM users
		WHERE email = ?
	`
	var user domain.User
	var isActiveInt int

	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.PasswordHash,
		&user.RoleID,
		&isActiveInt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query user by email: %w", err)
	}

	user.IsActive = (isActiveInt == 1)
	return &user, nil
}

func (r *sqliteRepository) CountUsers(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count users: %w", err)
	}
	return count, nil
}

func (r *sqliteRepository) CreateRole(ctx context.Context, role *domain.Role) error {
	permsJSON, err := json.Marshal(role.Permissions)
	if err != nil {
		return fmt.Errorf("failed to marshal permissions: %w", err)
	}

	query := `
		INSERT INTO roles (id, name, description, is_system, permissions, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	isSysInt := 0
	if role.IsSystem {
		isSysInt = 1
	}

	_, err = r.db.ExecContext(ctx, query,
		role.ID,
		role.Name,
		role.Description,
		isSysInt,
		string(permsJSON),
		role.CreatedAt,
		role.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert role: %w", err)
	}
	return nil
}

func (r *sqliteRepository) GetRoleByID(ctx context.Context, id string) (*domain.Role, error) {
	query := `
		SELECT id, name, description, is_system, permissions, created_at, updated_at
		FROM roles
		WHERE id = ?
	`
	var role domain.Role
	var isSysInt int
	var permsStr string

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&role.ID,
		&role.Name,
		&role.Description,
		&isSysInt,
		&permsStr,
		&role.CreatedAt,
		&role.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrRoleNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query role: %w", err)
	}

	role.IsSystem = (isSysInt == 1)
	if err := json.Unmarshal([]byte(permsStr), &role.Permissions); err != nil {
		return nil, fmt.Errorf("failed to parse role permissions json: %w", err)
	}
	return &role, nil
}

func (r *sqliteRepository) CreateRefreshToken(ctx context.Context, token *domain.RefreshToken) error {
	query := `
		INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at, revoked_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.ExecContext(ctx, query,
		token.ID,
		token.UserID,
		token.TokenHash,
		token.ExpiresAt,
		token.RevokedAt,
		token.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert refresh token: %w", err)
	}
	return nil
}

func (r *sqliteRepository) GetRefreshTokenByHash(ctx context.Context, tokenHash string) (*domain.RefreshToken, error) {
	query := `
		SELECT id, user_id, token_hash, expires_at, revoked_at, created_at
		FROM refresh_tokens
		WHERE token_hash = ?
	`
	var token domain.RefreshToken
	var revokedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, tokenHash).Scan(
		&token.ID,
		&token.UserID,
		&token.TokenHash,
		&token.ExpiresAt,
		&revokedAt,
		&token.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrRefreshTokenNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query refresh token: %w", err)
	}

	if revokedAt.Valid {
		t := revokedAt.Time
		token.RevokedAt = &t
	}
	return &token, nil
}

func (r *sqliteRepository) RevokeRefreshToken(ctx context.Context, id string) error {
	now := time.Now().UTC()
	query := `
		UPDATE refresh_tokens
		SET revoked_at = ?
		WHERE id = ? AND revoked_at IS NULL
	`
	_, err := r.db.ExecContext(ctx, query, now, id)
	if err != nil {
		return fmt.Errorf("failed to revoke refresh token: %w", err)
	}
	return nil
}
