package service

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ishaf/cubit/internal/domain"
)

// ServicesHandler handles HTTP requests for all 12 celld services.
type Handler struct {
	service *ServicesService
}

// NewServicesHandler creates a new ServicesHandler.
func NewHandler(s *ServicesService) *Handler {
	return &Handler{service: s}
}

// --- Response Helpers ---

func (h *Handler) respondJSON(c *gin.Context, code int, payload interface{}) {
	c.JSON(code, payload)
}

func (h *Handler) respondError(c *gin.Context, err error) {
	code := http.StatusInternalServerError
	errCode := "INTERNAL_ERROR"

	var domErr *domain.DomainError
	if errors.As(err, &domErr) {
		switch domErr.Code {
		case "NOT_FOUND":
			code = http.StatusNotFound
			errCode = "NOT_FOUND"
		case "VALIDATION_ERROR":
			code = http.StatusBadRequest
			errCode = "VALIDATION_ERROR"
		case "CONFLICT":
			code = http.StatusConflict
			errCode = "CONFLICT"
		case "INVALID_STATE":
			code = http.StatusBadRequest
			errCode = "INVALID_STATE"
		case "FORBIDDEN":
			code = http.StatusForbidden
			errCode = "FORBIDDEN"
		}
	} else if errors.Is(err, domain.ErrNotFound) {
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
	} else if errors.Is(err, domain.ErrForbidden) {
		code = http.StatusForbidden
		errCode = "FORBIDDEN"
	}

	h.respondJSON(c, code, map[string]string{
		"code":    errCode,
		"message": err.Error(),
		"error":   err.Error(),
	})
}

// --- KV Endpoints ---

type CreateKVNamespaceReq struct {
	Title string `json:"title"`
	Name  string `json:"name"`
}

func (h *Handler) ListKVNamespaces(c *gin.Context) {
	namespaces, err := h.service.ListKVNamespaces(c.Request.Context())
	if err != nil {
		h.respondError(c, err)
		return
	}
	h.respondJSON(c, http.StatusOK, namespaces)
}

func (h *Handler) CreateKVNamespace(c *gin.Context) {
	var req CreateKVNamespaceReq
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		h.respondError(c, domain.NewValidationError("invalid request body"))
		return
	}
	name := req.Title
	if name == "" {
		name = req.Name
	}
	ns, err := h.service.CreateKVNamespace(c.Request.Context(), name)
	if err != nil {
		h.respondError(c, err)
		return
	}
	h.respondJSON(c, http.StatusCreated, ns)
}

