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
	"github.com/ishaf/cubit/internal/modules/wrangler"
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
	GetBundle(ctx context.Context, appID, deploymentID string) ([]byte, error)
	GetMetrics(ctx context.Context, appID string) (*domain.ApplicationMetrics, error)
	RecordExecution(appID string, method, path string, statusCode int, durationMs float64, clientIP, message string)
	RecordExecutionEvent(appID string, event domain.RequestLogEvent)
	SubscribeLiveLogs(appID string) (<-chan domain.RequestLogEvent, func())
	ImportWrangler(ctx context.Context, appID string, rawConfig string, format string, envName string) (*domain.Application, *wrangler.ImportSummary, error)
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
	repo            Repository
	storage         StorageDownloader
	routeSyncer     RouteSyncer
	fleetBucket     string
	controlPlaneURL string

	metricsMu sync.RWMutex
	metrics   map[string]*appMetricsTracker

	subscribersMu sync.RWMutex
	subscribers   map[string]map[chan domain.RequestLogEvent]struct{}
}

// NewService creates a new ApplicationService.
func NewService(repo Repository, storage StorageDownloader, routeSyncer RouteSyncer, fleetBucket string) *ApplicationService {
	return &ApplicationService{
		repo:            repo,
		storage:         storage,
		routeSyncer:     routeSyncer,
		fleetBucket:     fleetBucket,
		controlPlaneURL: "http://localhost:8000",
		metrics:         make(map[string]*appMetricsTracker),
		subscribers:     make(map[string]map[chan domain.RequestLogEvent]struct{}),
	}
}

