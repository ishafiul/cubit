package deployment_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ishaf/cubit/internal/domain"
	"github.com/ishaf/cubit/internal/modules/deployment"
	"github.com/ishaf/cubit/internal/modules/wrangler"
)

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
	})
}
