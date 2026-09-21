package usecase

import (
	"context"

	"github.com/ishaf/cubit/internal/domain"
)

// NodeRepository defines the persistence contract for fleet nodes.
type NodeRepository interface {
	Save(ctx context.Context, node *domain.Node) error
	GetByID(ctx context.Context, id string) (*domain.Node, error)
	List(ctx context.Context) ([]*domain.Node, error)
	Delete(ctx context.Context, id string) error
	Update(ctx context.Context, node *domain.Node) error
}

// ApplicationRepository defines the persistence contract for worker applications.
type ApplicationRepository interface {
	Save(ctx context.Context, app *domain.Application) error
	GetByID(ctx context.Context, id string) (*domain.Application, error)
	GetBySubdomain(ctx context.Context, subdomain string) (*domain.Application, error)
	List(ctx context.Context) ([]*domain.Application, error)
	Delete(ctx context.Context, id string) error
	Update(ctx context.Context, app *domain.Application) error
}

// DeploymentRepository defines the persistence contract for deployments and logs.
type DeploymentRepository interface {
	Save(ctx context.Context, dep *domain.Deployment) error
	GetByID(ctx context.Context, id string) (*domain.Deployment, error)
	GetLatestBuildVersion(ctx context.Context, appID string) (int, error)
	ListByAppID(ctx context.Context, appID string) ([]*domain.Deployment, error)
	Update(ctx context.Context, dep *domain.Deployment) error
	AppendLog(ctx context.Context, deploymentID string, entry domain.DeploymentLog) error
	GetLogs(ctx context.Context, deploymentID string) ([]domain.DeploymentLog, error)
}

// DomainRepository defines the persistence contract for custom domain routes.
type DomainRepository interface {
	Save(ctx context.Context, dom *domain.Domain) error
	GetByID(ctx context.Context, id string) (*domain.Domain, error)
	List(ctx context.Context) ([]*domain.Domain, error)
	ListByAppID(ctx context.Context, appID string) ([]*domain.Domain, error)
	Delete(ctx context.Context, id string) error
	Update(ctx context.Context, dom *domain.Domain) error
}

// StoragePort defines the contract for S3-compatible storage (Garage S3 or Cloudflare R2 / AWS S3).
type StoragePort interface {
	EnsureBucket(ctx context.Context, bucketName string) error
	UploadBundle(ctx context.Context, bucketName, objectKey string, data []byte) error
	DownloadBundle(ctx context.Context, bucketName, objectKey string) ([]byte, error)
	CheckHealth(ctx context.Context) error
	DriverName() string
}

// RouteRule encapsulates an ingress route rule for Traefik.
type RouteRule struct {
	AppName    string
	Hostname   string
	PathPrefix string
	TargetURLs []string
	EnableTLS  bool
}

// ProxyPort defines the contract for synchronizing routing configurations with Traefik.
type ProxyPort interface {
	SyncRoutes(ctx context.Context, routes []RouteRule) error
}

// ContainerSupervisor defines the contract for managing celld containers on nodes.
type ContainerSupervisor interface {
	StartCelld(ctx context.Context, node *domain.Node, version string, bucketURL string) error
	StopCelld(ctx context.Context, node *domain.Node) error
	GracefulRestartCelld(ctx context.Context, node *domain.Node, newVersion string, bucketURL string) error
}
