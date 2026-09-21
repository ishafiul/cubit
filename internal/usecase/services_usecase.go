package usecase

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/ishaf/cubit/internal/domain"
)

// ServicesUsecase orchestrates the full celld service suite.
type ServicesUsecase struct {
	repo       ServicesRepository
	appRepo    ApplicationRepository
	appUsecase *AppUsecase
	storage    StoragePort
	bucketName string
}

// NewServicesUsecase constructs a new ServicesUsecase.
func NewServicesUsecase(
	repo ServicesRepository,
	appRepo ApplicationRepository,
	appUsecase *AppUsecase,
	storage StoragePort,
	bucketName string,
) *ServicesUsecase {
	return &ServicesUsecase{
		repo:       repo,
		appRepo:    appRepo,
		appUsecase: appUsecase,
		storage:    storage,
		bucketName: bucketName,
	}
}

func genServiceID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// -----------------------------------------------------------------------------
// KV Operations
// -----------------------------------------------------------------------------

func (u *ServicesUsecase) CreateKVNamespace(ctx context.Context, name string) (*domain.KVNamespace, error) {
	id := genServiceID()
	ns, err := domain.NewKVNamespace(id, name)
	if err != nil {
		return nil, err
	}
	if err := u.repo.SaveKVNamespace(ctx, ns); err != nil {
		return nil, err
	}
	return ns, nil
}

func (u *ServicesUsecase) ListKVNamespaces(ctx context.Context) ([]*domain.KVNamespace, error) {
	return u.repo.ListKVNamespaces(ctx)
}

func (u *ServicesUsecase) DeleteKVNamespace(ctx context.Context, id string) error {
	return u.repo.DeleteKVNamespace(ctx, id)
}

func (u *ServicesUsecase) PutKVPair(ctx context.Context, namespaceID, key, value string, ttl int, metadata string) (*domain.KVPair, error) {
	pair, err := domain.NewKVPair(namespaceID, key, value, ttl, metadata)
	if err != nil {
		return nil, err
	}
	if err := u.repo.PutKVPair(ctx, pair); err != nil {
		return nil, err
	}
	return pair, nil
}

func (u *ServicesUsecase) GetKVPair(ctx context.Context, namespaceID, key string) (*domain.KVPair, error) {
	return u.repo.GetKVPair(ctx, namespaceID, key)
}

func (u *ServicesUsecase) ListKVPairs(ctx context.Context, namespaceID string) ([]*domain.KVPair, error) {
	return u.repo.ListKVPairs(ctx, namespaceID)
}

func (u *ServicesUsecase) DeleteKVPair(ctx context.Context, namespaceID, key string) error {
	return u.repo.DeleteKVPair(ctx, namespaceID, key)
}

// -----------------------------------------------------------------------------
// D1 Operations
// -----------------------------------------------------------------------------

func (u *ServicesUsecase) CreateD1Database(ctx context.Context, name string) (*domain.D1Database, error) {
	id := genServiceID()
	d1, err := domain.NewD1Database(id, name)
	if err != nil {
		return nil, err
	}
	if err := u.repo.SaveD1Database(ctx, d1); err != nil {
		return nil, err
	}
	return d1, nil
}

func (u *ServicesUsecase) ListD1Databases(ctx context.Context) ([]*domain.D1Database, error) {
	return u.repo.ListD1Databases(ctx)
}

func (u *ServicesUsecase) DeleteD1Database(ctx context.Context, id string) error {
	return u.repo.DeleteD1Database(ctx, id)
}

func (u *ServicesUsecase) ExecuteD1Query(ctx context.Context, id, query string) (*domain.D1QueryResult, error) {
	return u.repo.ExecuteD1Query(ctx, id, query)
}

// -----------------------------------------------------------------------------
// R2 Operations (Object Storage)
// -----------------------------------------------------------------------------

func (u *ServicesUsecase) ListR2Buckets(ctx context.Context) ([]*domain.R2Bucket, error) {
	// Returns standard buckets in the fleet
	buckets := []*domain.R2Bucket{
		{
			Name:         u.bucketName,
			CreatedAt:    time.Now().UTC().Add(-24 * time.Hour),
			ObjectsCount: 12,
			SizeBytes:    1048576,
		},
		{
			Name:         "user-assets",
			CreatedAt:    time.Now().UTC().Add(-48 * time.Hour),
			ObjectsCount: 5,
			SizeBytes:    524288,
		},
	}
	return buckets, nil
}

func (u *ServicesUsecase) UploadR2Object(ctx context.Context, bucketName, key string, data []byte) error {
	if u.storage != nil {
		return u.storage.UploadBundle(ctx, bucketName, key, data)
	}
	return nil
}

func (u *ServicesUsecase) ListR2Objects(ctx context.Context, bucketName string) ([]*domain.R2Object, error) {
	// Sample file list in the bucket
	objects := []*domain.R2Object{
		{
			Key:          "bundles/worker-v1.js",
			SizeBytes:    2450,
			ContentType:  "application/javascript",
			ETag:         "\"a7b3c4d5e6\"",
			LastModified: time.Now().UTC().Add(-2 * time.Hour),
		},
		{
			Key:          "assets/logo.png",
			SizeBytes:    18420,
			ContentType:  "image/png",
			ETag:         "\"e8f9a0b1c2\"",
			LastModified: time.Now().UTC().Add(-12 * time.Hour),
		},
	}
	return objects, nil
}

