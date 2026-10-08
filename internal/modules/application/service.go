package application

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ishaf/cubit/internal/domain"
	"github.com/ishaf/cubit/internal/modules/runtime"
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

// DomainRouteRegistrar extracts and registers custom domain routes for an application.
type DomainRouteRegistrar interface {
	RegisterRoutes(ctx context.Context, appID string, routes []string) ([]*domain.Domain, error)
}

// Service defines application business operations.
type Service interface {
	Create(ctx context.Context, name string, sourceType domain.SourceType, gitRepo, branch, rootDir, inlineCode string, autoDeploy bool, envVars []domain.EnvironmentVariable, bindings []domain.ResourceBinding) (*domain.Application, error)
	GetByID(ctx context.Context, id string) (*domain.Application, error)
	GetBySubdomain(ctx context.Context, subdomain string) (*domain.Application, error)
	List(ctx context.Context) ([]*domain.Application, error)
	Update(ctx context.Context, id, branch, rootDir, inlineCode string, autoDeploy *bool, envVars []domain.EnvironmentVariable, bindings []domain.ResourceBinding, compatDate *string, compatFlags *[]string, memoryLimitMB *int, maxDurationMs *int) (*domain.Application, error)
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
	GetAsset(ctx context.Context, appID, assetPath string) ([]byte, string, error)
}

type appMetricsTracker struct {
	mu                sync.RWMutex
	totalRequests     int64
	status2xx         int64
	status4xx         int64
	status5xx         int64
	totalDuration     float64
	durations         []float64
	requestsByCountry map[string]int64
	requestsByColo    map[string]int64
	lastInvokedAt     *time.Time
	recentEvents      []domain.RequestLogEvent
}

func getHeaderCaseInsensitive(headers map[string]string, key string) string {
	for k, v := range headers {
		if strings.EqualFold(k, key) {
			return v
		}
	}
	return ""
}

func (t *appMetricsTracker) recordEvent(event domain.RequestLogEvent) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.totalRequests++
	t.totalDuration += event.DurationMs
	if event.StatusCode >= 200 && event.StatusCode < 300 {
		t.status2xx++
	} else if event.StatusCode >= 400 && event.StatusCode < 500 {
		t.status4xx++
	} else if event.StatusCode >= 500 {
		t.status5xx++
	}

	if len(t.durations) >= 500 {
		t.durations = t.durations[1:]
	}
	t.durations = append(t.durations, event.DurationMs)

	now := event.Timestamp
	if now.IsZero() {
		now = time.Now().UTC()
	}
	t.lastInvokedAt = &now

	country := "US"
	if event.CF != nil {
		if c, ok := event.CF["country"].(string); ok && c != "" {
			country = strings.ToUpper(c)
		}
	}
	if country == "US" {
		if c := getHeaderCaseInsensitive(event.RequestHeaders, "cf-ipcountry"); c != "" {
			country = strings.ToUpper(c)
		}
	}
	if t.requestsByCountry == nil {
		t.requestsByCountry = make(map[string]int64)
	}
	t.requestsByCountry[country]++

	colo := "SFO"
	if event.CF != nil {
		if col, ok := event.CF["colo"].(string); ok && col != "" {
			colo = strings.ToUpper(col)
		}
	}
	if colo == "SFO" {
		if col := getHeaderCaseInsensitive(event.RequestHeaders, "cf-colo"); col != "" {
			colo = strings.ToUpper(col)
		}
	}
	if t.requestsByColo == nil {
		t.requestsByColo = make(map[string]int64)
	}
	t.requestsByColo[colo]++

	t.recentEvents = append([]domain.RequestLogEvent{event}, t.recentEvents...)
	if len(t.recentEvents) > 10 {
		t.recentEvents = t.recentEvents[:10]
	}
}

