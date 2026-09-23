package github

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ishaf/cubit/internal/domain"
)

// Handler handles HTTP requests for GitHub App integration and webhooks using Gin.
type Handler struct {
	service *Service
}

// NewHandler creates a new GitHub Handler.
func NewHandler(s *Service) *Handler {
	return &Handler{service: s}
}

func (h *Handler) getBaseURL(c *gin.Context) string {
	proto := "http"
	if c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https" {
		proto = "https"
	}
	host := c.Request.Host
	if host == "" {
		host = "localhost:8000"
	}
	return proto + "://" + host
}

// GetSettings retrieves configured GitHub App details (secrets masked).
func (h *Handler) GetSettings(c *gin.Context) {
	settings, err := h.service.GetSettings(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	baseURL := h.getBaseURL(c)
	c.JSON(http.StatusOK, settings.MaskedSettings(baseURL))
}

// SaveSettings stores manual or updated GitHub App credentials.
func (h *Handler) SaveSettings(c *gin.Context) {
	var s domain.GitHubAppSettings
	if err := json.NewDecoder(c.Request.Body).Decode(&s); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if err := h.service.SaveSettings(c.Request.Context(), &s); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	baseURL := h.getBaseURL(c)
	c.JSON(http.StatusOK, s.MaskedSettings(baseURL))
}

// ClearSettings disconnects the GitHub App configuration.
func (h *Handler) ClearSettings(c *gin.Context) {
	if err := h.service.ClearSettings(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"message": "github app settings disconnected",
	})
}

// GetManifest returns the 1-click GitHub App manifest payload.
func (h *Handler) GetManifest(c *gin.Context) {
	baseURL := c.Query("baseUrl")
	if baseURL == "" {
		baseURL = h.getBaseURL(c)
	}
	webhookURL := c.Query("webhookUrl")

	manifest, err := h.service.GenerateManifest(c.Request.Context(), baseURL, webhookURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, manifest)
}

// ManifestCallback handles the redirect from GitHub after manifest registration or app installation.
func (h *Handler) ManifestCallback(c *gin.Context) {
	code := c.Query("code")
	if code != "" {
		_, err := h.service.ExchangeManifestCode(c.Request.Context(), code)
		if err != nil {
			c.Redirect(http.StatusTemporaryRedirect, "/github?error="+url.QueryEscape(err.Error()))
			return
		}
	}

	installationID := c.Query("installation_id")
	if installationID != "" {
		settings, err := h.service.GetSettings(c.Request.Context())
		if err == nil && settings.IsConfigured {
			settings.InstallationID = installationID
			_ = h.service.SaveSettings(c.Request.Context(), settings)
		}
	}

	// In background or before redirect, discover and bind active installations
	_, _ = h.service.DiscoverInstallations(c.Request.Context())

	c.Redirect(http.StatusTemporaryRedirect, "/github")
}

// SyncInstallations queries GitHub to discover installations and automatically binds the first active installation.
func (h *Handler) SyncInstallations(c *gin.Context) {
	installs, err := h.service.DiscoverInstallations(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	settings, err := h.service.GetSettings(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	baseURL := h.getBaseURL(c)
	c.JSON(http.StatusOK, gin.H{
		"installations": installs,
		"settings":      settings.MaskedSettings(baseURL),
	})
}

// ExchangeManifest converts the manifest creation code from GitHub callback.
func (h *Handler) ExchangeManifest(c *gin.Context) {
	var body struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(c.Request.Body).Decode(&body); err != nil || strings.TrimSpace(body.Code) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "code is required"})
		return
	}

	settings, err := h.service.ExchangeManifestCode(c.Request.Context(), body.Code)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	baseURL := h.getBaseURL(c)
	c.JSON(http.StatusOK, settings.MaskedSettings(baseURL))
}

// ListRepositories returns repositories accessible via GitHub App.
func (h *Handler) ListRepositories(c *gin.Context) {
	repos, err := h.service.ListRepositories(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if repos == nil {
		repos = []domain.GitHubRepository{}
	}

	c.JSON(http.StatusOK, repos)
}

// ListBranches returns branches for a repository.
func (h *Handler) ListBranches(c *gin.Context) {
	owner := c.Param("owner")
	repo := c.Param("repo")
	if owner == "" || repo == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "owner and repo parameters required"})
		return
	}

	branches, err := h.service.ListBranches(c.Request.Context(), owner, repo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, branches)
}

// ListFolders returns directories in a repository branch with wrangler detection.
func (h *Handler) ListFolders(c *gin.Context) {
	owner := c.Param("owner")
	repo := c.Param("repo")
	branch := c.Query("branch")
	if owner == "" || repo == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "owner and repo parameters required"})
		return
	}

	folders, err := h.service.ListFolders(c.Request.Context(), owner, repo, branch)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, folders)
}

// HandleWebhook receives push and ping webhooks from GitHub.
func (h *Handler) HandleWebhook(c *gin.Context) {
	payload, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed reading payload"})
		return
	}

	eventType := c.GetHeader("X-GitHub-Event")
	signatureHeader := c.GetHeader("X-Hub-Signature-256")

	if eventType == "" {
		eventType = "push"
	}

	result, err := h.service.HandlePushWebhook(c.Request.Context(), eventType, signatureHeader, payload)
	if err != nil {
		if strings.Contains(err.Error(), "signature") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized webhook signature"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}
