package http

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/ishaf/cubit/internal/domain"
	"github.com/ishaf/cubit/internal/usecase"
)

// ServicesHandler handles HTTP requests for all 12 celld services.
type ServicesHandler struct {
	usecase *usecase.ServicesUsecase
}

// NewServicesHandler creates a new ServicesHandler.
func NewServicesHandler(u *usecase.ServicesUsecase) *ServicesHandler {
	return &ServicesHandler{usecase: u}
}

// RegisterRoutes mounts all celld service routes under the provided router.
func (h *ServicesHandler) RegisterRoutes(r chi.Router) {
	// 1. KV Namespaces & Values
	r.Route("/kv", func(sub chi.Router) {
		sub.Get("/namespaces", h.ListKVNamespaces)
		sub.Post("/namespaces", h.CreateKVNamespace)
		sub.Delete("/namespaces/{id}", h.DeleteKVNamespace)
		sub.Get("/namespaces/{id}/keys", h.ListKVPairs)
		sub.Get("/namespaces/{id}/values/{key}", h.GetKVPair)
		sub.Put("/namespaces/{id}/values/{key}", h.PutKVPair)
		sub.Delete("/namespaces/{id}/values/{key}", h.DeleteKVPair)
	})

	// 2. D1 SQL Databases
	r.Route("/d1", func(sub chi.Router) {
		sub.Get("/databases", h.ListD1Databases)
		sub.Post("/databases", h.CreateD1Database)
		sub.Delete("/databases/{id}", h.DeleteD1Database)
		sub.Post("/databases/{id}/query", h.ExecuteD1Query)
	})

	// 3. R2 Object Storage
	r.Route("/r2", func(sub chi.Router) {
		sub.Get("/buckets", h.ListR2Buckets)
		sub.Get("/buckets/{name}/objects", h.ListR2Objects)
		sub.Post("/buckets/{name}/upload", h.UploadR2Object)
	})

	// 4. Queues
	r.Route("/queues", func(sub chi.Router) {
		sub.Get("/", h.ListQueues)
		sub.Post("/", h.CreateQueue)
		sub.Delete("/{id}", h.DeleteQueue)
		sub.Post("/{id}/messages", h.SendQueueMessage)
		sub.Get("/{id}/messages", h.ListQueueMessages)
	})

	// 5. Cron Triggers
	r.Route("/cron", func(sub chi.Router) {
		sub.Get("/", h.ListCronTriggers)
		sub.Post("/", h.CreateCronTrigger)
		sub.Delete("/{id}", h.DeleteCronTrigger)
		sub.Post("/{id}/run", h.RunTriggerNow)
		sub.Get("/{id}/runs", h.ListCronRuns)
	})

	// 6. Workflows
	r.Route("/workflows", func(sub chi.Router) {
		sub.Get("/", h.ListWorkflows)
		sub.Post("/", h.CreateWorkflow)
		sub.Delete("/{id}", h.DeleteWorkflow)
		sub.Post("/{id}/trigger", h.TriggerWorkflowRun)
		sub.Get("/{id}/runs", h.ListWorkflowRuns)
	})

	// 7. Durable Objects
	r.Route("/durable-objects", func(sub chi.Router) {
		sub.Get("/", h.ListDurableObjectClasses)
		sub.Post("/", h.CreateDurableObjectClass)
		sub.Delete("/{id}", h.DeleteDurableObjectClass)
		sub.Get("/{id}/instances", h.ListDOInstances)
		sub.Post("/{id}/instances", h.CreateDOInstance)
	})

	// 8. Containers
	r.Route("/containers", func(sub chi.Router) {
		sub.Get("/", h.ListContainers)
		sub.Post("/", h.CreateContainer)
		sub.Delete("/{id}", h.DeleteContainer)
	})

	// 9. Static Assets
	r.Route("/static-assets", func(sub chi.Router) {
		sub.Get("/", h.ListStaticSites)
		sub.Post("/", h.CreateStaticSite)
		sub.Delete("/{id}", h.DeleteStaticSite)
	})

	// 10. Dynamic Workers
	r.Route("/dynamic-workers", func(sub chi.Router) {
		sub.Post("/eval", h.EvalDynamicWorker)
	})
}

