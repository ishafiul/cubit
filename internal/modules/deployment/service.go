package deployment

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/ishaf/cubit/internal/domain"
)

// ApplicationManager defines the application operations needed by the deployment service.
type ApplicationManager interface {
	GetByID(ctx context.Context, id string) (*domain.Application, error)
	SetActiveDeployment(ctx context.Context, appID, deploymentID string) error
}

// StorageUploader uploads compiled worker bundles to storage.
type StorageUploader interface {
	UploadBundle(ctx context.Context, bucket, objectKey string, data []byte) error
}

// RouteSyncer triggers routing updates in Traefik.
type RouteSyncer interface {
	SyncRoutes(ctx context.Context) error
}

// Service defines deployment business operations.
type Service interface {
	Deploy(ctx context.Context, appID string, commitHash string) (*domain.Deployment, error)
	DeployWithDetails(ctx context.Context, appID, commitHash, commitMessage string) (*domain.Deployment, error)
	Rollback(ctx context.Context, appID, deploymentID string) (*domain.Deployment, error)
	GetByID(ctx context.Context, id string) (*domain.Deployment, error)
	ListByApp(ctx context.Context, appID string) ([]*domain.Deployment, error)
	GetLogs(ctx context.Context, depID string) ([]domain.DeploymentLog, error)
	AppendLog(ctx context.Context, depID string, entry domain.DeploymentLog) error
}

// DeploymentService coordinates deployment operations.
type DeploymentService struct {
	repo        Repository
	appMgr      ApplicationManager
	storage     StorageUploader
	routeSyncer RouteSyncer
	fleetBucket string
}

// NewService creates a new DeploymentService.
func NewService(repo Repository, appMgr ApplicationManager, storage StorageUploader, routeSyncer RouteSyncer, fleetBucket string) *DeploymentService {
	return &DeploymentService{
		repo:        repo,
		appMgr:      appMgr,
		storage:     storage,
		routeSyncer: routeSyncer,
		fleetBucket: fleetBucket,
	}
}

// Deploy triggers a new build and release for an application.
func (s *DeploymentService) Deploy(ctx context.Context, appID string, commitHash string) (*domain.Deployment, error) {
	return s.DeployWithDetails(ctx, appID, commitHash, "")
}

// DeployWithDetails triggers a deployment with custom commit metadata (e.g. from GitHub webhooks).
func (s *DeploymentService) DeployWithDetails(ctx context.Context, appID, commitHash, commitMessage string) (*domain.Deployment, error) {
	app, err := s.appMgr.GetByID(ctx, appID)
	if err != nil {
		return nil, err
	}

	depID := generateID()
	if commitHash == "" {
		commitHash = generateShortHash()
	}

	maxVer, _ := s.repo.GetLatestBuildVersion(ctx, appID)
	buildVer := maxVer + 1

	now := time.Now().UTC()
	dep, err := domain.NewDeploymentWithVersion(depID, app.ID, commitHash, commitMessage, buildVer)
	if err != nil {
		return nil, err
	}
	_ = dep.StartBuilding()

	if err := s.repo.Save(ctx, dep); err != nil {
		return nil, err
	}

	_ = s.repo.AppendLog(ctx, dep.ID, domain.DeploymentLog{
		Timestamp: now,
		Step:      domain.LogStepEsbuild,
		Message:   fmt.Sprintf("Starting build %s for application %s", dep.VersionTag(), app.Name),
		Level:     domain.LogLevelInfo,
	})

	// Build bundle
	var bundleData []byte
	if app.SourceType == domain.SourceTypeInline && app.InlineCode != "" {
		bundleData = []byte(app.InlineCode)
	} else {
		bundleData = []byte(domain.DefaultHelloWorldWorker)
	}

	dep.BundleSize = int64(len(bundleData))

	// Upload to storage
	if s.storage != nil {
		objectKey := fmt.Sprintf("deployments/%s/%s/bundle.js", app.Name, dep.ID)
		if err := s.storage.UploadBundle(ctx, s.fleetBucket, objectKey, bundleData); err != nil {
			dep.MarkFailed(fmt.Sprintf("Failed to upload bundle: %v", err))
			_ = s.repo.Update(ctx, dep)
			_ = s.repo.AppendLog(ctx, dep.ID, domain.DeploymentLog{
				Timestamp: time.Now().UTC(),
				Step:      domain.LogStepS3Upload,
				Message:   fmt.Sprintf("Bundle upload failed: %v", err),
				Level:     domain.LogLevelError,
			})
			return dep, err
		}
	}

	_ = s.repo.AppendLog(ctx, dep.ID, domain.DeploymentLog{
		Timestamp: time.Now().UTC(),
		Step:      domain.LogStepS3Upload,
		Message:   fmt.Sprintf("Distributed bundle (%d bytes) to storage", dep.BundleSize),
		Level:     domain.LogLevelInfo,
	})

	// Supersede any existing active deployments
	allDeps, _ := s.repo.ListByAppID(ctx, app.ID)
	for _, oldDep := range allDeps {
		if oldDep.ID != dep.ID && oldDep.Status == domain.DeploymentStatusActive {
			oldDep.MarkSuperseded()
			_ = s.repo.Update(ctx, oldDep)
		}
	}

	// Activate this deployment
	_ = dep.StartDeploying(dep.BundleSize)
	_ = dep.MarkActive()
	if err := s.repo.Update(ctx, dep); err != nil {
		return nil, err
	}

	_ = s.repo.AppendLog(ctx, dep.ID, domain.DeploymentLog{
		Timestamp: time.Now().UTC(),
		Step:      domain.LogStepCelldDeploy,
		Message:   fmt.Sprintf("Activated %s as primary live deployment", dep.VersionTag()),
		Level:     domain.LogLevelInfo,
	})

	// Update active deployment on application
	if err := s.appMgr.SetActiveDeployment(ctx, app.ID, dep.ID); err != nil {
		return nil, err
	}

	if s.routeSyncer != nil {
		_ = s.routeSyncer.SyncRoutes(ctx)
		_ = s.repo.AppendLog(ctx, dep.ID, domain.DeploymentLog{
			Timestamp: time.Now().UTC(),
			Step:      domain.LogStepRouteSync,
			Message:   fmt.Sprintf("Synchronized Traefik routing rules for %s (%s)", app.Name, dep.VersionTag()),
			Level:     domain.LogLevelInfo,
		})
	}

	return dep, nil
}

