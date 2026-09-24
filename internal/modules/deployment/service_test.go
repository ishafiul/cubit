package deployment_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ishaf/cubit/internal/domain"
	"github.com/ishaf/cubit/internal/modules/deployment"
	"github.com/ishaf/cubit/internal/modules/wrangler"
)

type mockStorageUploader struct {
	mu      sync.Mutex
	uploads map[string][]byte
}

func newMockStorageUploader() *mockStorageUploader {
	return &mockStorageUploader{
		uploads: make(map[string][]byte),
	}
}

func (m *mockStorageUploader) UploadBundle(ctx context.Context, bucket, objectKey string, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.uploads[objectKey] = data
	return nil
}

type mockFleetReloader struct {
	mu          sync.Mutex
	reloadCalls int
}

func (m *mockFleetReloader) ReloadFleet(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.reloadCalls++
	return nil
}

type mockNodeLister struct {
	nodes []*domain.Node
}

func (m *mockNodeLister) List(ctx context.Context) ([]*domain.Node, error) {
	return m.nodes, nil
}

type mockAppManager struct {
	app *domain.Application
}

func (m *mockAppManager) GetByID(ctx context.Context, id string) (*domain.Application, error) {
	if m.app != nil && m.app.ID == id {
		return m.app, nil
	}
	return nil, domain.NewNotFoundError("application not found: " + id)
}

func (m *mockAppManager) SetActiveDeployment(ctx context.Context, appID, deploymentID string) error {
	if m.app != nil && m.app.ID == appID {
		m.app.ActiveDeploymentID = deploymentID
		return nil
	}
	return domain.NewNotFoundError("application not found: " + appID)
}

func (m *mockAppManager) ImportWrangler(ctx context.Context, appID string, rawConfig string, format string, envName string) (*domain.Application, *wrangler.ImportSummary, error) {
	if m.app != nil && m.app.ID == appID {
		cfg, detectedFormat, err := wrangler.Parse([]byte(rawConfig), format)
		if err != nil {
			return nil, nil, err
		}
		summary, err := wrangler.ApplyToApplication(m.app, cfg, envName, detectedFormat)
		return m.app, summary, err
	}
	return nil, nil, domain.NewNotFoundError("application not found: " + appID)
}

type mockDeploymentRepo struct {
	deps map[string]*domain.Deployment
	logs map[string][]domain.DeploymentLog
}

func newMockDeploymentRepo() *mockDeploymentRepo {
	return &mockDeploymentRepo{
		deps: make(map[string]*domain.Deployment),
		logs: make(map[string][]domain.DeploymentLog),
	}
}

func (m *mockDeploymentRepo) Save(ctx context.Context, dep *domain.Deployment) error {
	m.deps[dep.ID] = dep
	return nil
}

func (m *mockDeploymentRepo) GetByID(ctx context.Context, id string) (*domain.Deployment, error) {
	if d, ok := m.deps[id]; ok {
		return d, nil
	}
	return nil, domain.NewNotFoundError("deployment not found: " + id)
}

func (m *mockDeploymentRepo) ListByAppID(ctx context.Context, appID string) ([]*domain.Deployment, error) {
	var list []*domain.Deployment
	for _, d := range m.deps {
		if d.ApplicationID == appID {
			list = append(list, d)
		}
	}
	return list, nil
}

func (m *mockDeploymentRepo) GetLatestBuildVersion(ctx context.Context, appID string) (int, error) {
	maxVer := 0
	for _, d := range m.deps {
		if d.ApplicationID == appID && d.BuildVersion > maxVer {
			maxVer = d.BuildVersion
		}
	}
	return maxVer, nil
}

func (m *mockDeploymentRepo) Update(ctx context.Context, dep *domain.Deployment) error {
	m.deps[dep.ID] = dep
	return nil
}

func (m *mockDeploymentRepo) AppendLog(ctx context.Context, deploymentID string, entry domain.DeploymentLog) error {
	m.logs[deploymentID] = append(m.logs[deploymentID], entry)
	return nil
}

func (m *mockDeploymentRepo) GetLogs(ctx context.Context, deploymentID string) ([]domain.DeploymentLog, error) {
	return m.logs[deploymentID], nil
}

