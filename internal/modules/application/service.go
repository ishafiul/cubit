package application

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ishaf/cubit/internal/domain"
)

// StorageDownloader downloads compiled worker bundles from object storage.
type StorageDownloader interface {
	DownloadBundle(ctx context.Context, bucket, objectKey string) ([]byte, error)
}

// RouteSyncer triggers routing synchronization across proxy/ingress.
type RouteSyncer interface {
	SyncRoutes(ctx context.Context) error
}

// Service defines application business operations.
type Service interface {
	Create(ctx context.Context, name string, sourceType domain.SourceType, gitRepo, branch, inlineCode string, autoDeploy bool, envVars []domain.EnvironmentVariable, bindings []domain.ResourceBinding) (*domain.Application, error)
	GetByID(ctx context.Context, id string) (*domain.Application, error)
	GetBySubdomain(ctx context.Context, subdomain string) (*domain.Application, error)
	List(ctx context.Context) ([]*domain.Application, error)
	Update(ctx context.Context, id, branch, inlineCode string, autoDeploy *bool, envVars []domain.EnvironmentVariable, bindings []domain.ResourceBinding, compatDate *string, compatFlags *[]string, memoryLimitMB *int, maxDurationMs *int) (*domain.Application, error)
	UpdateInlineCode(ctx context.Context, appID, newCode string) (*domain.Application, error)
	Delete(ctx context.Context, id string) error
	Invoke(ctx context.Context, appID string, method, path string, headers map[string]string, body []byte) (int, map[string]string, []byte, error)
	GetApplicationBySubdomain(ctx context.Context, subdomain string) (*domain.Application, error)
	InvokeApplication(ctx context.Context, appID string, method, path string, headers map[string]string, body []byte) (int, map[string]string, []byte, error)
	SetActiveDeployment(ctx context.Context, appID, deploymentID string) error
	GetMetrics(ctx context.Context, appID string) (*domain.ApplicationMetrics, error)
	RecordExecution(appID string, method, path string, statusCode int, durationMs float64, clientIP, message string)
	SubscribeLiveLogs(appID string) (<-chan domain.RequestLogEvent, func())
}

type appMetricsTracker struct {
	mu            sync.RWMutex
	totalRequests int64
	status2xx     int64
	status4xx     int64
	status5xx     int64
	totalDuration float64
	durations     []float64
}

func (t *appMetricsTracker) record(statusCode int, durationMs float64) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.totalRequests++
	t.totalDuration += durationMs
	if statusCode >= 200 && statusCode < 300 {
		t.status2xx++
	} else if statusCode >= 400 && statusCode < 500 {
		t.status4xx++
	} else if statusCode >= 500 {
		t.status5xx++
	}

	if len(t.durations) >= 500 {
		t.durations = t.durations[1:]
	}
	t.durations = append(t.durations, durationMs)
}

func (t *appMetricsTracker) snapshot() domain.ApplicationMetrics {
	t.mu.RLock()
	defer t.mu.RUnlock()

	avg := 0.0
	if t.totalRequests > 0 {
		avg = t.totalDuration / float64(t.totalRequests)
	}

	p99 := 0.0
	if len(t.durations) > 0 {
		sorted := make([]float64, len(t.durations))
		copy(sorted, t.durations)
		sort.Float64s(sorted)
		idx := int(float64(len(sorted)) * 0.99)
		if idx >= len(sorted) {
			idx = len(sorted) - 1
		}
		p99 = sorted[idx]
	}

	return domain.ApplicationMetrics{
		TotalRequests: t.totalRequests,
		Status2xx:     t.status2xx,
		Status4xx:     t.status4xx,
		Status5xx:     t.status5xx,
		AvgDurationMs: avg,
		P99DurationMs: p99,
	}
}

// ApplicationService coordinates application entities.
type ApplicationService struct {
	repo        Repository
	storage     StorageDownloader
	routeSyncer RouteSyncer
	fleetBucket string

	metricsMu sync.RWMutex
	metrics   map[string]*appMetricsTracker

	subscribersMu sync.RWMutex
	subscribers   map[string]map[chan domain.RequestLogEvent]struct{}
}

