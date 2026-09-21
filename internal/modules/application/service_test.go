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

		t.Run("When updating application configuration", func(t *testing.T) {
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
			updated, err := svc.Update(context.Background(), app.ID, "staging", "", &autoDeployFalse, nil, nil)

			t.Run("Then the update is persisted", func(t *testing.T) {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if updated.Branch != "staging" {
					t.Fatalf("expected branch staging, got %s", updated.Branch)
				}
				if updated.AutoDeploy != false {
					t.Fatal("expected autoDeploy to be false")
				}
			})
		})
	})
}
