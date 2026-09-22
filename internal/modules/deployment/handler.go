package deployment

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ishaf/cubit/internal/domain"
)

// Handler handles HTTP requests for deployments.
type Handler struct {
	service Service
}

// NewHandler creates a new Deployment Handler.
func NewHandler(s Service) *Handler {
	return &Handler{service: s}
}

type createDeploymentRequest struct {
	CommitHash   *string `json:"commitHash,omitempty"`
	ForceRebuild *bool   `json:"forceRebuild,omitempty"`
}

type rollbackApplicationRequest struct {
	DeploymentID string `json:"deploymentId" binding:"required"`
}

type deploymentResponse struct {
	ID            string  `json:"id"`
	ApplicationID string  `json:"applicationId"`
	BuildVersion  int     `json:"buildVersion"`
	CommitHash    string  `json:"commitHash"`
	CommitMessage *string `json:"commitMessage,omitempty"`
	Status        string  `json:"status"`
	BundleSize    int64   `json:"bundleSize"`
	ErrorMessage  *string `json:"errorMessage,omitempty"`
	CreatedAt     string  `json:"createdAt"`
	FinishedAt    *string `json:"finishedAt,omitempty"`
}

type logEntryResponse struct {
	Timestamp string `json:"timestamp"`
	Step      string `json:"step"`
	Message   string `json:"message"`
	Level     string `json:"level"`
}

func toDeploymentResponse(dep *domain.Deployment) deploymentResponse {
	var commitMsg *string
	if dep.CommitMessage != "" {
		commitMsg = &dep.CommitMessage
	}
	var errMsg *string
	if dep.ErrorMessage != "" {
		errMsg = &dep.ErrorMessage
	}
	var finishedAt *string
	if dep.FinishedAt != nil {
		t := dep.FinishedAt.Format("2006-01-02T15:04:05.999999999Z")
		finishedAt = &t
	}

	return deploymentResponse{
		ID:            dep.ID,
		ApplicationID: dep.ApplicationID,
		BuildVersion:  dep.BuildVersion,
		CommitHash:    dep.CommitHash,
		CommitMessage: commitMsg,
		Status:        string(dep.Status),
		BundleSize:    dep.BundleSize,
		ErrorMessage:  errMsg,
		CreatedAt:     dep.CreatedAt.Format("2006-01-02T15:04:05.999999999Z"),
		FinishedAt:    finishedAt,
	}
}

func toLogResponse(l domain.DeploymentLog) logEntryResponse {
	return logEntryResponse{
		Timestamp: l.Timestamp.Format("2006-01-02T15:04:05.999999999Z"),
		Step:      string(l.Step),
		Message:   l.Message,
		Level:     string(l.Level),
	}
}

// Deploy POST /applications/:id/deploy
func (h *Handler) Deploy(c *gin.Context) {
	appID := c.Param("id")
	var req createDeploymentRequest
	_ = c.ShouldBindJSON(&req)

	commitHash := ""
	if req.CommitHash != nil {
		commitHash = *req.CommitHash
	}

	dep, err := h.service.Deploy(c.Request.Context(), appID, commitHash)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, toDeploymentResponse(dep))
}

type deployDirectRequest struct {
	Bundle         string  `json:"bundle"`
	CommitMessage  *string `json:"commitMessage,omitempty"`
	WranglerConfig *string `json:"wranglerConfig,omitempty"`
}

