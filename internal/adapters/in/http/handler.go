package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/ishaf/cubit/internal/domain"
	"github.com/ishaf/cubit/internal/usecase"
)

var _ ServerInterface = (*APIHandler)(nil)

// APIHandler implements the generated ServerInterface.
type APIHandler struct {
	nodeUsecase    *usecase.NodeUsecase
	appUsecase     *usecase.AppUsecase
	domainUsecase  *usecase.DomainUsecase
	runtimeUsecase *usecase.RuntimeUsecase
	depRepo        usecase.DeploymentRepository
}

// NewAPIHandler creates a new APIHandler.
func NewAPIHandler(
	nodeUsecase *usecase.NodeUsecase,
	appUsecase *usecase.AppUsecase,
	domainUsecase *usecase.DomainUsecase,
	runtimeUsecase *usecase.RuntimeUsecase,
	depRepo usecase.DeploymentRepository,
) *APIHandler {
	return &APIHandler{
		nodeUsecase:    nodeUsecase,
		appUsecase:     appUsecase,
		domainUsecase:  domainUsecase,
		runtimeUsecase: runtimeUsecase,
		depRepo:        depRepo,
	}
}

// ListApplications GET /applications
func (h *APIHandler) ListApplications(w http.ResponseWriter, r *http.Request) {
	apps, err := h.appUsecase.ListApplications(r.Context())
	if err != nil {
		h.respondError(w, err)
		return
	}

	var result []Application
	for _, a := range apps {
		result = append(result, toAPIApplication(a))
	}
	h.respondJSON(w, http.StatusOK, result)
}

// CreateApplication POST /applications
func (h *APIHandler) CreateApplication(w http.ResponseWriter, r *http.Request) {
	var req CreateApplicationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, domain.NewValidationError("malformed request body: "+err.Error()))
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

	app, err := h.appUsecase.CreateApplicationWithSource(r.Context(), req.Name, sourceType, gitRepo, branch, inlineCode, envVars, bindings)
	if err != nil {
		h.respondError(w, err)
		return
	}

	h.respondJSON(w, http.StatusCreated, toAPIApplication(app))
}

// GetApplication GET /applications/{id}
func (h *APIHandler) GetApplication(w http.ResponseWriter, r *http.Request, id openapi_types.UUID) {
	app, err := h.appUsecase.GetApplication(r.Context(), id.String())
	if err != nil {
		h.respondError(w, err)
		return
	}
	h.respondJSON(w, http.StatusOK, toAPIApplication(app))
}

