package application

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
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

const selectApplicationColumns = `
	id, name, source_type, subdomain, git_repo, branch, COALESCE(root_dir, ''), inline_code, COALESCE(auto_deploy, 1), status, env_vars, bindings, COALESCE(migrations, '[]'), active_deployment_id,
	COALESCE(compatibility_date, '2024-09-23'), COALESCE(compatibility_flags, '[]'), COALESCE(memory_limit_mb, 128), COALESCE(max_duration_ms, 50), COALESCE(worker_port, 0),
	created_at, updated_at
`

type rowScanner interface {
	Scan(dest ...any) error
}

func scanApplicationRow(s rowScanner) (*domain.Application, error) {
	var (
		app                                                                           domain.Application
		sourceTypeStr, subdomainStr, statusStr, envJSON, bindingsJSON, migrationsJSON string
		createdAtStr, updatedAtStr                                                    string
		compatDateStr, compatFlagsJSON                                                string
		gitRepoNull, inlineCodeNull, activeDepID                                      sql.NullString
		autoDeployInt, memLimitInt, maxDurationInt, workerPortInt                      int
	)

	err := s.Scan(
		&app.ID, &app.Name, &sourceTypeStr, &subdomainStr, &gitRepoNull, &app.Branch, &app.RootDir, &inlineCodeNull, &autoDeployInt, &statusStr,
		&envJSON, &bindingsJSON, &migrationsJSON, &activeDepID,
		&compatDateStr, &compatFlagsJSON, &memLimitInt, &maxDurationInt, &workerPortInt,
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

	if envJSON != "" {
		if err := json.Unmarshal([]byte(envJSON), &app.EnvVars); err != nil {
			return nil, fmt.Errorf("unmarshal env_vars: %w", err)
		}
	}
	if bindingsJSON != "" {
		if err := json.Unmarshal([]byte(bindingsJSON), &app.Bindings); err != nil {
			return nil, fmt.Errorf("unmarshal bindings: %w", err)
		}
	}
	if migrationsJSON != "" {
		if err := json.Unmarshal([]byte(migrationsJSON), &app.Migrations); err != nil {
			return nil, fmt.Errorf("unmarshal migrations: %w", err)
		}
	}
	if app.Migrations == nil {
		app.Migrations = []domain.MigrationStep{}
	}
	app.CompatibilityDate = compatDateStr
	if compatFlagsJSON != "" {
		if err := json.Unmarshal([]byte(compatFlagsJSON), &app.CompatibilityFlags); err != nil {
			return nil, fmt.Errorf("unmarshal compatibility_flags: %w", err)
		}
	}
	app.MemoryLimitMB = memLimitInt
	app.MaxDurationMs = maxDurationInt
	app.WorkerPort = workerPortInt

	if createdAtStr != "" {
		t, err := time.Parse(time.RFC3339, createdAtStr)
		if err == nil {
			app.CreatedAt = t
		}
	}
	if updatedAtStr != "" {
		t, err := time.Parse(time.RFC3339, updatedAtStr)
		if err == nil {
			app.UpdatedAt = t
		}
	}

	return &app, nil
}

// Save persists an application entity to SQLite.
func (r *SQLiteRepository) Save(ctx context.Context, app *domain.Application) error {
	envVarsJSON, err := json.Marshal(app.EnvVars)
	if err != nil {
		return fmt.Errorf("marshal env_vars: %w", err)
	}
	bindingsJSON, err := json.Marshal(app.Bindings)
	if err != nil {
		return fmt.Errorf("marshal bindings: %w", err)
	}
	migrationsJSON, err := json.Marshal(app.Migrations)
	if err != nil {
		return fmt.Errorf("marshal migrations: %w", err)
	}
	compatFlagsJSON, err := json.Marshal(app.CompatibilityFlags)
	if err != nil {
		return fmt.Errorf("marshal compatibility_flags: %w", err)
	}

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
		INSERT INTO applications (id, name, source_type, subdomain, git_repo, branch, root_dir, inline_code, auto_deploy, status, env_vars, bindings, migrations, active_deployment_id, compatibility_date, compatibility_flags, memory_limit_mb, max_duration_ms, worker_port, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err = r.db.ExecContext(ctx, query,
		app.ID, app.Name, sourceType, subdomain, app.GitRepo, app.Branch, app.RootDir, app.InlineCode, autoDeployInt, string(app.Status),
		string(envVarsJSON), string(bindingsJSON), string(migrationsJSON), app.ActiveDeploymentID,
		compatDate, string(compatFlagsJSON), memLimit, maxDuration, app.WorkerPort,
		app.CreatedAt.Format(time.RFC3339), app.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("insert application: %w", err)
	}
	return nil
}

// GetByID retrieves an application by its ID.
func (r *SQLiteRepository) GetByID(ctx context.Context, id string) (*domain.Application, error) {
	query := fmt.Sprintf("SELECT %s FROM applications WHERE id = ?", selectApplicationColumns)
	row := r.db.QueryRowContext(ctx, query, id)

	app, err := scanApplicationRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.NewNotFoundError("application not found: " + id)
	}
	if err != nil {
		return nil, fmt.Errorf("get application by id %q: %w", id, err)
	}

	return app, nil
}

// GetBySubdomain retrieves an application by its subdomain or name.
func (r *SQLiteRepository) GetBySubdomain(ctx context.Context, subdomain string) (*domain.Application, error) {
	query := fmt.Sprintf("SELECT %s FROM applications WHERE subdomain = ? OR name = ?", selectApplicationColumns)
	row := r.db.QueryRowContext(ctx, query, subdomain, subdomain)

	app, err := scanApplicationRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.NewNotFoundError("application not found: " + subdomain)
	}
	if err != nil {
		return nil, fmt.Errorf("get application by subdomain %q: %w", subdomain, err)
	}

	return app, nil
}

