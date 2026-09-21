package domain

import (
	"strings"
	"time"
)

// DeploymentStatus represents the progress lifecycle of a deployment.
type DeploymentStatus string

const (
	DeploymentStatusPending    DeploymentStatus = "pending"
	DeploymentStatusBuilding   DeploymentStatus = "building"
	DeploymentStatusDeploying  DeploymentStatus = "deploying"
	DeploymentStatusActive     DeploymentStatus = "active"
	DeploymentStatusFailed     DeploymentStatus = "failed"
	DeploymentStatusSuperseded DeploymentStatus = "superseded"
)

// LogStep identifies the pipeline step where a log message originated.
type LogStep string

const (
	LogStepGitClone    LogStep = "git_clone"
	LogStepEsbuild     LogStep = "esbuild"
	LogStepS3Upload    LogStep = "s3_upload"
	LogStepCelldDeploy LogStep = "celld_deploy"
	LogStepRouteSync   LogStep = "route_sync"
)

// LogLevel indicates severity of a deployment log message.
type LogLevel string

const (
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
)

// DeploymentLog represents a discrete event or output line during deployment execution.
type DeploymentLog struct {
	Timestamp time.Time
	Step      LogStep
	Message   string
	Level     LogLevel
}

// Deployment represents an immutable release of an application.
type Deployment struct {
	ID            string
	ApplicationID string
	CommitHash    string
	CommitMessage string
	Status        DeploymentStatus
	BundleSize    int64
	ErrorMessage  string
	CreatedAt     time.Time
	FinishedAt    *time.Time
}

// NewDeployment creates and initializes a new Deployment.
func NewDeployment(id, applicationID, commitHash, commitMessage string) (*Deployment, error) {
	if strings.TrimSpace(id) == "" {
		return nil, NewValidationError("deployment ID cannot be empty")
	}
	if strings.TrimSpace(applicationID) == "" {
		return nil, NewValidationError("application ID cannot be empty")
	}

	commitHash = strings.TrimSpace(commitHash)
	if commitHash == "" {
		commitHash = "HEAD"
	}

	now := time.Now().UTC()
	return &Deployment{
		ID:            id,
		ApplicationID: applicationID,
		CommitHash:    commitHash,
		CommitMessage: strings.TrimSpace(commitMessage),
		Status:        DeploymentStatusPending,
		CreatedAt:     now,
	}, nil
}

// StartBuilding transitions deployment state from pending to building.
func (d *Deployment) StartBuilding() error {
	if d.Status != DeploymentStatusPending {
		return NewInvalidStateError("can only start building a pending deployment")
	}
	d.Status = DeploymentStatusBuilding
	return nil
}

// StartDeploying transitions deployment state from building to deploying.
func (d *Deployment) StartDeploying(bundleSize int64) error {
	if d.Status != DeploymentStatusBuilding {
		return NewInvalidStateError("can only deploy a built deployment")
	}
	d.Status = DeploymentStatusDeploying
	d.BundleSize = bundleSize
	return nil
}

// MarkActive marks the deployment as successfully active and serving traffic.
func (d *Deployment) MarkActive() error {
	if d.Status != DeploymentStatusDeploying {
		return NewInvalidStateError("can only activate a deployment that is in deploying state")
	}
	d.Status = DeploymentStatusActive
	now := time.Now().UTC()
	d.FinishedAt = &now
	return nil
}

// MarkFailed records a failure reason and moves deployment to failed state.
func (d *Deployment) MarkFailed(reason string) {
	d.Status = DeploymentStatusFailed
	d.ErrorMessage = strings.TrimSpace(reason)
	now := time.Now().UTC()
	d.FinishedAt = &now
}

// MarkSuperseded marks an active deployment as superseded by a newer deployment.
func (d *Deployment) MarkSuperseded() {
	if d.Status == DeploymentStatusActive {
		d.Status = DeploymentStatusSuperseded
	}
}