// UpdateApplication PUT /applications/{id}
func (h *APIHandler) UpdateApplication(w http.ResponseWriter, r *http.Request, id openapi_types.UUID) {
	var req UpdateApplicationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, domain.NewValidationError("malformed request body"))
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
			isSec := false
			if e.IsSecret != nil {
				isSec = *e.IsSecret
			}
			envVars = append(envVars, domain.EnvironmentVariable{
				Key:      e.Key,
				Value:    e.Value,
				IsSecret: isSec,
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

	app, err := h.appUsecase.UpdateApplication(r.Context(), id.String(), branch, inlineCode, envVars, bindings)
	if err != nil {
		h.respondError(w, err)
		return
	}

	h.respondJSON(w, http.StatusOK, toAPIApplication(app))
}

// DeleteApplication DELETE /applications/{id}
func (h *APIHandler) DeleteApplication(w http.ResponseWriter, r *http.Request, id openapi_types.UUID) {
	if err := h.appUsecase.DeleteApplication(r.Context(), id.String()); err != nil {
		h.respondError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// DeployApplication POST /applications/{id}/deploy
func (h *APIHandler) DeployApplication(w http.ResponseWriter, r *http.Request, id openapi_types.UUID) {
	var req CreateDeploymentRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	commitHash := "HEAD"
	if req.CommitHash != nil && *req.CommitHash != "" {
		commitHash = *req.CommitHash
	}

	dep, err := h.appUsecase.DeployApplication(r.Context(), id.String(), commitHash)
	if err != nil {
		h.respondError(w, err)
		return
	}

	h.respondJSON(w, http.StatusAccepted, toAPIDeployment(dep))
}

// ListDeployments GET /applications/{id}/deployments
func (h *APIHandler) ListDeployments(w http.ResponseWriter, r *http.Request, id openapi_types.UUID) {
	deps, err := h.depRepo.ListByAppID(r.Context(), id.String())
	if err != nil {
		h.respondError(w, err)
		return
	}

	var result []Deployment
	for _, d := range deps {
		result = append(result, toAPIDeployment(d))
	}
	h.respondJSON(w, http.StatusOK, result)
}

// TestApplication POST /applications/{id}/test
func (h *APIHandler) TestApplication(w http.ResponseWriter, r *http.Request, id openapi_types.UUID) {
	var req TestApplicationRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	method := "GET"
	if req.Method != nil && *req.Method != "" {
		method = *req.Method
	}

	path := "/"
	if req.Path != nil && *req.Path != "" {
		path = *req.Path
	}

	headers := make(map[string]string)
	if req.Headers != nil && *req.Headers != nil {
		headers = *req.Headers
	}

	var reqBody []byte
	if req.Body != nil {
		reqBody = []byte(*req.Body)
	}

	statusCode, respHeaders, respBody, err := h.appUsecase.InvokeApplication(r.Context(), id.String(), method, path, headers, reqBody)
	if err != nil {
		h.respondError(w, err)
		return
	}

	res := TestApplicationResponse{
		StatusCode: statusCode,
		Body:       string(respBody),
	}
	if len(respHeaders) > 0 {
		res.Headers = &respHeaders
	}

	h.respondJSON(w, http.StatusOK, res)
}

// GetDeployment GET /deployments/{id}
func (h *APIHandler) GetDeployment(w http.ResponseWriter, r *http.Request, id openapi_types.UUID) {
	dep, err := h.depRepo.GetByID(r.Context(), id.String())
	if err != nil {
		h.respondError(w, err)
		return
	}
	h.respondJSON(w, http.StatusOK, toAPIDeployment(dep))
}

// GetDeploymentLogs GET /deployments/{id}/logs
func (h *APIHandler) GetDeploymentLogs(w http.ResponseWriter, r *http.Request, id openapi_types.UUID) {
	logs, err := h.depRepo.GetLogs(r.Context(), id.String())
	if err != nil {
		h.respondError(w, err)
		return
	}

	var result []DeploymentLogEntry
	for _, l := range logs {
		result = append(result, DeploymentLogEntry{
			Timestamp: l.Timestamp,
			Step:      DeploymentLogEntryStep(l.Step),
			Message:   l.Message,
			Level:     DeploymentLogEntryLevel(l.Level),
		})
	}
	h.respondJSON(w, http.StatusOK, result)
}

// ListNodes GET /nodes
func (h *APIHandler) ListNodes(w http.ResponseWriter, r *http.Request) {
	nodes, err := h.nodeUsecase.ListNodes(r.Context())
	if err != nil {
		h.respondError(w, err)
		return
	}

	var result []Node
	for _, n := range nodes {
		result = append(result, toAPINode(n))
	}
	h.respondJSON(w, http.StatusOK, result)
}

// CreateNode POST /nodes
func (h *APIHandler) CreateNode(w http.ResponseWriter, r *http.Request) {
	var req CreateNodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, domain.NewValidationError("malformed request body"))
		return
	}

	intPort := 8081
	if req.InternalPort != nil {
		intPort = *req.InternalPort
	}
	workerPort := 8080
	if req.WorkerPort != nil {
		workerPort = *req.WorkerPort
	}

	node, err := h.nodeUsecase.RegisterNode(r.Context(), req.Name, req.IpAddress, intPort, workerPort, "0.2.0")
	if err != nil {
		h.respondError(w, err)
		return
	}

	h.respondJSON(w, http.StatusCreated, toAPINode(node))
}

// GetNode GET /nodes/{id}
func (h *APIHandler) GetNode(w http.ResponseWriter, r *http.Request, id openapi_types.UUID) {
	node, err := h.nodeUsecase.GetNode(r.Context(), id.String())
	if err != nil {
		h.respondError(w, err)
		return
	}
	h.respondJSON(w, http.StatusOK, toAPINode(node))
}

// DeleteNode DELETE /nodes/{id}
func (h *APIHandler) DeleteNode(w http.ResponseWriter, r *http.Request, id openapi_types.UUID) {
	if err := h.nodeUsecase.DeleteNode(r.Context(), id.String()); err != nil {
		h.respondError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// DrainNode POST /nodes/{id}/drain
func (h *APIHandler) DrainNode(w http.ResponseWriter, r *http.Request, id openapi_types.UUID) {
	node, err := h.nodeUsecase.DrainNode(r.Context(), id.String())
	if err != nil {
		h.respondError(w, err)
		return
	}
	h.respondJSON(w, http.StatusOK, toAPINode(node))
}

// ListDomains GET /domains
func (h *APIHandler) ListDomains(w http.ResponseWriter, r *http.Request) {
	domains, err := h.domainUsecase.ListDomains(r.Context())
	if err != nil {
		h.respondError(w, err)
		return
	}

	var result []Domain
	for _, d := range domains {
		result = append(result, toAPIDomain(d))
	}
	h.respondJSON(w, http.StatusOK, result)
}

// CreateDomain POST /domains
func (h *APIHandler) CreateDomain(w http.ResponseWriter, r *http.Request) {
	var req CreateDomainRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, domain.NewValidationError("malformed request body"))
		return
	}

	path := "/"
	if req.PathPrefix != nil && *req.PathPrefix != "" {
		path = *req.PathPrefix
	}

	dom, err := h.domainUsecase.AddDomain(r.Context(), req.ApplicationId.String(), req.Hostname, path)
	if err != nil {
		h.respondError(w, err)
		return
	}

	h.respondJSON(w, http.StatusCreated, toAPIDomain(dom))
}

// DeleteDomain DELETE /domains/{id}
func (h *APIHandler) DeleteDomain(w http.ResponseWriter, r *http.Request, id openapi_types.UUID) {
	if err := h.domainUsecase.DeleteDomain(r.Context(), id.String()); err != nil {
		h.respondError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GetRuntimeStatus GET /runtime/status
func (h *APIHandler) GetRuntimeStatus(w http.ResponseWriter, r *http.Request) {
	status, err := h.runtimeUsecase.GetStatus(r.Context())
	if err != nil {
		h.respondError(w, err)
		return
	}

	resp := RuntimeStatus{
		ActiveNodesCount:    status.ActiveNodesCount,
		CurrentCelldVersion: status.CurrentCelldVersion,
		IsUpgrading:         status.IsUpgrading,
	}
	if status.TargetCelldVersion != "" {
		resp.TargetCelldVersion = &status.TargetCelldVersion
	}
	backend := RuntimeStatusStorageBackend(status.StorageBackend)
	resp.StorageBackend = &backend

	h.respondJSON(w, http.StatusOK, resp)
}

// UpgradeCelldDaemon POST /runtime/upgrade
func (h *APIHandler) UpgradeCelldDaemon(w http.ResponseWriter, r *http.Request) {
	var req UpgradeDaemonRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, domain.NewValidationError("malformed request body"))
		return
	}

	go func() {
		_ = h.runtimeUsecase.UpgradeCelldDaemon(r.Context(), req.TargetVersion)
	}()

	status, _ := h.runtimeUsecase.GetStatus(r.Context())
	resp := RuntimeStatus{
		ActiveNodesCount:    status.ActiveNodesCount,
		CurrentCelldVersion: status.CurrentCelldVersion,
		IsUpgrading:         true,
		TargetCelldVersion:  &req.TargetVersion,
	}
	h.respondJSON(w, http.StatusAccepted, resp)
}

func (h *APIHandler) respondJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(payload)
}