func TestDeploymentService(t *testing.T) {
	t.Run("Given an application ready for deployment", func(t *testing.T) {
		app := &domain.Application{
			ID:         "app-1",
			Name:       "worker-one",
			SourceType: domain.SourceTypeInline,
			InlineCode: "export default { fetch() { return new Response('ok'); } }",
			CreatedAt:  time.Now().UTC(),
		}
		appMgr := &mockAppManager{app: app}
		repo := newMockDeploymentRepo()
		svc := deployment.NewService(repo, appMgr, nil, nil, "cubit-fleet")

		t.Run("When triggering a first deployment", func(t *testing.T) {
			dep, err := svc.Deploy(context.Background(), "app-1", "commit-abc")

			t.Run("Then build version is 1 and status is active", func(t *testing.T) {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if dep.BuildVersion != 1 {
					t.Fatalf("expected build version 1, got %d", dep.BuildVersion)
				}
				if dep.Status != domain.DeploymentStatusActive {
					t.Fatalf("expected status active, got %s", dep.Status)
				}
				if app.ActiveDeploymentID != dep.ID {
					t.Fatalf("expected active deployment on app %s, got %s", dep.ID, app.ActiveDeploymentID)
				}
			})
		})

		t.Run("When triggering a subsequent deployment", func(t *testing.T) {
			dep2, err := svc.DeployWithDetails(context.Background(), "app-1", "commit-def", "feat: updates")

			t.Run("Then build version increments to 2 and supersedes version 1", func(t *testing.T) {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if dep2.BuildVersion != 2 {
					t.Fatalf("expected build version 2, got %d", dep2.BuildVersion)
				}
				if dep2.CommitMessage != "feat: updates" {
					t.Fatalf("expected commit message 'feat: updates', got %s", dep2.CommitMessage)
				}
			})
		})

		t.Run("When rolling back to a previous deployment version", func(t *testing.T) {
			deps, _ := svc.ListByApp(context.Background(), "app-1")
			var dep1ID string
			for _, d := range deps {
				if d.BuildVersion == 1 {
					dep1ID = d.ID
					break
				}
			}

			rolledBack, err := svc.Rollback(context.Background(), "app-1", dep1ID)

			t.Run("Then deployment becomes active and app active deployment matches", func(t *testing.T) {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if rolledBack.ID != dep1ID {
					t.Fatalf("expected rolled back ID %s, got %s", dep1ID, rolledBack.ID)
				}
				if rolledBack.Status != domain.DeploymentStatusActive {
					t.Fatalf("expected status active, got %s", rolledBack.Status)
				}
				if app.ActiveDeploymentID != dep1ID {
					t.Fatalf("expected app active deployment to be %s, got %s", dep1ID, app.ActiveDeploymentID)
				}

				allDeps, _ := svc.ListByApp(context.Background(), "app-1")
				for _, d := range allDeps {
					if d.BuildVersion == 2 && d.Status != domain.DeploymentStatusSuperseded {
						t.Fatalf("expected version 2 to be superseded, got %s", d.Status)
					}
				}
			})
		})

		t.Run("When rolling back to a failed deployment", func(t *testing.T) {
			failedDep, _ := domain.NewDeploymentWithVersion("failed-dep", "app-1", "bad-hash", "failed", 3)
			failedDep.MarkFailed("syntax error")
			_ = repo.Save(context.Background(), failedDep)

			_, err := svc.Rollback(context.Background(), "app-1", "failed-dep")

			t.Run("Then rollback is rejected with validation error", func(t *testing.T) {
				if err == nil {
					t.Fatal("expected error rolling back to failed deployment, got nil")
				}
			})
		})

		t.Run("When deploying a Git application containing a wrangler.json file", func(t *testing.T) {
			// Create a temp repository folder simulating a cloned repo
			tempRepo, err := os.MkdirTemp("", "test-repo-*")
			if err != nil {
				t.Fatalf("failed to create temp repo dir: %v", err)
			}
			defer os.RemoveAll(tempRepo)

			wranglerContent := `{
				"name": "auto-pickup-worker",
				"compatibility_date": "2024-11-20",
				"compatibility_flags": ["nodejs_compat"],
				"vars": {
					"REPO_VAR": "detected-from-git",
					"EXISTING_SECRET": "wrangler-ignored"
				},
				"d1_databases": [
					{ "binding": "GIT_DB", "database_name": "prod-d1" }
				]
			}`
			_ = os.WriteFile(filepath.Join(tempRepo, "wrangler.json"), []byte(wranglerContent), 0644)
			_ = os.WriteFile(filepath.Join(tempRepo, "index.js"), []byte("export default { fetch() { return new Response('git worker'); } };"), 0644)

			gitApp, err := domain.NewApplicationWithSource(
				"git-app-1",
				"git-worker",
				domain.SourceTypeGit,
				"github.com/example/git-worker",
				"main",
				"",
				[]domain.EnvironmentVariable{
					{Key: "EXISTING_SECRET", Value: "topsecretpass", IsSecret: true},
				},
				nil,
			)
			if err != nil {
				t.Fatalf("failed to create git app: %v", err)
			}
			gitApp.GitRepo = tempRepo

			gitAppMgr := &mockAppManager{app: gitApp}
			gitDepRepo := newMockDeploymentRepo()
			gitSvc := deployment.NewService(gitDepRepo, gitAppMgr, nil, nil, "cubit-fleet")

			dep, err := gitSvc.Deploy(context.Background(), gitApp.ID, "commit123")

			t.Run("Then wrangler.json is automatically parsed and applied to the application", func(t *testing.T) {
				if err != nil {
					t.Fatalf("unexpected deployment error: %v", err)
				}
				if dep.Status != domain.DeploymentStatusActive {
					t.Fatalf("expected active deployment, got: %s", dep.Status)
				}

				// Verify logs recorded the wrangler pickup
				logs, _ := gitSvc.GetLogs(context.Background(), dep.ID)
				foundWranglerLog := false
				for _, l := range logs {
					if strings.Contains(l.Message, "[wrangler] Detected wrangler.json") {
						foundWranglerLog = true
						break
					}
				}
				if !foundWranglerLog {
					t.Errorf("expected wrangler detection log entry in deployment logs, got: %+v", logs)
				}

				// Verify application state was updated
				if gitApp.CompatibilityDate != "2024-11-20" {
					t.Errorf("expected compat date 2024-11-20, got: %s", gitApp.CompatibilityDate)
				}

				vMap := make(map[string]domain.EnvironmentVariable)
				for _, v := range gitApp.EnvVars {
					vMap[v.Key] = v
				}

				if vMap["REPO_VAR"].Value != "detected-from-git" {
					t.Errorf("expected REPO_VAR detected-from-git, got: %s", vMap["REPO_VAR"].Value)
				}
				if !vMap["EXISTING_SECRET"].IsSecret {
					t.Errorf("expected EXISTING_SECRET to maintain IsSecret=true")
				}
				if len(gitApp.Bindings) != 1 || gitApp.Bindings[0].Name != "GIT_DB" {
					t.Errorf("expected GIT_DB binding, got: %v", gitApp.Bindings)
				}
			})
		})

		t.Run("Given a Git application with custom bundler rules and defines", func(t *testing.T) {
			tempRepo, err := os.MkdirTemp("", "test-repo-rules-*")
			if err != nil {
				t.Fatalf("failed to create temp repo dir: %v", err)
			}
			defer os.RemoveAll(tempRepo)

			wranglerContent := `{
				"name": "bundler-rules-worker",
				"rules": [
					{ "type": "Text", "globs": ["**/*.txt"] },
					{ "type": "CompiledWasm", "globs": ["**/*.wasm"] }
				],
				"define": {
					"APP_ENV": "\"staging\""
				}
			}`
			_ = os.WriteFile(filepath.Join(tempRepo, "wrangler.json"), []byte(wranglerContent), 0644)
			_ = os.WriteFile(filepath.Join(tempRepo, "index.js"), []byte("export default { fetch() { return new Response('rules worker'); } };"), 0644)

			gitApp, err := domain.NewApplicationWithSource(
				"git-app-rules",
				"git-rules-worker",
				domain.SourceTypeGit,
				"github.com/example/rules-worker",
				"main",
				"",
				nil,
				nil,
			)
			if err != nil {
				t.Fatalf("failed to create git app: %v", err)
			}
			gitApp.GitRepo = tempRepo

			gitAppMgr := &mockAppManager{app: gitApp}
			gitDepRepo := newMockDeploymentRepo()
			gitSvc := deployment.NewService(gitDepRepo, gitAppMgr, nil, nil, "cubit-fleet")

			t.Run("When deploying the application", func(t *testing.T) {
				dep, err := gitSvc.Deploy(context.Background(), gitApp.ID, "commit456")
				if err != nil {
					t.Fatalf("unexpected deployment error: %v", err)
				}

				t.Run("Then custom bundler rules and defines are applied and recorded in deployment logs", func(t *testing.T) {
					if dep.Status != domain.DeploymentStatusActive {
						t.Fatalf("expected active deployment, got: %s", dep.Status)
					}

					logs, _ := gitSvc.GetLogs(context.Background(), dep.ID)
					foundBundlerLog := false
					for _, l := range logs {
						if strings.Contains(l.Message, "[esbuild] Applied custom bundler rules & defines:") {
							foundBundlerLog = true
							if !strings.Contains(l.Message, "--loader:.txt=text") {
								t.Errorf("expected --loader:.txt=text in log message, got %s", l.Message)
							}
							if !strings.Contains(l.Message, "--loader:.wasm=binary") {
								t.Errorf("expected --loader:.wasm=binary in log message, got %s", l.Message)
							}
							if !strings.Contains(l.Message, `--define:APP_ENV="staging"`) {
								t.Errorf(`expected --define:APP_ENV="staging" in log message, got %s`, l.Message)
							}
							break
						}
					}
					if !foundBundlerLog {
						t.Errorf("expected custom bundler log in deployment logs, got: %+v", logs)
					}
				})
			})
		})
	})
}

