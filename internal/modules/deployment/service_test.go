package deployment_test

import (
	"context"
	"testing"
	"time"

	"github.com/ishaf/cubit/internal/domain"
	"github.com/ishaf/cubit/internal/modules/deployment"
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
	})
}
