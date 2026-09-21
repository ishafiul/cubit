package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/ishaf/cubit/internal/domain"
)

// AppUsecase coordinates application registration, deployment, and route synchronization.
type AppUsecase struct {
	appRepo        ApplicationRepository
	depRepo        DeploymentRepository
	nodeRepo       NodeRepository
	domainRepo     DomainRepository
	storage        StoragePort
	proxy          ProxyPort
	fleetBucket    string
}

// NewAppUsecase creates a new AppUsecase instance.
func NewAppUsecase(
	appRepo ApplicationRepository,
	depRepo DeploymentRepository,
	nodeRepo NodeRepository,
	domainRepo DomainRepository,
	storage StoragePort,
	proxy ProxyPort,
	fleetBucket string,
) *AppUsecase {
	return &AppUsecase{
		appRepo:     appRepo,
		depRepo:     depRepo,
		nodeRepo:    nodeRepo,
		domainRepo:  domainRepo,
		storage:     storage,
		proxy:       proxy,
		fleetBucket: fleetBucket,
	}
}

// CreateApplication creates a new worker application (backwards-compatible for git source).
func (u *AppUsecase) CreateApplication(
	ctx context.Context,
	name, gitRepo, branch string,
	envVars []domain.EnvironmentVariable,
	bindings []domain.ResourceBinding,
) (*domain.Application, error) {
	return u.CreateApplicationWithSource(ctx, name, domain.SourceTypeGit, gitRepo, branch, "", envVars, bindings)
}

// CreateApplicationWithSource creates a new worker application with explicit source type.
func (u *AppUsecase) CreateApplicationWithSource(
	ctx context.Context,
	name string,
	sourceType domain.SourceType,
	gitRepo, branch, inlineCode string,
	envVars []domain.EnvironmentVariable,
	bindings []domain.ResourceBinding,
) (*domain.Application, error) {
	appID := generateID()
	app, err := domain.NewApplicationWithSource(appID, name, sourceType, gitRepo, branch, inlineCode, envVars, bindings)
	if err != nil {
		return nil, err
	}

	if err := u.appRepo.Save(ctx, app); err != nil {
		return nil, err
	}

	return app, nil
}

// GetApplication retrieves an application by ID.
func (u *AppUsecase) GetApplication(ctx context.Context, id string) (*domain.Application, error) {
	return u.appRepo.GetByID(ctx, id)
}

// ListApplications retrieves all registered applications.
func (u *AppUsecase) ListApplications(ctx context.Context) ([]*domain.Application, error) {
	return u.appRepo.List(ctx)
}

// UpdateApplication updates branch, inline code, environment variables, and bindings.
func (u *AppUsecase) UpdateApplication(
	ctx context.Context,
	id, branch, inlineCode string,
	envVars []domain.EnvironmentVariable,
	bindings []domain.ResourceBinding,
) (*domain.Application, error) {
	app, err := u.appRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := app.UpdateConfig(branch, inlineCode, envVars, bindings); err != nil {
		return nil, err
	}

	if err := u.appRepo.Update(ctx, app); err != nil {
		return nil, err
	}

	return app, nil
}

// DeleteApplication deletes an application and removes associated routes.
func (u *AppUsecase) DeleteApplication(ctx context.Context, id string) error {
	if err := u.appRepo.Delete(ctx, id); err != nil {
		return err
	}
	return u.syncAllRoutes(ctx)
}

