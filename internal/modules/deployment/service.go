package deployment

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/ishaf/cubit/internal/domain"
	"github.com/ishaf/cubit/internal/modules/wrangler"
)

// ApplicationManager defines the application operations needed by the deployment service.
type ApplicationManager interface {
	GetByID(ctx context.Context, id string) (*domain.Application, error)
	SetActiveDeployment(ctx context.Context, appID, deploymentID string) error
	ImportWrangler(ctx context.Context, appID string, rawConfig string, format string, envName string) (*domain.Application, *wrangler.ImportSummary, error)
}

// StorageUploader uploads compiled worker bundles to storage.
type StorageUploader interface {
	UploadBundle(ctx context.Context, bucket, objectKey string, data []byte) error
}

// RouteSyncer triggers routing updates in Traefik.
type RouteSyncer interface {
	SyncRoutes(ctx context.Context) error
}

// FleetReloader triggers zero-downtime hot reload across celld fleet nodes.
type FleetReloader interface {
	ReloadFleet(ctx context.Context) error
}

// NodeLister defines node listing capability for fleet reloaders.
type NodeLister interface {
	List(ctx context.Context) ([]*domain.Node, error)
}

// CelldFleetReloader triggers reload across all active nodes in the fleet.
type CelldFleetReloader struct {
	nodes      NodeLister
	httpClient HTTPClient
}

// NewCelldFleetReloader creates a new CelldFleetReloader.
func NewCelldFleetReloader(nodes NodeLister, client HTTPClient) *CelldFleetReloader {
	return &CelldFleetReloader{
		nodes:      nodes,
		httpClient: client,
	}
}

// ReloadFleet reloads all active celld fleet nodes.
func (r *CelldFleetReloader) ReloadFleet(ctx context.Context) error {
	if r.nodes == nil {
		return nil
	}
	nodes, err := r.nodes.List(ctx)
	if err != nil {
		return fmt.Errorf("failed to list nodes for reload: %w", err)
	}

	var urls []string
	for _, n := range nodes {
		if n.Status == domain.NodeStatusActive {
			port := n.InternalPort
			if port == 0 {
				port = 8081
			}
			urls = append(urls, fmt.Sprintf("http://%s:%d/reload", n.IPAddress, port))
		}
	}

	if len(urls) == 0 {
		return nil
	}

	errs := TriggerFleetReload(ctx, r.httpClient, urls)
	if len(errs) > 0 {
		return fmt.Errorf("failed to reload %d/%d nodes: %w", len(errs), len(urls), errors.Join(errs...))
	}
	return nil
}

