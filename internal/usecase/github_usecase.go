package usecase

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/ishaf/cubit/internal/domain"
)

// GitHubUsecase handles GitHub App setup, manifest exchange, repository sync, and webhook processing.
type GitHubUsecase struct {
	githubRepo GitHubRepository
	appRepo    ApplicationRepository
	appUsecase *AppUsecase
	httpClient *http.Client
}

// NewGitHubUsecase creates a new GitHubUsecase.
func NewGitHubUsecase(
	githubRepo GitHubRepository,
	appRepo ApplicationRepository,
	appUsecase *AppUsecase,
) *GitHubUsecase {
	return &GitHubUsecase{
		githubRepo: githubRepo,
		appRepo:    appRepo,
		appUsecase: appUsecase,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// GetSettings returns current GitHub App configuration.
func (u *GitHubUsecase) GetSettings(ctx context.Context) (*domain.GitHubAppSettings, error) {
	return u.githubRepo.GetSettings(ctx)
}

// SaveSettings stores or updates GitHub App configuration.
func (u *GitHubUsecase) SaveSettings(ctx context.Context, settings *domain.GitHubAppSettings) error {
	hasCredentials := strings.TrimSpace(settings.AppID) != "" ||
		strings.TrimSpace(settings.PrivateKey) != "" ||
		strings.TrimSpace(settings.ClientSecret) != ""

	settings.IsConfigured = hasCredentials
	return u.githubRepo.SaveSettings(ctx, settings)
}

// ClearSettings resets the GitHub App configuration.
func (u *GitHubUsecase) ClearSettings(ctx context.Context) error {
	return u.githubRepo.ClearSettings(ctx)
}

// GenerateManifest creates the GitHub App manifest JSON payload for 1-click app creation.
func (u *GitHubUsecase) GenerateManifest(ctx context.Context, baseURL string) (map[string]any, error) {
	baseURL = strings.TrimRight(baseURL, "/")
	if baseURL == "" {
		baseURL = "http://localhost:8000"
	}
	webhookURL := baseURL + "/api/v1/github/webhook"
	callbackURL := baseURL + "/api/v1/github/manifest/callback"

	manifest := map[string]any{
		"name": fmt.Sprintf("Cubit-Fleet-%d", time.Now().Unix()%10000),
		"url":  baseURL,
		"hook_attributes": map[string]any{
			"url": webhookURL,
		},
		"redirect_url":  callbackURL,
		"callback_urls": []string{callbackURL},
		"public":        false,
		"default_permissions": map[string]string{
			"contents":      "read",
			"metadata":      "read",
			"pull_requests": "read",
			"emails":        "read",
		},
		"default_events": []string{"push"},
	}

	return manifest, nil
}

// ExchangeManifestCode converts the manifest creation code from GitHub into permanent App credentials.
func (u *GitHubUsecase) ExchangeManifestCode(ctx context.Context, code string) (*domain.GitHubAppSettings, error) {
	if strings.TrimSpace(code) == "" {
		return nil, domain.NewValidationError("manifest exchange code is required")
	}

	url := fmt.Sprintf("https://api.github.com/app-manifests/%s/conversions", code)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := u.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed contacting github manifest conversion API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("github conversion error (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var conversion struct {
		ID            int64  `json:"id"`
		Name          string `json:"name"`
		ClientID      string `json:"client_id"`
		ClientSecret  string `json:"client_secret"`
		WebhookSecret string `json:"webhook_secret"`
		PEM           string `json:"pem"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&conversion); err != nil {
		return nil, fmt.Errorf("failed decoding github conversion response: %w", err)
	}

	settings := &domain.GitHubAppSettings{
		AppID:         fmt.Sprintf("%d", conversion.ID),
		AppName:       conversion.Name,
		ClientID:      conversion.ClientID,
		ClientSecret:  conversion.ClientSecret,
		WebhookSecret: conversion.WebhookSecret,
		PrivateKey:    conversion.PEM,
		IsConfigured:  true,
		UpdatedAt:     time.Now().UTC(),
	}

	if err := u.githubRepo.SaveSettings(ctx, settings); err != nil {
		return nil, err
	}

	return settings, nil
}

// GenerateAppJWT creates an RS256-signed JWT for GitHub App authentication.
func GenerateAppJWT(appID, privateKeyPEM string) (string, error) {
	block, _ := pem.Decode([]byte(privateKeyPEM))
	if block == nil {
		return "", errors.New("invalid private key PEM")
	}

	var priv *rsa.PrivateKey
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		priv = key
	} else if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		var ok bool
		priv, ok = key.(*rsa.PrivateKey)
		if !ok {
			return "", errors.New("private key is not RSA")
		}
	} else {
		return "", errors.New("failed parsing RSA private key")
	}

	now := time.Now().UTC()
	headerJSON := `{"alg":"RS256","typ":"JWT"}`
	payloadJSON := fmt.Sprintf(`{"iat":%d,"exp":%d,"iss":"%s"}`,
		now.Add(-60*time.Second).Unix(),
		now.Add(10*time.Minute).Unix(),
		appID,
	)

	b64Header := base64.RawURLEncoding.EncodeToString([]byte(headerJSON))
	b64Payload := base64.RawURLEncoding.EncodeToString([]byte(payloadJSON))
	unsignedToken := b64Header + "." + b64Payload

	h := sha256.New()
	h.Write([]byte(unsignedToken))
	sig, err := rsa.SignPKCS1v15(rand.Reader, priv, crypto.SHA256, h.Sum(nil))
	if err != nil {
		return "", fmt.Errorf("failed signing JWT: %w", err)
	}

	b64Sig := base64.RawURLEncoding.EncodeToString(sig)
	return unsignedToken + "." + b64Sig, nil
}

// GetInstallationAccessToken retrieves a GitHub App installation token.
func (u *GitHubUsecase) GetInstallationAccessToken(ctx context.Context, s *domain.GitHubAppSettings) (string, error) {
	if s.InstallationID == "" {
		// If ClientSecret or WebhookSecret is passed as a PAT or raw token
		if strings.HasPrefix(s.ClientSecret, "ghp_") || strings.HasPrefix(s.ClientSecret, "github_pat_") {
			return s.ClientSecret, nil
		}
		return "", errors.New("github installation ID not configured")
	}

	jwt, err := GenerateAppJWT(s.AppID, s.PrivateKey)
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf("https://api.github.com/app/installations/%s/access_tokens", s.InstallationID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+jwt)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := u.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed creating installation token: %s", string(b))
	}

	var res struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
	}
	return res.Token, nil
}

// ListRepositories retrieves accessible repositories from GitHub.
func (u *GitHubUsecase) ListRepositories(ctx context.Context) ([]domain.GitHubRepository, error) {
	settings, err := u.githubRepo.GetSettings(ctx)
	if err != nil {
		return nil, err
	}
	if !settings.IsConfigured {
		return []domain.GitHubRepository{}, nil
	}

	token, err := u.GetInstallationAccessToken(ctx, settings)
	if err != nil {
		return nil, err
	}

	// Try installation repositories endpoint first
	url := "https://api.github.com/installation/repositories?per_page=100"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "token "+token)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := u.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		var res struct {
			Repositories []struct {
				ID            int64  `json:"id"`
				Name          string `json:"name"`
				FullName      string `json:"full_name"`
				DefaultBranch string `json:"default_branch"`
				Private       bool   `json:"private"`
				CloneURL      string `json:"clone_url"`
				HTMLURL       string `json:"html_url"`
			} `json:"repositories"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&res); err == nil && len(res.Repositories) > 0 {
			var repos []domain.GitHubRepository
			for _, r := range res.Repositories {
				repos = append(repos, domain.GitHubRepository{
					ID:            r.ID,
					Name:          r.Name,
					FullName:      r.FullName,
					DefaultBranch: r.DefaultBranch,
					Private:       r.Private,
					CloneURL:      r.CloneURL,
					HTMLURL:       r.HTMLURL,
				})
			}
			return repos, nil
		}
	}

	// Fallback to /user/repos if using personal access token
	userReq, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user/repos?per_page=100", nil)
	userReq.Header.Set("Authorization", "token "+token)
	userReq.Header.Set("Accept", "application/vnd.github+json")
	userResp, err := u.httpClient.Do(userReq)
	if err == nil && userResp.StatusCode == http.StatusOK {
		defer userResp.Body.Close()
		var userRepos []struct {
			ID            int64  `json:"id"`
			Name          string `json:"name"`
			FullName      string `json:"full_name"`
			DefaultBranch string `json:"default_branch"`
			Private       bool   `json:"private"`
			CloneURL      string `json:"clone_url"`
			HTMLURL       string `json:"html_url"`
		}
		if err := json.NewDecoder(userResp.Body).Decode(&userRepos); err == nil {
			var repos []domain.GitHubRepository
			for _, r := range userRepos {
				repos = append(repos, domain.GitHubRepository{
					ID:            r.ID,
					Name:          r.Name,
					FullName:      r.FullName,
					DefaultBranch: r.DefaultBranch,
					Private:       r.Private,
					CloneURL:      r.CloneURL,
					HTMLURL:       r.HTMLURL,
				})
			}
			return repos, nil
		}
	}

	return []domain.GitHubRepository{}, nil
}

// ListBranches retrieves git branches for a given repository.
func (u *GitHubUsecase) ListBranches(ctx context.Context, owner, repo string) ([]string, error) {
	settings, err := u.githubRepo.GetSettings(ctx)
	if err != nil {
		return nil, err
	}
	if !settings.IsConfigured {
		return []string{"main"}, nil
	}

	token, _ := u.GetInstallationAccessToken(ctx, settings)
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/branches?per_page=100", owner, repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if token != "" {
		req.Header.Set("Authorization", "token "+token)
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := u.httpClient.Do(req)
	if err != nil {
		return []string{"main"}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return []string{"main"}, nil
	}

	var branches []struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&branches); err != nil {
		return []string{"main"}, nil
	}

	var names []string
	for _, b := range branches {
		names = append(names, b.Name)
	}
	if len(names) == 0 {
		names = append(names, "main")
	}
	return names, nil
}

