package application_test

import (
	"context"
	"testing"

	"github.com/ishaf/cubit/internal/domain"
	"github.com/ishaf/cubit/internal/modules/application"
)

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

func (m *mockAppRepo) Update(ctx context.Context, app *domain.Application) error {
	if _, ok := m.apps[app.ID]; !ok {
		return domain.NewNotFoundError("application not found: " + app.ID)
	}
	m.apps[app.ID] = app
	return nil
}

func (m *mockAppRepo) GetByID(ctx context.Context, id string) (*domain.Application, error) {
	if app, ok := m.apps[id]; ok {
		return app, nil
	}
	return nil, domain.NewNotFoundError("application not found: " + id)
}

func (m *mockAppRepo) GetBySubdomain(ctx context.Context, subdomain string) (*domain.Application, error) {
	for _, a := range m.apps {
		if a.Subdomain == subdomain || a.Name == subdomain {
			return a, nil
		}
	}
	return nil, domain.NewNotFoundError("application not found: " + subdomain)
}

func (m *mockAppRepo) List(ctx context.Context) ([]*domain.Application, error) {
	var list []*domain.Application
	for _, a := range m.apps {
		list = append(list, a)
	}
	return list, nil
}

func (m *mockAppRepo) Delete(ctx context.Context, id string) error {
	if _, ok := m.apps[id]; !ok {
		return domain.NewNotFoundError("application not found: " + id)
	}
	delete(m.apps, id)
	return nil
}

func (m *mockAppRepo) ListByGitRepoAndBranch(ctx context.Context, repo, branch string) ([]*domain.Application, error) {
	var list []*domain.Application
	for _, a := range m.apps {
		if a.GitRepo == repo && a.Branch == branch {
			list = append(list, a)
		}
	}
	return list, nil
}

func TestApplicationService(t *testing.T) {
	t.Run("Given a fresh ApplicationService", func(t *testing.T) {
		repo := newMockAppRepo()
		svc := application.NewService(repo, nil, nil, "cubit-fleet")

		t.Run("When creating an application with valid details", func(t *testing.T) {
			app, err := svc.Create(
				context.Background(),
				"test-worker",
				domain.SourceTypeGit,
				"https://github.com/example/worker",
				"main",
				"",
				true,
				nil,
				nil,
			)

			t.Run("Then creation succeeds without error", func(t *testing.T) {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if app == nil {
					t.Fatal("expected created application, got nil")
				}
				if app.Name != "test-worker" {
					t.Fatalf("expected name test-worker, got %s", app.Name)
				}
				if !app.AutoDeploy {
					t.Fatal("expected autoDeploy to be true")
				}
			})
		})

		t.Run("When updating application configuration and runtime settings", func(t *testing.T) {
			app, _ := svc.Create(
				context.Background(),
				"update-worker",
				domain.SourceTypeGit,
				"https://github.com/example/worker",
				"main",
				"",
				true,
				nil,
				nil,
			)

			autoDeployFalse := false
			cDate := "2024-09-23"
			flags := []string{"nodejs_compat"}
			mem := 256
			dur := 100
			updated, err := svc.Update(context.Background(), app.ID, "staging", "", &autoDeployFalse, nil, nil, &cDate, &flags, &mem, &dur)

			t.Run("Then configuration and runtime settings are persisted", func(t *testing.T) {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if updated.Branch != "staging" {
					t.Fatalf("expected branch staging, got %s", updated.Branch)
				}
				if updated.AutoDeploy != false {
					t.Fatal("expected autoDeploy to be false")
				}
				if updated.CompatibilityDate != "2024-09-23" {
					t.Fatalf("expected compat date 2024-09-23, got %s", updated.CompatibilityDate)
				}
				if len(updated.CompatibilityFlags) != 1 || updated.CompatibilityFlags[0] != "nodejs_compat" {
					t.Fatalf("expected nodejs_compat flag, got %v", updated.CompatibilityFlags)
				}
				if updated.MemoryLimitMB != 256 {
					t.Fatalf("expected memory limit 256, got %d", updated.MemoryLimitMB)
				}
				if updated.MaxDurationMs != 100 {
					t.Fatalf("expected max duration 100, got %d", updated.MaxDurationMs)
				}
			})
		})

		t.Run("When tracking request execution metrics", func(t *testing.T) {
			app, _ := svc.Create(
				context.Background(),
				"metrics-worker",
				domain.SourceTypeInline,
				"",
				"main",
				"export default {}",
				true,
				nil,
				nil,
			)

			svc.RecordExecution(app.ID, "GET", "/api", 200, 10.0, "127.0.0.1", "ok")
			svc.RecordExecution(app.ID, "POST", "/submit", 500, 20.0, "127.0.0.1", "error")

			metrics, err := svc.GetMetrics(context.Background(), app.ID)

			t.Run("Then aggregated request metrics are calculated accurately", func(t *testing.T) {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if metrics.TotalRequests != 2 {
					t.Fatalf("expected 2 total requests, got %d", metrics.TotalRequests)
				}
				if metrics.Status2xx != 1 {
					t.Fatalf("expected 1 2xx status, got %d", metrics.Status2xx)
				}
				if metrics.Status5xx != 1 {
					t.Fatalf("expected 1 5xx status, got %d", metrics.Status5xx)
				}
				if metrics.AvgDurationMs != 15.0 {
					t.Fatalf("expected avg duration 15.0, got %f", metrics.AvgDurationMs)
				}
			})
		})

		t.Run("When subscribing to live request logs", func(t *testing.T) {
			app, _ := svc.Create(
				context.Background(),
				"logs-worker",
				domain.SourceTypeInline,
				"",
				"main",
				"export default {}",
				true,
				nil,
				nil,
			)

			logsChan, unsubscribe := svc.SubscribeLiveLogs(app.ID)
			defer unsubscribe()

			svc.RecordExecution(app.ID, "GET", "/live", 200, 5.0, "127.0.0.1", "live test")

			t.Run("Then subscriber receives live log event", func(t *testing.T) {
				select {
				case event := <-logsChan:
					if event.Method != "GET" || event.Path != "/live" || event.StatusCode != 200 {
						t.Fatalf("unexpected event: %+v", event)
					}
				default:
					t.Fatal("expected to receive live log event, but channel was empty")
				}
			})
		})
	})
}