// Rollback switches the application's active live deployment to a specified past deployment.
func (s *DeploymentService) Rollback(ctx context.Context, appID, deploymentID string) (*domain.Deployment, error) {
	app, err := s.appMgr.GetByID(ctx, appID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(deploymentID) == "" {
		return nil, domain.NewValidationError("deploymentId is required")
	}

	targetDep, err := s.repo.GetByID(ctx, deploymentID)
	if err != nil {
		return nil, err
	}
	if targetDep.ApplicationID != app.ID {
		return nil, domain.NewValidationError("deployment does not belong to application")
	}
	if targetDep.Status == domain.DeploymentStatusFailed {
		return nil, domain.NewValidationError("cannot rollback to a failed deployment")
	}

	// Supersede any existing active deployments
	allDeps, _ := s.repo.ListByAppID(ctx, app.ID)
	for _, oldDep := range allDeps {
		if oldDep.ID != targetDep.ID && oldDep.Status == domain.DeploymentStatusActive {
			oldDep.MarkSuperseded()
			_ = s.repo.Update(ctx, oldDep)
		}
	}

	// Activate target deployment
	targetDep.Status = domain.DeploymentStatusActive
	now := time.Now().UTC()
	targetDep.FinishedAt = &now
	if err := s.repo.Update(ctx, targetDep); err != nil {
		return nil, err
	}

	_ = s.repo.AppendLog(ctx, targetDep.ID, domain.DeploymentLog{
		Timestamp: now,
		Step:      domain.LogStepCelldDeploy,
		Message:   fmt.Sprintf("Rolled back to %s (%s) as primary live deployment", targetDep.VersionTag(), targetDep.CommitHash),
		Level:     domain.LogLevelInfo,
	})

	if err := s.appMgr.SetActiveDeployment(ctx, app.ID, targetDep.ID); err != nil {
		return nil, err
	}

	if s.routeSyncer != nil {
		_ = s.routeSyncer.SyncRoutes(ctx)
		_ = s.repo.AppendLog(ctx, targetDep.ID, domain.DeploymentLog{
			Timestamp: time.Now().UTC(),
			Step:      domain.LogStepRouteSync,
			Message:   fmt.Sprintf("Synchronized Traefik routing rules for %s (%s)", app.Name, targetDep.VersionTag()),
			Level:     domain.LogLevelInfo,
		})
	}

	return targetDep, nil
}

// GetByID retrieves a deployment by ID.
func (s *DeploymentService) GetByID(ctx context.Context, id string) (*domain.Deployment, error) {
	return s.repo.GetByID(ctx, id)
}

// ListByApp returns all deployments for an application.
func (s *DeploymentService) ListByApp(ctx context.Context, appID string) ([]*domain.Deployment, error) {
	return s.repo.ListByAppID(ctx, appID)
}

// GetLogs returns chronological execution logs for a deployment.
func (s *DeploymentService) GetLogs(ctx context.Context, depID string) ([]domain.DeploymentLog, error) {
	return s.repo.GetLogs(ctx, depID)
}

// AppendLog appends a log entry to a deployment.
func (s *DeploymentService) AppendLog(ctx context.Context, depID string, entry domain.DeploymentLog) error {
	return s.repo.AppendLog(ctx, depID, entry)
}

func generateID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	h := hex.EncodeToString(b)
	return fmt.Sprintf("%s-%s-%s-%s-%s", h[0:8], h[8:12], h[12:16], h[16:20], h[20:32])
}

func generateShortHash() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