// Service defines deployment business operations.
type Service interface {
	Deploy(ctx context.Context, appID string, commitHash string) (*domain.Deployment, error)
	DeployWithDetails(ctx context.Context, appID, commitHash, commitMessage string) (*domain.Deployment, error)
	DeployDirect(ctx context.Context, appID string, bundle []byte, commitMessage string, rawWranglerConfig string) (*domain.Deployment, error)
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
	reloader    FleetReloader
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

// WithFleetReloader sets the fleet reloader for hot reloading celld nodes.
func (s *DeploymentService) WithFleetReloader(reloader FleetReloader) *DeploymentService {
	s.reloader = reloader
	return s
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
	var wranglerCfg *wrangler.WranglerConfig
	if app.SourceType == domain.SourceTypeInline && app.InlineCode != "" {
		bundleData = []byte(app.InlineCode)
	} else if app.SourceType == domain.SourceTypeGit && app.GitRepo != "" {
		bundleData, wranglerCfg = s.buildGitWorker(ctx, app, dep)
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

	// Publish celld deployment manifest, update deploy/current.json, and trigger fleet reload
	if err := s.publishCelldDeployment(ctx, app, dep, bundleData, wranglerCfg); err != nil {
		dep.MarkFailed(err.Error())
		_ = s.repo.Update(ctx, dep)
		_ = s.repo.AppendLog(ctx, dep.ID, domain.DeploymentLog{
			Timestamp: time.Now().UTC(),
			Step:      domain.LogStepCelldDeploy,
			Message:   fmt.Sprintf("Celld deployment publication failed: %v", err),
			Level:     domain.LogLevelError,
		})
		return dep, err
	}

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

	// Update celld active pointer for target deployment and trigger fleet reload
	if err := s.updatePointerAndReload(ctx, app, targetDep, nil); err != nil {
		return nil, fmt.Errorf("failed to sync celld deployment pointer during rollback: %w", err)
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

// DeployDirect creates and activates a deployment immediately from raw worker bundle bytes (CLI / CI deployment).
func (s *DeploymentService) DeployDirect(ctx context.Context, appID string, bundle []byte, commitMessage string, rawWranglerConfig string) (*domain.Deployment, error) {
	app, err := s.appMgr.GetByID(ctx, appID)
	if err != nil {
		return nil, err
	}

	depID := generateID()
	commitHash := generateShortHash()
	if commitMessage == "" {
		commitMessage = "Direct deployment via API/CLI"
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
		Message:   fmt.Sprintf("Direct release %s received for application %s", dep.VersionTag(), app.Name),
		Level:     domain.LogLevelInfo,
	})

	var wranglerCfg *wrangler.WranglerConfig
	// If wrangler configuration is supplied, sync variables, bindings, and compat settings
	if rawWranglerConfig != "" {
		if parsed, _, err := wrangler.Parse([]byte(rawWranglerConfig), "auto"); err == nil {
			wranglerCfg = parsed
		}
		_, summary, err := s.appMgr.ImportWrangler(ctx, app.ID, rawWranglerConfig, "auto", "")
		if err == nil && summary != nil {
			_ = s.repo.AppendLog(ctx, dep.ID, domain.DeploymentLog{
				Timestamp: time.Now().UTC(),
				Step:      domain.LogStepEsbuild,
				Message:   fmt.Sprintf("Synchronized %d environment variables and %d bindings from wrangler configuration", summary.ImportedVarsCount, summary.ImportedBindingsCount),
				Level:     domain.LogLevelInfo,
			})
		}
	}

	if len(bundle) == 0 {
		if app.InlineCode != "" {
			bundle = []byte(app.InlineCode)
		} else {
			bundle = []byte(domain.DefaultHelloWorldWorker)
		}
	}

	// Upload compiled bundle to storage
	if s.storage != nil {
		objectKey := fmt.Sprintf("deployments/%s/%s/bundle.js", app.Name, dep.ID)
		if err := s.storage.UploadBundle(ctx, s.fleetBucket, objectKey, bundle); err != nil {
			dep.MarkFailed(err.Error())
			_ = s.repo.Update(ctx, dep)
			return nil, fmt.Errorf("failed to upload bundle: %w", err)
		}
	}

	_ = s.repo.AppendLog(ctx, dep.ID, domain.DeploymentLog{
		Timestamp: time.Now().UTC(),
		Step:      domain.LogStepS3Upload,
		Message:   fmt.Sprintf("Worker bundle (%d bytes) stored successfully", len(bundle)),
		Level:     domain.LogLevelInfo,
	})

	// Publish celld deployment manifest, update deploy/current.json, and trigger fleet reload
	if err := s.publishCelldDeployment(ctx, app, dep, bundle, wranglerCfg); err != nil {
		dep.MarkFailed(err.Error())
		_ = s.repo.Update(ctx, dep)
		_ = s.repo.AppendLog(ctx, dep.ID, domain.DeploymentLog{
			Timestamp: time.Now().UTC(),
			Step:      domain.LogStepCelldDeploy,
			Message:   fmt.Sprintf("Celld direct deployment publication failed: %v", err),
			Level:     domain.LogLevelError,
		})
		return nil, err
	}

	// Supersede any existing active deployments
	allDeps, _ := s.repo.ListByAppID(ctx, app.ID)
	for _, oldDep := range allDeps {
		if oldDep.ID != dep.ID && oldDep.Status == domain.DeploymentStatusActive {
			oldDep.MarkSuperseded()
			_ = s.repo.Update(ctx, oldDep)
		}
	}

	_ = dep.StartDeploying(int64(len(bundle)))
	_ = dep.MarkActive()
	if err := s.repo.Update(ctx, dep); err != nil {
		return nil, err
	}

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

	_ = s.repo.AppendLog(ctx, dep.ID, domain.DeploymentLog{
		Timestamp: time.Now().UTC(),
		Step:      domain.LogStepCelldDeploy,
		Message:   fmt.Sprintf("Direct deployment %s is now active on %s", dep.VersionTag(), app.Subdomain),
		Level:     domain.LogLevelInfo,
	})

	return dep, nil
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

func (s *DeploymentService) buildGitWorker(ctx context.Context, app *domain.Application, dep *domain.Deployment) ([]byte, *wrangler.WranglerConfig) {
	repoPath := ""
	isTempDir := false

	// 1. Check if app.GitRepo points to a local folder or directory
	if stat, err := os.Stat(app.GitRepo); err == nil && stat.IsDir() {
		repoPath = app.GitRepo
	} else {
		// Attempt shallow clone
		tempDir, err := os.MkdirTemp("", "cubit-build-*")
		if err == nil {
			isTempDir = true
			defer func() {
				if isTempDir {
					_ = os.RemoveAll(tempDir)
				}
			}()

			repoURL := app.GitRepo
			if !strings.HasPrefix(repoURL, "http://") && !strings.HasPrefix(repoURL, "https://") && !strings.HasPrefix(repoURL, "git@") {
				repoURL = "https://github.com/" + repoURL
			}

			branch := app.Branch
			if branch == "" {
				branch = "main"
			}

			cloneCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
			defer cancel()

			cmd := exec.CommandContext(cloneCtx, "git", "clone", "--depth", "1", "-b", branch, repoURL, tempDir)
			if out, err := cmd.CombinedOutput(); err == nil {
				repoPath = tempDir
				_ = s.repo.AppendLog(ctx, dep.ID, domain.DeploymentLog{
					Timestamp: time.Now().UTC(),
					Step:      domain.LogStepGitClone,
					Message:   fmt.Sprintf("Cloned repository %s (%s)", app.GitRepo, branch),
					Level:     domain.LogLevelInfo,
				})
			} else {
				_ = s.repo.AppendLog(ctx, dep.ID, domain.DeploymentLog{
					Timestamp: time.Now().UTC(),
					Step:      domain.LogStepGitClone,
					Message:   fmt.Sprintf("Git clone skipped/fallback: %s", strings.TrimSpace(string(out))),
					Level:     domain.LogLevelWarn,
				})
			}
		}
	}

	if repoPath == "" {
		return []byte(domain.DefaultHelloWorldWorker), nil
	}

	workDir := repoPath
	if cleanRoot := domain.CleanRootDir(app.RootDir); cleanRoot != "" {
		targetDir := filepath.Join(repoPath, cleanRoot)
		if stat, err := os.Stat(targetDir); err == nil && stat.IsDir() {
			workDir = targetDir
			_ = s.repo.AppendLog(ctx, dep.ID, domain.DeploymentLog{
				Timestamp: time.Now().UTC(),
				Step:      domain.LogStepGitClone,
				Message:   fmt.Sprintf("Using configured root directory: %s", cleanRoot),
				Level:     domain.LogLevelInfo,
			})
		} else {
			_ = s.repo.AppendLog(ctx, dep.ID, domain.DeploymentLog{
				Timestamp: time.Now().UTC(),
				Step:      domain.LogStepGitClone,
				Message:   fmt.Sprintf("Configured root directory '%s' not found, falling back to repository root", cleanRoot),
				Level:     domain.LogLevelWarn,
			})
		}
	}

	// 2. Scan for wrangler.json, wrangler.jsonc, or wrangler.toml
	candidates := []string{
		"wrangler.json",
		"wrangler.jsonc",
		"wrangler.toml",
		"worker/wrangler.json",
		"worker/wrangler.toml",
	}

	var foundFile string
	var wranglerData []byte
	for _, c := range candidates {
		fp := filepath.Join(workDir, c)
		if data, err := os.ReadFile(fp); err == nil && len(data) > 0 {
			foundFile = c
			wranglerData = data
			break
		}
	}

	var entrypoint string
	var assetsDir string
	var parsedWrangler *wrangler.WranglerConfig
	if len(wranglerData) > 0 {
		if cfg, _, err := wrangler.Parse(wranglerData, "auto"); err == nil {
			if app.Branch != "" {
				parsedWrangler = wrangler.MergeEnvironment(cfg, app.Branch)
			} else {
				parsedWrangler = cfg
			}
		} else {
			_ = s.repo.AppendLog(ctx, dep.ID, domain.DeploymentLog{
				Timestamp: time.Now().UTC(),
				Step:      domain.LogStepGitClone,
				Message:   fmt.Sprintf("[wrangler] Warning: failed to parse config for bundler: %v", err),
				Level:     domain.LogLevelWarn,
			})
		}
		_, summary, err := s.appMgr.ImportWrangler(ctx, app.ID, string(wranglerData), "auto", app.Branch)
		if err == nil && summary != nil {
			_ = s.repo.AppendLog(ctx, dep.ID, domain.DeploymentLog{
				Timestamp: time.Now().UTC(),
				Step:      domain.LogStepGitClone,
				Message:   fmt.Sprintf("[wrangler] Detected %s: synced %d variables and %d bindings (compat: %s)", foundFile, summary.ImportedVarsCount, summary.ImportedBindingsCount, summary.CompatibilityDate),
				Level:     domain.LogLevelInfo,
			})
			if summary.Main != "" {
				entrypoint = summary.Main
			}
			if summary.AssetsDirectory != "" {
				assetsDir = summary.AssetsDirectory
			}
		} else if err != nil {
			_ = s.repo.AppendLog(ctx, dep.ID, domain.DeploymentLog{
				Timestamp: time.Now().UTC(),
				Step:      domain.LogStepGitClone,
				Message:   fmt.Sprintf("[wrangler] Warning: failed parsing %s: %v", foundFile, err),
				Level:     domain.LogLevelWarn,
			})
		}
	}

	if assetsDir == "" {
		for _, b := range app.Bindings {
			if b.Type == domain.BindingTypeAssets && b.ResourceID != "" {
				assetsDir = b.ResourceID
				break
			}
		}
	}

	// 3. Optional frontend build if package.json has a build script
	pkgJSONPath := filepath.Join(workDir, "package.json")
	if pkgData, err := os.ReadFile(pkgJSONPath); err == nil && len(pkgData) > 0 {
		var pkg struct {
			Scripts map[string]string `json:"scripts"`
		}
		if json.Unmarshal(pkgData, &pkg) == nil {
			if _, hasBuild := pkg.Scripts["build"]; hasBuild {
				targetAssetsDir := ""
				if assetsDir != "" {
					targetAssetsDir = filepath.Join(workDir, assetsDir)
				}
				needsBuild := true
				if targetAssetsDir != "" {
					if entries, err := os.ReadDir(targetAssetsDir); err == nil && len(entries) > 0 {
						needsBuild = false
					}
				}
				if needsBuild {
					buildCmdName := "npm"
					if _, err := exec.LookPath("pnpm"); err == nil {
						buildCmdName = "pnpm"
					}
					_ = s.repo.AppendLog(ctx, dep.ID, domain.DeploymentLog{
						Timestamp: time.Now().UTC(),
						Step:      domain.LogStepEsbuild,
						Message:   fmt.Sprintf("[%s] Installing dependencies and running build script...", buildCmdName),
						Level:     domain.LogLevelInfo,
					})
					instCtx, instCancel := context.WithTimeout(ctx, 60*time.Second)
					cmdInst := exec.CommandContext(instCtx, buildCmdName, "install")
					cmdInst.Dir = workDir
					_ = cmdInst.Run()
					instCancel()

					buildRunCtx, buildCancel := context.WithTimeout(ctx, 45*time.Second)
					cmdRun := exec.CommandContext(buildRunCtx, buildCmdName, "run", "build")
					cmdRun.Dir = workDir
					if out, err := cmdRun.CombinedOutput(); err == nil {
						_ = s.repo.AppendLog(ctx, dep.ID, domain.DeploymentLog{
							Timestamp: time.Now().UTC(),
							Step:      domain.LogStepEsbuild,
							Message:   fmt.Sprintf("[%s] Build script succeeded", buildCmdName),
							Level:     domain.LogLevelInfo,
						})
					} else {
						_ = s.repo.AppendLog(ctx, dep.ID, domain.DeploymentLog{
							Timestamp: time.Now().UTC(),
							Step:      domain.LogStepEsbuild,
							Message:   fmt.Sprintf("[%s] Build warning: %s", buildCmdName, strings.TrimSpace(string(out))),
							Level:     domain.LogLevelWarn,
						})
					}
					buildCancel()
				}
			}
		}
	}

	// 4. Upload static assets to storage if present
	if assetsDir != "" && s.storage != nil {
		targetAssetsDir := filepath.Join(workDir, assetsDir)
		if stat, err := os.Stat(targetAssetsDir); err == nil && stat.IsDir() {
			assetCount := 0
			var totalAssetBytes int64
			_ = filepath.Walk(targetAssetsDir, func(path string, info os.FileInfo, err error) error {
				if err != nil || info.IsDir() {
					return nil
				}
				rel, err := filepath.Rel(targetAssetsDir, path)
				if err != nil {
					return nil
				}
				relSlash := filepath.ToSlash(rel)
				data, err := os.ReadFile(path)
				if err != nil {
					return nil
				}
				objectKey := fmt.Sprintf("deployments/%s/%s/assets/%s", app.Name, dep.ID, strings.TrimPrefix(relSlash, "/"))
				if err := s.storage.UploadBundle(ctx, s.fleetBucket, objectKey, data); err == nil {
					assetCount++
					totalAssetBytes += int64(len(data))
				}
				return nil
			})
			if assetCount > 0 {
				_ = s.repo.AppendLog(ctx, dep.ID, domain.DeploymentLog{
					Timestamp: time.Now().UTC(),
					Step:      domain.LogStepS3Upload,
					Message:   fmt.Sprintf("Uploaded %d static assets (%d bytes) to storage", assetCount, totalAssetBytes),
					Level:     domain.LogLevelInfo,
				})
			}
		}
	}

	// 5. Detect and bundle worker entrypoint file
	if entrypoint == "" {
		entryCandidates := []string{
			"src/index.ts",
			"src/index.js",
			"src/worker.ts",
			"src/worker.js",
			"index.ts",
			"index.js",
		}
		for _, ec := range entryCandidates {
			if _, err := os.Stat(filepath.Join(workDir, ec)); err == nil {
				entrypoint = ec
				break
			}
		}
	}

	// Check if already bundled into dist-worker.js
	outPath := filepath.Join(workDir, "dist-worker.js")
	if bundled, err := os.ReadFile(outPath); err == nil && len(bundled) > 0 {
		_ = s.repo.AppendLog(ctx, dep.ID, domain.DeploymentLog{
			Timestamp: time.Now().UTC(),
			Step:      domain.LogStepEsbuild,
			Message:   fmt.Sprintf("[build] Using compiled worker bundle (%d bytes)", len(bundled)),
			Level:     domain.LogLevelInfo,
		})
		return bundled, parsedWrangler
	}

	if entrypoint != "" {
		entryPath := filepath.Join(workDir, entrypoint)

		buildCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
		defer cancel()
		buildArgs := wrangler.BuildEsbuildArgs(parsedWrangler, entryPath, outPath)
		if len(buildArgs) > 5 {
			customFlags := buildArgs[5:]
			_ = s.repo.AppendLog(ctx, dep.ID, domain.DeploymentLog{
				Timestamp: time.Now().UTC(),
				Step:      domain.LogStepEsbuild,
				Message:   fmt.Sprintf("[esbuild] Applied custom bundler rules & defines: %s", strings.Join(customFlags, " ")),
				Level:     domain.LogLevelInfo,
			})
		}

		cmdArgs := append([]string{"--yes", "esbuild"}, buildArgs...)
		cmd := exec.CommandContext(buildCtx, "npx", cmdArgs...)
		cmd.Dir = workDir
		if out, err := cmd.CombinedOutput(); err == nil {
			if bundled, err := os.ReadFile(outPath); err == nil && len(bundled) > 0 {
				_ = s.repo.AppendLog(ctx, dep.ID, domain.DeploymentLog{
					Timestamp: time.Now().UTC(),
					Step:      domain.LogStepEsbuild,
					Message:   fmt.Sprintf("[esbuild] Successfully bundled entrypoint '%s' (%d bytes)", entrypoint, len(bundled)),
					Level:     domain.LogLevelInfo,
				})
				return bundled, parsedWrangler
			}
		} else {
			_ = s.repo.AppendLog(ctx, dep.ID, domain.DeploymentLog{
				Timestamp: time.Now().UTC(),
				Step:      domain.LogStepEsbuild,
				Message:   fmt.Sprintf("[esbuild] Build error for '%s': %s", entrypoint, strings.TrimSpace(string(out))),
				Level:     domain.LogLevelWarn,
			})
		}
	}

	return []byte(domain.DefaultHelloWorldWorker), parsedWrangler
}

// updatePointerAndReload synchronizes the active deployment pointer in storage and triggers a zero-downtime hot reload.
func (s *DeploymentService) updatePointerAndReload(ctx context.Context, app *domain.Application, dep *domain.Deployment, manifest *CelldManifest) error {
	if s.storage != nil {
		manifestKey := fmt.Sprintf("deployments/%s/%s/manifest.json", app.Name, dep.ID)
		bundleKey := fmt.Sprintf("deployments/%s/%s/bundle.js", app.Name, dep.ID)

		if manifest != nil {
			manifestJSON, err := json.MarshalIndent(manifest, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal celld manifest: %w", err)
			}
			if err := s.storage.UploadBundle(ctx, s.fleetBucket, manifestKey, manifestJSON); err != nil {
				return fmt.Errorf("failed to upload celld manifest to %s: %w", manifestKey, err)
			}
		}

		pointer := BuildCurrentPointer(manifest, manifestKey, bundleKey)
		if pointer.Version == "" {
			pointer.Version = dep.ID
		}
		if pointer.ScriptName == "" {
			pointer.ScriptName = app.Name
		}
		pointerJSON, err := json.MarshalIndent(pointer, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to format current pointer: %w", err)
		}

		if err := s.storage.UploadBundle(ctx, s.fleetBucket, "deploy/current.json", pointerJSON); err != nil {
			return fmt.Errorf("failed to upload deploy/current.json: %w", err)
		}
		if err := s.storage.UploadBundle(ctx, s.fleetBucket, fmt.Sprintf("deployments/%s/current.json", app.Name), pointerJSON); err != nil {
			return fmt.Errorf("failed to upload app current.json: %w", err)
		}

		digest := ""
		if manifest != nil && len(manifest.Modules) > 0 {
			digest = manifest.Modules[0].Digest
		}
		_ = s.repo.AppendLog(ctx, dep.ID, domain.DeploymentLog{
			Timestamp: time.Now().UTC(),
			Step:      domain.LogStepS3Upload,
			Message:   fmt.Sprintf("Published celld manifest and updated deploy/current.json (%s)", digest),
			Level:     domain.LogLevelInfo,
		})
	}

	if s.reloader != nil {
		if err := s.reloader.ReloadFleet(ctx); err != nil {
			_ = s.repo.AppendLog(ctx, dep.ID, domain.DeploymentLog{
				Timestamp: time.Now().UTC(),
				Step:      domain.LogStepCelldDeploy,
				Message:   fmt.Sprintf("Celld fleet reload warning: %v", err),
				Level:     domain.LogLevelWarn,
			})
			return fmt.Errorf("celld fleet reload failed: %w", err)
		}
		_ = s.repo.AppendLog(ctx, dep.ID, domain.DeploymentLog{
			Timestamp: time.Now().UTC(),
			Step:      domain.LogStepCelldDeploy,
			Message:   "Issued zero-downtime hot reload across celld fleet (POST /reload)",
			Level:     domain.LogLevelInfo,
		})
	}

	return nil
}

// publishCelldDeployment builds and uploads the celld manifest, updates deploy/current.json, and triggers hot reload.
func (s *DeploymentService) publishCelldDeployment(ctx context.Context, app *domain.Application, dep *domain.Deployment, bundle []byte, cfg *wrangler.WranglerConfig) error {
	manifest, err := BuildCelldManifest(app, dep, bundle, cfg)
	if err != nil {
		_ = s.repo.AppendLog(ctx, dep.ID, domain.DeploymentLog{
			Timestamp: time.Now().UTC(),
			Step:      domain.LogStepS3Upload,
			Message:   fmt.Sprintf("Failed to generate celld manifest: %v", err),
			Level:     domain.LogLevelWarn,
		})
		return fmt.Errorf("failed to generate celld manifest: %w", err)
	}

	return s.updatePointerAndReload(ctx, app, dep, manifest)
}