func TestDeploymentService_CelldManifestAndReload(t *testing.T) {
	t.Run("Given an application with storage uploader and fleet reloader configured", func(t *testing.T) {
		app := &domain.Application{
			ID:         "app-celld-1",
			Name:       "celld-worker",
			SourceType: domain.SourceTypeInline,
			InlineCode: "export default { fetch() { return new Response('hello celld'); } }",
			Bindings: []domain.ResourceBinding{
				{Type: domain.BindingTypeDurableObject, ClassName: "CounterDO", Name: "COUNTER"},
			},
			CreatedAt: time.Now().UTC(),
		}

		appMgr := &mockAppManager{app: app}
		repo := newMockDeploymentRepo()
		storage := newMockStorageUploader()
		reloader := &mockFleetReloader{}

		svc := deployment.NewService(repo, appMgr, storage, nil, "cubit-fleet")
		svc.WithFleetReloader(reloader)

		var dep1 *domain.Deployment

		t.Run("When deploying the application via DeployWithDetails", func(t *testing.T) {
			var err error
			dep1, err = svc.DeployWithDetails(context.Background(), app.ID, "hash1", "initial deploy")
			if err != nil {
				t.Fatalf("unexpected deploy error: %v", err)
			}

			t.Run("Then manifest.json and deploy/current.json are uploaded to storage and fleet is reloaded", func(t *testing.T) {
				manifestKey := fmt.Sprintf("deployments/%s/%s/manifest.json", app.Name, dep1.ID)
				manifestBytes, ok := storage.uploads[manifestKey]
				if !ok {
					t.Fatalf("expected manifest uploaded to %s", manifestKey)
				}

				var manifest deployment.CelldManifest
				if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
					t.Fatalf("failed to parse manifest JSON: %v", err)
				}

				if manifest.ScriptName != app.Name {
					t.Errorf("expected script_name %s, got %s", app.Name, manifest.ScriptName)
				}
				if len(manifest.DOClasses) != 1 || manifest.DOClasses[0] != "CounterDO" {
					t.Errorf("expected DO class CounterDO, got %v", manifest.DOClasses)
				}
				if len(manifest.Modules) == 0 || !strings.HasPrefix(manifest.Modules[0].Digest, "sha256:") {
					t.Errorf("expected module with sha256 digest, got: %+v", manifest.Modules)
				}

				// Check deploy/current.json
				currentBytes, ok := storage.uploads["deploy/current.json"]
				if !ok {
					t.Fatalf("expected deploy/current.json to be uploaded")
				}

				var currentPtr deployment.CelldCurrentPointer
				if err := json.Unmarshal(currentBytes, &currentPtr); err != nil {
					t.Fatalf("failed to parse current.json: %v", err)
				}

				if currentPtr.Version != dep1.ID {
					t.Errorf("expected current.json version %s, got %s", dep1.ID, currentPtr.Version)
				}
				if currentPtr.ManifestPath != manifestKey {
					t.Errorf("expected current.json manifest_path %s, got %s", manifestKey, currentPtr.ManifestPath)
				}

				// Check fleet reload was triggered
				if reloader.reloadCalls != 1 {
					t.Errorf("expected 1 reload call, got %d", reloader.reloadCalls)
				}
			})
		})

		t.Run("When deploying directly via DeployDirect", func(t *testing.T) {
			directBundle := []byte("export default { fetch() { return new Response('direct v2'); } };")
			dep2, err := svc.DeployDirect(context.Background(), app.ID, directBundle, "direct v2 release", "")
			if err != nil {
				t.Fatalf("unexpected direct deploy error: %v", err)
			}

			t.Run("Then deploy/current.json is updated to dep2 and reload is called again", func(t *testing.T) {
				currentBytes := storage.uploads["deploy/current.json"]
				var currentPtr deployment.CelldCurrentPointer
				_ = json.Unmarshal(currentBytes, &currentPtr)

				if currentPtr.Version != dep2.ID {
					t.Errorf("expected current.json updated to %s, got %s", dep2.ID, currentPtr.Version)
				}
				if reloader.reloadCalls != 2 {
					t.Errorf("expected 2 reload calls, got %d", reloader.reloadCalls)
				}
			})
		})

		t.Run("When rolling back to the first deployment", func(t *testing.T) {
			reloader.reloadCalls = 0
			rbDep, err := svc.Rollback(context.Background(), app.ID, dep1.ID)
			if err != nil {
				t.Fatalf("unexpected rollback error: %v", err)
			}

			t.Run("Then deploy/current.json points back to dep1 and reload is triggered", func(t *testing.T) {
				if rbDep.ID != dep1.ID {
					t.Fatalf("expected rolled back dep %s, got %s", dep1.ID, rbDep.ID)
				}

				currentBytes := storage.uploads["deploy/current.json"]
				var currentPtr deployment.CelldCurrentPointer
				_ = json.Unmarshal(currentBytes, &currentPtr)

				if currentPtr.Version != dep1.ID {
					t.Errorf("expected current.json rolled back to %s, got %s", dep1.ID, currentPtr.Version)
				}
				if reloader.reloadCalls != 1 {
					t.Errorf("expected 1 reload call during rollback, got %d", reloader.reloadCalls)
				}
			})
		})
	})
}