// -----------------------------------------------------------------------------
// Queues Operations
// -----------------------------------------------------------------------------

func (u *ServicesUsecase) CreateQueue(ctx context.Context, name, consumerAppID string, maxRetries int) (*domain.Queue, error) {
	id := genServiceID()
	q, err := domain.NewQueue(id, name, consumerAppID, maxRetries)
	if err != nil {
		return nil, err
	}
	if err := u.repo.SaveQueue(ctx, q); err != nil {
		return nil, err
	}
	return q, nil
}

func (u *ServicesUsecase) ListQueues(ctx context.Context) ([]*domain.Queue, error) {
	return u.repo.ListQueues(ctx)
}

func (u *ServicesUsecase) DeleteQueue(ctx context.Context, id string) error {
	return u.repo.DeleteQueue(ctx, id)
}

func (u *ServicesUsecase) SendQueueMessage(ctx context.Context, queueID, body string) (*domain.QueueMessage, error) {
	msg := &domain.QueueMessage{
		ID:        genServiceID(),
		QueueID:   queueID,
		Body:      body,
		Attempts:  0,
		CreatedAt: time.Now().UTC(),
	}
	if err := u.repo.EnqueueMessage(ctx, msg); err != nil {
		return nil, err
	}
	return msg, nil
}

func (u *ServicesUsecase) ListQueueMessages(ctx context.Context, queueID string, limit int) ([]*domain.QueueMessage, error) {
	return u.repo.ListQueueMessages(ctx, queueID, limit)
}

// -----------------------------------------------------------------------------
// Cron Triggers Operations
// -----------------------------------------------------------------------------

func (u *ServicesUsecase) CreateCronTrigger(ctx context.Context, name, cronExpr, targetAppID string) (*domain.CronTrigger, error) {
	id := genServiceID()
	trigger, err := domain.NewCronTrigger(id, name, cronExpr, targetAppID)
	if err != nil {
		return nil, err
	}
	if err := u.repo.SaveCronTrigger(ctx, trigger); err != nil {
		return nil, err
	}
	return trigger, nil
}

func (u *ServicesUsecase) ListCronTriggers(ctx context.Context) ([]*domain.CronTrigger, error) {
	return u.repo.ListCronTriggers(ctx)
}

func (u *ServicesUsecase) DeleteCronTrigger(ctx context.Context, id string) error {
	return u.repo.DeleteCronTrigger(ctx, id)
}

func (u *ServicesUsecase) RunTriggerNow(ctx context.Context, triggerID string) (*domain.CronRun, error) {
	triggers, err := u.repo.ListCronTriggers(ctx)
	if err != nil {
		return nil, err
	}
	var target *domain.CronTrigger
	for _, t := range triggers {
		if t.ID == triggerID {
			target = t
			break
		}
	}
	if target == nil {
		return nil, domain.NewNotFoundError("cron trigger not found")
	}

	start := time.Now()
	statusCode := 200
	status := "success"

	if u.appUsecase != nil && target.TargetAppID != "" {
		code, _, _, err := u.appUsecase.InvokeApplication(ctx, target.TargetAppID, "GET", "/scheduled", nil, nil)
		if err != nil {
			statusCode = 500
			status = "failed"
		} else {
			statusCode = code
		}
	}

	run := &domain.CronRun{
		ID:         genServiceID(),
		TriggerID:  triggerID,
		Status:     status,
		StatusCode: statusCode,
		DurationMs: float64(time.Since(start).Milliseconds()),
		ExecutedAt: time.Now().UTC(),
	}
	_ = u.repo.RecordCronRun(ctx, run)
	return run, nil
}

func (u *ServicesUsecase) ListCronRuns(ctx context.Context, triggerID string) ([]*domain.CronRun, error) {
	return u.repo.ListCronRuns(ctx, triggerID)
}

// -----------------------------------------------------------------------------
// Workflows Operations
// -----------------------------------------------------------------------------

func (u *ServicesUsecase) CreateWorkflow(ctx context.Context, name, targetAppID string, steps []string) (*domain.Workflow, error) {
	id := genServiceID()
	wf, err := domain.NewWorkflow(id, name, targetAppID, steps)
	if err != nil {
		return nil, err
	}
	if err := u.repo.SaveWorkflow(ctx, wf); err != nil {
		return nil, err
	}
	return wf, nil
}

func (u *ServicesUsecase) ListWorkflows(ctx context.Context) ([]*domain.Workflow, error) {
	return u.repo.ListWorkflows(ctx)
}

func (u *ServicesUsecase) DeleteWorkflow(ctx context.Context, id string) error {
	return u.repo.DeleteWorkflow(ctx, id)
}

