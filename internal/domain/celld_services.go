package domain

import (
	"strings"
	"time"
)

// -----------------------------------------------------------------------------
// 1. KV (Key-Value)
// -----------------------------------------------------------------------------

// KVNamespace represents an isolated key-value storage namespace.
type KVNamespace struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	KeyCount  int       `json:"keyCount"`
	CreatedAt time.Time `json:"createdAt"`
}

// NewKVNamespace constructs a valid KV namespace.
func NewKVNamespace(id, name string) (*KVNamespace, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, NewValidationError("KV namespace name cannot be empty")
	}
	return &KVNamespace{
		ID:        id,
		Name:      name,
		KeyCount:  0,
		CreatedAt: time.Now().UTC(),
	}, nil
}

// KVPair represents a key-value entry inside a namespace.
type KVPair struct {
	NamespaceID   string    `json:"namespaceId"`
	Key           string    `json:"key"`
	Value         string    `json:"value"`
	ExpirationTTL int       `json:"expirationTtl,omitempty"` // TTL in seconds
	Metadata      string    `json:"metadata,omitempty"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

// NewKVPair constructs a valid key-value entry.
func NewKVPair(namespaceID, key, value string, ttl int, metadata string) (*KVPair, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, NewValidationError("KV key cannot be empty")
	}
	return &KVPair{
		NamespaceID:   namespaceID,
		Key:           key,
		Value:         value,
		ExpirationTTL: ttl,
		Metadata:      metadata,
		UpdatedAt:     time.Now().UTC(),
	}, nil
}

// -----------------------------------------------------------------------------
// 2. D1 (Serverless SQL Database)
// -----------------------------------------------------------------------------

// D1Database represents a serverless SQLite database.
type D1Database struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	SizeBytes   int64     `json:"sizeBytes"`
	TablesCount int       `json:"tablesCount"`
	CreatedAt   time.Time `json:"createdAt"`
}

// NewD1Database constructs a valid D1 database entity.
func NewD1Database(id, name string) (*D1Database, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, NewValidationError("D1 database name cannot be empty")
	}
	return &D1Database{
		ID:          id,
		Name:        name,
		SizeBytes:   0,
		TablesCount: 0,
		CreatedAt:   time.Now().UTC(),
	}, nil
}

// D1QueryResult encapsulates query execution results.
type D1QueryResult struct {
	Columns      []string                 `json:"columns"`
	Rows         []map[string]interface{} `json:"rows"`
	RowsAffected int64                    `json:"rowsAffected"`
	DurationMs   float64                  `json:"durationMs"`
}

// -----------------------------------------------------------------------------
// 3. R2 (S3-Compatible Object Storage)
// -----------------------------------------------------------------------------

// R2Bucket represents an object storage bucket.
type R2Bucket struct {
	Name         string    `json:"name"`
	CreatedAt    time.Time `json:"createdAt"`
	ObjectsCount int       `json:"objectsCount"`
	SizeBytes    int64     `json:"sizeBytes"`
	IsSystem     bool      `json:"isSystem"`
}

// R2Object represents a stored file object.
type R2Object struct {
	Key          string    `json:"key"`
	SizeBytes    int64     `json:"sizeBytes"`
	ContentType  string    `json:"contentType"`
	ETag         string    `json:"etag"`
	LastModified time.Time `json:"lastModified"`
}

// -----------------------------------------------------------------------------
// 4. Queues (Asynchronous Messaging)
// -----------------------------------------------------------------------------

// Queue represents a decoupled message queue.
type Queue struct {
	ID                string    `json:"id"`
	Name              string    `json:"name"`
	ConsumerAppID     string    `json:"consumerAppId,omitempty"`
	MaxRetries        int       `json:"maxRetries"`
	BacklogCount      int       `json:"backlogCount"`
	MessagesProcessed int64     `json:"messagesProcessed"`
	CreatedAt         time.Time `json:"createdAt"`
}

// NewQueue constructs a valid Queue entity.
func NewQueue(id, name, consumerAppID string, maxRetries int) (*Queue, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, NewValidationError("queue name cannot be empty")
	}
	if maxRetries < 0 {
		maxRetries = 3
	}
	return &Queue{
		ID:            id,
		Name:          name,
		ConsumerAppID: consumerAppID,
		MaxRetries:    maxRetries,
		CreatedAt:     time.Now().UTC(),
	}, nil
}

// QueueMessage represents a single message in a queue.
type QueueMessage struct {
	ID        string    `json:"id"`
	QueueID   string    `json:"queueId"`
	Body      string    `json:"body"`
	Attempts  int       `json:"attempts"`
	CreatedAt time.Time `json:"createdAt"`
}

// -----------------------------------------------------------------------------
// 5. Cron Triggers (Scheduled Invocations)
// -----------------------------------------------------------------------------

// CronTrigger represents a scheduled worker execution rule.
type CronTrigger struct {
	ID             string     `json:"id"`
	Name           string     `json:"name"`
	CronExpression string     `json:"cronExpression"`
	TargetAppID    string     `json:"targetAppId"`
	Status         string     `json:"status"` // "active", "paused"
	LastRunAt      *time.Time `json:"lastRunAt,omitempty"`
	NextRunAt      *time.Time `json:"nextRunAt,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
}

