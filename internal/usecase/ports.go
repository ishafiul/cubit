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

// ServicesRepository defines the persistence contract for all celld services.
type ServicesRepository interface {
	// KV
	SaveKVNamespace(ctx context.Context, ns *domain.KVNamespace) error
	ListKVNamespaces(ctx context.Context) ([]*domain.KVNamespace, error)
	GetKVNamespace(ctx context.Context, id string) (*domain.KVNamespace, error)
	DeleteKVNamespace(ctx context.Context, id string) error
	PutKVPair(ctx context.Context, pair *domain.KVPair) error
	GetKVPair(ctx context.Context, namespaceID, key string) (*domain.KVPair, error)
	ListKVPairs(ctx context.Context, namespaceID string) ([]*domain.KVPair, error)
	DeleteKVPair(ctx context.Context, namespaceID, key string) error

	// D1
	SaveD1Database(ctx context.Context, db *domain.D1Database) error
	ListD1Databases(ctx context.Context) ([]*domain.D1Database, error)
	GetD1Database(ctx context.Context, id string) (*domain.D1Database, error)
	DeleteD1Database(ctx context.Context, id string) error
	ExecuteD1Query(ctx context.Context, id, query string) (*domain.D1QueryResult, error)

	// Queues
	SaveQueue(ctx context.Context, q *domain.Queue) error
	ListQueues(ctx context.Context) ([]*domain.Queue, error)
	DeleteQueue(ctx context.Context, id string) error
	EnqueueMessage(ctx context.Context, msg *domain.QueueMessage) error
	ListQueueMessages(ctx context.Context, queueID string, limit int) ([]*domain.QueueMessage, error)

	// Cron Triggers
	SaveCronTrigger(ctx context.Context, trigger *domain.CronTrigger) error
	ListCronTriggers(ctx context.Context) ([]*domain.CronTrigger, error)
	DeleteCronTrigger(ctx context.Context, id string) error
	RecordCronRun(ctx context.Context, run *domain.CronRun) error
	ListCronRuns(ctx context.Context, triggerID string) ([]*domain.CronRun, error)

	// Workflows
	SaveWorkflow(ctx context.Context, wf *domain.Workflow) error
	ListWorkflows(ctx context.Context) ([]*domain.Workflow, error)
	DeleteWorkflow(ctx context.Context, id string) error
	SaveWorkflowRun(ctx context.Context, run *domain.WorkflowRun) error
	ListWorkflowRuns(ctx context.Context, workflowID string) ([]*domain.WorkflowRun, error)

	// Durable Objects
	SaveDurableObjectClass(ctx context.Context, doc *domain.DurableObjectClass) error
	ListDurableObjectClasses(ctx context.Context) ([]*domain.DurableObjectClass, error)
	DeleteDurableObjectClass(ctx context.Context, id string) error
	SaveDOInstance(ctx context.Context, inst *domain.DurableObjectInstance) error
	ListDOInstances(ctx context.Context, classID string) ([]*domain.DurableObjectInstance, error)

	// Containers
	SaveContainer(ctx context.Context, ct *domain.ContainerWorkload) error
	ListContainers(ctx context.Context) ([]*domain.ContainerWorkload, error)
	DeleteContainer(ctx context.Context, id string) error

	// Static Assets
	SaveStaticSite(ctx context.Context, site *domain.StaticSite) error
	ListStaticSites(ctx context.Context) ([]*domain.StaticSite, error)
	DeleteStaticSite(ctx context.Context, id string) error
}