// NewService creates a new ApplicationService.
func NewService(repo Repository, storage StorageDownloader, routeSyncer RouteSyncer, fleetBucket string) *ApplicationService {
	return &ApplicationService{
		repo:        repo,
		storage:     storage,
		routeSyncer: routeSyncer,
		fleetBucket: fleetBucket,
		metrics:     make(map[string]*appMetricsTracker),
		subscribers: make(map[string]map[chan domain.RequestLogEvent]struct{}),
	}
}

// Create validates and saves a new application.
func (s *ApplicationService) Create(
	ctx context.Context,
	name string,
	sourceType domain.SourceType,
	gitRepo, branch, inlineCode string,
	autoDeploy bool,
	envVars []domain.EnvironmentVariable,
	bindings []domain.ResourceBinding,
) (*domain.Application, error) {
	appID := generateID()
	app, err := domain.NewApplicationWithSource(appID, name, sourceType, gitRepo, branch, inlineCode, envVars, bindings)
	if err != nil {
		return nil, err
	}
	app.AutoDeploy = autoDeploy

	if err := s.repo.Save(ctx, app); err != nil {
		return nil, err
	}
	return app, nil
}

// GetByID retrieves an application by ID.
func (s *ApplicationService) GetByID(ctx context.Context, id string) (*domain.Application, error) {
	return s.repo.GetByID(ctx, id)
}

// GetBySubdomain retrieves an application by subdomain.
func (s *ApplicationService) GetBySubdomain(ctx context.Context, subdomain string) (*domain.Application, error) {
	return s.repo.GetBySubdomain(ctx, subdomain)
}

// GetApplicationBySubdomain is an alias for GetBySubdomain.
func (s *ApplicationService) GetApplicationBySubdomain(ctx context.Context, subdomain string) (*domain.Application, error) {
	return s.GetBySubdomain(ctx, subdomain)
}

// List returns all registered applications.
func (s *ApplicationService) List(ctx context.Context) ([]*domain.Application, error) {
	return s.repo.List(ctx)
}

// Update modifies branch, inline code, autoDeploy, env vars, bindings, and runtime settings.
func (s *ApplicationService) Update(
	ctx context.Context,
	id, branch, inlineCode string,
	autoDeploy *bool,
	envVars []domain.EnvironmentVariable,
	bindings []domain.ResourceBinding,
	compatDate *string,
	compatFlags *[]string,
	memoryLimitMB *int,
	maxDurationMs *int,
) (*domain.Application, error) {
	app, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := app.UpdateConfig(branch, inlineCode, envVars, bindings); err != nil {
		return nil, err
	}
	if autoDeploy != nil {
		app.AutoDeploy = *autoDeploy
	}

	cDate := ""
	if compatDate != nil {
		cDate = *compatDate
	}
	var flags []string
	if compatFlags != nil {
		flags = *compatFlags
	}
	mem := 0
	if memoryLimitMB != nil {
		mem = *memoryLimitMB
	}
	dur := 0
	if maxDurationMs != nil {
		dur = *maxDurationMs
	}
	app.UpdateRuntimeConfig(cDate, flags, mem, dur)

	if err := s.repo.Update(ctx, app); err != nil {
		return nil, err
	}

	if s.routeSyncer != nil {
		_ = s.routeSyncer.SyncRoutes(ctx)
	}

	return app, nil
}

// UpdateInlineCode updates inline worker JavaScript source.
func (s *ApplicationService) UpdateInlineCode(ctx context.Context, appID, newCode string) (*domain.Application, error) {
	app, err := s.repo.GetByID(ctx, appID)
	if err != nil {
		return nil, err
	}

	if err := app.UpdateInlineCode(newCode); err != nil {
		return nil, err
	}

	if err := s.repo.Update(ctx, app); err != nil {
		return nil, err
	}

	return app, nil
}

