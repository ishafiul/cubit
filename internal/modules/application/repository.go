package application

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/ishaf/cubit/internal/domain"
)

// Repository defines data access operations for applications.
type Repository interface {
	Save(ctx context.Context, app *domain.Application) error
	Update(ctx context.Context, app *domain.Application) error
	GetByID(ctx context.Context, id string) (*domain.Application, error)
	GetBySubdomain(ctx context.Context, subdomain string) (*domain.Application, error)
	List(ctx context.Context) ([]*domain.Application, error)
	Delete(ctx context.Context, id string) error
	ListByGitRepoAndBranch(ctx context.Context, repo, branch string) ([]*domain.Application, error)
}

// SQLiteRepository implements Repository using SQLite.
type SQLiteRepository struct {
	db *sql.DB
}

// NewRepository creates a new SQLite application repository.
func NewRepository(db *sql.DB) *SQLiteRepository {
	return &SQLiteRepository{db: db}
}

// Save persists an application entity to SQLite.
func (r *SQLiteRepository) Save(ctx context.Context, app *domain.Application) error {
	envVarsJSON, _ := json.Marshal(app.EnvVars)
	bindingsJSON, _ := json.Marshal(app.Bindings)
	compatFlagsJSON, _ := json.Marshal(app.CompatibilityFlags)

	sourceType := string(app.SourceType)
	if sourceType == "" {
		sourceType = string(domain.SourceTypeGit)
	}
	subdomain := app.Subdomain
	if subdomain == "" {
		subdomain = domain.SanitizeSubdomain(app.Name)
	}

	autoDeployInt := 0
	if app.AutoDeploy {
		autoDeployInt = 1
	}

	compatDate := app.CompatibilityDate
	if compatDate == "" {
		compatDate = "2024-09-23"
	}
	memLimit := app.MemoryLimitMB
	if memLimit <= 0 {
		memLimit = 128
	}
	maxDuration := app.MaxDurationMs
	if maxDuration <= 0 {
		maxDuration = 50
	}

	query := `
		INSERT INTO applications (id, name, source_type, subdomain, git_repo, branch, inline_code, auto_deploy, status, env_vars, bindings, active_deployment_id, compatibility_date, compatibility_flags, memory_limit_mb, max_duration_ms, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.ExecContext(ctx, query,
		app.ID, app.Name, sourceType, subdomain, app.GitRepo, app.Branch, app.InlineCode, autoDeployInt, string(app.Status),
		string(envVarsJSON), string(bindingsJSON), app.ActiveDeploymentID,
		compatDate, string(compatFlagsJSON), memLimit, maxDuration,
		app.CreatedAt.Format(time.RFC3339), app.UpdatedAt.Format(time.RFC3339),
	)
	return err
}

// GetByID retrieves an application by its ID.
func (r *SQLiteRepository) GetByID(ctx context.Context, id string) (*domain.Application, error) {
	query := `
		SELECT id, name, source_type, subdomain, git_repo, branch, inline_code, COALESCE(auto_deploy, 1), status, env_vars, bindings, active_deployment_id,
		       COALESCE(compatibility_date, '2024-09-23'), COALESCE(compatibility_flags, '[]'), COALESCE(memory_limit_mb, 128), COALESCE(max_duration_ms, 50),
		       created_at, updated_at
		FROM applications WHERE id = ?
	`
	row := r.db.QueryRowContext(ctx, query, id)

	var (
		app                                                                domain.Application
		sourceTypeStr, subdomainStr, statusStr, envJSON, bindingsJSON, createdAtStr, updatedAtStr string
		compatDateStr, compatFlagsJSON                                     string
		gitRepoNull, inlineCodeNull, activeDepID                           sql.NullString
		autoDeployInt, memLimitInt, maxDurationInt                         int
	)

	err := row.Scan(
		&app.ID, &app.Name, &sourceTypeStr, &subdomainStr, &gitRepoNull, &app.Branch, &inlineCodeNull, &autoDeployInt, &statusStr,
		&envJSON, &bindingsJSON, &activeDepID,
		&compatDateStr, &compatFlagsJSON, &memLimitInt, &maxDurationInt,
		&createdAtStr, &updatedAtStr,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.NewNotFoundError("application not found: " + id)
	}
	if err != nil {
		return nil, err
	}

	app.SourceType = domain.SourceType(sourceTypeStr)
	app.Subdomain = subdomainStr
	if gitRepoNull.Valid {
		app.GitRepo = gitRepoNull.String
	}
	if inlineCodeNull.Valid {
		app.InlineCode = inlineCodeNull.String
	}
	app.AutoDeploy = autoDeployInt == 1
	app.Status = domain.ApplicationStatus(statusStr)
	if activeDepID.Valid {
		app.ActiveDeploymentID = activeDepID.String
	}
	_ = json.Unmarshal([]byte(envJSON), &app.EnvVars)
	_ = json.Unmarshal([]byte(bindingsJSON), &app.Bindings)
	app.CompatibilityDate = compatDateStr
	_ = json.Unmarshal([]byte(compatFlagsJSON), &app.CompatibilityFlags)
	app.MemoryLimitMB = memLimitInt
	app.MaxDurationMs = maxDurationInt
	app.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
	app.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)

	return &app, nil
}

// GetBySubdomain retrieves an application by its subdomain or name.
func (r *SQLiteRepository) GetBySubdomain(ctx context.Context, subdomain string) (*domain.Application, error) {
	query := `
		SELECT id, name, source_type, subdomain, git_repo, branch, inline_code, COALESCE(auto_deploy, 1), status, env_vars, bindings, active_deployment_id,
		       COALESCE(compatibility_date, '2024-09-23'), COALESCE(compatibility_flags, '[]'), COALESCE(memory_limit_mb, 128), COALESCE(max_duration_ms, 50),
		       created_at, updated_at
		FROM applications WHERE subdomain = ? OR name = ?
	`
	row := r.db.QueryRowContext(ctx, query, subdomain, subdomain)

	var (
		app                                                                domain.Application
		sourceTypeStr, subdomainStr, statusStr, envJSON, bindingsJSON, createdAtStr, updatedAtStr string
		compatDateStr, compatFlagsJSON                                     string
		gitRepoNull, inlineCodeNull, activeDepID                           sql.NullString
		autoDeployInt, memLimitInt, maxDurationInt                         int
	)

	err := row.Scan(
		&app.ID, &app.Name, &sourceTypeStr, &subdomainStr, &gitRepoNull, &app.Branch, &inlineCodeNull, &autoDeployInt, &statusStr,
		&envJSON, &bindingsJSON, &activeDepID,
		&compatDateStr, &compatFlagsJSON, &memLimitInt, &maxDurationInt,
		&createdAtStr, &updatedAtStr,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.NewNotFoundError("application not found: " + subdomain)
	}
	if err != nil {
		return nil, err
	}

	app.SourceType = domain.SourceType(sourceTypeStr)
	app.Subdomain = subdomainStr
	if gitRepoNull.Valid {
		app.GitRepo = gitRepoNull.String
	}
	if inlineCodeNull.Valid {
		app.InlineCode = inlineCodeNull.String
	}
	app.AutoDeploy = autoDeployInt == 1
	app.Status = domain.ApplicationStatus(statusStr)
	if activeDepID.Valid {
		app.ActiveDeploymentID = activeDepID.String
	}
	_ = json.Unmarshal([]byte(envJSON), &app.EnvVars)
	_ = json.Unmarshal([]byte(bindingsJSON), &app.Bindings)
	app.CompatibilityDate = compatDateStr
	_ = json.Unmarshal([]byte(compatFlagsJSON), &app.CompatibilityFlags)
	app.MemoryLimitMB = memLimitInt
	app.MaxDurationMs = maxDurationInt
	app.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
	app.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)

	return &app, nil
}

// List retrieves all registered applications.
func (r *SQLiteRepository) List(ctx context.Context) ([]*domain.Application, error) {
	query := `
		SELECT id, name, source_type, subdomain, git_repo, branch, inline_code, COALESCE(auto_deploy, 1), status, env_vars, bindings, active_deployment_id,
		       COALESCE(compatibility_date, '2024-09-23'), COALESCE(compatibility_flags, '[]'), COALESCE(memory_limit_mb, 128), COALESCE(max_duration_ms, 50),
		       created_at, updated_at
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
			sourceTypeStr, subdomainStr, statusStr, envJSON, bindingsJSON, createdAtStr, updatedAtStr string
			compatDateStr, compatFlagsJSON                                     string
			gitRepoNull, inlineCodeNull, activeDepID                           sql.NullString
			autoDeployInt, memLimitInt, maxDurationInt                         int
		)

		err := rows.Scan(
			&app.ID, &app.Name, &sourceTypeStr, &subdomainStr, &gitRepoNull, &app.Branch, &inlineCodeNull, &autoDeployInt, &statusStr,
			&envJSON, &bindingsJSON, &activeDepID,
			&compatDateStr, &compatFlagsJSON, &memLimitInt, &maxDurationInt,
			&createdAtStr, &updatedAtStr,
		)
		if err != nil {
			return nil, err
		}

		app.SourceType = domain.SourceType(sourceTypeStr)
		app.Subdomain = subdomainStr
		if gitRepoNull.Valid {
			app.GitRepo = gitRepoNull.String
		}
		if inlineCodeNull.Valid {
			app.InlineCode = inlineCodeNull.String
		}
		app.AutoDeploy = autoDeployInt == 1
		app.Status = domain.ApplicationStatus(statusStr)
		if activeDepID.Valid {
			app.ActiveDeploymentID = activeDepID.String
		}
		_ = json.Unmarshal([]byte(envJSON), &app.EnvVars)
		_ = json.Unmarshal([]byte(bindingsJSON), &app.Bindings)
		app.CompatibilityDate = compatDateStr
		_ = json.Unmarshal([]byte(compatFlagsJSON), &app.CompatibilityFlags)
		app.MemoryLimitMB = memLimitInt
		app.MaxDurationMs = maxDurationInt
		app.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
		app.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)

		apps = append(apps, &app)
	}

	return apps, nil
}