// List retrieves all registered applications.
func (r *SQLiteRepository) List(ctx context.Context) ([]*domain.Application, error) {
	query := fmt.Sprintf("SELECT %s FROM applications ORDER BY created_at DESC", selectApplicationColumns)
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list applications: %w", err)
	}
	defer rows.Close()

	var apps []*domain.Application
	for rows.Next() {
		app, err := scanApplicationRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan application row: %w", err)
		}
		apps = append(apps, app)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate application rows: %w", err)
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
	envVarsJSON, err := json.Marshal(app.EnvVars)
	if err != nil {
		return fmt.Errorf("marshal env_vars: %w", err)
	}
	bindingsJSON, err := json.Marshal(app.Bindings)
	if err != nil {
		return fmt.Errorf("marshal bindings: %w", err)
	}
	migrationsJSON, err := json.Marshal(app.Migrations)
	if err != nil {
		return fmt.Errorf("marshal migrations: %w", err)
	}
	compatFlagsJSON, err := json.Marshal(app.CompatibilityFlags)
	if err != nil {
		return fmt.Errorf("marshal compatibility_flags: %w", err)
	}

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
		SET name = ?, source_type = ?, subdomain = ?, git_repo = ?, branch = ?, root_dir = ?, inline_code = ?, auto_deploy = ?, status = ?,
		    env_vars = ?, bindings = ?, migrations = ?, active_deployment_id = ?, compatibility_date = ?, compatibility_flags = ?, memory_limit_mb = ?, max_duration_ms = ?, worker_port = ?, updated_at = ?
		WHERE id = ?
	`
	res, err := r.db.ExecContext(ctx, query,
		app.Name, sourceType, subdomain, app.GitRepo, app.Branch, app.RootDir, app.InlineCode, autoDeployInt, string(app.Status),
		string(envVarsJSON), string(bindingsJSON), string(migrationsJSON), app.ActiveDeploymentID,
		compatDate, string(compatFlagsJSON), memLimit, maxDuration, app.WorkerPort,
		app.UpdatedAt.Format(time.RFC3339), app.ID,
	)
	if err != nil {
		return fmt.Errorf("update application %q: %w", app.ID, err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected for update %q: %w", app.ID, err)
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
		return fmt.Errorf("delete application %q: %w", id, err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected for delete %q: %w", id, err)
	}
	if rows == 0 {
		return domain.NewNotFoundError("application not found: " + id)
	}
	return nil
}
