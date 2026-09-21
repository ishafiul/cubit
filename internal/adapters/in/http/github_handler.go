package http

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/ishaf/cubit/internal/domain"
	"github.com/ishaf/cubit/internal/usecase"
)

// GitHubHandler handles HTTP requests for GitHub App integration and webhooks.
type GitHubHandler struct {
	usecase *usecase.GitHubUsecase
}

// NewGitHubHandler creates a new GitHubHandler.
func NewGitHubHandler(u *usecase.GitHubUsecase) *GitHubHandler {
	return &GitHubHandler{usecase: u}
}

// RegisterRoutes mounts all GitHub App and webhook endpoints.
func (h *GitHubHandler) RegisterRoutes(r chi.Router) {
	r.Route("/github", func(sub chi.Router) {
		sub.Get("/settings", h.GetSettings)
		sub.Post("/settings", h.SaveSettings)
		sub.Delete("/settings", h.ClearSettings)
		sub.Get("/manifest", h.GetManifest)
		sub.Post("/manifest/exchange", h.ExchangeManifest)
		sub.Get("/repositories", h.ListRepositories)
		sub.Get("/repositories/{owner}/{repo}/branches", h.ListBranches)
		sub.Post("/webhook", h.HandleWebhook)
	})
}

func (h *GitHubHandler) getBaseURL(r *http.Request) string {
	proto := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		proto = "https"
	}
	host := r.Host
	if host == "" {
		host = "localhost:8000"
	}
	return proto + "://" + host
}

// GetSettings retrieves configured GitHub App details (secrets masked).
func (h *GitHubHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := h.usecase.GetSettings(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	baseURL := h.getBaseURL(r)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(settings.MaskedSettings(baseURL))
}

// SaveSettings stores manual or updated GitHub App credentials.
func (h *GitHubHandler) SaveSettings(w http.ResponseWriter, r *http.Request) {
	var s domain.GitHubAppSettings
	if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.usecase.SaveSettings(r.Context(), &s); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	baseURL := h.getBaseURL(r)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(s.MaskedSettings(baseURL))
}

// ClearSettings disconnects the GitHub App configuration.
func (h *GitHubHandler) ClearSettings(w http.ResponseWriter, r *http.Request) {
	if err := h.usecase.ClearSettings(r.Context()); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":  "ok",
		"message": "github app settings disconnected",
	})
}

// GetManifest returns the 1-click GitHub App manifest payload.
func (h *GitHubHandler) GetManifest(w http.ResponseWriter, r *http.Request) {
	baseURL := r.URL.Query().Get("baseUrl")
	if baseURL == "" {
		baseURL = h.getBaseURL(r)
	}

	manifest, err := h.usecase.GenerateManifest(r.Context(), baseURL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(manifest)
}

// ExchangeManifest converts the manifest creation code from GitHub callback.
func (h *GitHubHandler) ExchangeManifest(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Code) == "" {
		http.Error(w, "code is required", http.StatusBadRequest)
		return
	}

	settings, err := h.usecase.ExchangeManifestCode(r.Context(), body.Code)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	baseURL := h.getBaseURL(r)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(settings.MaskedSettings(baseURL))
}

// ListRepositories returns repositories accessible via GitHub App.
func (h *GitHubHandler) ListRepositories(w http.ResponseWriter, r *http.Request) {
	repos, err := h.usecase.ListRepositories(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if repos == nil {
		repos = []domain.GitHubRepository{}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(repos)
}

// ListBranches returns branches for a repository.
func (h *GitHubHandler) ListBranches(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repo := chi.URLParam(r, "repo")
	if owner == "" || repo == "" {
		http.Error(w, "owner and repo parameters required", http.StatusBadRequest)
		return
	}

	branches, err := h.usecase.ListBranches(r.Context(), owner, repo)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(branches)
}

// HandleWebhook receives push and ping webhooks from GitHub.
func (h *GitHubHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed reading payload", http.StatusBadRequest)
		return
	}

	eventType := r.Header.Get("X-GitHub-Event")
	signatureHeader := r.Header.Get("X-Hub-Signature-256")

	if eventType == "" {
		eventType = "push"
	}

	result, err := h.usecase.HandlePushWebhook(r.Context(), eventType, signatureHeader, payload)
	if err != nil {
		if strings.Contains(err.Error(), "signature") {
			http.Error(w, "unauthorized webhook signature", http.StatusUnauthorized)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(result)
}