func (h *APIHandler) respondError(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError
	errCode := "INTERNAL_ERROR"

	if errors.Is(err, domain.ErrNotFound) {
		code = http.StatusNotFound
		errCode = "NOT_FOUND"
	} else if errors.Is(err, domain.ErrValidation) {
		code = http.StatusBadRequest
		errCode = "VALIDATION_ERROR"
	} else if errors.Is(err, domain.ErrConflict) {
		code = http.StatusConflict
		errCode = "CONFLICT"
	} else if errors.Is(err, domain.ErrInvalidState) {
		code = http.StatusBadRequest
		errCode = "INVALID_STATE"
	}

	h.respondJSON(w, code, ErrorResponse{
		Code:    errCode,
		Message: err.Error(),
	})
}

// parseUUID parses string into openapi_types.UUID
func parseUUID(s string) openapi_types.UUID {
	u, err := uuid.Parse(s)
	if err != nil {
		return openapi_types.UUID(uuid.New())
	}
	return openapi_types.UUID(u)
}

// Mappers to API DTOs
func toAPINode(n *domain.Node) Node {
	uid := parseUUID(n.ID)
	cpu := n.Specs.CPUCores
	mem := n.Specs.MemoryBytes
	disk := n.Specs.DiskFreeBytes

	return Node{
		Id:           uid,
		Name:         n.Name,
		IpAddress:    n.IPAddress,
		InternalPort: n.InternalPort,
		WorkerPort:   n.WorkerPort,
		Status:       NodeStatus(n.Status),
		CelldVersion: n.CelldVersion,
		Specs: &NodeSpecs{
			CpuCores:      &cpu,
			MemoryBytes:   &mem,
			DiskFreeBytes: &disk,
		},
		CreatedAt: n.CreatedAt,
	}
}

