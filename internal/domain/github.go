package domain

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

// GitHubAppSettings holds configuration and credentials for the connected GitHub App.
type GitHubAppSettings struct {
	ID             string    `json:"id"`
	AppID          string    `json:"appId"`
	AppName        string    `json:"appName"`
	AppSlug        string    `json:"appSlug"`
	ClientID       string    `json:"clientId"`
	ClientSecret   string    `json:"clientSecret"`
	WebhookSecret  string    `json:"webhookSecret"`
	PrivateKey     string    `json:"privateKey"`
	InstallationID string    `json:"installationId"`
	IsConfigured   bool      `json:"isConfigured"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

// Slug returns the URL-friendly slug for the GitHub App.
func (s *GitHubAppSettings) Slug() string {
	if s.AppSlug != "" {
		return s.AppSlug
	}
	slug := strings.ToLower(strings.TrimSpace(s.AppName))
	slug = strings.ReplaceAll(slug, " ", "-")
	return slug
}

// MaskedSettings returns a sanitized version of settings safe to return in HTTP responses.
func (s *GitHubAppSettings) MaskedSettings(baseURL string) map[string]any {
	webhookURL := strings.TrimRight(baseURL, "/") + "/api/v1/github/webhook"
	hasKey := strings.TrimSpace(s.PrivateKey) != ""
	hasSecret := strings.TrimSpace(s.WebhookSecret) != ""
	slug := s.Slug()
	installURL := ""
	if slug != "" {
		installURL = "https://github.com/apps/" + slug + "/installations/new"
	}

	return map[string]any{
		"id":               s.ID,
		"appId":            s.AppID,
		"appName":          s.AppName,
		"appSlug":          slug,
		"installUrl":       installURL,
		"clientId":         s.ClientID,
		"installationId":   s.InstallationID,
		"isConfigured":     s.IsConfigured,
		"hasPrivateKey":    hasKey,
		"hasWebhookSecret": hasSecret,
		"webhookUrl":       webhookURL,
		"createdAt":        s.CreatedAt,
		"updatedAt":        s.UpdatedAt,
	}
}

// GitHubInstallation represents an account/org installation of the GitHub App.
type GitHubInstallation struct {
	ID            int64  `json:"id"`
	AccountLogin  string `json:"accountLogin"`
	AccountType   string `json:"accountType"`
	AccountAvatar string `json:"accountAvatar"`
	HTMLURL       string `json:"htmlUrl"`
	TargetID      int64  `json:"targetId"`
	TargetType    string `json:"targetType"`
}

// GitHubRepository represents a repository accessible through GitHub App or token.
type GitHubRepository struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	FullName      string `json:"fullName"` // e.g. "octocat/hello-world"
	DefaultBranch string `json:"defaultBranch"`
	Private       bool   `json:"private"`
	CloneURL      string `json:"cloneUrl"`
	HTMLURL       string `json:"htmlUrl"`
}

// GitHubCommitAuthor contains author information in a GitHub commit.
type GitHubCommitAuthor struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// GitHubCommit represents head commit information in a push event.
type GitHubCommit struct {
	ID        string             `json:"id"`
	Message   string             `json:"message"`
	Timestamp string             `json:"timestamp"`
	Author    GitHubCommitAuthor `json:"author"`
}

// GitHubPushEvent represents the payload received from GitHub on push webhooks.
type GitHubPushEvent struct {
	Ref        string `json:"ref"` // e.g. "refs/heads/main"
	Before     string `json:"before"`
	After      string `json:"after"`
	Repository struct {
		ID       int64  `json:"id"`
		Name     string `json:"name"`
		FullName string `json:"full_name"`
		CloneURL string `json:"clone_url"`
		HTMLURL  string `json:"html_url"`
		Private  bool   `json:"private"`
	} `json:"repository"`
	HeadCommit *GitHubCommit `json:"head_commit"`
	Pusher     struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	} `json:"pusher"`
}

// BranchName extracts the short branch name from a ref (e.g. "refs/heads/main" -> "main").
func (e *GitHubPushEvent) BranchName() string {
	const prefix = "refs/heads/"
	if strings.HasPrefix(e.Ref, prefix) {
		return strings.TrimPrefix(e.Ref, prefix)
	}
	return e.Ref
}

// ParseGitHubPushPayload parses raw JSON payload into a GitHubPushEvent.
func ParseGitHubPushPayload(data []byte) (*GitHubPushEvent, error) {
	var event GitHubPushEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return nil, errors.New("invalid github push payload")
	}
	if event.Repository.FullName == "" && event.Repository.Name == "" {
		return nil, errors.New("push event missing repository info")
	}
	return &event, nil
}

// VerifyGitHubHMAC validates the X-Hub-Signature-256 header against the payload and secret.
// Signature format: "sha256=<hex-digest>"
func VerifyGitHubHMAC(payload []byte, secret, signatureHeader string) bool {
	if secret == "" {
		// If no secret configured, accept signature (optional local dev mode)
		return true
	}
	if signatureHeader == "" {
		return false
	}

	const prefix = "sha256="
	if !strings.HasPrefix(signatureHeader, prefix) {
		return false
	}
	expectedHex := strings.TrimPrefix(signatureHeader, prefix)

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expectedMAC := mac.Sum(nil)
	actualMAC, err := hex.DecodeString(expectedHex)
	if err != nil {
		return false
	}

	return hmac.Equal(actualMAC, expectedMAC)
}
