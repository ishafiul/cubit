package node

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ishaf/cubit/internal/domain"
)

// Handler handles HTTP requests for fleet nodes.
type Handler struct {
	service Service
}

// NewHandler creates a new Node Handler.
func NewHandler(s Service) *Handler {
	return &Handler{service: s}
}

type createNodeRequest struct {
	Name         string  `json:"name" binding:"required"`
	IpAddress    string  `json:"ipAddress" binding:"required"`
	InternalPort *int    `json:"internalPort,omitempty"`
	WorkerPort   *int    `json:"workerPort,omitempty"`
	CelldVersion *string `json:"celldVersion,omitempty"`
}

type nodeSpecsResponse struct {
	CpuCores      int   `json:"cpuCores"`
	MemoryBytes   int64 `json:"memoryBytes"`
	DiskFreeBytes int64 `json:"diskFreeBytes"`
}

type nodeResponse struct {
	Id           string            `json:"id"`
	Name         string            `json:"name"`
	IpAddress    string            `json:"ipAddress"`
	InternalPort int               `json:"internalPort"`
	WorkerPort   int               `json:"workerPort"`
	Status       string            `json:"status"`
	CelldVersion string            `json:"celldVersion"`
	Specs        nodeSpecsResponse `json:"specs"`
	CreatedAt    string            `json:"createdAt"`
	UpdatedAt    string            `json:"updatedAt"`
}

func toNodeResponse(n *domain.Node) nodeResponse {
	return nodeResponse{
		Id:           n.ID,
		Name:         n.Name,
		IpAddress:    n.IPAddress,
		InternalPort: n.InternalPort,
		WorkerPort:   n.WorkerPort,
		Status:       string(n.Status),
		CelldVersion: n.CelldVersion,
		Specs: nodeSpecsResponse{
			CpuCores:      n.Specs.CPUCores,
			MemoryBytes:   n.Specs.MemoryBytes,
			DiskFreeBytes: n.Specs.DiskFreeBytes,
		},
		CreatedAt: n.CreatedAt.Format("2006-01-02T15:04:05.999999999Z"),
		UpdatedAt: n.UpdatedAt.Format("2006-01-02T15:04:05.999999999Z"),
	}
}

// List GET /nodes
func (h *Handler) List(c *gin.Context) {
	nodes, err := h.service.ListNodes(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}

	result := make([]nodeResponse, 0, len(nodes))
	for _, n := range nodes {
		result = append(result, toNodeResponse(n))
	}
	c.JSON(http.StatusOK, result)
}

// Create POST /nodes
func (h *Handler) Create(c *gin.Context) {
	var req createNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "malformed request body: " + err.Error()})
		return
	}

	internalPort := 9090
	if req.InternalPort != nil && *req.InternalPort > 0 {
		internalPort = *req.InternalPort
	}
	workerPort := 8080
	if req.WorkerPort != nil && *req.WorkerPort > 0 {
		workerPort = *req.WorkerPort
	}
	celldVer := domain.DefaultCelldVersion
	if req.CelldVersion != nil && *req.CelldVersion != "" {
		celldVer = *req.CelldVersion
	}

	node, err := h.service.RegisterNode(c.Request.Context(), req.Name, req.IpAddress, internalPort, workerPort, celldVer)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, toNodeResponse(node))
}

// GetByID GET /nodes/:id
func (h *Handler) GetByID(c *gin.Context) {
	id := c.Param("id")
	node, err := h.service.GetNode(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, toNodeResponse(node))
}

// Delete DELETE /nodes/:id
func (h *Handler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.service.DeleteNode(c.Request.Context(), id); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// Drain POST /nodes/:id/drain
func (h *Handler) Drain(c *gin.Context) {
	id := c.Param("id")
	node, err := h.service.DrainNode(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, toNodeResponse(node))
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