func (h *Handler) DeleteKVNamespace(c *gin.Context) {
	id := c.Param("id")
	if err := h.service.DeleteKVNamespace(c.Request.Context(), id); err != nil {
		h.respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) ListKVPairs(c *gin.Context) {
	id := c.Param("id")
	pairs, err := h.service.ListKVPairs(c.Request.Context(), id)
	if err != nil {
		h.respondError(c, err)
		return
	}
	h.respondJSON(c, http.StatusOK, pairs)
}

func (h *Handler) GetKVPair(c *gin.Context) {
	id := c.Param("id")
	key := c.Param("key")
	pair, err := h.service.GetKVPair(c.Request.Context(), id, key)
	if err != nil {
		h.respondError(c, err)
		return
	}
	h.respondJSON(c, http.StatusOK, pair)
}

type PutKVPairReq struct {
	Value         string `json:"value"`
	ExpirationTTL int    `json:"expiration_ttl"`
	Metadata      string `json:"metadata"`
}

func (h *Handler) PutKVPair(c *gin.Context) {
	id := c.Param("id")
	key := c.Param("key")

	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		h.respondError(c, domain.NewValidationError("failed reading body"))
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

	pair, err := h.service.PutKVPair(c.Request.Context(), id, key, value, ttl, metadata)
	if err != nil {
		h.respondError(c, err)
		return
	}
	h.respondJSON(c, http.StatusOK, pair)
}

func (h *Handler) DeleteKVPair(c *gin.Context) {
	id := c.Param("id")
	key := c.Param("key")
	if err := h.service.DeleteKVPair(c.Request.Context(), id, key); err != nil {
		h.respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// --- D1 Endpoints ---

type CreateD1Req struct {
	Name string `json:"name"`
}

func (h *Handler) ListD1Databases(c *gin.Context) {
	dbs, err := h.service.ListD1Databases(c.Request.Context())
	if err != nil {
		h.respondError(c, err)
		return
	}
	h.respondJSON(c, http.StatusOK, dbs)
}

func (h *Handler) CreateD1Database(c *gin.Context) {
	var req CreateD1Req
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		h.respondError(c, domain.NewValidationError("invalid request body"))
		return
	}
	db, err := h.service.CreateD1Database(c.Request.Context(), req.Name)
	if err != nil {
		h.respondError(c, err)
		return
	}
	h.respondJSON(c, http.StatusCreated, db)
}

func (h *Handler) DeleteD1Database(c *gin.Context) {
	id := c.Param("id")
	if err := h.service.DeleteD1Database(c.Request.Context(), id); err != nil {
		h.respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

type QueryD1Req struct {
	SQL   string `json:"sql"`
	Query string `json:"query"`
}

func (h *Handler) ExecuteD1Query(c *gin.Context) {
	id := c.Param("id")
	var req QueryD1Req
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		h.respondError(c, domain.NewValidationError("invalid request body"))
		return
	}
	sqlQuery := req.SQL
	if sqlQuery == "" {
		sqlQuery = req.Query
	}
	result, err := h.service.ExecuteD1Query(c.Request.Context(), id, sqlQuery)
	if err != nil {
		h.respondError(c, err)
		return
	}
	h.respondJSON(c, http.StatusOK, result)
}

// --- R2 Endpoints ---

type CreateR2BucketReq struct {
	Name string `json:"name" binding:"required"`
}

func (h *Handler) CreateR2Bucket(c *gin.Context) {
	var req CreateR2BucketReq
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		h.respondError(c, domain.NewValidationError("invalid request body"))
		return
	}
	bucket, err := h.service.CreateR2Bucket(c.Request.Context(), req.Name)
	if err != nil {
		h.respondError(c, err)
		return
	}
	h.respondJSON(c, http.StatusCreated, bucket)
}

func (h *Handler) DeleteR2Bucket(c *gin.Context) {
	bucketName := c.Param("name")
	if err := h.service.DeleteR2Bucket(c.Request.Context(), bucketName); err != nil {
		h.respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) ListR2Buckets(c *gin.Context) {
	buckets, err := h.service.ListR2Buckets(c.Request.Context())
	if err != nil {
		h.respondError(c, err)
		return
	}
	h.respondJSON(c, http.StatusOK, buckets)
}

func (h *Handler) ListR2Objects(c *gin.Context) {
	bucketName := c.Param("name")
	objects, err := h.service.ListR2Objects(c.Request.Context(), bucketName)
	if err != nil {
		h.respondError(c, err)
		return
	}
	h.respondJSON(c, http.StatusOK, objects)
}

type UploadR2Req struct {
	Key     string `json:"key"`
	Content string `json:"content"`
}

func (h *Handler) UploadR2Object(c *gin.Context) {
	bucketName := c.Param("name")
	var req UploadR2Req
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		h.respondError(c, domain.NewValidationError("invalid request body"))
		return
	}
	if err := h.service.UploadR2Object(c.Request.Context(), bucketName, req.Key, []byte(req.Content)); err != nil {
		h.respondError(c, err)
		return
	}
	h.respondJSON(c, http.StatusOK, map[string]string{"status": "uploaded", "key": req.Key})
}

func (h *Handler) GetR2Object(c *gin.Context) {
	bucketName := c.Param("name")
	key := strings.TrimPrefix(c.Param("key"), "/")
	data, err := h.service.GetR2Object(c.Request.Context(), bucketName, key)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	contentType := mime.TypeByExtension(filepath.Ext(key))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	c.Data(http.StatusOK, contentType, data)
}

func (h *Handler) DeleteR2Object(c *gin.Context) {
	bucketName := c.Param("name")
	key := strings.TrimPrefix(c.Param("key"), "/")
	if err := h.service.DeleteR2Object(c.Request.Context(), bucketName, key); err != nil {
		h.respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// --- Queues Endpoints ---

type CreateQueueReq struct {
	Name           string `json:"name"`
	ConsumerAppID  string `json:"consumer_app_id"`
	MaxRetries     int    `json:"max_retries"`
}

func (h *Handler) ListQueues(c *gin.Context) {
	queues, err := h.service.ListQueues(c.Request.Context())
	if err != nil {
		h.respondError(c, err)
		return
	}
	h.respondJSON(c, http.StatusOK, queues)
}

func (h *Handler) CreateQueue(c *gin.Context) {
	var req CreateQueueReq
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		h.respondError(c, domain.NewValidationError("invalid request body"))
		return
	}
	q, err := h.service.CreateQueue(c.Request.Context(), req.Name, req.ConsumerAppID, req.MaxRetries)
	if err != nil {
		h.respondError(c, err)
		return
	}
	h.respondJSON(c, http.StatusCreated, q)
}

func (h *Handler) DeleteQueue(c *gin.Context) {
	id := c.Param("id")
	if err := h.service.DeleteQueue(c.Request.Context(), id); err != nil {
		h.respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

type SendQueueMsgReq struct {
	Body string `json:"body"`
}

func (h *Handler) SendQueueMessage(c *gin.Context) {
	queueID := c.Param("id")
	var req SendQueueMsgReq
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		h.respondError(c, domain.NewValidationError("invalid request body"))
		return
	}
	msg, err := h.service.SendQueueMessage(c.Request.Context(), queueID, req.Body)
	if err != nil {
		h.respondError(c, err)
		return
	}
	h.respondJSON(c, http.StatusCreated, msg)
}

func (h *Handler) ListQueueMessages(c *gin.Context) {
	queueID := c.Param("id")
	messages, err := h.service.ListQueueMessages(c.Request.Context(), queueID, 50)
	if err != nil {
		h.respondError(c, err)
		return
	}
	h.respondJSON(c, http.StatusOK, messages)
}

// --- Cron Triggers Endpoints ---

type CreateCronReq struct {
	Name        string `json:"name"`
	Cron        string `json:"cron"`
	TargetAppID string `json:"target_app_id"`
}

func (h *Handler) ListCronTriggers(c *gin.Context) {
	triggers, err := h.service.ListCronTriggers(c.Request.Context())
	if err != nil {
		h.respondError(c, err)
		return
	}
	h.respondJSON(c, http.StatusOK, triggers)
}

func (h *Handler) CreateCronTrigger(c *gin.Context) {
	var req CreateCronReq
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		h.respondError(c, domain.NewValidationError("invalid request body"))
		return
	}
	trigger, err := h.service.CreateCronTrigger(c.Request.Context(), req.Name, req.Cron, req.TargetAppID)
	if err != nil {
		h.respondError(c, err)
		return
	}
	h.respondJSON(c, http.StatusCreated, trigger)
}

func (h *Handler) DeleteCronTrigger(c *gin.Context) {
	id := c.Param("id")
	if err := h.service.DeleteCronTrigger(c.Request.Context(), id); err != nil {
		h.respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) RunTriggerNow(c *gin.Context) {
	id := c.Param("id")
	run, err := h.service.RunTriggerNow(c.Request.Context(), id)
	if err != nil {
		h.respondError(c, err)
		return
	}
	h.respondJSON(c, http.StatusOK, run)
}

func (h *Handler) ListCronRuns(c *gin.Context) {
	id := c.Param("id")
	runs, err := h.service.ListCronRuns(c.Request.Context(), id)
	if err != nil {
		h.respondError(c, err)
		return
	}
	h.respondJSON(c, http.StatusOK, runs)
}

// --- Workflows Endpoints ---

type CreateWorkflowReq struct {
	Name        string   `json:"name"`
	TargetAppID string   `json:"target_app_id"`
	Steps       []string `json:"steps"`
}

func (h *Handler) ListWorkflows(c *gin.Context) {
	wfs, err := h.service.ListWorkflows(c.Request.Context())
	if err != nil {
		h.respondError(c, err)
		return
	}
	h.respondJSON(c, http.StatusOK, wfs)
}

func (h *Handler) CreateWorkflow(c *gin.Context) {
	var req CreateWorkflowReq
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		h.respondError(c, domain.NewValidationError("invalid request body"))
		return
	}
	wf, err := h.service.CreateWorkflow(c.Request.Context(), req.Name, req.TargetAppID, req.Steps)
	if err != nil {
		h.respondError(c, err)
		return
	}
	h.respondJSON(c, http.StatusCreated, wf)
}

func (h *Handler) DeleteWorkflow(c *gin.Context) {
	id := c.Param("id")
	if err := h.service.DeleteWorkflow(c.Request.Context(), id); err != nil {
		h.respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) TriggerWorkflowRun(c *gin.Context) {
	id := c.Param("id")
	run, err := h.service.TriggerWorkflowRun(c.Request.Context(), id)
	if err != nil {
		h.respondError(c, err)
		return
	}
	h.respondJSON(c, http.StatusOK, run)
}

func (h *Handler) ListWorkflowRuns(c *gin.Context) {
	id := c.Param("id")
	runs, err := h.service.ListWorkflowRuns(c.Request.Context(), id)
	if err != nil {
		h.respondError(c, err)
		return
	}
	h.respondJSON(c, http.StatusOK, runs)
}

// --- Durable Objects Endpoints ---

type CreateDOClassReq struct {
	Name   string            `json:"name"`
	AppID  string            `json:"app_id"`
	Facets []domain.DOFacet `json:"facets"`
}

func (h *Handler) ListDurableObjectClasses(c *gin.Context) {
	classes, err := h.service.ListDurableObjectClasses(c.Request.Context())
	if err != nil {
		h.respondError(c, err)
		return
	}
	h.respondJSON(c, http.StatusOK, classes)
}

func (h *Handler) CreateDurableObjectClass(c *gin.Context) {
	var req CreateDOClassReq
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		h.respondError(c, domain.NewValidationError("invalid request body"))
		return
	}
	doc, err := h.service.CreateDurableObjectClass(c.Request.Context(), req.Name, req.AppID, req.Facets)
	if err != nil {
		h.respondError(c, err)
		return
	}
	h.respondJSON(c, http.StatusCreated, doc)
}

func (h *Handler) DeleteDurableObjectClass(c *gin.Context) {
	id := c.Param("id")
	if err := h.service.DeleteDurableObjectClass(c.Request.Context(), id); err != nil {
		h.respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

type CreateDOInstanceReq struct {
	ObjectID string `json:"object_id"`
}

func (h *Handler) ListDOInstances(c *gin.Context) {
	classID := c.Param("id")
	instances, err := h.service.ListDOInstances(c.Request.Context(), classID)
	if err != nil {
		h.respondError(c, err)
		return
	}
	h.respondJSON(c, http.StatusOK, instances)
}

func (h *Handler) CreateDOInstance(c *gin.Context) {
	classID := c.Param("id")
	var req CreateDOInstanceReq
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		h.respondError(c, domain.NewValidationError("invalid request body"))
		return
	}
	inst, err := h.service.CreateDOInstance(c.Request.Context(), classID, req.ObjectID)
	if err != nil {
		h.respondError(c, err)
		return
	}
	h.respondJSON(c, http.StatusCreated, inst)
}

// --- Containers Endpoints ---

type CreateContainerReq struct {
	Name    string            `json:"name"`
	Image   string            `json:"image"`
	Port    int               `json:"port"`
	EnvVars map[string]string `json:"env_vars"`
}

func (h *Handler) ListContainers(c *gin.Context) {
	ctList, err := h.service.ListContainers(c.Request.Context())
	if err != nil {
		h.respondError(c, err)
		return
	}
	h.respondJSON(c, http.StatusOK, ctList)
}

func (h *Handler) CreateContainer(c *gin.Context) {
	var req CreateContainerReq
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		h.respondError(c, domain.NewValidationError("invalid request body"))
		return
	}
	ct, err := h.service.CreateContainer(c.Request.Context(), req.Name, req.Image, req.Port, req.EnvVars)
	if err != nil {
		h.respondError(c, err)
		return
	}
	h.respondJSON(c, http.StatusCreated, ct)
}

func (h *Handler) DeleteContainer(c *gin.Context) {
	id := c.Param("id")
	if err := h.service.DeleteContainer(c.Request.Context(), id); err != nil {
		h.respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// --- Static Assets Endpoints ---

type CreateStaticSiteReq struct {
	Name      string `json:"name"`
	Subdomain string `json:"subdomain"`
}

func (h *Handler) ListStaticSites(c *gin.Context) {
	sites, err := h.service.ListStaticSites(c.Request.Context())
	if err != nil {
		h.respondError(c, err)
		return
	}
	h.respondJSON(c, http.StatusOK, sites)
}

func (h *Handler) CreateStaticSite(c *gin.Context) {
	var req CreateStaticSiteReq
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		h.respondError(c, domain.NewValidationError("invalid request body"))
		return
	}
	site, err := h.service.CreateStaticSite(c.Request.Context(), req.Name, req.Subdomain)
	if err != nil {
		h.respondError(c, err)
		return
	}
	h.respondJSON(c, http.StatusCreated, site)
}

func (h *Handler) DeleteStaticSite(c *gin.Context) {
	id := c.Param("id")
	if err := h.service.DeleteStaticSite(c.Request.Context(), id); err != nil {
		h.respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
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

func (h *Handler) EvalDynamicWorker(c *gin.Context) {
	var req EvalDynamicWorkerReq
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		h.respondError(c, domain.NewValidationError("invalid request body"))
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

	status, headers, respBody, err := h.service.EvalDynamicWorker(
		c.Request.Context(),
		req.Code,
		method,
		path,
		req.Headers,
		[]byte(req.Body),
	)
	if err != nil {
		h.respondError(c, err)
		return
	}

	h.respondJSON(c, http.StatusOK, EvalDynamicWorkerResp{
		Status:  status,
		Headers: headers,
		Body:    string(respBody),
	})
}