// NewCronTrigger constructs and validates a CronTrigger entity.
func NewCronTrigger(id, name, expression, targetAppID string) (*CronTrigger, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, NewValidationError("cron trigger name cannot be empty")
	}
	expression = strings.TrimSpace(expression)
	if expression == "" {
		return nil, NewValidationError("cron expression cannot be empty")
	}
	if targetAppID == "" {
		return nil, NewValidationError("target application ID cannot be empty")
	}
	return &CronTrigger{
		ID:             id,
		Name:           name,
		CronExpression: expression,
		TargetAppID:    targetAppID,
		Status:         "active",
		CreatedAt:      time.Now().UTC(),
	}, nil
}

// CronRun represents an execution log entry for a scheduled trigger.
type CronRun struct {
	ID         string    `json:"id"`
	TriggerID  string    `json:"triggerId"`
	Status     string    `json:"status"` // "success", "failed"
	StatusCode int       `json:"statusCode"`
	DurationMs float64   `json:"durationMs"`
	ExecutedAt time.Time `json:"executedAt"`
}

// -----------------------------------------------------------------------------
// 6. Workflows (Durable Multi-Step Orchestrations)
// -----------------------------------------------------------------------------

// Workflow represents a defined multi-step workflow.
type Workflow struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	TargetAppID string    `json:"targetAppId"`
	Steps       []string  `json:"steps"`
	CreatedAt   time.Time `json:"createdAt"`
}

// NewWorkflow constructs a valid Workflow entity.
func NewWorkflow(id, name, targetAppID string, steps []string) (*Workflow, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, NewValidationError("workflow name cannot be empty")
	}
	if len(steps) == 0 {
		steps = []string{"init", "process", "finalize"}
	}
	return &Workflow{
		ID:          id,
		Name:        name,
		TargetAppID: targetAppID,
		Steps:       steps,
		CreatedAt:   time.Now().UTC(),
	}, nil
}

// WorkflowRun represents an active or historical execution instance of a workflow.
type WorkflowRun struct {
	ID          string     `json:"id"`
	WorkflowID  string     `json:"workflowId"`
	Status      string     `json:"status"` // "running", "completed", "failed"
	CurrentStep string     `json:"currentStep"`
	Logs        []string   `json:"logs"`
	StartedAt   time.Time  `json:"startedAt"`
	FinishedAt  *time.Time `json:"finishedAt,omitempty"`
}

// -----------------------------------------------------------------------------
// 7. Durable Objects & Facets (Stateful Actors)
// -----------------------------------------------------------------------------