// ListByGitRepoAndBranch finds all applications connected to a given repository and branch.
func (r *SQLiteRepository) ListByGitRepoAndBranch(ctx context.Context, repo, branch string) ([]*domain.Application, error) {
	allApps, err := r.List(ctx)
	if err != nil {
		return nil, err
	}

	normTarget := normalizeGitRepo(repo)
	var matched []*domain.Application

	for _, a := range allApps {
		if normalizeGitRepo(a.GitRepo) == normTarget && a.Branch == branch {
			matched = append(matched, a)
		}
	}

	return matched, nil
}

func normalizeGitRepo(url string) string {
	u := strings.TrimSpace(url)
	u = strings.TrimSuffix(u, ".git")
	u = strings.TrimPrefix(u, "https://github.com/")
	u = strings.TrimPrefix(u, "http://github.com/")
	u = strings.TrimPrefix(u, "git@github.com:")
	return strings.ToLower(strings.Trim(u, "/"))
}

// Update persists changes to an application.
func (r *SQLiteRepository) Update(ctx context.Context, app *domain.Application) error {
	envVarsJSON, _ := json.Marshal(app.EnvVars)
	bindingsJSON, _ := json.Marshal(app.Bindings)
	compatFlagsJSON, _ := json.Marshal(app.CompatibilityFlags)

	sourceType := string(app.SourceType)
	if sourceType == "" {
		sourceType = string(domain.SourceTypeGit)
	}
	subdomain := app.Subdomain
	if subdomain == "" {
		subdomain = domain.SanitizeSubdomain(app.Name)
	}

	autoDeployInt := 0
	if app.AutoDeploy {
		autoDeployInt = 1
	}

	compatDate := app.CompatibilityDate
	if compatDate == "" {
		compatDate = "2024-09-23"
	}
	memLimit := app.MemoryLimitMB
	if memLimit <= 0 {
		memLimit = 128
	}
	maxDuration := app.MaxDurationMs
	if maxDuration <= 0 {
		maxDuration = 50
	}

	query := `
		UPDATE applications
		SET name = ?, source_type = ?, subdomain = ?, git_repo = ?, branch = ?, inline_code = ?, auto_deploy = ?, status = ?,
		    env_vars = ?, bindings = ?, active_deployment_id = ?, compatibility_date = ?, compatibility_flags = ?, memory_limit_mb = ?, max_duration_ms = ?, updated_at = ?
		WHERE id = ?
	`
	res, err := r.db.ExecContext(ctx, query,
		app.Name, sourceType, subdomain, app.GitRepo, app.Branch, app.InlineCode, autoDeployInt, string(app.Status),
		string(envVarsJSON), string(bindingsJSON), app.ActiveDeploymentID,
		compatDate, string(compatFlagsJSON), memLimit, maxDuration,
		app.UpdatedAt.Format(time.RFC3339), app.ID,
	)
	if err != nil {
		return err
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return domain.NewNotFoundError("application not found: " + app.ID)
	}

	return nil
}

// Delete removes an application.
func (r *SQLiteRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM applications WHERE id = ?`
	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return domain.NewNotFoundError("application not found: " + id)
	}
	return nil
}
