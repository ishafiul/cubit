package domain

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

var (
	appNameRegex   = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)
	envKeyRegex    = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	gitRepoURLRegex = regexp.MustCompile(`^(https?://|git@|[a-zA-Z0-9_.-]+/[a-zA-Z0-9_.-]+)`)
)

// ApplicationStatus represents the state of a worker application.
type ApplicationStatus string

const (
	AppStatusCreated  ApplicationStatus = "created"
	AppStatusBuilding ApplicationStatus = "building"
	AppStatusRunning  ApplicationStatus = "running"
	AppStatusStopped  ApplicationStatus = "stopped"
	AppStatusFailed   ApplicationStatus = "failed"
)

// SourceType defines whether the application code originates from Git or inline code.
type SourceType string

const (
	SourceTypeGit    SourceType = "git"
	SourceTypeInline SourceType = "inline"
)

// DefaultHelloWorldWorker is the initial standard template for inline Cloudflare Workers.
const DefaultHelloWorldWorker = `export default {
  async fetch(request, env, ctx) {
    return new Response("Hello World from Cubit Worker!", {
      headers: { "content-type": "text/plain; charset=utf-8" },
    });
  },
};`

// BindingType identifies Cloudflare Worker resource bindings.
type BindingType string

const (
	BindingTypeKV            BindingType = "kv_namespace"
	BindingTypeD1            BindingType = "d1_database"
	BindingTypeR2            BindingType = "r2_bucket"
	BindingTypeQueue         BindingType = "queue"
	BindingTypeWorkflow      BindingType = "workflow"
	BindingTypeService       BindingType = "service"
	BindingTypeAssets        BindingType = "assets"
	BindingTypeDurableObject BindingType = "durable_object"
	BindingTypeContainer     BindingType = "container"
)

// EnvironmentVariable represents a key-value pair injected into a Worker isolate.
type EnvironmentVariable struct {
	Key      string `json:"key"`
	Value    string `json:"value"`
	IsSecret bool   `json:"isSecret"`
}

// ResourceBinding represents a cloud binding to a Worker (e.g. KV, R2, D1, Service RPC, Durable Object, Workflow, Container).
type ResourceBinding struct {
	Type             BindingType       `json:"type"`
	Name             string            `json:"name"`
	ResourceID       string            `json:"resourceId"`
	ClassName        string            `json:"className,omitempty"`
	ScriptName       string            `json:"scriptName,omitempty"`
	Environment      string            `json:"environment,omitempty"`
	WorkflowName     string            `json:"workflowName,omitempty"`
	ContainerName    string            `json:"containerName,omitempty"`
	Image            string            `json:"image,omitempty"`
	Port             int               `json:"port,omitempty"`
	ContainerEnvVars map[string]string `json:"containerEnvVars,omitempty"`
}

// MigrationRenamedClass represents a renamed class in a Durable Object migration step.
type MigrationRenamedClass struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// MigrationStep represents a Durable Object class migration history entry.
type MigrationStep struct {
	Tag            string                  `json:"tag"`
	NewClasses     []string                `json:"newClasses,omitempty"`
	RenamedClasses []MigrationRenamedClass `json:"renamedClasses,omitempty"`
	DeletedClasses []string                `json:"deletedClasses,omitempty"`
}

// ApplicationMetrics encapsulates real-time worker execution telemetry.
type ApplicationMetrics struct {
	TotalRequests     int64                  `json:"totalRequests"`
	Status2xx         int64                  `json:"status2xx"`
	Status4xx         int64                  `json:"status4xx"`
	Status5xx         int64                  `json:"status5xx"`
	AvgDurationMs     float64                `json:"avgDurationMs"`
	P99DurationMs     float64                `json:"p99DurationMs"`
	SuccessRate       float64                `json:"successRate"`
	ErrorRate         float64                `json:"errorRate"`
	RequestsByCountry map[string]int64       `json:"requestsByCountry,omitempty"`
	RequestsByColo    map[string]int64       `json:"requestsByColo,omitempty"`
	LastInvokedAt     *time.Time             `json:"lastInvokedAt,omitempty"`
	RecentEvents      []RequestLogEvent      `json:"recentEvents,omitempty"`
}