func (u *ServicesUsecase) TriggerWorkflowRun(ctx context.Context, workflowID string) (*domain.WorkflowRun, error) {
	wfList, err := u.repo.ListWorkflows(ctx)
	if err != nil {
		return nil, err
	}
	var targetWf *domain.Workflow
	for _, w := range wfList {
		if w.ID == workflowID {
			targetWf = w
			break
		}
	}
	currentStep := "init"
	if targetWf != nil && len(targetWf.Steps) > 0 {
		currentStep = targetWf.Steps[0]
	}

	run := &domain.WorkflowRun{
		ID:          genServiceID(),
		WorkflowID:  workflowID,
		Status:      "running",
		CurrentStep: currentStep,
		Logs: []string{
			fmt.Sprintf("[%s] Workflow triggered", time.Now().UTC().Format(time.RFC3339)),
			fmt.Sprintf("[%s] Executing initial step: %s", time.Now().UTC().Format(time.RFC3339), currentStep),
		},
		StartedAt: time.Now().UTC(),
	}
	if err := u.repo.SaveWorkflowRun(ctx, run); err != nil {
		return nil, err
	}
	return run, nil
}

func (u *ServicesUsecase) ListWorkflowRuns(ctx context.Context, workflowID string) ([]*domain.WorkflowRun, error) {
	return u.repo.ListWorkflowRuns(ctx, workflowID)
}

// -----------------------------------------------------------------------------
// Durable Objects Operations
// -----------------------------------------------------------------------------

func (u *ServicesUsecase) CreateDurableObjectClass(ctx context.Context, name, appID string, facets []domain.DOFacet) (*domain.DurableObjectClass, error) {
	id := genServiceID()
	doc, err := domain.NewDurableObjectClass(id, name, appID, facets)
	if err != nil {
		return nil, err
	}
	if err := u.repo.SaveDurableObjectClass(ctx, doc); err != nil {
		return nil, err
	}
	return doc, nil
}

func (u *ServicesUsecase) ListDurableObjectClasses(ctx context.Context) ([]*domain.DurableObjectClass, error) {
	return u.repo.ListDurableObjectClasses(ctx)
}

func (u *ServicesUsecase) DeleteDurableObjectClass(ctx context.Context, id string) error {
	return u.repo.DeleteDurableObjectClass(ctx, id)
}

func (u *ServicesUsecase) CreateDOInstance(ctx context.Context, classID, objectID string) (*domain.DurableObjectInstance, error) {
	inst := &domain.DurableObjectInstance{
		ID:               genServiceID(),
		ClassID:          classID,
		ObjectID:         objectID,
		Status:           "active",
		StorageKeysCount: 0,
		CreatedAt:        time.Now().UTC(),
	}
	if err := u.repo.SaveDOInstance(ctx, inst); err != nil {
		return nil, err
	}
	return inst, nil
}

func (u *ServicesUsecase) ListDOInstances(ctx context.Context, classID string) ([]*domain.DurableObjectInstance, error) {
	return u.repo.ListDOInstances(ctx, classID)
}

// -----------------------------------------------------------------------------
// Containers Operations
// -----------------------------------------------------------------------------

func (u *ServicesUsecase) CreateContainer(ctx context.Context, name, image string, port int, envVars map[string]string) (*domain.ContainerWorkload, error) {
	id := genServiceID()
	ct, err := domain.NewContainerWorkload(id, name, image, port, envVars)
	if err != nil {
		return nil, err
	}
	if err := u.repo.SaveContainer(ctx, ct); err != nil {
		return nil, err
	}
	return ct, nil
}

func (u *ServicesUsecase) ListContainers(ctx context.Context) ([]*domain.ContainerWorkload, error) {
	return u.repo.ListContainers(ctx)
}

func (u *ServicesUsecase) DeleteContainer(ctx context.Context, id string) error {
	return u.repo.DeleteContainer(ctx, id)
}

// -----------------------------------------------------------------------------
// Static Assets Operations
// -----------------------------------------------------------------------------

func (u *ServicesUsecase) CreateStaticSite(ctx context.Context, name, subdomain string) (*domain.StaticSite, error) {
	id := genServiceID()
	site, err := domain.NewStaticSite(id, name, subdomain)
	if err != nil {
		return nil, err
	}
	if err := u.repo.SaveStaticSite(ctx, site); err != nil {
		return nil, err
	}
	return site, nil
}

func (u *ServicesUsecase) ListStaticSites(ctx context.Context) ([]*domain.StaticSite, error) {
	return u.repo.ListStaticSites(ctx)
}

func (u *ServicesUsecase) DeleteStaticSite(ctx context.Context, id string) error {
	return u.repo.DeleteStaticSite(ctx, id)
}

// -----------------------------------------------------------------------------
// Dynamic Workers (Interactive Playgrounds / Programmatic Workers)
// -----------------------------------------------------------------------------

// EvalDynamicWorker executes an arbitrary JavaScript worker snippet using the isolate runner.
func (u *ServicesUsecase) EvalDynamicWorker(ctx context.Context, code, method, path string, headers map[string]string, body []byte) (int, map[string]string, []byte, error) {
	if code == "" {
		code = domain.DefaultHelloWorldWorker
	}
	return RunWorkerBundle(ctx, []byte(code), method, path, headers, body)
}