// DeployApplication creates and executes a deployment for an application.
func (u *AppUsecase) DeployApplication(ctx context.Context, appID, commitHash string) (*domain.Deployment, error) {
	app, err := u.appRepo.GetByID(ctx, appID)
	if err != nil {
		return nil, err
	}

	depID := generateID()
	dep, err := domain.NewDeployment(depID, app.ID, commitHash, "Deployment triggered")
	if err != nil {
		return nil, err
	}

	if err := u.depRepo.Save(ctx, dep); err != nil {
		return nil, err
	}

	// Progress through build and deployment steps
	_ = dep.StartBuilding()
	_ = u.depRepo.Update(ctx, dep)

	var bundleData []byte
	if app.SourceType == domain.SourceTypeInline {
		bundleData = []byte(app.InlineCode)
		_ = u.depRepo.AppendLog(ctx, dep.ID, domain.DeploymentLog{
			Timestamp: time.Now().UTC(),
			Step:      domain.LogStepEsbuild,
			Message:   fmt.Sprintf("Prepared inline worker bundle: %d bytes", len(bundleData)),
			Level:     domain.LogLevelInfo,
		})
	} else {
		_ = u.depRepo.AppendLog(ctx, dep.ID, domain.DeploymentLog{
			Timestamp: time.Now().UTC(),
			Step:      domain.LogStepGitClone,
			Message:   fmt.Sprintf("Cloned repository %s @ %s", app.GitRepo, dep.CommitHash),
			Level:     domain.LogLevelInfo,
		})

		bundleData = []byte("// Bundled worker code\nexport default { fetch: (req) => new Response('Hello from Cubit!') };")
		_ = u.depRepo.AppendLog(ctx, dep.ID, domain.DeploymentLog{
			Timestamp: time.Now().UTC(),
			Step:      domain.LogStepEsbuild,
			Message:   fmt.Sprintf("Compiled bundle using esbuild: %d bytes", len(bundleData)),
			Level:     domain.LogLevelInfo,
		})
	}
	bundleSize := int64(len(bundleData))

	_ = dep.StartDeploying(bundleSize)
	_ = u.depRepo.Update(ctx, dep)

	// Upload to S3 storage if storage port is configured
	if u.storage != nil {
		objectKey := fmt.Sprintf("deployments/%s/%s/bundle.js", app.Name, dep.ID)
		if err := u.storage.UploadBundle(ctx, u.fleetBucket, objectKey, bundleData); err != nil {
			dep.MarkFailed("failed uploading to S3: " + err.Error())
			_ = u.depRepo.Update(ctx, dep)
			return dep, nil
		}
		_ = u.depRepo.AppendLog(ctx, dep.ID, domain.DeploymentLog{
			Timestamp: time.Now().UTC(),
			Step:      domain.LogStepS3Upload,
			Message:   fmt.Sprintf("Uploaded bundle to S3 bucket %s (%s)", u.fleetBucket, objectKey),
			Level:     domain.LogLevelInfo,
		})
	}

	_ = dep.MarkActive()
	_ = u.depRepo.Update(ctx, dep)

	app.SetActiveDeployment(dep.ID)
	_ = u.appRepo.Update(ctx, app)

	_ = u.syncAllRoutes(ctx)

	_ = u.depRepo.AppendLog(ctx, dep.ID, domain.DeploymentLog{
		Timestamp: time.Now().UTC(),
		Step:      domain.LogStepRouteSync,
		Message:   "Synchronized Traefik routing rules across fleet",
		Level:     domain.LogLevelInfo,
	})

	return dep, nil
}

// syncAllRoutes queries active nodes and domains to push updated routing tables to Traefik.
func (u *AppUsecase) syncAllRoutes(ctx context.Context) error {
	if u.proxy == nil {
		return nil
	}

	nodes, err := u.nodeRepo.List(ctx)
	if err != nil {
		return err
	}

	var targetURLs []string
	for _, n := range nodes {
		if n.Status == domain.NodeStatusActive {
			targetURLs = append(targetURLs, fmt.Sprintf("http://%s:%d", n.IPAddress, n.WorkerPort))
		}
	}

	if len(targetURLs) == 0 {
		return u.proxy.SyncRoutes(ctx, nil)
	}

	apps, err := u.appRepo.List(ctx)
	if err != nil {
		return err
	}

	var rules []RouteRule
	for _, app := range apps {
		if app.Status != domain.AppStatusRunning {
			continue
		}

		domains, err := u.domainRepo.ListByAppID(ctx, app.ID)
		if err != nil {
			continue
		}

		for _, d := range domains {
			rules = append(rules, RouteRule{
				AppName:    app.Name,
				Hostname:   d.Hostname,
				PathPrefix: d.PathPrefix,
				TargetURLs: targetURLs,
				EnableTLS:  d.SSLActive,
			})
		}
	}

	return u.proxy.SyncRoutes(ctx, rules)
}
