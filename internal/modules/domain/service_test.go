package domain_test

import (
	"context"
	"testing"

	coreDomain "github.com/ishaf/cubit/internal/domain"
	domModule "github.com/ishaf/cubit/internal/modules/domain"
)

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

type mockAppVerifier struct{}

func (m *mockAppVerifier) GetByID(ctx context.Context, id string) (*coreDomain.Application, error) {
	return &coreDomain.Application{ID: id, Name: "app-test"}, nil
}

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
}
