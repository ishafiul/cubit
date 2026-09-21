package github

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/ishaf/cubit/internal/domain"
)

// Repository defines the storage contract for GitHub App settings.
type Repository interface {
	GetSettings(ctx context.Context) (*domain.GitHubAppSettings, error)
	SaveSettings(ctx context.Context, settings *domain.GitHubAppSettings) error
	ClearSettings(ctx context.Context) error
}

// SQLiteRepository implements Repository using SQLite.
type SQLiteRepository struct {
	db *sql.DB
}

// NewRepository creates a new SQLite-backed GitHub settings repository.
func NewRepository(db *sql.DB) *SQLiteRepository {
	return &SQLiteRepository{db: db}
}

const defaultGitHubSettingsID = "default"

// GetSettings retrieves the current GitHub App settings or returns an unconfigured record.
func (r *SQLiteRepository) GetSettings(ctx context.Context) (*domain.GitHubAppSettings, error) {
	query := `
		SELECT id, app_id, app_name, client_id, client_secret, webhook_secret, private_key, installation_id, is_configured, created_at, updated_at
		FROM github_app_settings
		WHERE id = ?
		LIMIT 1;
	`
	var s domain.GitHubAppSettings
	var isConfInt int

	err := r.db.QueryRowContext(ctx, query, defaultGitHubSettingsID).Scan(
		&s.ID,
		&s.AppID,
		&s.AppName,
		&s.ClientID,
		&s.ClientSecret,
		&s.WebhookSecret,
		&s.PrivateKey,
		&s.InstallationID,
		&isConfInt,
		&s.CreatedAt,
		&s.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		now := time.Now().UTC()
		return &domain.GitHubAppSettings{
			ID:           defaultGitHubSettingsID,
			IsConfigured: false,
			CreatedAt:    now,
			UpdatedAt:    now,
		}, nil
	}
	if err != nil {
		return nil, err
	}
	s.IsConfigured = isConfInt == 1
	return &s, nil
}

// SaveSettings inserts or updates the GitHub App configuration.
func (r *SQLiteRepository) SaveSettings(ctx context.Context, s *domain.GitHubAppSettings) error {
	query := `
		INSERT INTO github_app_settings (
			id, app_id, app_name, client_id, client_secret, webhook_secret, private_key, installation_id, is_configured, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			app_id = excluded.app_id,
			app_name = excluded.app_name,
			client_id = excluded.client_id,
			client_secret = excluded.client_secret,
			webhook_secret = excluded.webhook_secret,
			private_key = excluded.private_key,
			installation_id = excluded.installation_id,
			is_configured = excluded.is_configured,
			updated_at = excluded.updated_at;
	`
	now := time.Now().UTC()
	if s.CreatedAt.IsZero() {
		s.CreatedAt = now
	}
	s.UpdatedAt = now
	if s.ID == "" {
		s.ID = defaultGitHubSettingsID
	}

	isConfInt := 0
	if s.IsConfigured {
		isConfInt = 1
	}

	_, err := r.db.ExecContext(ctx, query,
		s.ID,
		s.AppID,
		s.AppName,
		s.ClientID,
		s.ClientSecret,
		s.WebhookSecret,
		s.PrivateKey,
		s.InstallationID,
		isConfInt,
		s.CreatedAt.Format(time.RFC3339),
		s.UpdatedAt.Format(time.RFC3339),
	)
	return err
}

// ClearSettings resets the GitHub App configuration back to unconfigured state.
func (r *SQLiteRepository) ClearSettings(ctx context.Context) error {
	query := `
		UPDATE github_app_settings
		SET app_id = '',
		    app_name = '',
		    client_id = '',
		    client_secret = '',
		    webhook_secret = '',
		    private_key = '',
		    installation_id = '',
		    is_configured = 0,
		    updated_at = ?
		WHERE id = ?;
	`
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := r.db.ExecContext(ctx, query, now, defaultGitHubSettingsID)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return r.SaveSettings(ctx, &domain.GitHubAppSettings{
			ID:           defaultGitHubSettingsID,
			IsConfigured: false,
		})
	}
	return nil
}