// --- Response Helpers ---

func (h *ServicesHandler) respondJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(payload)
}

func (h *ServicesHandler) respondError(w http.ResponseWriter, err error) {
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

	h.respondJSON(w, code, map[string]string{
		"code":    errCode,
		"message": err.Error(),
	})
}

// --- KV Endpoints ---

type CreateKVNamespaceReq struct {
	Title string `json:"title"`
	Name  string `json:"name"`
}

func (h *ServicesHandler) ListKVNamespaces(w http.ResponseWriter, r *http.Request) {
	namespaces, err := h.usecase.ListKVNamespaces(r.Context())
	if err != nil {
		h.respondError(w, err)
		return
	}
	h.respondJSON(w, http.StatusOK, namespaces)
}

func (h *ServicesHandler) CreateKVNamespace(w http.ResponseWriter, r *http.Request) {
	var req CreateKVNamespaceReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, domain.NewValidationError("invalid request body"))
		return
	}
	name := req.Title
	if name == "" {
		name = req.Name
	}
	ns, err := h.usecase.CreateKVNamespace(r.Context(), name)
	if err != nil {
		h.respondError(w, err)
		return
	}
	h.respondJSON(w, http.StatusCreated, ns)
}

func (h *ServicesHandler) DeleteKVNamespace(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.usecase.DeleteKVNamespace(r.Context(), id); err != nil {
		h.respondError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ServicesHandler) ListKVPairs(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	pairs, err := h.usecase.ListKVPairs(r.Context(), id)
	if err != nil {
		h.respondError(w, err)
		return
	}
	h.respondJSON(w, http.StatusOK, pairs)
}

func (h *ServicesHandler) GetKVPair(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	key := chi.URLParam(r, "key")
	pair, err := h.usecase.GetKVPair(r.Context(), id, key)
	if err != nil {
		h.respondError(w, err)
		return
	}
	h.respondJSON(w, http.StatusOK, pair)
}

type PutKVPairReq struct {
	Value         string `json:"value"`
	ExpirationTTL int    `json:"expiration_ttl"`
	Metadata      string `json:"metadata"`
}

func (h *ServicesHandler) PutKVPair(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	key := chi.URLParam(r, "key")

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		h.respondError(w, domain.NewValidationError("failed reading body"))
		return
	}

	var req PutKVPairReq
	value := string(bodyBytes)
	ttl := 0
	metadata := ""

	if json.Unmarshal(bodyBytes, &req) == nil && req.Value != "" {
		value = req.Value
		ttl = req.ExpirationTTL
		metadata = req.Metadata
	}

	pair, err := h.usecase.PutKVPair(r.Context(), id, key, value, ttl, metadata)
	if err != nil {
		h.respondError(w, err)
		return
	}
	h.respondJSON(w, http.StatusOK, pair)
}

func (h *ServicesHandler) DeleteKVPair(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	key := chi.URLParam(r, "key")
	if err := h.usecase.DeleteKVPair(r.Context(), id, key); err != nil {
		h.respondError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- D1 Endpoints ---

type CreateD1Req struct {
	Name string `json:"name"`
}

func (h *ServicesHandler) ListD1Databases(w http.ResponseWriter, r *http.Request) {
	dbs, err := h.usecase.ListD1Databases(r.Context())
	if err != nil {
		h.respondError(w, err)
		return
	}
	h.respondJSON(w, http.StatusOK, dbs)
}

func (h *ServicesHandler) CreateD1Database(w http.ResponseWriter, r *http.Request) {
	var req CreateD1Req
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, domain.NewValidationError("invalid request body"))
		return
	}
	db, err := h.usecase.CreateD1Database(r.Context(), req.Name)
	if err != nil {
		h.respondError(w, err)
		return
	}
	h.respondJSON(w, http.StatusCreated, db)
}