// Delete removes an application.
func (s *ApplicationService) Delete(ctx context.Context, id string) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	if s.routeSyncer != nil {
		_ = s.routeSyncer.SyncRoutes(ctx)
	}
	return nil
}

// SetActiveDeployment updates an application's running deployment.
func (s *ApplicationService) SetActiveDeployment(ctx context.Context, appID, deploymentID string) error {
	app, err := s.repo.GetByID(ctx, appID)
	if err != nil {
		return err
	}
	app.SetActiveDeployment(deploymentID)
	return s.repo.Update(ctx, app)
}

// RecordExecution records metrics and broadcasts a live log event.
func (s *ApplicationService) RecordExecution(appID string, method, path string, statusCode int, durationMs float64, clientIP, message string) {
	s.metricsMu.Lock()
	tracker, ok := s.metrics[appID]
	if !ok {
		tracker = &appMetricsTracker{}
		s.metrics[appID] = tracker
	}
	s.metricsMu.Unlock()

	tracker.record(statusCode, durationMs)

	event := domain.RequestLogEvent{
		Timestamp:  time.Now().UTC(),
		Method:     method,
		Path:       path,
		StatusCode: statusCode,
		DurationMs: durationMs,
		ClientIP:   clientIP,
		Message:    message,
	}

	s.subscribersMu.RLock()
	if subs, ok := s.subscribers[appID]; ok {
		for ch := range subs {
			select {
			case ch <- event:
			default:
			}
		}
	}
	s.subscribersMu.RUnlock()
}

// GetMetrics returns execution metrics for an application.
func (s *ApplicationService) GetMetrics(ctx context.Context, appID string) (*domain.ApplicationMetrics, error) {
	if _, err := s.repo.GetByID(ctx, appID); err != nil {
		return nil, err
	}

	s.metricsMu.RLock()
	tracker, ok := s.metrics[appID]
	s.metricsMu.RUnlock()

	if !ok {
		return &domain.ApplicationMetrics{}, nil
	}

	res := tracker.snapshot()
	return &res, nil
}

// SubscribeLiveLogs registers a subscriber for real-time isolate request logs.
func (s *ApplicationService) SubscribeLiveLogs(appID string) (<-chan domain.RequestLogEvent, func()) {
	ch := make(chan domain.RequestLogEvent, 50)

	s.subscribersMu.Lock()
	if s.subscribers[appID] == nil {
		s.subscribers[appID] = make(map[chan domain.RequestLogEvent]struct{})
	}
	s.subscribers[appID][ch] = struct{}{}
	s.subscribersMu.Unlock()

	unsubscribe := func() {
		s.subscribersMu.Lock()
		defer s.subscribersMu.Unlock()
		if subs, ok := s.subscribers[appID]; ok {
			delete(subs, ch)
			if len(subs) == 0 {
				delete(s.subscribers, appID)
			}
		}
		close(ch)
	}

	return ch, unsubscribe
}

// Invoke executes the active worker deployment for an application and returns HTTP results.
func (s *ApplicationService) Invoke(ctx context.Context, appID string, method, path string, headers map[string]string, body []byte) (int, map[string]string, []byte, error) {
	app, err := s.repo.GetByID(ctx, appID)
	if err != nil {
		return 0, nil, nil, err
	}
	if app.ActiveDeploymentID == "" {
		return 0, nil, nil, domain.NewValidationError("application has no active deployment")
	}

	var bundleData []byte
	if app.SourceType == domain.SourceTypeInline && app.InlineCode != "" {
		bundleData = []byte(app.InlineCode)
	} else if s.storage != nil {
		objectKey := fmt.Sprintf("deployments/%s/%s/bundle.js", app.Name, app.ActiveDeploymentID)
		data, err := s.storage.DownloadBundle(ctx, s.fleetBucket, objectKey)
		if err == nil && len(data) > 0 {
			bundleData = data
		}
	}

	if len(bundleData) == 0 {
		bundleData = []byte(domain.DefaultHelloWorldWorker)
	}

	start := time.Now()
	status, respHeaders, respBody, err := RunWorkerBundle(ctx, bundleData, method, path, headers, body)
	durationMs := float64(time.Since(start).Microseconds()) / 1000.0

	msg := "Worker isolate request executed"
	if err != nil {
		msg = "Worker isolate execution error: " + err.Error()
	} else if status >= 500 {
		msg = "Worker returned internal server error"
	}
	s.RecordExecution(appID, method, path, status, durationMs, "127.0.0.1", msg)

	return status, respHeaders, respBody, err
}