// WebhookResult represents the outcome of processing a GitHub webhook.
type WebhookResult struct {
	Event               string   `json:"event"`
	MatchedApplications []string `json:"matchedApplications"`
	TriggeredDeployments []string `json:"triggeredDeployments"`
	Message             string   `json:"message"`
}

// HandlePushWebhook processes an incoming GitHub webhook.
func (u *GitHubUsecase) HandlePushWebhook(
	ctx context.Context,
	eventType string,
	signatureHeader string,
	payload []byte,
) (*WebhookResult, error) {
	settings, err := u.githubRepo.GetSettings(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed loading github settings: %w", err)
	}

	// Verify HMAC signature
	if settings.IsConfigured && settings.WebhookSecret != "" {
		if !domain.VerifyGitHubHMAC(payload, settings.WebhookSecret, signatureHeader) {
			return nil, errors.New("invalid webhook signature")
		}
	}

	// Handle ping event
	if eventType == "ping" {
		return &WebhookResult{
			Event:   "ping",
			Message: "pong",
		}, nil
	}

	if eventType != "push" {
		return &WebhookResult{
			Event:   eventType,
			Message: fmt.Sprintf("ignored event type: %s", eventType),
		}, nil
	}

	event, err := domain.ParseGitHubPushPayload(payload)
	if err != nil {
		return nil, err
	}

	targetRepo := event.Repository.FullName
	targetBranch := event.BranchName()

	// Find applications registered with this repo and branch
	matchedApps, err := u.appRepo.ListByGitRepoAndBranch(ctx, targetRepo, targetBranch)
	if err != nil {
		return nil, fmt.Errorf("failed querying applications for repo: %w", err)
	}

	// Also fallback to clone URL if needed
	if len(matchedApps) == 0 && event.Repository.CloneURL != "" {
		matchedApps, _ = u.appRepo.ListByGitRepoAndBranch(ctx, event.Repository.CloneURL, targetBranch)
	}

	commitHash := event.After
	if commitHash == "" && event.HeadCommit != nil {
		commitHash = event.HeadCommit.ID
	}
	if commitHash == "" {
		commitHash = "HEAD"
	}

	commitMsg := "Webhook push deployment"
	if event.HeadCommit != nil && event.HeadCommit.Message != "" {
		commitMsg = event.HeadCommit.Message
	}
	if event.Pusher.Name != "" {
		commitMsg = fmt.Sprintf("%s (pushed by %s)", commitMsg, event.Pusher.Name)
	}

	var appIDs []string
	var depIDs []string

	for _, app := range matchedApps {
		appIDs = append(appIDs, app.ID)
		dep, err := u.appUsecase.DeployApplicationWithDetails(ctx, app.ID, commitHash, commitMsg)
		if err == nil && dep != nil {
			depIDs = append(depIDs, dep.ID)
		}
	}

	return &WebhookResult{
		Event:                "push",
		MatchedApplications:  appIDs,
		TriggeredDeployments: depIDs,
		Message:              fmt.Sprintf("triggered %d deployment(s) for %s@%s", len(depIDs), targetRepo, targetBranch),
	}, nil
}
