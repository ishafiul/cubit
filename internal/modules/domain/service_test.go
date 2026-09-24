package domain_test

import (
	"context"
	"testing"

	coreDomain "github.com/ishaf/cubit/internal/domain"
	domModule "github.com/ishaf/cubit/internal/modules/domain"
)

func TestDomainService(t *testing.T) {
	t.Run("Given a fresh DomainService", func(t *testing.T) {
		repo := newMockDomainRepo()
		svc := domModule.NewService(repo, &mockAppVerifier{}, nil)

		t.Run("When adding a custom domain with valid hostname", func(t *testing.T) {
			dom, err := svc.AddDomain(context.Background(), "app-123", "api.example.com", "/")

			t.Run("Then domain is created successfully", func(t *testing.T) {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if dom.Hostname != "api.example.com" {
					t.Fatalf("expected hostname api.example.com, got %s", dom.Hostname)
				}
			})
		})

		t.Run("When adding an invalid hostname", func(t *testing.T) {
			_, err := svc.AddDomain(context.Background(), "app-123", "invalid_host_name!", "/")

			t.Run("Then validation error is returned", func(t *testing.T) {
				if err == nil {
					t.Fatal("expected validation error, got nil")
				}
			})
		})
	})

	t.Run("Given an application and a list of extracted route strings", func(t *testing.T) {
		repo := newMockDomainRepo()
		appVerifier := &mockAppVerifier{existingAppID: "app-target"}
		syncer := &mockRouteSyncer{}
		svc := domModule.NewService(repo, appVerifier, syncer)

		routes := []string{
			"api.example.com/*",
			"*sub.domain.com/path*",
			"custom.domain.org",
		}

		t.Run("When RegisterRoutes is called", func(t *testing.T) {
			registered, err := svc.RegisterRoutes(context.Background(), "app-target", routes)

			t.Run("Then domain entities are created and attached to the application", func(t *testing.T) {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				// 1: api.example.com -> /
				// 2: sub.domain.com -> /path
				// 3: *.sub.domain.com -> /path
				// 4: custom.domain.org -> /
				if len(registered) != 4 {
					t.Fatalf("expected 4 registered domains, got %d", len(registered))
				}

				hostMap := make(map[string]*coreDomain.Domain)
				for _, d := range registered {
					if d.ApplicationID != "app-target" {
						t.Errorf("expected application ID app-target, got %s", d.ApplicationID)
					}
					hostMap[d.Hostname] = d
				}

				if d, ok := hostMap["api.example.com"]; !ok || d.PathPrefix != "/" {
					t.Errorf("missing or invalid api.example.com route: %+v", d)
				}
				if d, ok := hostMap["sub.domain.com"]; !ok || d.PathPrefix != "/path" {
					t.Errorf("missing or invalid sub.domain.com route: %+v", d)
				}
				if d, ok := hostMap["*.sub.domain.com"]; !ok || d.PathPrefix != "/path" {
					t.Errorf("missing or invalid *.sub.domain.com route: %+v", d)
				}
				if d, ok := hostMap["custom.domain.org"]; !ok || d.PathPrefix != "/" {
					t.Errorf("missing or invalid custom.domain.org route: %+v", d)
				}
			})

			t.Run("Then RouteSyncer.SyncRoutes is invoked", func(t *testing.T) {
				if syncer.syncCalls == 0 {
					t.Error("expected RouteSyncer.SyncRoutes to be invoked")
				}
			})

			t.Run("Then all domains are persisted in repository", func(t *testing.T) {
				list, err := repo.ListByAppID(context.Background(), "app-target")
				if err != nil {
					t.Fatalf("expected no error listing domains, got %v", err)
				}
				if len(list) != 4 {
					t.Errorf("expected 4 persisted domains in repo, got %d", len(list))
				}
			})
		})

		t.Run("When RegisterRoutes is called idempotently with existing routes", func(t *testing.T) {
			prevCalls := syncer.syncCalls
			reRegistered, err := svc.RegisterRoutes(context.Background(), "app-target", []string{"api.example.com/*"})

			t.Run("Then existing domain is reused without duplicate error", func(t *testing.T) {
				if err != nil {
					t.Fatalf("expected no error for idempotent registration, got %v", err)
				}
				if len(reRegistered) != 1 {
					t.Fatalf("expected 1 domain returned, got %d", len(reRegistered))
				}
				if reRegistered[0].Hostname != "api.example.com" {
					t.Errorf("expected hostname api.example.com, got %s", reRegistered[0].Hostname)
				}
				// Total domains in repo should still be 4
				list, _ := repo.ListByAppID(context.Background(), "app-target")
				if len(list) != 4 {
					t.Errorf("expected total domains to remain 4, got %d", len(list))
				}
				if syncer.syncCalls <= prevCalls {
					t.Error("expected RouteSyncer.SyncRoutes to be called")
				}
			})
		})

		t.Run("When a route belongs to another application", func(t *testing.T) {
			_, err := svc.RegisterRoutes(context.Background(), "app-other", []string{"api.example.com/*"})

			t.Run("Then conflict error is returned", func(t *testing.T) {
				if err == nil {
					t.Fatal("expected conflict error when registering domain belonging to another app, got nil")
				}
			})
		})

		t.Run("When registering invalid route patterns", func(t *testing.T) {
			_, err := svc.RegisterRoutes(context.Background(), "app-target", []string{"invalid_host_no_tld"})

			t.Run("Then validation error is returned", func(t *testing.T) {
				if err == nil {
					t.Fatal("expected validation error, got nil")
				}
			})
		})

		t.Run("When registering multiple distinct paths on the same hostname", func(t *testing.T) {
			routes := []string{
				"api.example.com/v1/*",
				"api.example.com/v2/*",
			}
			registered, err := svc.RegisterRoutes(context.Background(), "app-target", routes)

			t.Run("Then both routes are stored as distinct entities without overwriting", func(t *testing.T) {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if len(registered) != 2 {
					t.Fatalf("expected 2 registered domains, got %d", len(registered))
				}
				p1, err1 := repo.GetByHostAndPath(context.Background(), "api.example.com", "/v1")
				if err1 != nil || p1.PathPrefix != "/v1" {
					t.Errorf("expected /v1 route to be preserved: %+v", p1)
				}
				p2, err2 := repo.GetByHostAndPath(context.Background(), "api.example.com", "/v2")
				if err2 != nil || p2.PathPrefix != "/v2" {
					t.Errorf("expected /v2 route to be preserved: %+v", p2)
				}
			})
		})

		t.Run("When registering routes for non-existent application", func(t *testing.T) {
			_, err := svc.RegisterRoutes(context.Background(), "app-nonexistent", []string{"brand-new.example.com/*"})

			t.Run("Then not found error is returned", func(t *testing.T) {
				if err == nil {
					t.Fatal("expected error for non-existent application, got nil")
				}
			})
		})
	})
}