// DOFacet represents an exposed RPC interface on a Durable Object.
type DOFacet struct {
	Name        string   `json:"name"`
	Methods     []string `json:"methods"`
	Permissions string   `json:"permissions"`
}

// DurableObjectClass represents a defined Durable Object class.
type DurableObjectClass struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	AppID          string    `json:"appId"`
	Facets         []DOFacet `json:"facets"`
	StorageBackend string    `json:"storageBackend"` // "sqlite"
	InstancesCount int       `json:"instancesCount"`
	CreatedAt      time.Time `json:"createdAt"`
}

// NewDurableObjectClass constructs a valid Durable Object class.
func NewDurableObjectClass(id, name, appID string, facets []DOFacet) (*DurableObjectClass, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, NewValidationError("Durable Object class name cannot be empty")
	}
	if len(facets) == 0 {
		facets = []DOFacet{
			{Name: "default", Methods: []string{"fetch", "alarm"}, Permissions: "read-write"},
		}
	}
	return &DurableObjectClass{
		ID:             id,
		Name:           name,
		AppID:          appID,
		Facets:         facets,
		StorageBackend: "sqlite",
		InstancesCount: 0,
		CreatedAt:      time.Now().UTC(),
	}, nil
}

// DurableObjectInstance represents an individual stateful actor instance.
type DurableObjectInstance struct {
	ID               string     `json:"id"`
	ClassID          string     `json:"classId"`
	ObjectID         string     `json:"objectId"`
	Status           string     `json:"status"` // "active", "hibernating"
	StorageKeysCount int        `json:"storageKeysCount"`
	AlarmAt          *time.Time `json:"alarmAt,omitempty"`
	CreatedAt        time.Time  `json:"createdAt"`
}

// -----------------------------------------------------------------------------
// 8. Containers (Experimental Workloads)
// -----------------------------------------------------------------------------

// ContainerWorkload represents an experimental container instance managed alongside isolates.
type ContainerWorkload struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Image     string            `json:"image"`
	Port      int               `json:"port"`
	Status    string            `json:"status"` // "running", "stopped"
	EnvVars   map[string]string `json:"envVars"`
	CreatedAt time.Time         `json:"createdAt"`
}

// NewContainerWorkload constructs a valid container workload entity.
func NewContainerWorkload(id, name, image string, port int, envVars map[string]string) (*ContainerWorkload, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, NewValidationError("container name cannot be empty")
	}
	image = strings.TrimSpace(image)
	if image == "" {
		return nil, NewValidationError("container image cannot be empty")
	}
	if port <= 0 || port > 65535 {
		port = 8080
	}
	if envVars == nil {
		envVars = make(map[string]string)
	}
	return &ContainerWorkload{
		ID:        id,
		Name:      name,
		Image:     image,
		Port:      port,
		Status:    "running",
		EnvVars:   envVars,
		CreatedAt: time.Now().UTC(),
	}, nil
}

// -----------------------------------------------------------------------------
// 9. Static Assets (SPAs & Static Sites)
// -----------------------------------------------------------------------------

// StaticSite represents a static assets deployment for a website or SPA.
type StaticSite struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Subdomain     string    `json:"subdomain"`
	FilesCount    int       `json:"filesCount"`
	TotalSize     int64     `json:"totalSize"`
	IndexDocument string    `json:"indexDocument"`
	SPARouting    bool      `json:"spaRouting"`
	CreatedAt     time.Time `json:"createdAt"`
}

// NewStaticSite constructs a valid StaticSite entity.
func NewStaticSite(id, name, subdomain string) (*StaticSite, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, NewValidationError("static site name cannot be empty")
	}
	if subdomain == "" {
		subdomain = SanitizeSubdomain(name)
	}
	return &StaticSite{
		ID:            id,
		Name:          name,
		Subdomain:     subdomain,
		FilesCount:    0,
		TotalSize:     0,
		IndexDocument: "index.html",
		SPARouting:    true,
		CreatedAt:     time.Now().UTC(),
	}, nil
}
