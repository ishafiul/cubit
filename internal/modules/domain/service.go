package domain

import (
	"context"
	"crypto/rand"
	"fmt"

	coreDomain "github.com/ishaf/cubit/internal/domain"
)

// ApplicationVerifier checks application existence.
type ApplicationVerifier interface {
	GetByID(ctx context.Context, id string) (*coreDomain.Application, error)
}

// RouteSyncer triggers Traefik routing rules refresh.
type RouteSyncer interface {
	SyncRoutes(ctx context.Context) error
}

// Service defines custom domain business operations.
type Service interface {
	AddDomain(ctx context.Context, appID, hostname, pathPrefix string) (*coreDomain.Domain, error)
	GetDomain(ctx context.Context, id string) (*coreDomain.Domain, error)
	ListDomains(ctx context.Context) ([]*coreDomain.Domain, error)
	DeleteDomain(ctx context.Context, id string) error
}

// DomainService manages custom domains.
type DomainService struct {
	repo        Repository
	appVerifier ApplicationVerifier
	routeSyncer RouteSyncer
}

// NewService creates a new DomainService.
func NewService(repo Repository, appVerifier ApplicationVerifier, routeSyncer RouteSyncer) *DomainService {
	return &DomainService{
		repo:        repo,
		appVerifier: appVerifier,
		routeSyncer: routeSyncer,
	}
}

func generateID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

// AddDomain binds a new domain to an application.
func (s *DomainService) AddDomain(ctx context.Context, appID, hostname, pathPrefix string) (*coreDomain.Domain, error) {
	if s.appVerifier != nil {
		if _, err := s.appVerifier.GetByID(ctx, appID); err != nil {
			return nil, err
		}
	}

	domID := generateID()
	dom, err := coreDomain.NewDomain(domID, appID, hostname, pathPrefix)
	if err != nil {
		return nil, err
	}

	if err := s.repo.Save(ctx, dom); err != nil {
		return nil, err
	}

	if s.routeSyncer != nil {
		_ = s.routeSyncer.SyncRoutes(ctx)
	}

	return dom, nil
}

// GetDomain retrieves a domain by ID.
func (s *DomainService) GetDomain(ctx context.Context, id string) (*coreDomain.Domain, error) {
	return s.repo.GetByID(ctx, id)
}

// ListDomains retrieves all registered custom domains.
func (s *DomainService) ListDomains(ctx context.Context) ([]*coreDomain.Domain, error) {
	return s.repo.List(ctx)
}

// DeleteDomain unbinds and removes a domain.
func (s *DomainService) DeleteDomain(ctx context.Context, id string) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	if s.routeSyncer != nil {
		_ = s.routeSyncer.SyncRoutes(ctx)
	}
	return nil
}