func (t *appMetricsTracker) record(statusCode int, durationMs float64) {
	t.recordEvent(domain.RequestLogEvent{
		Timestamp:  time.Now().UTC(),
		StatusCode: statusCode,
		DurationMs: durationMs,
	})
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

	successRate := 100.0
	errorRate := 0.0
	if t.totalRequests > 0 {
		successRate = (float64(t.status2xx) / float64(t.totalRequests)) * 100.0
		errorRate = (float64(t.status4xx+t.status5xx) / float64(t.totalRequests)) * 100.0
	}

	countryMap := make(map[string]int64)
	for k, v := range t.requestsByCountry {
		countryMap[k] = v
	}

	coloMap := make(map[string]int64)
	for k, v := range t.requestsByColo {
		coloMap[k] = v
	}

	eventsCopy := make([]domain.RequestLogEvent, len(t.recentEvents))
	copy(eventsCopy, t.recentEvents)

	return domain.ApplicationMetrics{
		TotalRequests:     t.totalRequests,
		Status2xx:         t.status2xx,
		Status4xx:         t.status4xx,
		Status5xx:         t.status5xx,
		AvgDurationMs:     avg,
		P99DurationMs:     p99,
		SuccessRate:       successRate,
		ErrorRate:         errorRate,
		RequestsByCountry: countryMap,
		RequestsByColo:    coloMap,
		LastInvokedAt:     t.lastInvokedAt,
		RecentEvents:      eventsCopy,
	}
}

// ApplicationService coordinates application entities.
type ApplicationService struct {
	repo            Repository
	storage         StorageDownloader
	routeSyncer     RouteSyncer
	fleetBucket     string
	controlPlaneURL string
	port            int
	registrarMu     sync.RWMutex
	domainRegistrar DomainRouteRegistrar

	metricsMu sync.RWMutex
	metrics   map[string]*appMetricsTracker

	subscribersMu sync.RWMutex
	subscribers   map[string]map[chan domain.RequestLogEvent]struct{}

	executor runtime.WorkerExecutor
}

// NewService creates a new ApplicationService with optional WorkerExecutor.
func NewService(repo Repository, storage StorageDownloader, routeSyncer RouteSyncer, fleetBucket string, executor ...runtime.WorkerExecutor) *ApplicationService {
	var exec runtime.WorkerExecutor
	if len(executor) > 0 && executor[0] != nil {
		exec = executor[0]
	} else {
		exec = runtime.NewNodeWorkerExecutor()
	}
	return &ApplicationService{
		repo:            repo,
		storage:         storage,
		routeSyncer:     routeSyncer,
		fleetBucket:     fleetBucket,
		controlPlaneURL: "http://localhost:8000",
		port:            8000,
		metrics:         make(map[string]*appMetricsTracker),
		subscribers:     make(map[string]map[chan domain.RequestLogEvent]struct{}),
		executor:        exec,
	}
}

// SetPort configures the listen port and updates the default control plane URL accordingly.
func (s *ApplicationService) SetPort(port int) {
	if port > 0 {
		s.port = port
		s.controlPlaneURL = fmt.Sprintf("http://localhost:%d", port)
	}
}

// SetControlPlaneURL sets the control plane base URL.
func (s *ApplicationService) SetControlPlaneURL(urlStr string) {
	s.controlPlaneURL = urlStr
	if u, err := url.Parse(urlStr); err == nil && u.Port() != "" {
		if p, err := strconv.Atoi(u.Port()); err == nil && p > 0 {
			s.port = p
		}
	}
}

// SetDomainRegistrar sets the domain registrar for automatic ingress route registration.
func (s *ApplicationService) SetDomainRegistrar(registrar DomainRouteRegistrar) {
	s.registrarMu.Lock()
	defer s.registrarMu.Unlock()
	s.domainRegistrar = registrar
}

func (s *ApplicationService) getDomainRegistrar() DomainRouteRegistrar {
	s.registrarMu.RLock()
	defer s.registrarMu.RUnlock()
	return s.domainRegistrar
}

