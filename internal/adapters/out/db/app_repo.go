package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/ishaf/cubit/internal/domain"
	"github.com/ishaf/cubit/internal/usecase"
)

var _ usecase.ApplicationRepository = (*AppRepo)(nil)

// AppRepo implements usecase.ApplicationRepository using SQLite.
type AppRepo struct {
	db *sql.DB
}

// NewAppRepo creates a new AppRepo.
func NewAppRepo(db *sql.DB) *AppRepo {
	return &AppRepo{db: db}
}

// Save persists an application.
func (r *AppRepo) Save(ctx context.Context, app *domain.Application) error {
	envVarsJSON, _ := json.Marshal(app.EnvVars)
	bindingsJSON, _ := json.Marshal(app.Bindings)

	sourceType := string(app.SourceType)
	if sourceType == "" {
		sourceType = string(domain.SourceTypeGit)
	}

	query := `
		INSERT INTO applications (id, name, source_type, git_repo, branch, inline_code, status, env_vars, bindings, active_deployment_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.ExecContext(ctx, query,
		app.ID, app.Name, sourceType, app.GitRepo, app.Branch, app.InlineCode, string(app.Status),
		string(envVarsJSON), string(bindingsJSON), app.ActiveDeploymentID,
		app.CreatedAt.Format(time.RFC3339), app.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return err
	}
	return nil
}

// GetByID retrieves an application by ID.
func (r *AppRepo) GetByID(ctx context.Context, id string) (*domain.Application, error) {
	query := `
		SELECT id, name, source_type, git_repo, branch, inline_code, status, env_vars, bindings, active_deployment_id, created_at, updated_at
		FROM applications WHERE id = ?
	`
	row := r.db.QueryRowContext(ctx, query, id)

	var (
		app                                                                domain.Application
		sourceTypeStr, statusStr, envJSON, bindingsJSON, createdAtStr, updatedAtStr string
		gitRepoNull, inlineCodeNull, activeDepID                           sql.NullString
	)

	err := row.Scan(
		&app.ID, &app.Name, &sourceTypeStr, &gitRepoNull, &app.Branch, &inlineCodeNull, &statusStr,
		&envJSON, &bindingsJSON, &activeDepID, &createdAtStr, &updatedAtStr,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.NewNotFoundError("application not found")
	}
	if err != nil {
		return nil, err
	}

	app.SourceType = domain.SourceType(sourceTypeStr)
	if app.SourceType == "" {
		app.SourceType = domain.SourceTypeGit
	}
	if gitRepoNull.Valid {
		app.GitRepo = gitRepoNull.String
	}
	if inlineCodeNull.Valid {
		app.InlineCode = inlineCodeNull.String
	}
	app.Status = domain.ApplicationStatus(statusStr)
	if activeDepID.Valid {
		app.ActiveDeploymentID = activeDepID.String
	}
	_ = json.Unmarshal([]byte(envJSON), &app.EnvVars)
	_ = json.Unmarshal([]byte(bindingsJSON), &app.Bindings)
	app.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
	app.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)
	return &app, nil
}

// List returns all registered applications.
func (r *AppRepo) List(ctx context.Context) ([]*domain.Application, error) {
	query := `
		SELECT id, name, source_type, git_repo, branch, inline_code, status, env_vars, bindings, active_deployment_id, created_at, updated_at
		FROM applications ORDER BY created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var apps []*domain.Application
	for rows.Next() {
		var (
			app                                                                domain.Application
			sourceTypeStr, statusStr, envJSON, bindingsJSON, createdAtStr, updatedAtStr string
			gitRepoNull, inlineCodeNull, activeDepID                           sql.NullString
		)
		if err := rows.Scan(
			&app.ID, &app.Name, &sourceTypeStr, &gitRepoNull, &app.Branch, &inlineCodeNull, &statusStr,
			&envJSON, &bindingsJSON, &activeDepID, &createdAtStr, &updatedAtStr,
		); err != nil {
			return nil, err
		}
		app.SourceType = domain.SourceType(sourceTypeStr)
		if app.SourceType == "" {
			app.SourceType = domain.SourceTypeGit
		}
		if gitRepoNull.Valid {
			app.GitRepo = gitRepoNull.String
		}
		if inlineCodeNull.Valid {
			app.InlineCode = inlineCodeNull.String
		}
		app.Status = domain.ApplicationStatus(statusStr)
		if activeDepID.Valid {
			app.ActiveDeploymentID = activeDepID.String
		}
		_ = json.Unmarshal([]byte(envJSON), &app.EnvVars)
		_ = json.Unmarshal([]byte(bindingsJSON), &app.Bindings)
		app.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
		app.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)
		apps = append(apps, &app)
	}
	return apps, rows.Err()
}

// Delete removes an application by ID.
func (r *AppRepo) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM applications WHERE id = ?`
	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return domain.NewNotFoundError("application not found")
	}
	return nil
}

// Update updates an application.
func (r *AppRepo) Update(ctx context.Context, app *domain.Application) error {
	envVarsJSON, _ := json.Marshal(app.EnvVars)
	bindingsJSON, _ := json.Marshal(app.Bindings)

	sourceType := string(app.SourceType)
	if sourceType == "" {
		sourceType = string(domain.SourceTypeGit)
	}

	query := `
		UPDATE applications
		SET name = ?, source_type = ?, git_repo = ?, branch = ?, inline_code = ?, status = ?, env_vars = ?, bindings = ?, active_deployment_id = ?, updated_at = ?
		WHERE id = ?
	`
	res, err := r.db.ExecContext(ctx, query,
		app.Name, sourceType, app.GitRepo, app.Branch, app.InlineCode, string(app.Status),
		string(envVarsJSON), string(bindingsJSON), app.ActiveDeploymentID,
		time.Now().UTC().Format(time.RFC3339), app.ID,
	)
	if err != nil {
		return err
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return domain.NewNotFoundError("application not found")
	}
	return nil
}
