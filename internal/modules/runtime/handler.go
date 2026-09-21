package runtime

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ishaf/cubit/internal/domain"
)

// Handler handles HTTP requests for celld runtime status and upgrades.
type Handler struct {
	service Service
}

// NewHandler creates a new Runtime Handler.
func NewHandler(s Service) *Handler {
	return &Handler{service: s}
}

type upgradeRequest struct {
	TargetVersion string `json:"targetVersion" binding:"required"`
}

// Status GET /runtime/status
func (h *Handler) Status(c *gin.Context) {
	st, err := h.service.GetStatus(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, st)
}

// Upgrade POST /runtime/upgrade
func (h *Handler) Upgrade(c *gin.Context) {
	var req upgradeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "malformed request body: " + err.Error()})
		return
	}

	if err := h.service.UpgradeCelldDaemon(c.Request.Context(), req.TargetVersion); err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "Upgrade to " + req.TargetVersion + " completed successfully",
		"targetVersion": req.TargetVersion,
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