// Test utilities and mocks

type mockDomainRepo struct {
	domains map[string]*coreDomain.Domain
}

func newMockDomainRepo() *mockDomainRepo {
	return &mockDomainRepo{domains: make(map[string]*coreDomain.Domain)}
}

func (m *mockDomainRepo) Save(ctx context.Context, d *coreDomain.Domain) error {
	m.domains[d.ID] = d
	return nil
}

func (m *mockDomainRepo) GetByID(ctx context.Context, id string) (*coreDomain.Domain, error) {
	if d, ok := m.domains[id]; ok {
		return d, nil
	}
	return nil, coreDomain.NewNotFoundError("domain not found: " + id)
}

func (m *mockDomainRepo) GetByHostname(ctx context.Context, hostname string) (*coreDomain.Domain, error) {
	for _, d := range m.domains {
		if d.Hostname == hostname {
			return d, nil
		}
	}
	return nil, coreDomain.NewNotFoundError("domain not found for hostname: " + hostname)
}

func (m *mockDomainRepo) GetByHostAndPath(ctx context.Context, hostname, pathPrefix string) (*coreDomain.Domain, error) {
	for _, d := range m.domains {
		if d.Hostname == hostname && d.PathPrefix == pathPrefix {
			return d, nil
		}
	}
	return nil, coreDomain.NewNotFoundError("domain not found for host and path: " + hostname + pathPrefix)
}

func (m *mockDomainRepo) ListByHostname(ctx context.Context, hostname string) ([]*coreDomain.Domain, error) {
	var list []*coreDomain.Domain
	for _, d := range m.domains {
		if d.Hostname == hostname {
			list = append(list, d)
		}
	}
	return list, nil
}

func (m *mockDomainRepo) List(ctx context.Context) ([]*coreDomain.Domain, error) {
	var list []*coreDomain.Domain
	for _, d := range m.domains {
		list = append(list, d)
	}
	return list, nil
}

func (m *mockDomainRepo) ListByAppID(ctx context.Context, appID string) ([]*coreDomain.Domain, error) {
	var list []*coreDomain.Domain
	for _, d := range m.domains {
		if d.ApplicationID == appID {
			list = append(list, d)
		}
	}
	return list, nil
}

func (m *mockDomainRepo) Delete(ctx context.Context, id string) error {
	if _, ok := m.domains[id]; !ok {
		return coreDomain.NewNotFoundError("domain not found: " + id)
	}
	delete(m.domains, id)
	return nil
}

func (m *mockDomainRepo) Update(ctx context.Context, d *coreDomain.Domain) error {
	m.domains[d.ID] = d
	return nil
}

type mockAppVerifier struct {
	existingAppID string
}

func (m *mockAppVerifier) GetByID(ctx context.Context, id string) (*coreDomain.Application, error) {
	if m.existingAppID != "" && id != m.existingAppID {
		return nil, coreDomain.NewNotFoundError("application not found: " + id)
	}
	return &coreDomain.Application{ID: id, Name: "app-test"}, nil
}

type mockRouteSyncer struct {
	syncCalls int
}

func (m *mockRouteSyncer) SyncRoutes(ctx context.Context) error {
	m.syncCalls++
	return nil
}