func TestCelldFleetReloader(t *testing.T) {
	t.Run("Given a fleet with active and non-active nodes", func(t *testing.T) {
		activeNodeHit := false

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost && r.URL.Path == "/reload" {
				activeNodeHit = true
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"status":"reloaded"}`))
			}
		}))
		defer server.Close()

		hostPort := strings.TrimPrefix(server.URL, "http://")
		parts := strings.Split(hostPort, ":")
		ip := parts[0]
		port := 8081
		if len(parts) > 1 {
			fmt.Sscanf(parts[1], "%d", &port)
		}

		activeNode := &domain.Node{
			ID:           "node-active-1",
			IPAddress:    ip,
			InternalPort: port,
			Status:       domain.NodeStatusActive,
		}
		drainingNode := &domain.Node{
			ID:           "node-draining-2",
			IPAddress:    "192.0.2.1",
			InternalPort: 8081,
			Status:       domain.NodeStatusDraining,
		}

		nodeLister := &mockNodeLister{
			nodes: []*domain.Node{activeNode, drainingNode},
		}

		reloader := deployment.NewCelldFleetReloader(nodeLister, server.Client())

		t.Run("When ReloadFleet is invoked", func(t *testing.T) {
			err := reloader.ReloadFleet(context.Background())

			t.Run("Then only the active node receives POST /reload", func(t *testing.T) {
				if err != nil {
					t.Fatalf("expected no error, got: %v", err)
				}
				if !activeNodeHit {
					t.Errorf("expected active node to receive POST /reload")
				}
			})
		})
	})
}

