package usecase_test

import (
	"context"
	"testing"

	"github.com/ishaf/cubit/internal/domain"
	"github.com/ishaf/cubit/internal/usecase"
)

// Mock ApplicationRepository
type mockAppRepo struct {
	apps map[string]*domain.Application
}

func newMockAppRepo() *mockAppRepo {
	return &mockAppRepo{apps: make(map[string]*domain.Application)}
}

func (m *mockAppRepo) Save(ctx context.Context, app *domain.Application) error {
	m.apps[app.ID] = app
	return nil
}

func (m *mockAppRepo) GetByID(ctx context.Context, id string) (*domain.Application, error) {
	app, exists := m.apps[id]
	if !exists {
		return nil, domain.NewNotFoundError("app not found")
	}
	return app, nil
}

func (m *mockAppRepo) List(ctx context.Context) ([]*domain.Application, error) {
	var list []*domain.Application
	for _, a := range m.apps {
		list = append(list, a)
	}
	return list, nil
}

func (m *mockAppRepo) Delete(ctx context.Context, id string) error {
	delete(m.apps, id)
	return nil
}

func (m *mockAppRepo) Update(ctx context.Context, app *domain.Application) error {
	m.apps[app.ID] = app
	return nil
}

// Mock DeploymentRepository
type mockDepRepo struct {
	deps map[string]*domain.Deployment
	logs map[string][]domain.DeploymentLog
}

func newMockDepRepo() *mockDepRepo {
	return &mockDepRepo{
		deps: make(map[string]*domain.Deployment),
		logs: make(map[string][]domain.DeploymentLog),
	}
}

func (m *mockDepRepo) Save(ctx context.Context, dep *domain.Deployment) error {
	m.deps[dep.ID] = dep
	return nil
}

func (m *mockDepRepo) GetByID(ctx context.Context, id string) (*domain.Deployment, error) {
	dep, exists := m.deps[id]
	if !exists {
		return nil, domain.NewNotFoundError("deployment not found")
	}
	return dep, nil
}

func (m *mockDepRepo) ListByAppID(ctx context.Context, appID string) ([]*domain.Deployment, error) {
	var list []*domain.Deployment
	for _, d := range m.deps {
		if d.ApplicationID == appID {
			list = append(list, d)
		}
	}
	return list, nil
}

func (m *mockDepRepo) Update(ctx context.Context, dep *domain.Deployment) error {
	m.deps[dep.ID] = dep
	return nil
}

func (m *mockDepRepo) AppendLog(ctx context.Context, deploymentID string, entry domain.DeploymentLog) error {
	m.logs[deploymentID] = append(m.logs[deploymentID], entry)
	return nil
}

func (m *mockDepRepo) GetLogs(ctx context.Context, deploymentID string) ([]domain.DeploymentLog, error) {
	return m.logs[deploymentID], nil
}

// Mock StoragePort
type mockStoragePort struct {
	uploadedBundles map[string][]byte
}

func newMockStoragePort() *mockStoragePort {
	return &mockStoragePort{uploadedBundles: make(map[string][]byte)}
}

func (m *mockStoragePort) EnsureBucket(ctx context.Context, bucketName string) error {
	return nil
}

func (m *mockStoragePort) UploadBundle(ctx context.Context, bucketName, objectKey string, data []byte) error {
	m.uploadedBundles[objectKey] = data
	return nil
}

func (m *mockStoragePort) CheckHealth(ctx context.Context) error {
	return nil
}

func (m *mockStoragePort) DriverName() string {
	return "mock_storage"
}

// Mock ProxyPort
type mockProxyPort struct {
	syncedRules []usecase.RouteRule
}

func newMockProxyPort() *mockProxyPort {
	return &mockProxyPort{}
}

func (m *mockProxyPort) SyncRoutes(ctx context.Context, routes []usecase.RouteRule) error {
	m.syncedRules = routes
	return nil
}

// Mock DomainRepository
type mockDomainRepo struct {
	domains map[string]*domain.Domain
}

func newMockDomainRepo() *mockDomainRepo {
	return &mockDomainRepo{domains: make(map[string]*domain.Domain)}
}

func (m *mockDomainRepo) Save(ctx context.Context, dom *domain.Domain) error {
	m.domains[dom.ID] = dom
	return nil
}

func (m *mockDomainRepo) GetByID(ctx context.Context, id string) (*domain.Domain, error) {
	dom, exists := m.domains[id]
	if !exists {
		return nil, domain.NewNotFoundError("domain not found")
	}
	return dom, nil
}

func (m *mockDomainRepo) List(ctx context.Context) ([]*domain.Domain, error) {
	var list []*domain.Domain
	for _, d := range m.domains {
		list = append(list, d)
	}
	return list, nil
}

func (m *mockDomainRepo) ListByAppID(ctx context.Context, appID string) ([]*domain.Domain, error) {
	var list []*domain.Domain
	for _, d := range m.domains {
		if d.ApplicationID == appID {
			list = append(list, d)
		}
	}
	return list, nil
}

func (m *mockDomainRepo) Delete(ctx context.Context, id string) error {
	delete(m.domains, id)
	return nil
}

func (m *mockDomainRepo) Update(ctx context.Context, dom *domain.Domain) error {
	m.domains[dom.ID] = dom
	return nil
}

func TestAppUsecase(t *testing.T) {
	ctx := context.Background()

	t.Run("Given initialized repositories, storage and proxy", func(t *testing.T) {
		appRepo := newMockAppRepo()
		depRepo := newMockDepRepo()
		nodeRepo := newMockNodeRepo()
		domainRepo := newMockDomainRepo()
		storage := newMockStoragePort()
		proxy := newMockProxyPort()

		// Register an active node so deployment has a routing target
		activeNode, _ := domain.NewNode("n1", "node-1", "10.0.0.1", 8081, 8080, "0.2.0")
		_ = nodeRepo.Save(ctx, activeNode)

		uc := usecase.NewAppUsecase(appRepo, depRepo, nodeRepo, domainRepo, storage, proxy, "fleet-bucket")

		t.Run("When creating an application then it saves with created status", func(t *testing.T) {
			app, err := uc.CreateApplication(ctx, "my-worker", "https://github.com/org/worker", "main", nil, nil)

			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if app.Name != "my-worker" {
				t.Errorf("expected name my-worker, got %s", app.Name)
			}
		})

		t.Run("When deploying an existing application", func(t *testing.T) {
			app, _ := uc.CreateApplication(ctx, "api-worker", "https://github.com/org/api", "main", nil, nil)

			dep, err := uc.DeployApplication(ctx, app.ID, "commit-123")

			t.Run("Then deployment succeeds and transitions to active status", func(t *testing.T) {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if dep.Status != domain.DeploymentStatusActive {
					t.Errorf("expected active deployment, got %s", dep.Status)
				}
			})

			t.Run("Then application marks the deployment as active", func(t *testing.T) {
				updatedApp, _ := appRepo.GetByID(ctx, app.ID)
				if updatedApp.ActiveDeploymentID != dep.ID {
					t.Errorf("expected active deployment %s, got %s", dep.ID, updatedApp.ActiveDeploymentID)
				}
				if updatedApp.Status != domain.AppStatusRunning {
					t.Errorf("expected status running, got %s", updatedApp.Status)
				}
			})

			t.Run("Then bundle is uploaded to storage port", func(t *testing.T) {
				if len(storage.uploadedBundles) == 0 {
					t.Error("expected at least one bundle uploaded to storage")
				}
			})
		})
	})
}
