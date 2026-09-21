package domain

import (
	"regexp"
	"strings"
	"time"
)

var (
	appNameRegex   = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)
	envKeyRegex    = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	gitRepoURLRegex = regexp.MustCompile(`^(https?://|git@).+`)
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

// SourceType defines the origin of the application worker code.
type SourceType string

const (
	SourceTypeGit    SourceType = "git"
	SourceTypeInline SourceType = "inline"
)

// DefaultHelloWorldWorker is the standard ES module template for inline worker applications.
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
	BindingTypeKV       BindingType = "kv_namespace"
	BindingTypeD1       BindingType = "d1_database"
	BindingTypeR2       BindingType = "r2_bucket"
	BindingTypeQueue    BindingType = "queue"
	BindingTypeWorkflow BindingType = "workflow"
)

// EnvironmentVariable represents an injected application runtime variable.
type EnvironmentVariable struct {
	Key      string
	Value    string
	IsSecret bool
}

// ResourceBinding represents a binding to a stateful cell or storage.
type ResourceBinding struct {
	Type       BindingType
	Name       string
	ResourceID string
}

// Application represents a Cloudflare Worker application running on the celld fleet.
type Application struct {
	ID                 string
	Name               string
	SourceType         SourceType
	GitRepo            string
	Branch             string
	InlineCode         string
	Status             ApplicationStatus
	EnvVars            []EnvironmentVariable
	Bindings           []ResourceBinding
	ActiveDeploymentID string
	CreatedAt          time.Time
	UpdatedAt          time.Time
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
		ID:         id,
		Name:       name,
		SourceType: sourceType,
		GitRepo:    gitRepo,
		Branch:     branch,
		InlineCode: inlineCode,
		Status:     AppStatusCreated,
		EnvVars:    envVars,
		Bindings:   bindings,
		CreatedAt:  now,
		UpdatedAt:  now,
	}, nil
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