// ConsoleLogEntry represents an isolate console log output.
type ConsoleLogEntry struct {
	Level     string `json:"level"` // "log", "info", "warn", "error"
	Message   string `json:"message"`
	Timestamp int64  `json:"timestamp"`
}

// RequestLogEvent represents a live streaming event from a worker isolate request.
type RequestLogEvent struct {
	ID              string                 `json:"id"`
	Timestamp       time.Time              `json:"timestamp"`
	Method          string                 `json:"method"`
	Path            string                 `json:"path"`
	URL             string                 `json:"url"`
	StatusCode      int                    `json:"statusCode"`
	DurationMs      float64                `json:"durationMs"`
	ClientIP        string                 `json:"clientIp"`
	Message         string                 `json:"message"`
	Outcome         string                 `json:"outcome"`
	RequestHeaders  map[string]string      `json:"requestHeaders"`
	RequestBody     string                 `json:"requestBody,omitempty"`
	ResponseHeaders map[string]string      `json:"responseHeaders"`
	ResponseBody    string                 `json:"responseBody,omitempty"`
	Logs            []ConsoleLogEntry      `json:"logs,omitempty"`
	Exceptions      []string               `json:"exceptions,omitempty"`
	CF              map[string]interface{} `json:"cf,omitempty"`
}

// Application represents a Cloudflare Worker application running on the celld fleet.
type Application struct {
	ID                 string
	Name               string
	SourceType         SourceType
	Subdomain          string
	GitRepo            string
	Branch             string
	RootDir            string
	InlineCode         string
	AutoDeploy         bool
	Status             ApplicationStatus
	EnvVars            []EnvironmentVariable
	Bindings           []ResourceBinding
	Migrations         []MigrationStep
	ActiveDeploymentID string
	CompatibilityDate  string
	CompatibilityFlags []string
	MemoryLimitMB      int
	MaxDurationMs      int
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// CleanRootDir normalizes a relative directory path.
func CleanRootDir(dir string) string {
	d := strings.TrimSpace(dir)
	d = strings.TrimPrefix(d, "/")
	d = strings.TrimPrefix(d, "./")
	d = strings.TrimSuffix(d, "/")
	if d == "." {
		return ""
	}
	return d
}

// SanitizeSubdomain converts an application name into a valid RFC-1123 subdomain label.
func SanitizeSubdomain(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	s = strings.ReplaceAll(s, "_", "-")
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, ".", "-")
	s = strings.Trim(s, "-")
	if s == "" {
		s = "worker"
	}
	return s
}

// DefaultTestURL generates the standard local test URL for this application.
func (a *Application) DefaultTestURL(basePort int) string {
	if basePort <= 0 {
		basePort = 8000
	}
	return fmt.Sprintf("http://%s.localhost:%d", a.Subdomain, basePort)
}

// NewApplication constructs and validates a new Git-backed Application entity (backwards-compatible).
func NewApplication(id, name, gitRepo, branch string, envVars []EnvironmentVariable, bindings []ResourceBinding) (*Application, error) {
	return NewApplicationWithSource(id, name, SourceTypeGit, gitRepo, branch, "", envVars, bindings)
}

// NewApplicationWithSource constructs and validates an Application with explicit source type.
func NewApplicationWithSource(
	id, name string,
	sourceType SourceType,
	gitRepo, branch, inlineCode string,
	envVars []EnvironmentVariable,
	bindings []ResourceBinding,
) (*Application, error) {
	return NewApplicationWithSourceAndDir(id, name, sourceType, gitRepo, branch, "", inlineCode, envVars, bindings)
}

