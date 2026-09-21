package deployment

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/ishaf/cubit/internal/domain"
)

// Repository defines persistence operations for deployments and build logs.
type Repository interface {
	Save(ctx context.Context, dep *domain.Deployment) error
	GetByID(ctx context.Context, id string) (*domain.Deployment, error)
	ListByAppID(ctx context.Context, appID string) ([]*domain.Deployment, error)
	GetLatestBuildVersion(ctx context.Context, appID string) (int, error)
	Update(ctx context.Context, dep *domain.Deployment) error
	AppendLog(ctx context.Context, deploymentID string, entry domain.DeploymentLog) error
	GetLogs(ctx context.Context, deploymentID string) ([]domain.DeploymentLog, error)
}

// SQLiteRepository implements Repository using SQLite.
type SQLiteRepository struct {
	db *sql.DB
}

// NewRepository creates a new SQLite deployment repository.
func NewRepository(db *sql.DB) *SQLiteRepository {
	return &SQLiteRepository{db: db}
}

// Save persists a new deployment.
func (r *SQLiteRepository) Save(ctx context.Context, dep *domain.Deployment) error {
	var finishedAtStr sql.NullString
	if dep.FinishedAt != nil {
		finishedAtStr.String = dep.FinishedAt.Format(time.RFC3339)
		finishedAtStr.Valid = true
	}

	buildVersion := dep.BuildVersion
	if buildVersion < 1 {
		buildVersion = 1
	}

	query := `
		INSERT INTO deployments (id, application_id, build_version, commit_hash, commit_message, status, bundle_size, error_message, created_at, finished_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.ExecContext(ctx, query,
		dep.ID, dep.ApplicationID, buildVersion, dep.CommitHash, dep.CommitMessage,
		string(dep.Status), dep.BundleSize, dep.ErrorMessage,
		dep.CreatedAt.Format(time.RFC3339), finishedAtStr,
	)
	return err
}

// GetByID retrieves a deployment by ID.
func (r *SQLiteRepository) GetByID(ctx context.Context, id string) (*domain.Deployment, error) {
	query := `
		SELECT id, application_id, build_version, commit_hash, commit_message, status, bundle_size, error_message, created_at, finished_at
		FROM deployments WHERE id = ?
	`
	row := r.db.QueryRowContext(ctx, query, id)

	var (
		dep                                domain.Deployment
		statusStr, createdAtStr            string
		commitMsg, errorMsg, finishedAtStr sql.NullString
	)

	err := row.Scan(
		&dep.ID, &dep.ApplicationID, &dep.BuildVersion, &dep.CommitHash, &commitMsg,
		&statusStr, &dep.BundleSize, &errorMsg,
		&createdAtStr, &finishedAtStr,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.NewNotFoundError("deployment not found: " + id)
	}
	if err != nil {
		return nil, err
	}

	if dep.BuildVersion < 1 {
		dep.BuildVersion = 1
	}
	dep.Status = domain.DeploymentStatus(statusStr)
	if commitMsg.Valid {
		dep.CommitMessage = commitMsg.String
	}
	if errorMsg.Valid {
		dep.ErrorMessage = errorMsg.String
	}
	dep.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
	if finishedAtStr.Valid {
		t, _ := time.Parse(time.RFC3339, finishedAtStr.String)
		dep.FinishedAt = &t
	}

	return &dep, nil
}

// ListByAppID returns all deployments for an application.
func (r *SQLiteRepository) ListByAppID(ctx context.Context, appID string) ([]*domain.Deployment, error) {
	query := `
		SELECT id, application_id, build_version, commit_hash, commit_message, status, bundle_size, error_message, created_at, finished_at
		FROM deployments WHERE application_id = ? ORDER BY build_version DESC, created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query, appID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var deps []*domain.Deployment
	for rows.Next() {
		var (
			dep                                domain.Deployment
			statusStr, createdAtStr            string
			commitMsg, errorMsg, finishedAtStr sql.NullString
		)
		if err := rows.Scan(
			&dep.ID, &dep.ApplicationID, &dep.BuildVersion, &dep.CommitHash, &commitMsg,
			&statusStr, &dep.BundleSize, &errorMsg,
			&createdAtStr, &finishedAtStr,
		); err != nil {
			return nil, err
		}
		if dep.BuildVersion < 1 {
			dep.BuildVersion = 1
		}
		dep.Status = domain.DeploymentStatus(statusStr)
		if commitMsg.Valid {
			dep.CommitMessage = commitMsg.String
		}
		if errorMsg.Valid {
			dep.ErrorMessage = errorMsg.String
		}
		dep.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
		if finishedAtStr.Valid {
			t, _ := time.Parse(time.RFC3339, finishedAtStr.String)
			dep.FinishedAt = &t
		}
		deps = append(deps, &dep)
	}

	return deps, rows.Err()
}

// GetLatestBuildVersion returns the highest build version recorded for an application, or 0 if none exist.
func (r *SQLiteRepository) GetLatestBuildVersion(ctx context.Context, appID string) (int, error) {
	query := `SELECT COALESCE(MAX(build_version), 0) FROM deployments WHERE application_id = ?`
	var maxVer int
	err := r.db.QueryRowContext(ctx, query, appID).Scan(&maxVer)
	if err != nil {
		return 0, err
	}
	return maxVer, nil
}

// Update updates deployment status, finish time, or bundle size.
func (r *SQLiteRepository) Update(ctx context.Context, dep *domain.Deployment) error {
	var finishedAtStr sql.NullString
	if dep.FinishedAt != nil {
		finishedAtStr.String = dep.FinishedAt.Format(time.RFC3339)
		finishedAtStr.Valid = true
	}

	query := `
		UPDATE deployments
		SET status = ?, bundle_size = ?, error_message = ?, finished_at = ?
		WHERE id = ?
	`
	_, err := r.db.ExecContext(ctx, query,
		string(dep.Status), dep.BundleSize, dep.ErrorMessage, finishedAtStr, dep.ID,
	)
	return err
}

// AppendLog appends a new execution log entry.
func (r *SQLiteRepository) AppendLog(ctx context.Context, deploymentID string, entry domain.DeploymentLog) error {
	query := `
		INSERT INTO deployment_logs (deployment_id, timestamp, step, message, level)
		VALUES (?, ?, ?, ?, ?)
	`
	_, err := r.db.ExecContext(ctx, query,
		deploymentID, entry.Timestamp.Format(time.RFC3339),
		string(entry.Step), entry.Message, string(entry.Level),
	)
	return err
}

// GetLogs retrieves chronological logs for a deployment.
func (r *SQLiteRepository) GetLogs(ctx context.Context, deploymentID string) ([]domain.DeploymentLog, error) {
	query := `
		SELECT timestamp, step, message, level
		FROM deployment_logs
		WHERE deployment_id = ?
		ORDER BY id ASC
	`
	rows, err := r.db.QueryContext(ctx, query, deploymentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []domain.DeploymentLog
	for rows.Next() {
		var (
			entry                    domain.DeploymentLog
			tsStr, stepStr, levelStr string
		)
		if err := rows.Scan(&tsStr, &stepStr, &entry.Message, &levelStr); err != nil {
			return nil, err
		}
		entry.Timestamp, _ = time.Parse(time.RFC3339, tsStr)
		entry.Step = domain.LogStep(stepStr)
		entry.Level = domain.LogLevel(levelStr)
		logs = append(logs, entry)
	}
	return logs, rows.Err()
}