// DeployDirect POST /applications/:id/deploy/direct
func (h *Handler) DeployDirect(c *gin.Context) {
	appID := c.Param("id")

	var bundleBytes []byte
	var commitMessage string
	var wranglerConfig string

	if strings.HasPrefix(c.GetHeader("Content-Type"), "multipart/form-data") {
		form, err := c.MultipartForm()
		if err == nil {
			if files := form.File["bundle"]; len(files) > 0 {
				f, err := files[0].Open()
				if err == nil {
					bundleBytes, _ = io.ReadAll(f)
					_ = f.Close()
				}
			} else if files := form.File["file"]; len(files) > 0 {
				f, err := files[0].Open()
				if err == nil {
					bundleBytes, _ = io.ReadAll(f)
					_ = f.Close()
				}
			}
			if len(form.Value["commitMessage"]) > 0 {
				commitMessage = form.Value["commitMessage"][0]
			}
			if len(form.Value["wranglerConfig"]) > 0 {
				wranglerConfig = form.Value["wranglerConfig"][0]
			}
		}
	} else {
		var req deployDirectRequest
		if err := c.ShouldBindJSON(&req); err == nil {
			bundleBytes = []byte(req.Bundle)
			if req.CommitMessage != nil {
				commitMessage = *req.CommitMessage
			}
			if req.WranglerConfig != nil {
				wranglerConfig = *req.WranglerConfig
			}
		}
	}

	dep, err := h.service.DeployDirect(c.Request.Context(), appID, bundleBytes, commitMessage, wranglerConfig)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusCreated, toDeploymentResponse(dep))
}

// Rollback POST /applications/:id/rollback
func (h *Handler) Rollback(c *gin.Context) {
	appID := c.Param("id")
	var req rollbackApplicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "deploymentId is required"})
		return
	}

	dep, err := h.service.Rollback(c.Request.Context(), appID, req.DeploymentID)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, toDeploymentResponse(dep))
}

// ListByApp GET /applications/:id/deployments
func (h *Handler) ListByApp(c *gin.Context) {
	appID := c.Param("id")
	deps, err := h.service.ListByApp(c.Request.Context(), appID)
	if err != nil {
		respondError(c, err)
		return
	}

	result := make([]deploymentResponse, 0, len(deps))
	for _, d := range deps {
		result = append(result, toDeploymentResponse(d))
	}
	c.JSON(http.StatusOK, result)
}

// GetByID GET /deployments/:id
func (h *Handler) GetByID(c *gin.Context) {
	id := c.Param("id")
	dep, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, toDeploymentResponse(dep))
}

// GetLogs GET /deployments/:id/logs
func (h *Handler) GetLogs(c *gin.Context) {
	id := c.Param("id")
	logs, err := h.service.GetLogs(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}

	result := make([]logEntryResponse, 0, len(logs))
	for _, l := range logs {
		result = append(result, toLogResponse(l))
	}
	c.JSON(http.StatusOK, result)
}

// StreamLogs GET /deployments/:id/logs/stream (Server-Sent Events)
func (h *Handler) StreamLogs(c *gin.Context) {
	depID := c.Param("id")
	if depID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing deployment id"})
		return
	}

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
	c.Writer.Flush()

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Minute)
	defer cancel()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	lastCount := 0

	// Immediate initial flush of existing logs
	if initialLogs, err := h.service.GetLogs(ctx, depID); err == nil && len(initialLogs) > 0 {
		for _, entry := range initialLogs {
			data, _ := json.Marshal(entry)
			fmt.Fprintf(c.Writer, "data: %s\n\n", data)
		}
		lastCount = len(initialLogs)
		c.Writer.Flush()
	}

	// Check if already in terminal state
	if dep, err := h.service.GetByID(ctx, depID); err == nil && (dep.Status == domain.DeploymentStatusActive || dep.Status == domain.DeploymentStatusFailed) {
		fmt.Fprintf(c.Writer, "event: complete\ndata: {\"status\":\"%s\"}\n\n", dep.Status)
		c.Writer.Flush()
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			logs, err := h.service.GetLogs(ctx, depID)
			if err != nil {
				return
			}

			if len(logs) > lastCount {
				for _, entry := range logs[lastCount:] {
					data, _ := json.Marshal(entry)
					fmt.Fprintf(c.Writer, "data: %s\n\n", data)
				}
				lastCount = len(logs)
				c.Writer.Flush()
			}

			dep, err := h.service.GetByID(ctx, depID)
			if err == nil && (dep.Status == domain.DeploymentStatusActive || dep.Status == domain.DeploymentStatusFailed) {
				fmt.Fprintf(c.Writer, "event: complete\ndata: {\"status\":\"%s\"}\n\n", dep.Status)
				c.Writer.Flush()
				return
			}
		}
	}
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
