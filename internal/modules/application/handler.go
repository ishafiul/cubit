package application

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ishaf/cubit/internal/domain"
)

// Handler handles HTTP requests for applications using Gin.
type Handler struct {
	service Service
}

// NewHandler creates a new Application Handler.
func NewHandler(s Service) *Handler {
	return &Handler{service: s}
}

type envVarReq struct {
	Key      string `json:"key"`
	Value    string `json:"value"`
	IsSecret *bool  `json:"isSecret,omitempty"`
}

type bindingReq struct {
	Type       string  `json:"type"`
	Name       string  `json:"name"`
	ResourceId *string `json:"resourceId,omitempty"`
}

type createApplicationRequest struct {
	Name       string        `json:"name" binding:"required"`
	SourceType *string       `json:"sourceType,omitempty"`
	GitRepo    *string       `json:"gitRepo,omitempty"`
	Branch     *string       `json:"branch,omitempty"`
	InlineCode *string       `json:"inlineCode,omitempty"`
	AutoDeploy *bool         `json:"autoDeploy,omitempty"`
	EnvVars    *[]envVarReq  `json:"envVars,omitempty"`
	Bindings   *[]bindingReq `json:"bindings,omitempty"`
}

type updateApplicationRequest struct {
	Branch     *string       `json:"branch,omitempty"`
	InlineCode *string       `json:"inlineCode,omitempty"`
	AutoDeploy *bool         `json:"autoDeploy,omitempty"`
	EnvVars    *[]envVarReq  `json:"envVars,omitempty"`
	Bindings   *[]bindingReq `json:"bindings,omitempty"`
}

type testApplicationRequest struct {
	Method  *string            `json:"method,omitempty"`
	Path    *string            `json:"path,omitempty"`
	Headers *map[string]string `json:"headers,omitempty"`
	Body    *string            `json:"body,omitempty"`
}

type applicationResponse struct {
	ID                 string                       `json:"id"`
	Name               string                       `json:"name"`
	SourceType         string                       `json:"sourceType"`
	Subdomain          string                       `json:"subdomain"`
	TestURL            string                       `json:"testUrl"`
	GitRepo            *string                      `json:"gitRepo,omitempty"`
	Branch             string                       `json:"branch"`
	InlineCode         *string                      `json:"inlineCode,omitempty"`
	AutoDeploy         bool                         `json:"autoDeploy"`
	Status             string                       `json:"status"`
	EnvVars            []domain.EnvironmentVariable `json:"envVars,omitempty"`
	Bindings           []domain.ResourceBinding     `json:"bindings,omitempty"`
	ActiveDeploymentID *string                      `json:"activeDeploymentId,omitempty"`
	CreatedAt          string                       `json:"createdAt"`
	UpdatedAt          string                       `json:"updatedAt"`
}

func toResponse(app *domain.Application) applicationResponse {
	var gitRepo *string
	if app.GitRepo != "" {
		gitRepo = &app.GitRepo
	}
	var inlineCode *string
	if app.InlineCode != "" {
		inlineCode = &app.InlineCode
	}
	var activeDep *string
	if app.ActiveDeploymentID != "" {
		activeDep = &app.ActiveDeploymentID
	}

	subdomain := app.Subdomain
	if subdomain == "" {
		subdomain = domain.SanitizeSubdomain(app.Name)
	}

	return applicationResponse{
		ID:                 app.ID,
		Name:               app.Name,
		SourceType:         string(app.SourceType),
		Subdomain:          subdomain,
		TestURL:            app.DefaultTestURL(8000),
		GitRepo:            gitRepo,
		Branch:             app.Branch,
		InlineCode:         inlineCode,
		AutoDeploy:         app.AutoDeploy,
		Status:             string(app.Status),
		EnvVars:            app.EnvVars,
		Bindings:           app.Bindings,
		ActiveDeploymentID: activeDep,
		CreatedAt:          app.CreatedAt.Format("2006-01-02T15:04:05.999999999Z"),
		UpdatedAt:          app.UpdatedAt.Format("2006-01-02T15:04:05.999999999Z"),
	}
}

// List returns all registered applications.
func (h *Handler) List(c *gin.Context) {
	apps, err := h.service.List(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}

	result := make([]applicationResponse, 0, len(apps))
	for _, app := range apps {
		result = append(result, toResponse(app))
	}
	c.JSON(http.StatusOK, result)
}