func (h *ServicesHandler) DeleteD1Database(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.usecase.DeleteD1Database(r.Context(), id); err != nil {
		h.respondError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type QueryD1Req struct {
	SQL   string `json:"sql"`
	Query string `json:"query"`
}

func (h *ServicesHandler) ExecuteD1Query(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req QueryD1Req
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, domain.NewValidationError("invalid request body"))
		return
	}
	sqlQuery := req.SQL
	if sqlQuery == "" {
		sqlQuery = req.Query
	}
	result, err := h.usecase.ExecuteD1Query(r.Context(), id, sqlQuery)
	if err != nil {
		h.respondError(w, err)
		return
	}
	h.respondJSON(w, http.StatusOK, result)
}

// --- R2 Endpoints ---

func (h *ServicesHandler) ListR2Buckets(w http.ResponseWriter, r *http.Request) {
	buckets, err := h.usecase.ListR2Buckets(r.Context())
	if err != nil {
		h.respondError(w, err)
		return
	}
	h.respondJSON(w, http.StatusOK, buckets)
}

func (h *ServicesHandler) ListR2Objects(w http.ResponseWriter, r *http.Request) {
	bucketName := chi.URLParam(r, "name")
	objects, err := h.usecase.ListR2Objects(r.Context(), bucketName)
	if err != nil {
		h.respondError(w, err)
		return
	}
	h.respondJSON(w, http.StatusOK, objects)
}

type UploadR2Req struct {
	Key     string `json:"key"`
	Content string `json:"content"`
}

func (h *ServicesHandler) UploadR2Object(w http.ResponseWriter, r *http.Request) {
	bucketName := chi.URLParam(r, "name")
	var req UploadR2Req
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, domain.NewValidationError("invalid request body"))
		return
	}
	if err := h.usecase.UploadR2Object(r.Context(), bucketName, req.Key, []byte(req.Content)); err != nil {
		h.respondError(w, err)
		return
	}
	h.respondJSON(w, http.StatusOK, map[string]string{"status": "uploaded", "key": req.Key})
}

// --- Queues Endpoints ---

type CreateQueueReq struct {
	Name           string `json:"name"`
	ConsumerAppID  string `json:"consumer_app_id"`
	MaxRetries     int    `json:"max_retries"`
}

func (h *ServicesHandler) ListQueues(w http.ResponseWriter, r *http.Request) {
	queues, err := h.usecase.ListQueues(r.Context())
	if err != nil {
		h.respondError(w, err)
		return
	}
	h.respondJSON(w, http.StatusOK, queues)
}

func (h *ServicesHandler) CreateQueue(w http.ResponseWriter, r *http.Request) {
	var req CreateQueueReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, domain.NewValidationError("invalid request body"))
		return
	}
	q, err := h.usecase.CreateQueue(r.Context(), req.Name, req.ConsumerAppID, req.MaxRetries)
	if err != nil {
		h.respondError(w, err)
		return
	}
	h.respondJSON(w, http.StatusCreated, q)
}

func (h *ServicesHandler) DeleteQueue(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.usecase.DeleteQueue(r.Context(), id); err != nil {
		h.respondError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type SendQueueMsgReq struct {
	Body string `json:"body"`
}

func (h *ServicesHandler) SendQueueMessage(w http.ResponseWriter, r *http.Request) {
	queueID := chi.URLParam(r, "id")
	var req SendQueueMsgReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, domain.NewValidationError("invalid request body"))
		return
	}
	msg, err := h.usecase.SendQueueMessage(r.Context(), queueID, req.Body)
	if err != nil {
		h.respondError(w, err)
		return
	}
	h.respondJSON(w, http.StatusCreated, msg)
}

func (h *ServicesHandler) ListQueueMessages(w http.ResponseWriter, r *http.Request) {
	queueID := chi.URLParam(r, "id")
	messages, err := h.usecase.ListQueueMessages(r.Context(), queueID, 50)
	if err != nil {
		h.respondError(w, err)
		return
	}
	h.respondJSON(w, http.StatusOK, messages)
}

// --- Cron Triggers Endpoints ---

type CreateCronReq struct {
	Name        string `json:"name"`
	Cron        string `json:"cron"`
	TargetAppID string `json:"target_app_id"`
}