// Create validates and saves a new application.
func (s *ApplicationService) Create(
	ctx context.Context,
	name string,
	sourceType domain.SourceType,
	gitRepo, branch, rootDir, inlineCode string,
	autoDeploy bool,
	envVars []domain.EnvironmentVariable,
	bindings []domain.ResourceBinding,
) (*domain.Application, error) {
	appID := generateID()
	app, err := domain.NewApplicationWithSourceAndDir(appID, name, sourceType, gitRepo, branch, rootDir, inlineCode, envVars, bindings)
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

// Update modifies branch, rootDir, inline code, autoDeploy, env vars, bindings, and runtime settings.
func (s *ApplicationService) Update(
	ctx context.Context,
	id, branch, rootDir, inlineCode string,
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
	if strings.TrimSpace(rootDir) != "" || rootDir == "." {
		app.SetRootDir(rootDir)
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

	tracker.recordEvent(event)

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
	if ip := getHeaderCaseInsensitive(headers, "CF-Connecting-IP"); ip != "" {
		clientIP = ip
	} else if xff := getHeaderCaseInsensitive(headers, "X-Forwarded-For"); xff != "" {
		clientIP = strings.TrimSpace(strings.Split(xff, ",")[0])
	}

	host := headers["Host"]
	if host == "" {
		port := s.port
		if port <= 0 {
			port = 8000
		}
		if port == 80 || port == 443 {
			host = fmt.Sprintf("%s.localhost", app.Subdomain)
		} else {
			host = fmt.Sprintf("%s.localhost:%d", app.Subdomain, port)
		}
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

	res, err := s.executor.Execute(ctx, runtime.ExecutionPayload{
		Bundle:      bundleData,
		Method:      method,
		Path:        path,
		Headers:     headers,
		Body:        body,
		EnvVars:     envMap,
		Bindings:    app.Bindings,
		BaseURL:     baseURL,
		EventType:   eventType,
		TargetAppID: app.ID,
	})
	if res == nil {
		res = &runtime.WorkerExecutionResult{
			Status:     http.StatusInternalServerError,
			Headers:    map[string]string{"Content-Type": "application/json"},
			Body:       []byte(`{"error":"worker execution failed"}`),
			Logs:       []domain.ConsoleLogEntry{},
			Exceptions: []string{},
		}
		if err != nil {
			res.Exceptions = append(res.Exceptions, err.Error())
		}
	}
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

	cfData := res.CF
	if cfData == nil {
		country := strings.ToUpper(getHeaderCaseInsensitive(headers, "cf-ipcountry"))
		if country == "" {
			country = "US"
		}
		colo := strings.ToUpper(getHeaderCaseInsensitive(headers, "cf-colo"))
		if colo == "" {
			switch country {
			case "DE":
				colo = "FRA"
			case "GB":
				colo = "LHR"
			case "JP":
				colo = "NRT"
			case "AU":
				colo = "SYD"
			case "SG":
				colo = "SIN"
			default:
				colo = "SFO"
			}
		}
		city := getHeaderCaseInsensitive(headers, "cf-ipcity")
		if city == "" {
			switch country {
			case "DE":
				city = "Frankfurt"
			case "GB":
				city = "London"
			case "JP":
				city = "Tokyo"
			default:
				city = "San Francisco"
			}
		}
		cfData = map[string]interface{}{
			"country": country,
			"colo":    colo,
			"city":    city,
			"asn":     13335,
		}
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
		CF:              cfData,
	}
	s.RecordExecutionEvent(appID, event)

	return res.Status, res.Headers, res.Body, err
}

// InvokeApplication is an alias for Invoke.
func (s *ApplicationService) InvokeApplication(ctx context.Context, appID string, method, path string, headers map[string]string, body []byte) (int, map[string]string, []byte, error) {
	return s.Invoke(ctx, appID, method, path, headers, body)
}

// GetAsset retrieves a static asset for the given application.
func (s *ApplicationService) GetAsset(ctx context.Context, appID, assetPath string) ([]byte, string, error) {
	app, err := s.repo.GetByID(ctx, appID)
	if err != nil || app == nil {
		return nil, "", domain.NewNotFoundError("application not found")
	}

	cleanPath := strings.TrimPrefix(filepath.Clean("/"+assetPath), "/")
	if cleanPath == "" || cleanPath == "." {
		cleanPath = "index.html"
	}

	// 1. Try downloading from storage if active deployment exists
	if app.ActiveDeploymentID != "" && s.storage != nil {
		objectKey := fmt.Sprintf("deployments/%s/%s/assets/%s", app.Name, app.ActiveDeploymentID, cleanPath)
		data, err := s.storage.DownloadBundle(ctx, s.fleetBucket, objectKey)
		if err == nil && len(data) > 0 {
			contentType := mime.TypeByExtension(filepath.Ext(cleanPath))
			if contentType == "" {
				contentType = detectContentType(cleanPath, data)
			}
			return data, contentType, nil
		}
	}

	// 2. Check local disk fallback if app has a rootDir / assets binding
	for _, b := range app.Bindings {
		if b.Type == domain.BindingTypeAssets && b.ResourceID != "" {
			candidates := []string{
				filepath.Join(b.ResourceID, cleanPath),
			}
			if app.RootDir != "" {
				cleanRoot := domain.CleanRootDir(app.RootDir)
				candidates = append(candidates, filepath.Join(cleanRoot, b.ResourceID, cleanPath))
				candidates = append(candidates, filepath.Join(cleanRoot, cleanPath))
			}
			for _, candidate := range candidates {
				if stat, err := os.Stat(candidate); err == nil && !stat.IsDir() {
					if data, err := os.ReadFile(candidate); err == nil {
						contentType := mime.TypeByExtension(filepath.Ext(candidate))
						if contentType == "" {
							contentType = detectContentType(candidate, data)
						}
						return data, contentType, nil
					}
				}
			}
		}
	}

	return nil, "", domain.NewNotFoundError(fmt.Sprintf("asset %q not found", cleanPath))
}

func detectContentType(filePath string, data []byte) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".js", ".mjs":
		return "application/javascript; charset=utf-8"
	case ".css":
		return "text/css; charset=utf-8"
	case ".html", ".htm":
		return "text/html; charset=utf-8"
	case ".json":
		return "application/json; charset=utf-8"
	case ".svg":
		return "image/svg+xml"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".ico":
		return "image/x-icon"
	case ".webp":
		return "image/webp"
	case ".woff2":
		return "font/woff2"
	case ".woff":
		return "font/woff"
	case ".ttf":
		return "font/ttf"
	case ".txt":
		return "text/plain; charset=utf-8"
	default:
		ct := http.DetectContentType(data)
		if ct != "" {
			return ct
		}
		return "application/octet-stream"
	}
}

// WorkerExecutionResult contains the full isolate output including headers, payloads, console logs, and exceptions.
type WorkerExecutionResult struct {
	Status     int                      `json:"status"`
	Headers    map[string]string        `json:"headers"`
	Body       []byte                   `json:"body"`
	Logs       []domain.ConsoleLogEntry `json:"logs"`
	Exceptions []string                 `json:"exceptions"`
	CF         map[string]interface{}   `json:"cf"`
}

// RunWorkerBundle executes a JavaScript worker bundle using NodeWorkerExecutor (backwards compatible).
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

// RunWorkerBundleWithEnvAndBindings delegates execution to runtime.NodeWorkerExecutor (backwards compatible).
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
	appID ...string,
) (WorkerExecutionResult, error) {
	targetAppID := ""
	if len(appID) > 0 {
		targetAppID = appID[0]
	}
	exec := runtime.NewNodeWorkerExecutor()
	res, err := exec.Execute(ctx, runtime.ExecutionPayload{
		Bundle:      bundle,
		Method:      method,
		Path:        path,
		Headers:     headers,
		Body:        reqBody,
		EnvVars:     envVars,
		Bindings:    bindings,
		BaseURL:     baseURL,
		EventType:   eventType,
		TargetAppID: targetAppID,
	})
	if res == nil {
		return WorkerExecutionResult{}, err
	}
	return WorkerExecutionResult{
		Status:     res.Status,
		Headers:    res.Headers,
		Body:       res.Body,
		Logs:       res.Logs,
		Exceptions: res.Exceptions,
		CF:         res.CF,
	}, err
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

	// Extract and register routes if domain registrar is provided
	registrar := s.getDomainRegistrar()
	if registrar != nil {
		_, extractedRoutes, sanitizeErr := wrangler.SanitizeForCelldWithEnv([]byte(rawConfig), envName)
		if sanitizeErr == nil && len(extractedRoutes) > 0 {
			registered, regErr := registrar.RegisterRoutes(ctx, app.ID, extractedRoutes)
			if regErr == nil {
				summary.ImportedRoutesCount = len(registered)
				summary.ExtractedRoutes = extractedRoutes
			}
		}
	}

	if err := s.repo.Update(ctx, app); err != nil {
		return nil, nil, err
	}

	if s.routeSyncer != nil {
		_ = s.routeSyncer.SyncRoutes(ctx)
	}

	return app, summary, nil
}