func toAPIApplication(a *domain.Application) Application {
	uid := parseUUID(a.ID)
	var envs []EnvironmentVariable
	for _, e := range a.EnvVars {
		isSec := e.IsSecret
		envs = append(envs, EnvironmentVariable{
			Key:      e.Key,
			Value:    e.Value,
			IsSecret: &isSec,
		})
	}
	var bindings []ResourceBinding
	for _, b := range a.Bindings {
		resID := b.ResourceID
		bindings = append(bindings, ResourceBinding{
			Type:       ResourceBindingType(b.Type),
			Name:       b.Name,
			ResourceId: &resID,
		})
	}
	var activeDepUUID *openapi_types.UUID
	if a.ActiveDeploymentID != "" {
		p := parseUUID(a.ActiveDeploymentID)
		activeDepUUID = &p
	}

	sourceType := SourceType(a.SourceType)
	if sourceType == "" {
		sourceType = Git
	}

	var gitRepo *string
	if a.GitRepo != "" {
		gitRepo = &a.GitRepo
	}

	branch := a.Branch

	var inlineCode *string
	if a.InlineCode != "" {
		inlineCode = &a.InlineCode
	}

	subdomain := a.Subdomain
	if subdomain == "" {
		subdomain = domain.SanitizeSubdomain(a.Name)
	}
	testURL := a.DefaultTestURL(8000)

	return Application{
		Id:                 uid,
		Name:               a.Name,
		SourceType:         sourceType,
		Subdomain:          &subdomain,
		TestUrl:            &testURL,
		GitRepo:            gitRepo,
		Branch:             &branch,
		InlineCode:         inlineCode,
		Status:             ApplicationStatus(a.Status),
		EnvVars:            &envs,
		Bindings:           &bindings,
		ActiveDeploymentId: activeDepUUID,
		CreatedAt:          a.CreatedAt,
		UpdatedAt:          a.UpdatedAt,
	}
}

func toAPIDeployment(d *domain.Deployment) Deployment {
	uid := parseUUID(d.ID)
	appUUID := parseUUID(d.ApplicationID)
	commitMsg := d.CommitMessage
	bundleSize := int(d.BundleSize)
	errMsg := d.ErrorMessage

	var buildVer *int
	if d.BuildVersion > 0 {
		bv := d.BuildVersion
		buildVer = &bv
	}

	dep := Deployment{
		Id:            uid,
		ApplicationId: appUUID,
		BuildVersion:  buildVer,
		CommitHash:    d.CommitHash,
		CommitMessage: &commitMsg,
		Status:        DeploymentStatus(d.Status),
		BundleSize:    &bundleSize,
		CreatedAt:     d.CreatedAt,
		FinishedAt:    d.FinishedAt,
	}
	if errMsg != "" {
		dep.ErrorMessage = &errMsg
	}
	return dep
}

func toAPIDomain(d *domain.Domain) Domain {
	uid := parseUUID(d.ID)
	appUUID := parseUUID(d.ApplicationID)

	return Domain{
		Id:            uid,
		ApplicationId: appUUID,
		Hostname:      d.Hostname,
		PathPrefix:    d.PathPrefix,
		SslActive:     d.SSLActive,
		CreatedAt:     d.CreatedAt,
	}
}