func (h *ServicesHandler) ListCronTriggers(w http.ResponseWriter, r *http.Request) {
	triggers, err := h.usecase.ListCronTriggers(r.Context())
	if err != nil {
		h.respondError(w, err)
		return
	}
	h.respondJSON(w, http.StatusOK, triggers)
}

func (h *ServicesHandler) CreateCronTrigger(w http.ResponseWriter, r *http.Request) {
	var req CreateCronReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, domain.NewValidationError("invalid request body"))
		return
	}
	trigger, err := h.usecase.CreateCronTrigger(r.Context(), req.Name, req.Cron, req.TargetAppID)
	if err != nil {
		h.respondError(w, err)
		return
	}
	h.respondJSON(w, http.StatusCreated, trigger)
}

func (h *ServicesHandler) DeleteCronTrigger(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.usecase.DeleteCronTrigger(r.Context(), id); err != nil {
		h.respondError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ServicesHandler) RunTriggerNow(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	run, err := h.usecase.RunTriggerNow(r.Context(), id)
	if err != nil {
		h.respondError(w, err)
		return
	}
	h.respondJSON(w, http.StatusOK, run)
}

func (h *ServicesHandler) ListCronRuns(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	runs, err := h.usecase.ListCronRuns(r.Context(), id)
	if err != nil {
		h.respondError(w, err)
		return
	}
	h.respondJSON(w, http.StatusOK, runs)
}

// --- Workflows Endpoints ---

type CreateWorkflowReq struct {
	Name        string   `json:"name"`
	TargetAppID string   `json:"target_app_id"`
	Steps       []string `json:"steps"`
}

func (h *ServicesHandler) ListWorkflows(w http.ResponseWriter, r *http.Request) {
	wfs, err := h.usecase.ListWorkflows(r.Context())
	if err != nil {
		h.respondError(w, err)
		return
	}
	h.respondJSON(w, http.StatusOK, wfs)
}

func (h *ServicesHandler) CreateWorkflow(w http.ResponseWriter, r *http.Request) {
	var req CreateWorkflowReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, domain.NewValidationError("invalid request body"))
		return
	}
	wf, err := h.usecase.CreateWorkflow(r.Context(), req.Name, req.TargetAppID, req.Steps)
	if err != nil {
		h.respondError(w, err)
		return
	}
	h.respondJSON(w, http.StatusCreated, wf)
}

func (h *ServicesHandler) DeleteWorkflow(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.usecase.DeleteWorkflow(r.Context(), id); err != nil {
		h.respondError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ServicesHandler) TriggerWorkflowRun(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	run, err := h.usecase.TriggerWorkflowRun(r.Context(), id)
	if err != nil {
		h.respondError(w, err)
		return
	}
	h.respondJSON(w, http.StatusOK, run)
}

func (h *ServicesHandler) ListWorkflowRuns(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	runs, err := h.usecase.ListWorkflowRuns(r.Context(), id)
	if err != nil {
		h.respondError(w, err)
		return
	}
	h.respondJSON(w, http.StatusOK, runs)
}

// --- Durable Objects Endpoints ---

type CreateDOClassReq struct {
	Name   string            `json:"name"`
	AppID  string            `json:"app_id"`
	Facets []domain.DOFacet `json:"facets"`
}

func (h *ServicesHandler) ListDurableObjectClasses(w http.ResponseWriter, r *http.Request) {
	classes, err := h.usecase.ListDurableObjectClasses(r.Context())
	if err != nil {
		h.respondError(w, err)
		return
	}
	h.respondJSON(w, http.StatusOK, classes)
}

func (h *ServicesHandler) CreateDurableObjectClass(w http.ResponseWriter, r *http.Request) {
	var req CreateDOClassReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, domain.NewValidationError("invalid request body"))
		return
	}
	doc, err := h.usecase.CreateDurableObjectClass(r.Context(), req.Name, req.AppID, req.Facets)
	if err != nil {
		h.respondError(w, err)
		return
	}
	h.respondJSON(w, http.StatusCreated, doc)
}