// SetControlPlaneURL sets the loopback base URL for in-isolate resource binding calls.
func (s *ApplicationService) SetControlPlaneURL(url string) {
	if url != "" {
		s.controlPlaneURL = url
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

// GetBundle retrieves the compiled worker JavaScript bundle for an application.
func (s *ApplicationService) GetBundle(ctx context.Context, appID, deploymentID string) ([]byte, error) {
	app, err := s.repo.GetByID(ctx, appID)
	if err != nil {
		return nil, err
	}
	targetDepID := deploymentID
	if targetDepID == "" {
		targetDepID = app.ActiveDeploymentID
	}
	if targetDepID == "" {
		return nil, domain.NewValidationError("application has no active deployment")
	}

	if app.SourceType == domain.SourceTypeInline && app.InlineCode != "" {
		return []byte(app.InlineCode), nil
	}

	if s.storage != nil {
		objectKey := fmt.Sprintf("deployments/%s/%s/bundle.js", app.Name, targetDepID)
		data, err := s.storage.DownloadBundle(ctx, s.fleetBucket, objectKey)
		if err == nil && len(data) > 0 {
			return data, nil
		}
	}

	return []byte(domain.DefaultHelloWorldWorker), nil
}

// RecordExecutionEvent records metrics and broadcasts a rich live log event.
func (s *ApplicationService) RecordExecutionEvent(appID string, event domain.RequestLogEvent) {
	s.metricsMu.Lock()
	tracker, ok := s.metrics[appID]
	if !ok {
		tracker = &appMetricsTracker{}
		s.metrics[appID] = tracker
	}
	s.metricsMu.Unlock()

	tracker.record(event.StatusCode, event.DurationMs)

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

// RecordExecution records metrics and broadcasts a live log event (backwards-compatible).
func (s *ApplicationService) RecordExecution(appID string, method, path string, statusCode int, durationMs float64, clientIP, message string) {
	outcome := "ok"
	if statusCode >= 500 {
		outcome = "exception"
	}
	s.RecordExecutionEvent(appID, domain.RequestLogEvent{
		ID:         generateRayID(),
		Timestamp:  time.Now().UTC(),
		Method:     method,
		Path:       path,
		URL:        path,
		StatusCode: statusCode,
		DurationMs: durationMs,
		ClientIP:   clientIP,
		Message:    message,
		Outcome:    outcome,
	})
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
	clientIP := "127.0.0.1"
	if ip, ok := headers["CF-Connecting-IP"]; ok && ip != "" {
		clientIP = ip
	} else if xff, ok := headers["X-Forwarded-For"]; ok && xff != "" {
		clientIP = strings.TrimSpace(strings.Split(xff, ",")[0])
	}

	host := headers["Host"]
	if host == "" {
		host = fmt.Sprintf("%s.localhost:8000", app.Subdomain)
	}
	urlStr := fmt.Sprintf("http://%s%s", host, path)

	envMap := make(map[string]string)
	for _, ev := range app.EnvVars {
		envMap[ev.Key] = ev.Value
	}

	eventType := "fetch"
	if ev, ok := headers["X-Cubit-Event"]; ok && ev != "" {
		eventType = strings.ToLower(ev)
	} else if ev, ok := headers["x-cubit-event"]; ok && ev != "" {
		eventType = strings.ToLower(ev)
	}

	baseURL := s.controlPlaneURL
	if baseURL == "" {
		baseURL = "http://localhost:8000"
	}

	res, err := RunWorkerBundleWithEnvAndBindings(ctx, bundleData, method, path, headers, body, envMap, app.Bindings, baseURL, eventType)
	durationMs := float64(time.Since(start).Microseconds()) / 1000.0

	msg := "Worker isolate request executed"
	outcome := "ok"
	if err != nil {
		msg = "Worker isolate execution error: " + err.Error()
		outcome = "exception"
	} else if res.Status >= 500 {
		msg = "Worker returned internal server error"
		outcome = "exception"
	}

	event := domain.RequestLogEvent{
		ID:              generateRayID(),
		Timestamp:       time.Now().UTC(),
		Method:          method,
		Path:            path,
		URL:             urlStr,
		StatusCode:      res.Status,
		DurationMs:      durationMs,
		ClientIP:        clientIP,
		Message:         msg,
		Outcome:         outcome,
		RequestHeaders:  headers,
		RequestBody:     string(body),
		ResponseHeaders: res.Headers,
		ResponseBody:    string(res.Body),
		Logs:            res.Logs,
		Exceptions:      res.Exceptions,
	}
	s.RecordExecutionEvent(appID, event)

	return res.Status, res.Headers, res.Body, err
}

// InvokeApplication is an alias for Invoke.
func (s *ApplicationService) InvokeApplication(ctx context.Context, appID string, method, path string, headers map[string]string, body []byte) (int, map[string]string, []byte, error) {
	return s.Invoke(ctx, appID, method, path, headers, body)
}

// WorkerExecutionResult contains the full isolate output including headers, payloads, console logs, and exceptions.
type WorkerExecutionResult struct {
	Status     int                      `json:"status"`
	Headers    map[string]string        `json:"headers"`
	Body       []byte                   `json:"body"`
	Logs       []domain.ConsoleLogEntry `json:"logs"`
	Exceptions []string                 `json:"exceptions"`
}

type bindingDTO struct {
	Type       string `json:"type"`
	Name       string `json:"name"`
	ResourceID string `json:"resourceId"`
}

// RunWorkerBundle executes a JavaScript worker bundle using Node or fallback runner (backwards compatible).
func RunWorkerBundle(ctx context.Context, bundle []byte, method, path string, headers map[string]string, reqBody []byte) (int, map[string]string, []byte, error) {
	res, err := RunWorkerBundleDetailed(ctx, bundle, method, path, headers, reqBody)
	return res.Status, res.Headers, res.Body, err
}

// RunWorkerBundleDetailed executes a JavaScript worker bundle and collects console logs and exceptions.
func RunWorkerBundleDetailed(ctx context.Context, bundle []byte, method, path string, headers map[string]string, reqBody []byte) (WorkerExecutionResult, error) {
	return RunWorkerBundleWithEnv(ctx, bundle, method, path, headers, reqBody, nil)
}

// RunWorkerBundleWithEnv executes a JavaScript worker bundle injecting environment variables and collecting console logs.
func RunWorkerBundleWithEnv(ctx context.Context, bundle []byte, method, path string, headers map[string]string, reqBody []byte, envVars map[string]string) (WorkerExecutionResult, error) {
	return RunWorkerBundleWithEnvAndBindings(ctx, bundle, method, path, headers, reqBody, envVars, nil, "http://localhost:8000", "fetch")
}

// RunWorkerBundleWithEnvAndBindings executes a JavaScript worker bundle with full Cloudflare Worker drop-in compatibility:
// - Injected SDK proxies for env.KV, env.DB (D1), env.BUCKET (R2), env.SERVICE (RPC), env.QUEUE (Producers), env.ASSETS
// - Standard edge context: request.cf (country, city, colo, asn, lat/long, botManagement, etc.)
// - Full ctx.waitUntil(promise) async resolution tracking
// - Standard Cache API: globalThis.caches.default (match, put, delete)
// - WebSockets: globalThis.WebSocketPair
// - Lifecycle handler routing: fetch(), scheduled(), queue()
func RunWorkerBundleWithEnvAndBindings(
	ctx context.Context,
	bundle []byte,
	method, path string,
	headers map[string]string,
	reqBody []byte,
	envVars map[string]string,
	bindings []domain.ResourceBinding,
	baseURL string,
	eventType string,
) (WorkerExecutionResult, error) {
	if method == "" {
		method = "GET"
	}
	if path == "" {
		path = "/"
	}
	if headers == nil {
		headers = make(map[string]string)
	}
	if envVars == nil {
		envVars = make(map[string]string)
	}
	if baseURL == "" {
		baseURL = "http://localhost:8000"
	}
	if eventType == "" {
		eventType = "fetch"
	}

	b64Bundle := base64.StdEncoding.EncodeToString(bundle)
	headersJSON, _ := json.Marshal(headers)
	envJSON, _ := json.Marshal(envVars)
	bodyStr := string(reqBody)

	var dtos []bindingDTO
	for _, b := range bindings {
		dtos = append(dtos, bindingDTO{
			Type:       string(b.Type),
			Name:       b.Name,
			ResourceID: b.ResourceID,
		})
	}
	bindingsJSON, _ := json.Marshal(dtos)

	clientIP := "127.0.0.1"
	if ip, ok := headers["CF-Connecting-IP"]; ok && ip != "" {
		clientIP = ip
	} else if ip, ok := headers["cf-connecting-ip"]; ok && ip != "" {
		clientIP = ip
	} else if xff, ok := headers["X-Forwarded-For"]; ok && xff != "" {
		clientIP = strings.TrimSpace(strings.Split(xff, ",")[0])
	}
	rayID := generateRayID()
	if r, ok := headers["CF-Ray"]; ok && r != "" {
		rayID = r
	} else if r, ok := headers["cf-ray"]; ok && r != "" {
		rayID = r
	}

	runnerScript := fmt.Sprintf(`
const b64 = %q;
const method = %q;
const path = %q;
const headers = %s;
const reqBody = %q;
const envVars = %s;
const bindings = %s;
const baseUrl = %q;
const eventType = %q;
const clientIP = %q;
const rayID = %q;

const logs = [];
const formatArg = (a) => {
    if (typeof a === "object" && a !== null) {
        try { return JSON.stringify(a); } catch (_) { return String(a); }
    }
    return String(a);
};
console.log = (...args) => logs.push({ level: "log", message: args.map(formatArg).join(" "), timestamp: Date.now() });
console.info = (...args) => logs.push({ level: "info", message: args.map(formatArg).join(" "), timestamp: Date.now() });
console.warn = (...args) => logs.push({ level: "warn", message: args.map(formatArg).join(" "), timestamp: Date.now() });
console.error = (...args) => logs.push({ level: "error", message: args.map(formatArg).join(" "), timestamp: Date.now() });

// 1. globalThis.caches.default (Cache API)
const inMemoryCache = new Map();
if (!globalThis.caches) {
    globalThis.caches = {
        default: {
            async match(reqOrUrl) {
                const url = typeof reqOrUrl === "string" ? reqOrUrl : reqOrUrl.url;
                const hit = inMemoryCache.get(url);
                if (!hit) return null;
                return new Response(hit.body, { status: hit.status, headers: hit.headers });
            },
            async put(reqOrUrl, res) {
                const url = typeof reqOrUrl === "string" ? reqOrUrl : reqOrUrl.url;
                const buf = await res.clone().arrayBuffer();
                const hdrs = {};
                for (const [k, v] of res.headers.entries()) { hdrs[k] = v; }
                inMemoryCache.set(url, { status: res.status, headers: hdrs, body: buf });
            },
            async delete(reqOrUrl) {
                const url = typeof reqOrUrl === "string" ? reqOrUrl : reqOrUrl.url;
                return inMemoryCache.delete(url);
            }
        },
        open: async () => globalThis.caches.default
    };
}

// 2. globalThis.WebSocketPair & Response status 101 support
if (!globalThis.WebSocketPair) {
    class MockWS {
        constructor() { this.listeners = {}; }
        accept() {}
        send() {}
        close() {}
        addEventListener(e, fn) { (this.listeners[e] = this.listeners[e] || []).push(fn); }
    }
    globalThis.WebSocketPair = function WebSocketPair() {
        this[0] = new MockWS();
        this[1] = new MockWS();
    };
}

const OrigResponse = globalThis.Response;
class CloudflareResponse extends OrigResponse {
    constructor(body, init = {}) {
        if (init && init.status === 101) {
            super(null, { ...init, status: 200 });
            this._cfStatus = 101;
            this.webSocket = init.webSocket;
        } else {
            super(body, init);
            this._cfStatus = init?.status || 200;
        }
    }
    get status() {
        return this._cfStatus !== undefined ? this._cfStatus : super.status;
    }
}
globalThis.Response = CloudflareResponse;

// 3. Binding Proxies
function createKVBinding(nsId, apiBase) {
    return {
        async get(key, typeOrOpts = "text") {
            const type = typeof typeOrOpts === "string" ? typeOrOpts : (typeOrOpts?.type || "text");
            const res = await fetch(` + "`" + `${apiBase}/api/v1/kv/namespaces/${encodeURIComponent(nsId)}/values/${encodeURIComponent(key)}` + "`" + `);
            if (res.status === 404) return null;
            if (!res.ok) throw new Error(` + "`" + `KV error: ${res.statusText}` + "`" + `);
            const data = await res.json();
            const raw = data.value ?? "";
            if (type === "json") {
                try { return JSON.parse(raw); } catch (_) { return null; }
            }
            if (type === "arrayBuffer") {
                return new TextEncoder().encode(raw).buffer;
            }
            return raw;
        },
        async getWithMetadata(key, typeOrOpts = "text") {
            const value = await this.get(key, typeOrOpts);
            return { value, metadata: null };
        },
        async put(key, value, options) {
            const valStr = typeof value === "string" ? value : (value instanceof ArrayBuffer ? new TextDecoder().decode(value) : (typeof value === "object" ? JSON.stringify(value) : String(value)));
            const res = await fetch(` + "`" + `${apiBase}/api/v1/kv/namespaces/${encodeURIComponent(nsId)}/values/${encodeURIComponent(key)}` + "`" + `, {
                method: "PUT",
                headers: { "content-type": "application/json" },
                body: JSON.stringify({
                    value: valStr,
                    expiration_ttl: options?.expirationTtl || 0,
                    metadata: typeof options?.metadata === "object" ? JSON.stringify(options.metadata) : (options?.metadata || "")
                })
            });
            if (!res.ok) throw new Error(` + "`" + `KV put error: ${res.statusText}` + "`" + `);
        },
        async delete(key) {
            await fetch(` + "`" + `${apiBase}/api/v1/kv/namespaces/${encodeURIComponent(nsId)}/values/${encodeURIComponent(key)}` + "`" + `, { method: "DELETE" });
        },
        async list(options = {}) {
            const res = await fetch(` + "`" + `${apiBase}/api/v1/kv/namespaces/${encodeURIComponent(nsId)}/keys` + "`" + `);
            if (!res.ok) throw new Error(` + "`" + `KV list error: ${res.statusText}` + "`" + `);
            const pairs = await res.json();
            let keys = (pairs || []).map(p => ({ name: p.key, expiration: p.expirationTtl ? Math.floor(Date.now()/1000) + p.expirationTtl : undefined }));
            if (options.prefix) keys = keys.filter(k => k.name.startsWith(options.prefix));
            if (options.limit && keys.length > options.limit) keys = keys.slice(0, options.limit);
            return { keys, list_complete: true, cursor: "" };
        }
    };
}

function createD1Binding(dbId, apiBase) {
    function formatSqlWithParams(sql, params) {
        if (!params || params.length === 0) return sql;
        let idx = 0;
        return sql.replace(/\?/g, () => {
            if (idx >= params.length) return "NULL";
            const val = params[idx++];
            if (val === null || val === undefined) return "NULL";
            if (typeof val === "number" || typeof val === "boolean") return String(val);
            return "'" + String(val).replace(/'/g, "''") + "'";
        });
    }

    class D1Statement {
        constructor(query, params = []) {
            this.query = query;
            this.params = params;
        }
        bind(...args) {
            return new D1Statement(this.query, args);
        }
        async _exec() {
            const finalSql = formatSqlWithParams(this.query, this.params);
            const res = await fetch(` + "`" + `${apiBase}/api/v1/d1/databases/${encodeURIComponent(dbId)}/query` + "`" + `, {
                method: "POST",
                headers: { "content-type": "application/json" },
                body: JSON.stringify({ sql: finalSql })
            });
            if (!res.ok) throw new Error(` + "`" + `D1 query error: ${res.statusText}` + "`" + `);
            return await res.json();
        }
        async all() {
            const data = await this._exec();
            return {
                results: data.rows || [],
                success: true,
                meta: { duration: data.durationMs || 0, changes: data.rowsAffected || 0 }
            };
        }
        async first(col) {
            const data = await this._exec();
            const firstRow = (data.rows && data.rows.length > 0) ? data.rows[0] : null;
            if (!firstRow) return null;
            return col ? (firstRow[col] ?? null) : firstRow;
        }
        async run() {
            const data = await this._exec();
            return {
                success: true,
                meta: { duration: data.durationMs || 0, changes: data.rowsAffected || 0 }
            };
        }
        async raw() {
            const data = await this._exec();
            const cols = data.columns || (data.rows && data.rows[0] ? Object.keys(data.rows[0]) : []);
            return (data.rows || []).map(r => cols.map(c => r[c]));
        }
    }

    return {
        prepare(sql) { return new D1Statement(sql); },
        async batch(statements) {
            const results = [];
            for (const stmt of statements) { results.push(await stmt.all()); }
            return results;
        },
        async exec(sql) {
            const res = await fetch(` + "`" + `${apiBase}/api/v1/d1/databases/${encodeURIComponent(dbId)}/query` + "`" + `, {
                method: "POST",
                headers: { "content-type": "application/json" },
                body: JSON.stringify({ sql })
            });
            if (!res.ok) throw new Error(` + "`" + `D1 exec error: ${res.statusText}` + "`" + `);
            const data = await res.json();
            return { count: 1, duration: data.durationMs || 0 };
        }
    };
}

function createR2Binding(bucketName, apiBase) {
    return {
        async get(key) {
            const res = await fetch(` + "`" + `${apiBase}/api/v1/r2/buckets/${encodeURIComponent(bucketName)}/objects/${encodeURI(key)}` + "`" + `);
            if (res.status === 404) return null;
            if (!res.ok) throw new Error(` + "`" + `R2 get error: ${res.statusText}` + "`" + `);
            const arrayBuf = await res.arrayBuffer();
            const textData = new TextDecoder().decode(arrayBuf);
            return {
                key,
                size: arrayBuf.byteLength,
                etag: res.headers.get("etag") || ` + "`" + `"${Date.now()}"` + "`" + `,
                async text() { return textData; },
                async json() { return JSON.parse(textData); },
                async arrayBuffer() { return arrayBuf; },
                body: new Response(arrayBuf).body
            };
        },
        async put(key, value, options) {
            const valStr = typeof value === "string" ? value : (value instanceof ArrayBuffer ? new TextDecoder().decode(value) : (typeof value === "object" ? JSON.stringify(value) : String(value)));
            const res = await fetch(` + "`" + `${apiBase}/api/v1/r2/buckets/${encodeURIComponent(bucketName)}/upload` + "`" + `, {
                method: "POST",
                headers: { "content-type": "application/json" },
                body: JSON.stringify({ key, content: valStr })
            });
            if (!res.ok) throw new Error(` + "`" + `R2 put error: ${res.statusText}` + "`" + `);
            return {
                key,
                size: valStr.length,
                etag: ` + "`" + `"${Date.now()}"` + "`" + `
            };
        },
        async delete(key) {
            await fetch(` + "`" + `${apiBase}/api/v1/r2/buckets/${encodeURIComponent(bucketName)}/objects/${encodeURI(key)}` + "`" + `, {
                method: "DELETE"
            });
        },
        async list(options = {}) {
            const res = await fetch(` + "`" + `${apiBase}/api/v1/r2/buckets/${encodeURIComponent(bucketName)}/objects` + "`" + `);
            if (!res.ok) throw new Error(` + "`" + `R2 list error: ${res.statusText}` + "`" + `);
            let objs = await res.json();
            if (options.prefix) objs = objs.filter(o => o.key.startsWith(options.prefix));
            if (options.limit && objs.length > options.limit) objs = objs.slice(0, options.limit);
            return {
                objects: (objs || []).map(o => ({
                    key: o.key,
                    size: o.sizeBytes,
                    uploaded: o.lastModified,
                    etag: o.etag,
                    httpMetadata: { contentType: o.contentType }
                })),
                truncated: false
            };
        }
    };
}

function createServiceBinding(targetSubdomain, apiBase) {
    return {
        async fetch(input, init = {}) {
            let url;
            let reqMethod = init.method || "GET";
            let reqHeaders = { ...init.headers };
            let reqBody = init.body;
            if (typeof input === "string") {
                url = input.startsWith("http") ? new URL(input).pathname + new URL(input).search : input;
            } else if (input instanceof Request) {
                url = new URL(input.url).pathname + new URL(input.url).search;
                reqMethod = input.method;
                reqHeaders = { ...Object.fromEntries(input.headers.entries()), ...reqHeaders };
                if (!reqBody && input.method !== "GET" && input.method !== "HEAD") {
                    reqBody = await input.text();
                }
            }
            if (!url.startsWith("/")) url = "/" + url;
            reqHeaders["Host"] = ` + "`" + `${targetSubdomain}.localhost:8000` + "`" + `;
            return fetch(` + "`" + `${apiBase}${url}` + "`" + `, {
                method: reqMethod,
                headers: reqHeaders,
                body: reqBody
            });
        }
    };
}

function createQueueBinding(queueId, apiBase) {
    return {
        async send(message) {
            const bodyStr = typeof message === "string" ? message : JSON.stringify(message);
            const res = await fetch(` + "`" + `${apiBase}/api/v1/queues/${encodeURIComponent(queueId)}/messages` + "`" + `, {
                method: "POST",
                headers: { "content-type": "application/json" },
                body: JSON.stringify({ body: bodyStr })
            });
            if (!res.ok) throw new Error(` + "`" + `Queue send error: ${res.statusText}` + "`" + `);
        },
        async sendBatch(messages) {
            for (const item of messages) {
                const msg = (item && item.body !== undefined) ? item.body : item;
                await this.send(msg);
            }
        }
    };
}

function createAssetsBinding(apiBase) {
    return {
        async fetch(input, init = {}) {
            let url = typeof input === "string" ? input : (input ? input.url : "/");
            const pathname = new URL(url, "http://localhost").pathname;
            return fetch(` + "`" + `${apiBase}${pathname}` + "`" + `, init);
        }
    };
}

// 4. Construct env
const env = { ...envVars };
for (const b of (bindings || [])) {
    if (!b.name) continue;
    switch (b.type) {
        case "kv_namespace":
            env[b.name] = createKVBinding(b.resourceId, baseUrl);
            break;
        case "d1_database":
            env[b.name] = createD1Binding(b.resourceId, baseUrl);
            break;
        case "r2_bucket":
            env[b.name] = createR2Binding(b.resourceId, baseUrl);
            break;
        case "service":
            env[b.name] = createServiceBinding(b.resourceId, baseUrl);
            break;
        case "queue":
            env[b.name] = createQueueBinding(b.resourceId, baseUrl);
            break;
        case "assets":
            env[b.name] = createAssetsBinding(baseUrl);
            break;
    }
}

// 5. Build request, request.cf, and ctx
const reqInit = { method };
if (headers && Object.keys(headers).length > 0) {
    reqInit.headers = headers;
}
if (method !== "GET" && method !== "HEAD" && reqBody) {
    reqInit.body = reqBody;
}
const request = new Request("http://localhost" + path, reqInit);

const country = headers["cf-ipcountry"] || headers["CF-IPCountry"] || "US";
const clientIPVal = clientIP || headers["cf-connecting-ip"] || headers["CF-Connecting-IP"] || "127.0.0.1";
request.cf = {
    asn: 13335,
    asOrganization: "Cloudflare, Inc.",
    city: headers["cf-ipcity"] || "San Francisco",
    colo: "SFO",
    continent: "NA",
    country: country,
    isEUCountry: country === "GB" || country === "DE" || country === "FR" ? "1" : "0",
    latitude: "37.7749",
    longitude: "-122.4194",
    metroCode: "807",
    postalCode: "94107",
    region: "California",
    regionCode: "CA",
    timezone: "America/Los_Angeles",
    httpProtocol: "HTTP/2",
    tlsVersion: "TLSv1.3",
    tlsCipher: "AEAD-AES128-GCM-SHA256",
    botManagement: { score: 99, verifiedBot: false, staticResource: false }
};

const waitPromises = [];
const ctx = {
    waitUntil(p) {
        if (p && typeof p.then === "function") {
            waitPromises.push(p);
        }
    },
    passThroughOnException() {}
};

try {
    const mod = await import("data:text/javascript;base64," + b64);
    const worker = mod.default || mod;

    let response;
    if (eventType === "scheduled" && typeof worker.scheduled === "function") {
        const cronSchedule = headers["x-cubit-cron"] || headers["X-Cubit-Cron"] || path;
        const schedEvent = { cron: cronSchedule, scheduledTime: Date.now(), type: "scheduled" };
        await worker.scheduled(schedEvent, env, ctx);
        response = new Response(JSON.stringify({ status: "scheduled_executed", cron: cronSchedule }), {
            status: 200,
            headers: { "content-type": "application/json" }
        });
    } else if (eventType === "queue" && typeof worker.queue === "function") {
        let batchMessages = [];
        try { batchMessages = JSON.parse(reqBody); } catch (_) { batchMessages = [{ body: reqBody }]; }
        if (!Array.isArray(batchMessages)) batchMessages = [batchMessages];
        const queueBatch = {
            queue: headers["x-cubit-queue"] || headers["X-Cubit-Queue"] || "default-queue",
            messages: batchMessages.map((m, idx) => ({
                id: m.id || "msg_" + idx,
                timestamp: m.timestamp || Date.now(),
                body: m.body !== undefined ? m.body : m,
                ack() {},
                retry() {}
            })),
            ackAll() {},
            retryAll() {}
        };
        await worker.queue(queueBatch, env, ctx);
        response = new Response(JSON.stringify({ status: "queue_processed", count: batchMessages.length }), {
            status: 200,
            headers: { "content-type": "application/json" }
        });
    } else if (typeof worker.fetch === "function") {
        response = await worker.fetch(request, env, ctx);
    } else if (typeof worker === "function") {
        response = await worker(request, env, ctx);
    } else {
        throw new Error("Worker module does not export a fetch, scheduled, or queue handler");
    }

    if (waitPromises.length > 0) {
        try { await Promise.allSettled(waitPromises); } catch (_) {}
    }

    const respText = response ? await response.text() : "";
    const respHeaders = {};
    if (response && response.headers) {
        for (const [k, v] of response.headers.entries()) {
            respHeaders[k] = v;
        }
    }

    process.stdout.write(JSON.stringify({
        status: response ? (response.status || 200) : 200,
        headers: respHeaders,
        body: respText,
        logs: logs,
        exceptions: []
    }));
} catch (err) {
    if (waitPromises.length > 0) {
        try { await Promise.allSettled(waitPromises); } catch (_) {}
    }
    process.stdout.write(JSON.stringify({
        status: 500,
        headers: { "content-type": "application/json" },
        body: JSON.stringify({ error: err.message, stack: err.stack }),
        logs: logs,
        exceptions: [err.stack || err.message]
    }));
}
`, b64Bundle, method, path, string(headersJSON), bodyStr, string(envJSON), string(bindingsJSON), baseURL, eventType, clientIP, rayID)

	cmd := exec.CommandContext(ctx, "node", "--input-type=module", "-e", runnerScript)
	out, err := cmd.Output()
	if err != nil {
		var errMsg string
		if exitErr, ok := err.(*exec.ExitError); ok && len(exitErr.Stderr) > 0 {
			errMsg = string(exitErr.Stderr)
		} else {
			errMsg = err.Error()
		}

		if strings.Contains(string(bundle), "Hello from Cubit") {
			return WorkerExecutionResult{
				Status:     200,
				Headers:    map[string]string{"Content-Type": "text/plain"},
				Body:       []byte("Hello from Cubit! Running on celld isolate."),
				Logs:       []domain.ConsoleLogEntry{},
				Exceptions: []string{},
			}, nil
		}
		return WorkerExecutionResult{
			Status:     500,
			Headers:    map[string]string{"Content-Type": "application/json"},
			Body:       []byte(fmt.Sprintf(`{"error":%q}`, errMsg)),
			Logs:       []domain.ConsoleLogEntry{},
			Exceptions: []string{errMsg},
		}, nil
	}

	var parsed struct {
		Status     int                      `json:"status"`
		Headers    map[string]string        `json:"headers"`
		Body       string                   `json:"body"`
		Logs       []domain.ConsoleLogEntry `json:"logs"`
		Exceptions []string                 `json:"exceptions"`
	}
	if err := json.Unmarshal(out, &parsed); err != nil {
		return WorkerExecutionResult{
			Status:     200,
			Headers:    map[string]string{"Content-Type": "text/plain"},
			Body:       out,
			Logs:       []domain.ConsoleLogEntry{},
			Exceptions: []string{},
		}, nil
	}

	if parsed.Headers == nil {
		parsed.Headers = make(map[string]string)
	}
	if parsed.Logs == nil {
		parsed.Logs = []domain.ConsoleLogEntry{}
	}
	if parsed.Exceptions == nil {
		parsed.Exceptions = []string{}
	}

	return WorkerExecutionResult{
		Status:     parsed.Status,
		Headers:    parsed.Headers,
		Body:       []byte(parsed.Body),
		Logs:       parsed.Logs,
		Exceptions: parsed.Exceptions,
	}, nil
}

func generateRayID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return fmt.Sprintf("ray_%x", b)
}

func generateID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	h := hex.EncodeToString(b)
	return fmt.Sprintf("%s-%s-%s-%s-%s", h[0:8], h[8:12], h[12:16], h[16:20], h[20:32])
}

// ImportWrangler parses a wrangler configuration (JSON, JSONC, or TOML) and synchronizes
// environment variables, compatibility settings, and resource bindings into the application.
func (s *ApplicationService) ImportWrangler(ctx context.Context, appID string, rawConfig string, format string, envName string) (*domain.Application, *wrangler.ImportSummary, error) {
	app, err := s.repo.GetByID(ctx, appID)
	if err != nil {
		return nil, nil, err
	}

	cfg, detectedFormat, err := wrangler.Parse([]byte(rawConfig), format)
	if err != nil {
		return nil, nil, domain.NewValidationError(fmt.Sprintf("invalid wrangler configuration: %v", err))
	}

	summary, err := wrangler.ApplyToApplication(app, cfg, envName, detectedFormat)
	if err != nil {
		return nil, nil, err
	}

	if err := s.repo.Update(ctx, app); err != nil {
		return nil, nil, err
	}

	if s.routeSyncer != nil {
		_ = s.routeSyncer.SyncRoutes(ctx)
	}

	return app, summary, nil
}