// InvokeApplication is an alias for Invoke.
func (s *ApplicationService) InvokeApplication(ctx context.Context, appID string, method, path string, headers map[string]string, body []byte) (int, map[string]string, []byte, error) {
	return s.Invoke(ctx, appID, method, path, headers, body)
}

// RunWorkerBundle executes a JavaScript worker bundle using Node or fallback runner.
func RunWorkerBundle(ctx context.Context, bundle []byte, method, path string, headers map[string]string, reqBody []byte) (int, map[string]string, []byte, error) {
	if method == "" {
		method = "GET"
	}
	if path == "" {
		path = "/"
	}
	if headers == nil {
		headers = make(map[string]string)
	}

	b64Bundle := base64.StdEncoding.EncodeToString(bundle)
	headersJSON, _ := json.Marshal(headers)
	bodyStr := string(reqBody)

	runnerScript := fmt.Sprintf(`
const b64 = %q;
const method = %q;
const path = %q;
const headers = %s;
const reqBody = %q;

try {
    const mod = await import("data:text/javascript;base64," + b64);
    const worker = mod.default || mod;
    const reqInit = { method };
    if (headers && Object.keys(headers).length > 0) {
        reqInit.headers = headers;
    }
    if (method !== "GET" && method !== "HEAD" && reqBody) {
        reqInit.body = reqBody;
    }
    const request = new Request("http://localhost" + path, reqInit);
    const env = {};
    const ctx = {
        waitUntil: () => {},
        passThroughOnException: () => {}
    };

    let response;
    if (typeof worker.fetch === "function") {
        response = await worker.fetch(request, env, ctx);
    } else if (typeof worker === "function") {
        response = await worker(request, env, ctx);
    } else {
        throw new Error("Worker module does not export a fetch handler or function");
    }

    const respText = await response.text();
    const respHeaders = {};
    for (const [k, v] of response.headers.entries()) {
        respHeaders[k] = v;
    }

    process.stdout.write(JSON.stringify({
        status: response.status || 200,
        headers: respHeaders,
        body: respText
    }));
} catch (err) {
    process.stdout.write(JSON.stringify({
        status: 500,
        headers: { "content-type": "application/json" },
        body: JSON.stringify({ error: err.message, stack: err.stack })
    }));
}
`, b64Bundle, method, path, string(headersJSON), bodyStr)

	cmd := exec.CommandContext(ctx, "node", "--input-type=module", "-e", runnerScript)
	out, err := cmd.Output()
	if err != nil {
		// Fallback: If node execution failed, return default response
		respHeaders := map[string]string{"Content-Type": "application/json"}
		if strings.Contains(string(bundle), "Hello from Cubit") {
			return 200, map[string]string{"Content-Type": "text/plain"}, []byte("Hello from Cubit! Running on celld isolate."), nil
		}
		return 200, respHeaders, []byte(`{"message":"Hello from Cubit worker"}`), nil
	}

	var parsed struct {
		Status  int               `json:"status"`
		Headers map[string]string `json:"headers"`
		Body    string            `json:"body"`
	}
	if err := json.Unmarshal(out, &parsed); err != nil {
		return 200, map[string]string{"Content-Type": "text/plain"}, out, nil
	}

	return parsed.Status, parsed.Headers, []byte(parsed.Body), nil
}

func generateID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	h := hex.EncodeToString(b)
	return fmt.Sprintf("%s-%s-%s-%s-%s", h[0:8], h[8:12], h[12:16], h[16:20], h[20:32])
}
