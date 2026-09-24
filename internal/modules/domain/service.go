package domain

import (
	"context"
	"crypto/rand"
	"fmt"

	coreDomain "github.com/ishaf/cubit/internal/domain"
	"github.com/ishaf/cubit/internal/modules/wrangler"
)

// ApplicationVerifier checks application existence.
type ApplicationVerifier interface {
	GetByID(ctx context.Context, id string) (*coreDomain.Application, error)
}

// RouteSyncer triggers Traefik routing rules refresh.
type RouteSyncer interface {
	SyncRoutes(ctx context.Context) error
}

// RouteRegistrar defines operations for registering ingress routes as domain entities.
type RouteRegistrar interface {
	RegisterRoutes(ctx context.Context, appID string, routes []string) ([]*coreDomain.Domain, error)
}

// Service defines custom domain business operations.
type Service interface {
	AddDomain(ctx context.Context, appID, hostname, pathPrefix string) (*coreDomain.Domain, error)
	GetDomain(ctx context.Context, id string) (*coreDomain.Domain, error)
	ListDomains(ctx context.Context) ([]*coreDomain.Domain, error)
	DeleteDomain(ctx context.Context, id string) error
	RegisterRoutes(ctx context.Context, appID string, routes []string) ([]*coreDomain.Domain, error)
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

// RegisterRoutes extracts and normalizes route rules, validates target application ownership,
// persists distinct domain route entities (supporting multi-path routing on the same host),
// and synchronizes Traefik ingress rules.
func (s *DomainService) RegisterRoutes(ctx context.Context, appID string, routes []string) ([]*coreDomain.Domain, error) {
	if s.appVerifier != nil {
		if _, err := s.appVerifier.GetByID(ctx, appID); err != nil {
			return nil, err
		}
	}

	if len(routes) == 0 {
		return []*coreDomain.Domain{}, nil
	}

	extracted, err := wrangler.ExtractRouteRules(routes)
	if err != nil {
		return nil, coreDomain.NewValidationError(fmt.Sprintf("failed to parse routes: %v", err))
	}

	var registered []*coreDomain.Domain

	for _, rule := range extracted {
		// 1. Verify hostname ownership: no other application may own this hostname
		domainsForHost, err := s.repo.ListByHostname(ctx, rule.Hostname)
		if err == nil {
			for _, d := range domainsForHost {
				if d.ApplicationID != appID {
					return nil, coreDomain.NewConflictError(fmt.Sprintf("hostname %q is already registered to another application (%s)", rule.Hostname, d.ApplicationID))
				}
			}
		}

		// 2. Check if this exact (hostname, pathPrefix) is already registered for this app
		exact, err := s.repo.GetByHostAndPath(ctx, rule.Hostname, rule.PathPrefix)
		if err == nil && exact != nil {
			registered = append(registered, exact)
			continue
		}

		// 3. New route rule for this application
		domID := generateID()
		dom, err := coreDomain.NewDomain(domID, appID, rule.Hostname, rule.PathPrefix)
		if err != nil {
			return nil, err
		}

		if err := s.repo.Save(ctx, dom); err != nil {
			return nil, fmt.Errorf("failed saving route %s%s: %w", rule.Hostname, rule.PathPrefix, err)
		}
		registered = append(registered, dom)
	}

	if s.routeSyncer != nil {
		_ = s.routeSyncer.SyncRoutes(ctx)
	}

	return registered, nil
}