func (h *ServicesHandler) DeleteDurableObjectClass(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.usecase.DeleteDurableObjectClass(r.Context(), id); err != nil {
		h.respondError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type CreateDOInstanceReq struct {
	ObjectID string `json:"object_id"`
}

func (h *ServicesHandler) ListDOInstances(w http.ResponseWriter, r *http.Request) {
	classID := chi.URLParam(r, "id")
	instances, err := h.usecase.ListDOInstances(r.Context(), classID)
	if err != nil {
		h.respondError(w, err)
		return
	}
	h.respondJSON(w, http.StatusOK, instances)
}

func (h *ServicesHandler) CreateDOInstance(w http.ResponseWriter, r *http.Request) {
	classID := chi.URLParam(r, "id")
	var req CreateDOInstanceReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, domain.NewValidationError("invalid request body"))
		return
	}
	inst, err := h.usecase.CreateDOInstance(r.Context(), classID, req.ObjectID)
	if err != nil {
		h.respondError(w, err)
		return
	}
	h.respondJSON(w, http.StatusCreated, inst)
}

// --- Containers Endpoints ---

type CreateContainerReq struct {
	Name    string            `json:"name"`
	Image   string            `json:"image"`
	Port    int               `json:"port"`
	EnvVars map[string]string `json:"env_vars"`
}

func (h *ServicesHandler) ListContainers(w http.ResponseWriter, r *http.Request) {
	ctList, err := h.usecase.ListContainers(r.Context())
	if err != nil {
		h.respondError(w, err)
		return
	}
	h.respondJSON(w, http.StatusOK, ctList)
}

func (h *ServicesHandler) CreateContainer(w http.ResponseWriter, r *http.Request) {
	var req CreateContainerReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, domain.NewValidationError("invalid request body"))
		return
	}
	ct, err := h.usecase.CreateContainer(r.Context(), req.Name, req.Image, req.Port, req.EnvVars)
	if err != nil {
		h.respondError(w, err)
		return
	}
	h.respondJSON(w, http.StatusCreated, ct)
}

func (h *ServicesHandler) DeleteContainer(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.usecase.DeleteContainer(r.Context(), id); err != nil {
		h.respondError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Static Assets Endpoints ---

type CreateStaticSiteReq struct {
	Name      string `json:"name"`
	Subdomain string `json:"subdomain"`
}

func (h *ServicesHandler) ListStaticSites(w http.ResponseWriter, r *http.Request) {
	sites, err := h.usecase.ListStaticSites(r.Context())
	if err != nil {
		h.respondError(w, err)
		return
	}
	h.respondJSON(w, http.StatusOK, sites)
}

func (h *ServicesHandler) CreateStaticSite(w http.ResponseWriter, r *http.Request) {
	var req CreateStaticSiteReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, domain.NewValidationError("invalid request body"))
		return
	}
	site, err := h.usecase.CreateStaticSite(r.Context(), req.Name, req.Subdomain)
	if err != nil {
		h.respondError(w, err)
		return
	}
	h.respondJSON(w, http.StatusCreated, site)
}

func (h *ServicesHandler) DeleteStaticSite(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.usecase.DeleteStaticSite(r.Context(), id); err != nil {
		h.respondError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Dynamic Workers (Eval) ---

type EvalDynamicWorkerReq struct {
	Code    string            `json:"code"`
	Method  string            `json:"method"`
	Path    string            `json:"path"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
}

type EvalDynamicWorkerResp struct {
	Status  int               `json:"status"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
}

func (h *ServicesHandler) EvalDynamicWorker(w http.ResponseWriter, r *http.Request) {
	var req EvalDynamicWorkerReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, domain.NewValidationError("invalid request body"))
		return
	}

	method := req.Method
	if method == "" {
		method = "GET"
	}
	path := req.Path
	if path == "" {
		path = "/"
	}

	status, headers, respBody, err := h.usecase.EvalDynamicWorker(
		r.Context(),
		req.Code,
		method,
		path,
		req.Headers,
		[]byte(req.Body),
	)
	if err != nil {
		h.respondError(w, err)
		return
	}

	h.respondJSON(w, http.StatusOK, EvalDynamicWorkerResp{
		Status:  status,
		Headers: headers,
		Body:    string(respBody),
	})
}
