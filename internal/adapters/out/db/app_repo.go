package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
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
	subdomain := app.Subdomain
	if subdomain == "" {
		subdomain = domain.SanitizeSubdomain(app.Name)
	}

	autoDeployInt := 0
	if app.AutoDeploy {
		autoDeployInt = 1
	}

	query := `
		INSERT INTO applications (id, name, source_type, subdomain, git_repo, branch, inline_code, auto_deploy, status, env_vars, bindings, active_deployment_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.ExecContext(ctx, query,
		app.ID, app.Name, sourceType, subdomain, app.GitRepo, app.Branch, app.InlineCode, autoDeployInt, string(app.Status),
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
		SELECT id, name, source_type, subdomain, git_repo, branch, inline_code, COALESCE(auto_deploy, 1), status, env_vars, bindings, active_deployment_id, created_at, updated_at
		FROM applications WHERE id = ?
	`
	row := r.db.QueryRowContext(ctx, query, id)

	var (
		app                                                                domain.Application
		sourceTypeStr, subdomainStr, statusStr, envJSON, bindingsJSON, createdAtStr, updatedAtStr string
		gitRepoNull, inlineCodeNull, activeDepID                           sql.NullString
		autoDeployInt                                                      int
	)

	err := row.Scan(
		&app.ID, &app.Name, &sourceTypeStr, &subdomainStr, &gitRepoNull, &app.Branch, &inlineCodeNull, &autoDeployInt, &statusStr,
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
	app.Subdomain = subdomainStr
	if app.Subdomain == "" {
		app.Subdomain = domain.SanitizeSubdomain(app.Name)
	}
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
	app.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
	app.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)
	return &app, nil
}

// GetBySubdomain retrieves an application by its unique subdomain.
func (r *AppRepo) GetBySubdomain(ctx context.Context, subdomain string) (*domain.Application, error) {
	query := `
		SELECT id, name, source_type, subdomain, git_repo, branch, inline_code, COALESCE(auto_deploy, 1), status, env_vars, bindings, active_deployment_id, created_at, updated_at
		FROM applications WHERE subdomain = ? OR name = ?
	`
	row := r.db.QueryRowContext(ctx, query, subdomain, subdomain)

	var (
		app                                                                domain.Application
		sourceTypeStr, subdomainStr, statusStr, envJSON, bindingsJSON, createdAtStr, updatedAtStr string
		gitRepoNull, inlineCodeNull, activeDepID                           sql.NullString
		autoDeployInt                                                      int
	)

	err := row.Scan(
		&app.ID, &app.Name, &sourceTypeStr, &subdomainStr, &gitRepoNull, &app.Branch, &inlineCodeNull, &autoDeployInt, &statusStr,
		&envJSON, &bindingsJSON, &activeDepID, &createdAtStr, &updatedAtStr,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.NewNotFoundError("application not found for subdomain")
	}
	if err != nil {
		return nil, err
	}

	app.SourceType = domain.SourceType(sourceTypeStr)
	if app.SourceType == "" {
		app.SourceType = domain.SourceTypeGit
	}
	app.Subdomain = subdomainStr
	if app.Subdomain == "" {
		app.Subdomain = domain.SanitizeSubdomain(app.Name)
	}
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
	app.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
	app.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)
	return &app, nil
}

// List returns all registered applications.
func (r *AppRepo) List(ctx context.Context) ([]*domain.Application, error) {
	query := `
		SELECT id, name, source_type, subdomain, git_repo, branch, inline_code, COALESCE(auto_deploy, 1), status, env_vars, bindings, active_deployment_id, created_at, updated_at
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
			gitRepoNull, inlineCodeNull, activeDepID                           sql.NullString
			autoDeployInt                                                      int
		)
		if err := rows.Scan(
			&app.ID, &app.Name, &sourceTypeStr, &subdomainStr, &gitRepoNull, &app.Branch, &inlineCodeNull, &autoDeployInt, &statusStr,
			&envJSON, &bindingsJSON, &activeDepID, &createdAtStr, &updatedAtStr,
		); err != nil {
			return nil, err
		}
		app.SourceType = domain.SourceType(sourceTypeStr)
		if app.SourceType == "" {
			app.SourceType = domain.SourceTypeGit
		}
		app.Subdomain = subdomainStr
		if app.Subdomain == "" {
			app.Subdomain = domain.SanitizeSubdomain(app.Name)
		}
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
	subdomain := app.Subdomain
	if subdomain == "" {
		subdomain = domain.SanitizeSubdomain(app.Name)
	}

	autoDeployInt := 0
	if app.AutoDeploy {
		autoDeployInt = 1
	}

	query := `
		UPDATE applications
		SET name = ?, source_type = ?, subdomain = ?, git_repo = ?, branch = ?, inline_code = ?, auto_deploy = ?, status = ?, env_vars = ?, bindings = ?, active_deployment_id = ?, updated_at = ?
		WHERE id = ?
	`
	res, err := r.db.ExecContext(ctx, query,
		app.Name, sourceType, subdomain, app.GitRepo, app.Branch, app.InlineCode, autoDeployInt, string(app.Status),
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

// ListByGitRepoAndBranch returns all Git applications configured with auto-deploy matching a repository and branch.
func (r *AppRepo) ListByGitRepoAndBranch(ctx context.Context, repo, branch string) ([]*domain.Application, error) {
	all, err := r.List(ctx)
	if err != nil {
		return nil, err
	}

	var matched []*domain.Application
	normalizedTargetRepo := strings.ToLower(strings.TrimSpace(repo))
	normalizedTargetRepo = strings.TrimSuffix(normalizedTargetRepo, ".git")

	for _, app := range all {
		if app.SourceType != domain.SourceTypeGit || !app.AutoDeploy {
			continue
		}

		appRepo := strings.ToLower(strings.TrimSpace(app.GitRepo))
		appRepo = strings.TrimSuffix(appRepo, ".git")

		// Match if equal, or if appRepo ends with "/owner/repo", or ":owner/repo" (SSH git@github.com:owner/repo)
		isRepoMatch := appRepo == normalizedTargetRepo ||
			strings.HasSuffix(appRepo, "/"+normalizedTargetRepo) ||
			strings.HasSuffix(appRepo, ":"+normalizedTargetRepo)

		isBranchMatch := app.Branch == branch || (app.Branch == "" && branch == "main")

		if isRepoMatch && isBranchMatch {
			matched = append(matched, app)
		}
	}
	return matched, nil
}