// Create creates a new application.
func (h *Handler) Create(c *gin.Context) {
	var req createApplicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "malformed request body: " + err.Error()})
		return
	}

	branch := "main"
	if req.Branch != nil && *req.Branch != "" {
		branch = *req.Branch
	}
	sourceType := domain.SourceTypeGit
	if req.SourceType != nil && *req.SourceType != "" {
		sourceType = domain.SourceType(*req.SourceType)
	}
	gitRepo := ""
	if req.GitRepo != nil {
		gitRepo = *req.GitRepo
	}
	inlineCode := ""
	if req.InlineCode != nil {
		inlineCode = *req.InlineCode
	}
	autoDeploy := true
	if req.AutoDeploy != nil {
		autoDeploy = *req.AutoDeploy
	}

	var envVars []domain.EnvironmentVariable
	if req.EnvVars != nil {
		for _, e := range *req.EnvVars {
			isSecret := false
			if e.IsSecret != nil {
				isSecret = *e.IsSecret
			}
			envVars = append(envVars, domain.EnvironmentVariable{
				Key:      e.Key,
				Value:    e.Value,
				IsSecret: isSecret,
			})
		}
	}

	var bindings []domain.ResourceBinding
	if req.Bindings != nil {
		for _, b := range *req.Bindings {
			resID := ""
			if b.ResourceId != nil {
				resID = *b.ResourceId
			}
			bindings = append(bindings, domain.ResourceBinding{
				Type:       domain.BindingType(b.Type),
				Name:       b.Name,
				ResourceID: resID,
			})
		}
	}

	app, err := h.service.Create(c.Request.Context(), req.Name, sourceType, gitRepo, branch, inlineCode, autoDeploy, envVars, bindings)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, toResponse(app))
}

// GetByID retrieves a single application.
func (h *Handler) GetByID(c *gin.Context) {
	id := c.Param("id")
	app, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, toResponse(app))
}

// Update updates an application configuration.
func (h *Handler) Update(c *gin.Context) {
	id := c.Param("id")
	var req updateApplicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "malformed request body: " + err.Error()})
		return
	}

	branch := ""
	if req.Branch != nil {
		branch = *req.Branch
	}
	inlineCode := ""
	if req.InlineCode != nil {
		inlineCode = *req.InlineCode
	}

	var envVars []domain.EnvironmentVariable
	if req.EnvVars != nil {
		for _, e := range *req.EnvVars {
			isSecret := false
			if e.IsSecret != nil {
				isSecret = *e.IsSecret
			}
			envVars = append(envVars, domain.EnvironmentVariable{
				Key:      e.Key,
				Value:    e.Value,
				IsSecret: isSecret,
			})
		}
	}

	var bindings []domain.ResourceBinding
	if req.Bindings != nil {
		for _, b := range *req.Bindings {
			resID := ""
			if b.ResourceId != nil {
				resID = *b.ResourceId
			}
			bindings = append(bindings, domain.ResourceBinding{
				Type:       domain.BindingType(b.Type),
				Name:       b.Name,
				ResourceID: resID,
			})
		}
	}

	app, err := h.service.Update(c.Request.Context(), id, branch, inlineCode, req.AutoDeploy, envVars, bindings)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, toResponse(app))
}

// Delete removes an application.
func (h *Handler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// Test invokes the active isolate for an application.
func (h *Handler) Test(c *gin.Context) {
	id := c.Param("id")
	var req testApplicationRequest
	_ = c.ShouldBindJSON(&req)

	method := "GET"
	if req.Method != nil && *req.Method != "" {
		method = *req.Method
	}
	path := "/"
	if req.Path != nil && *req.Path != "" {
		path = *req.Path
	}
	headers := make(map[string]string)
	if req.Headers != nil {
		headers = *req.Headers
	}
	var body []byte
	if req.Body != nil {
		body = []byte(*req.Body)
	}

	status, respHeaders, respBody, err := h.service.Invoke(c.Request.Context(), id, method, path, headers, body)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  status,
		"headers": respHeaders,
		"body":    string(respBody),
	})
}

func respondError(c *gin.Context, err error) {
	var domErr *domain.DomainError
	if errors.As(err, &domErr) {
		switch domErr.Code {
		case "VALIDATION_ERROR":
			c.JSON(http.StatusBadRequest, gin.H{"error": domErr.Error()})
			return
		case "NOT_FOUND":
			c.JSON(http.StatusNotFound, gin.H{"error": domErr.Error()})
			return
		case "CONFLICT":
			c.JSON(http.StatusConflict, gin.H{"error": domErr.Error()})
			return
		}
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
}
