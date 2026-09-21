package domain

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	coreDomain "github.com/ishaf/cubit/internal/domain"
)

// Handler handles HTTP requests for custom domains.
type Handler struct {
	service Service
}

// NewHandler creates a new Domain Handler.
func NewHandler(s Service) *Handler {
	return &Handler{service: s}
}

type createDomainRequest struct {
	ApplicationId string  `json:"applicationId" binding:"required"`
	Hostname      string  `json:"hostname" binding:"required"`
	PathPrefix    *string `json:"pathPrefix,omitempty"`
}

type domainResponse struct {
	Id            string `json:"id"`
	ApplicationId string `json:"applicationId"`
	Hostname      string `json:"hostname"`
	PathPrefix    string `json:"pathPrefix"`
	SslActive     bool   `json:"sslActive"`
	CreatedAt     string `json:"createdAt"`
	UpdatedAt     string `json:"updatedAt"`
}

func toDomainResponse(d *coreDomain.Domain) domainResponse {
	return domainResponse{
		Id:            d.ID,
		ApplicationId: d.ApplicationID,
		Hostname:      d.Hostname,
		PathPrefix:    d.PathPrefix,
		SslActive:     d.SSLActive,
		CreatedAt:     d.CreatedAt.Format("2006-01-02T15:04:05.999999999Z"),
		UpdatedAt:     d.UpdatedAt.Format("2006-01-02T15:04:05.999999999Z"),
	}
}

// List GET /domains
func (h *Handler) List(c *gin.Context) {
	domains, err := h.service.ListDomains(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}

	result := make([]domainResponse, 0, len(domains))
	for _, d := range domains {
		result = append(result, toDomainResponse(d))
	}
	c.JSON(http.StatusOK, result)
}

// Create POST /domains
func (h *Handler) Create(c *gin.Context) {
	var req createDomainRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "malformed request body: " + err.Error()})
		return
	}

	prefix := "/"
	if req.PathPrefix != nil && *req.PathPrefix != "" {
		prefix = *req.PathPrefix
	}

	dom, err := h.service.AddDomain(c.Request.Context(), req.ApplicationId, req.Hostname, prefix)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, toDomainResponse(dom))
}

// GetByID GET /domains/:id
func (h *Handler) GetByID(c *gin.Context) {
	id := c.Param("id")
	dom, err := h.service.GetDomain(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, toDomainResponse(dom))
}

// Delete DELETE /domains/:id
func (h *Handler) Delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.service.DeleteDomain(c.Request.Context(), id); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func respondError(c *gin.Context, err error) {
	var domErr *coreDomain.DomainError
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