// NewApplicationWithSourceAndDir constructs and validates an Application with explicit source type and root directory.
func NewApplicationWithSourceAndDir(
	id, name string,
	sourceType SourceType,
	gitRepo, branch, rootDir, inlineCode string,
	envVars []EnvironmentVariable,
	bindings []ResourceBinding,
) (*Application, error) {
	name = strings.TrimSpace(name)
	if !appNameRegex.MatchString(name) {
		return nil, NewValidationError("application name must be lowercase alphanumeric and may contain dashes, e.g. 'my-app'")
	}

	if sourceType == "" {
		sourceType = SourceTypeGit
	}

	switch sourceType {
	case SourceTypeGit:
		gitRepo = strings.TrimSpace(gitRepo)
		if !gitRepoURLRegex.MatchString(gitRepo) {
			return nil, NewValidationError("invalid git repository URL format")
		}
		branch = strings.TrimSpace(branch)
		if branch == "" {
			branch = "main"
		}
	case SourceTypeInline:
		inlineCode = strings.TrimSpace(inlineCode)
		if inlineCode == "" {
			inlineCode = DefaultHelloWorldWorker
		}
	default:
		return nil, NewValidationError("unsupported source type: " + string(sourceType))
	}

	for _, env := range envVars {
		if !envKeyRegex.MatchString(env.Key) {
			return nil, NewValidationError("invalid environment variable key: " + env.Key)
		}
	}

	now := time.Now().UTC()
	return &Application{
		ID:                 id,
		Name:               name,
		SourceType:         sourceType,
		Subdomain:          SanitizeSubdomain(name),
		GitRepo:            gitRepo,
		Branch:             branch,
		RootDir:            CleanRootDir(rootDir),
		InlineCode:         inlineCode,
		AutoDeploy:         true,
		Status:             AppStatusCreated,
		EnvVars:            envVars,
		Bindings:           bindings,
		Migrations:         []MigrationStep{},
		CompatibilityDate:  "2024-09-23",
		CompatibilityFlags: []string{},
		MemoryLimitMB:      128,
		MaxDurationMs:      50,
		CreatedAt:          now,
		UpdatedAt:          now,
	}, nil
}

// SetRootDir updates the application root directory.
func (a *Application) SetRootDir(dir string) {
	a.RootDir = CleanRootDir(dir)
	a.UpdatedAt = time.Now().UTC()
}

// RecordMigration idempotently appends or updates a migration step by tag.
func (a *Application) RecordMigration(step MigrationStep) {
	for i, existing := range a.Migrations {
		if existing.Tag == step.Tag {
			a.Migrations[i] = step
			a.UpdatedAt = time.Now().UTC()
			return
		}
	}
	a.Migrations = append(a.Migrations, step)
	a.UpdatedAt = time.Now().UTC()
}

// SetActiveDeployment updates the running deployment for the application.
func (a *Application) SetActiveDeployment(deploymentID string) {
	a.ActiveDeploymentID = deploymentID
	a.Status = AppStatusRunning
	a.UpdatedAt = time.Now().UTC()
}

// SetStatus updates application lifecycle status.
func (a *Application) SetStatus(status ApplicationStatus) {
	a.Status = status
	a.UpdatedAt = time.Now().UTC()
}

// UpdateInlineCode updates the script content for an inline application.
func (a *Application) UpdateInlineCode(code string) error {
	code = strings.TrimSpace(code)
	if code == "" {
		return NewValidationError("inline code cannot be empty")
	}
	a.InlineCode = code
	a.UpdatedAt = time.Now().UTC()
	return nil
}

// UpdateConfig updates branch, inline code, environment variables, and resource bindings.
func (a *Application) UpdateConfig(branch, inlineCode string, envVars []EnvironmentVariable, bindings []ResourceBinding) error {
	branch = strings.TrimSpace(branch)
	if branch != "" {
		a.Branch = branch
	}

	inlineCode = strings.TrimSpace(inlineCode)
	if inlineCode != "" {
		a.InlineCode = inlineCode
	}

	for _, env := range envVars {
		if !envKeyRegex.MatchString(env.Key) {
			return NewValidationError("invalid environment variable key: " + env.Key)
		}
	}

	if envVars != nil {
		a.EnvVars = envVars
	}
	if bindings != nil {
		a.Bindings = bindings
	}
	a.UpdatedAt = time.Now().UTC()
	return nil
}

// UpdateRuntimeConfig updates compatibility date, compatibility flags, memory limit, and execution timeout.
func (a *Application) UpdateRuntimeConfig(compatDate string, flags []string, memoryLimitMB, maxDurationMs int) {
	if strings.TrimSpace(compatDate) != "" {
		a.CompatibilityDate = strings.TrimSpace(compatDate)
	}
	if flags != nil {
		a.CompatibilityFlags = flags
	}
	if memoryLimitMB > 0 {
		a.MemoryLimitMB = memoryLimitMB
	}
	if maxDurationMs > 0 {
		a.MaxDurationMs = maxDurationMs
	}
	a.UpdatedAt = time.Now().UTC()
}
