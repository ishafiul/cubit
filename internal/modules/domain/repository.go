package domain

import (
	"context"
	"database/sql"
	"errors"
	"time"

	coreDomain "github.com/ishaf/cubit/internal/domain"
)

// Repository defines persistence operations for custom domain routes.
type Repository interface {
	Save(ctx context.Context, dom *coreDomain.Domain) error
	GetByID(ctx context.Context, id string) (*coreDomain.Domain, error)
	List(ctx context.Context) ([]*coreDomain.Domain, error)
	ListByAppID(ctx context.Context, appID string) ([]*coreDomain.Domain, error)
	Delete(ctx context.Context, id string) error
	Update(ctx context.Context, dom *coreDomain.Domain) error
}

// SQLiteRepository implements Repository using SQLite.
type SQLiteRepository struct {
	db *sql.DB
}

// NewRepository creates a new SQLite domain repository.
func NewRepository(db *sql.DB) *SQLiteRepository {
	return &SQLiteRepository{db: db}
}

// Save persists a new custom domain route.
func (r *SQLiteRepository) Save(ctx context.Context, dom *coreDomain.Domain) error {
	sslInt := 0
	if dom.SSLActive {
		sslInt = 1
	}

	query := `
		INSERT INTO domains (id, application_id, hostname, path_prefix, ssl_active, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.ExecContext(ctx, query,
		dom.ID, dom.ApplicationID, dom.Hostname, dom.PathPrefix, sslInt,
		dom.CreatedAt.Format(time.RFC3339), dom.UpdatedAt.Format(time.RFC3339),
	)
	return err
}

// GetByID retrieves a domain by ID.
func (r *SQLiteRepository) GetByID(ctx context.Context, id string) (*coreDomain.Domain, error) {
	query := `
		SELECT id, application_id, hostname, path_prefix, ssl_active, created_at, updated_at
		FROM domains WHERE id = ?
	`
	row := r.db.QueryRowContext(ctx, query, id)

	var (
		dom                        coreDomain.Domain
		sslInt                     int
		createdAtStr, updatedAtStr string
	)

	err := row.Scan(&dom.ID, &dom.ApplicationID, &dom.Hostname, &dom.PathPrefix, &sslInt, &createdAtStr, &updatedAtStr)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, coreDomain.NewNotFoundError("domain not found: " + id)
	}
	if err != nil {
		return nil, err
	}

	dom.SSLActive = sslInt == 1
	dom.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
	dom.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)
	return &dom, nil
}

// List returns all registered domains.
func (r *SQLiteRepository) List(ctx context.Context) ([]*coreDomain.Domain, error) {
	query := `
		SELECT id, application_id, hostname, path_prefix, ssl_active, created_at, updated_at
		FROM domains ORDER BY created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*coreDomain.Domain
	for rows.Next() {
		var (
			dom                        coreDomain.Domain
			sslInt                     int
			createdAtStr, updatedAtStr string
		)
		if err := rows.Scan(&dom.ID, &dom.ApplicationID, &dom.Hostname, &dom.PathPrefix, &sslInt, &createdAtStr, &updatedAtStr); err != nil {
			return nil, err
		}
		dom.SSLActive = sslInt == 1
		dom.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
		dom.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)
		list = append(list, &dom)
	}

	return list, rows.Err()
}

// ListByAppID returns all custom domains for a specific application.
func (r *SQLiteRepository) ListByAppID(ctx context.Context, appID string) ([]*coreDomain.Domain, error) {
	query := `
		SELECT id, application_id, hostname, path_prefix, ssl_active, created_at, updated_at
		FROM domains WHERE application_id = ? ORDER BY created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query, appID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*coreDomain.Domain
	for rows.Next() {
		var (
			dom                        coreDomain.Domain
			sslInt                     int
			createdAtStr, updatedAtStr string
		)
		if err := rows.Scan(&dom.ID, &dom.ApplicationID, &dom.Hostname, &dom.PathPrefix, &sslInt, &createdAtStr, &updatedAtStr); err != nil {
			return nil, err
		}
		dom.SSLActive = sslInt == 1
		dom.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
		dom.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)
		list = append(list, &dom)
	}

	return list, rows.Err()
}

// Delete removes a domain.
func (r *SQLiteRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM domains WHERE id = ?`
	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return coreDomain.NewNotFoundError("domain not found: " + id)
	}
	return nil
}

// Update updates a domain's SSL state.
func (r *SQLiteRepository) Update(ctx context.Context, dom *coreDomain.Domain) error {
	sslInt := 0
	if dom.SSLActive {
		sslInt = 1
	}

	query := `
		UPDATE domains
		SET hostname = ?, path_prefix = ?, ssl_active = ?, updated_at = ?
		WHERE id = ?
	`
	_, err := r.db.ExecContext(ctx, query,
		dom.Hostname, dom.PathPrefix, sslInt, time.Now().UTC().Format(time.RFC3339), dom.ID,
	)
	return err
}
