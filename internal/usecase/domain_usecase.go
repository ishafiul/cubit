package usecase

import (
	"context"

	"github.com/ishaf/cubit/internal/domain"
)

// DomainUsecase manages custom domain bindings and route updates.
type DomainUsecase struct {
	domainRepo DomainRepository
	appRepo    ApplicationRepository
	appUsecase *AppUsecase
}

// NewDomainUsecase creates a new DomainUsecase instance.
func NewDomainUsecase(domainRepo DomainRepository, appRepo ApplicationRepository, appUsecase *AppUsecase) *DomainUsecase {
	return &DomainUsecase{
		domainRepo: domainRepo,
		appRepo:    appRepo,
		appUsecase: appUsecase,
	}
}

// AddDomain binds a new domain to an application.
func (u *DomainUsecase) AddDomain(ctx context.Context, appID, hostname, pathPrefix string) (*domain.Domain, error) {
	_, err := u.appRepo.GetByID(ctx, appID)
	if err != nil {
		return nil, err
	}

	domID := generateID()
	dom, err := domain.NewDomain(domID, appID, hostname, pathPrefix)
	if err != nil {
		return nil, err
	}

	if err := u.domainRepo.Save(ctx, dom); err != nil {
		return nil, err
	}

	if u.appUsecase != nil {
		_ = u.appUsecase.syncAllRoutes(ctx)
	}

	return dom, nil
}

// ListDomains retrieves all registered custom domains.
func (u *DomainUsecase) ListDomains(ctx context.Context) ([]*domain.Domain, error) {
	return u.domainRepo.List(ctx)
}

// DeleteDomain unbinds and removes a domain.
func (u *DomainUsecase) DeleteDomain(ctx context.Context, id string) error {
	if err := u.domainRepo.Delete(ctx, id); err != nil {
		return err
	}
	if u.appUsecase != nil {
		_ = u.appUsecase.syncAllRoutes(ctx)
	}
	return nil
}
