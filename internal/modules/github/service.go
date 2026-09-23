package github

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
	"net"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ishaf/cubit/internal/domain"
)

// ApplicationFinder locates applications configured for specific git repositories and branches.
type ApplicationFinder interface {
	ListByGitRepoAndBranch(ctx context.Context, repo, branch string) ([]*domain.Application, error)
}

// DeploymentTrigger initiates application deployments.
type DeploymentTrigger interface {
	DeployWithDetails(ctx context.Context, appID, commitHash, commitMessage string) (*domain.Deployment, error)
}

// Service handles GitHub App setup, manifest exchange, repository sync, and webhook processing.
type Service struct {
	repo       Repository
	appFinder  ApplicationFinder
	deployer   DeploymentTrigger
	httpClient *http.Client
}

// NewService creates a new GitHub Service.
func NewService(
	repo Repository,
	appFinder ApplicationFinder,
	deployer DeploymentTrigger,
) *Service {
	return &Service{
		repo:       repo,
		appFinder:  appFinder,
		deployer:   deployer,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// GetSettings returns current GitHub App configuration.
func (s *Service) GetSettings(ctx context.Context) (*domain.GitHubAppSettings, error) {
	return s.repo.GetSettings(ctx)
}

// SaveSettings stores or updates GitHub App configuration.
func (s *Service) SaveSettings(ctx context.Context, settings *domain.GitHubAppSettings) error {
	hasCredentials := strings.TrimSpace(settings.AppID) != "" ||
		strings.TrimSpace(settings.PrivateKey) != "" ||
		strings.TrimSpace(settings.ClientSecret) != ""

	settings.IsConfigured = hasCredentials
	return s.repo.SaveSettings(ctx, settings)
}

// ClearSettings resets the GitHub App configuration.
func (s *Service) ClearSettings(ctx context.Context) error {
	return s.repo.ClearSettings(ctx)
}

func isLocalhostOrPrivate(rawURL string) bool {
	if rawURL == "" {
		return true
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		lower := strings.ToLower(rawURL)
		return strings.Contains(lower, "localhost") || strings.Contains(lower, "127.0.0.1")
	}
	host := u.Hostname()
	if host == "" {
		host = rawURL
	}
	host = strings.ToLower(host)
	if host == "localhost" || host == "127.0.0.1" || host == "::1" || strings.HasSuffix(host, ".localhost") || strings.HasSuffix(host, ".local") {
		return true
	}
	ip := net.ParseIP(host)
	if ip != nil {
		return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast()
	}
	return false
}

// GenerateManifest creates the GitHub App manifest JSON payload for 1-click app creation.
func (s *Service) GenerateManifest(ctx context.Context, baseURL string, webhookURL string) (map[string]any, error) {
	baseURL = strings.TrimRight(baseURL, "/")
	if baseURL == "" {
		baseURL = "http://localhost:8000"
	}
	callbackURL := baseURL + "/api/v1/github/manifest/callback"

	manifest := map[string]any{
		"name":          fmt.Sprintf("Cubit-Fleet-%d", time.Now().Unix()%10000),
		"url":           baseURL,
		"redirect_url":  callbackURL,
		"callback_urls": []string{callbackURL},
		"setup_url":     callbackURL,
		"public":        false,
		"default_permissions": map[string]string{
			"contents":      "read",
			"metadata":      "read",
			"pull_requests": "read",
			"emails":        "read",
		},
	}

	webhookURL = strings.TrimSpace(webhookURL)
	if webhookURL == "" && !isLocalhostOrPrivate(baseURL) {
		webhookURL = baseURL + "/api/v1/github/webhook"
	}

	// GitHub requires hook_attributes.url to be publicly reachable.
	// For localhost / private network addresses, hook_attributes must be omitted so registration succeeds.
	if webhookURL != "" && !isLocalhostOrPrivate(webhookURL) {
		manifest["hook_attributes"] = map[string]any{
			"url": webhookURL,
		}
		manifest["default_events"] = []string{"push"}
	}

	return manifest, nil
}

// ExchangeManifestCode converts the manifest creation code from GitHub into permanent App credentials.
func (s *Service) ExchangeManifestCode(ctx context.Context, code string) (*domain.GitHubAppSettings, error) {
	if strings.TrimSpace(code) == "" {
		return nil, domain.NewValidationError("manifest exchange code is required")
	}

	url := fmt.Sprintf("https://api.github.com/app-manifests/%s/conversions", code)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := s.httpClient.Do(req)
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
		Slug          string `json:"slug"`
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
		AppSlug:       conversion.Slug,
		ClientID:      conversion.ClientID,
		ClientSecret:  conversion.ClientSecret,
		WebhookSecret: conversion.WebhookSecret,
		PrivateKey:    conversion.PEM,
		IsConfigured:  true,
		UpdatedAt:     time.Now().UTC(),
	}

	if err := s.repo.SaveSettings(ctx, settings); err != nil {
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

// DiscoverInstallations queries GitHub for all installations of this GitHub App using JWT authentication.
// If settings.InstallationID is empty and installations exist, it auto-configures the first installation ID.
func (s *Service) DiscoverInstallations(ctx context.Context) ([]domain.GitHubInstallation, error) {
	settings, err := s.repo.GetSettings(ctx)
	if err != nil {
		return nil, err
	}
	if !settings.IsConfigured || settings.AppID == "" || settings.PrivateKey == "" {
		return []domain.GitHubInstallation{}, nil
	}

	jwt, err := GenerateAppJWT(settings.AppID, settings.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed generating app jwt: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/app/installations?per_page=100", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+jwt)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed contacting github installations API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("github error fetching installations (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var rawList []struct {
		ID      int64  `json:"id"`
		HTMLURL string `json:"html_url"`
		Account struct {
			Login     string `json:"login"`
			Type      string `json:"type"`
			AvatarURL string `json:"avatar_url"`
		} `json:"account"`
		TargetID   int64  `json:"target_id"`
		TargetType string `json:"target_type"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&rawList); err != nil {
		return nil, fmt.Errorf("failed decoding installations response: %w", err)
	}

	var installations []domain.GitHubInstallation
	for _, raw := range rawList {
		installations = append(installations, domain.GitHubInstallation{
			ID:            raw.ID,
			AccountLogin:  raw.Account.Login,
			AccountType:   raw.Account.Type,
			AccountAvatar: raw.Account.AvatarURL,
			HTMLURL:       raw.HTMLURL,
			TargetID:      raw.TargetID,
			TargetType:    raw.TargetType,
		})
	}

	if settings.InstallationID == "" && len(installations) > 0 {
		settings.InstallationID = strconv.FormatInt(installations[0].ID, 10)
		_ = s.repo.SaveSettings(ctx, settings)
	}

	return installations, nil
}

// GetInstallationAccessToken retrieves a GitHub App installation token.
func (s *Service) GetInstallationAccessToken(ctx context.Context, appSettings *domain.GitHubAppSettings) (string, error) {
	if appSettings.InstallationID == "" {
		if strings.HasPrefix(appSettings.ClientSecret, "ghp_") || strings.HasPrefix(appSettings.ClientSecret, "github_pat_") {
			return appSettings.ClientSecret, nil
		}

		// Auto-discover installations if not yet bound
		installs, err := s.DiscoverInstallations(ctx)
		if err == nil && len(installs) > 0 {
			appSettings.InstallationID = strconv.FormatInt(installs[0].ID, 10)
		} else {
			slug := appSettings.Slug()
			return "", fmt.Errorf("GitHub App is not yet installed on any account or repository. Please install it at https://github.com/apps/%s/installations/new", slug)
		}
	}

	jwt, err := GenerateAppJWT(appSettings.AppID, appSettings.PrivateKey)
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf("https://api.github.com/app/installations/%s/access_tokens", appSettings.InstallationID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+jwt)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := s.httpClient.Do(req)
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
func (s *Service) ListRepositories(ctx context.Context) ([]domain.GitHubRepository, error) {
	settings, err := s.repo.GetSettings(ctx)
	if err != nil {
		return nil, err
	}
	if !settings.IsConfigured {
		return []domain.GitHubRepository{}, nil
	}

	token, err := s.GetInstallationAccessToken(ctx, settings)
	if err != nil {
		return []domain.GitHubRepository{}, nil
	}

	var repos []domain.GitHubRepository
	page := 1

	for {
		url := fmt.Sprintf("https://api.github.com/installation/repositories?per_page=100&page=%d", page)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "token "+token)
		req.Header.Set("Accept", "application/vnd.github+json")

		resp, err := s.httpClient.Do(req)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			break
		}

		var res struct {
			TotalCount   int `json:"total_count"`
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

		decodeErr := json.NewDecoder(resp.Body).Decode(&res)
		resp.Body.Close()
		if decodeErr != nil || len(res.Repositories) == 0 {
			break
		}

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

		if len(repos) >= res.TotalCount || len(res.Repositories) < 100 {
			break
		}
		page++
	}

	if len(repos) > 0 {
		sort.Slice(repos, func(i, j int) bool {
			return strings.ToLower(repos[i].FullName) < strings.ToLower(repos[j].FullName)
		})
		return repos, nil
	}

	userReq, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user/repos?per_page=100", nil)
	userReq.Header.Set("Authorization", "token "+token)
	userReq.Header.Set("Accept", "application/vnd.github+json")
	userResp, err := s.httpClient.Do(userReq)
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
func (s *Service) ListBranches(ctx context.Context, owner, repo string) ([]string, error) {
	settings, err := s.repo.GetSettings(ctx)
	if err != nil {
		return nil, err
	}
	if !settings.IsConfigured {
		return []string{"main"}, nil
	}

	token, _ := s.GetInstallationAccessToken(ctx, settings)
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/branches?per_page=100", owner, repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if token != "" {
		req.Header.Set("Authorization", "token "+token)
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := s.httpClient.Do(req)
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

func isIgnoredGitPath(p string) bool {
	parts := strings.Split(p, "/")
	for _, part := range parts {
		if part == "" {
			continue
		}
		if strings.HasPrefix(part, ".") ||
			part == "node_modules" ||
			part == "dist" ||
			part == "build" ||
			part == "vendor" ||
			part == "coverage" ||
			part == "target" ||
			part == ".dart_tool" {
			return true
		}
	}
	return false
}

// ListFolders retrieves selectable directories in a repository branch and detects wrangler/package.json.
func (s *Service) ListFolders(ctx context.Context, owner, repo, branch string) ([]domain.RepositoryFolder, error) {
	fallback := []domain.RepositoryFolder{
		{
			Path:           "",
			Name:           "Root (/)",
			HasWrangler:    false,
			HasPackageJSON: false,
		},
	}

	settings, err := s.repo.GetSettings(ctx)
	if err != nil {
		return fallback, nil
	}

	token := ""
	if settings.IsConfigured {
		token, _ = s.GetInstallationAccessToken(ctx, settings)
	}

	if branch == "" {
		branch = "main"
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/git/trees/%s?recursive=1", owner, repo, url.PathEscape(branch))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fallback, nil
	}
	if token != "" {
		req.Header.Set("Authorization", "token "+token)
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fallback, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fallback, nil
	}

	var res struct {
		SHA       string `json:"sha"`
		Truncated bool   `json:"truncated"`
		Tree      []struct {
			Path string `json:"path"`
			Mode string `json:"mode"`
			Type string `json:"type"`
			SHA  string `json:"sha"`
		} `json:"tree"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return fallback, nil
	}

	dirMap := make(map[string]*domain.RepositoryFolder)
	dirMap[""] = &domain.RepositoryFolder{
		Path:           "",
		Name:           "Root (/)",
		HasWrangler:    false,
		HasPackageJSON: false,
	}

	for _, item := range res.Tree {
		if isIgnoredGitPath(item.Path) {
			continue
		}

		if item.Type == "tree" {
			cleanDir := domain.CleanRootDir(item.Path)
			if cleanDir != "" {
				if _, exists := dirMap[cleanDir]; !exists {
					dirMap[cleanDir] = &domain.RepositoryFolder{
						Path: cleanDir,
						Name: cleanDir,
					}
				}
			}
		} else if item.Type == "blob" {
			cleanPath := strings.Trim(item.Path, "/")
			lastSlash := strings.LastIndex(cleanPath, "/")
			var parentDir string
			var fileName string
			if lastSlash == -1 {
				parentDir = ""
				fileName = cleanPath
			} else {
				parentDir = domain.CleanRootDir(cleanPath[:lastSlash])
				fileName = cleanPath[lastSlash+1:]
			}

			if _, exists := dirMap[parentDir]; !exists {
				dirMap[parentDir] = &domain.RepositoryFolder{
					Path: parentDir,
					Name: parentDir,
				}
			}

			lower := strings.ToLower(fileName)
			if lower == "wrangler.json" || lower == "wrangler.jsonc" || lower == "wrangler.toml" {
				dirMap[parentDir].HasWrangler = true
			} else if lower == "package.json" {
				dirMap[parentDir].HasPackageJSON = true
			}
		}
	}

	var list []domain.RepositoryFolder
	for path, folder := range dirMap {
		if path == "" {
			continue
		}
		depth := strings.Count(path, "/") + 1
		if folder.HasWrangler || folder.HasPackageJSON || depth <= 3 {
			list = append(list, *folder)
		}
	}

	sort.Slice(list, func(i, j int) bool {
		if list[i].HasWrangler != list[j].HasWrangler {
			return list[i].HasWrangler
		}
		if list[i].HasPackageJSON != list[j].HasPackageJSON {
			return list[i].HasPackageJSON
		}
		return list[i].Path < list[j].Path
	})

	finalList := make([]domain.RepositoryFolder, 0, len(list)+1)
	finalList = append(finalList, *dirMap[""])
	finalList = append(finalList, list...)

	return finalList, nil
}

// WebhookResult represents the outcome of processing a GitHub webhook.
type WebhookResult struct {
	Event                string   `json:"event"`
	MatchedApplications  []string `json:"matchedApplications"`
	TriggeredDeployments []string `json:"triggeredDeployments"`
	Message              string   `json:"message"`
}

// HandlePushWebhook processes an incoming GitHub webhook.
func (s *Service) HandlePushWebhook(
	ctx context.Context,
	eventType string,
	signatureHeader string,
	payload []byte,
) (*WebhookResult, error) {
	settings, err := s.repo.GetSettings(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed loading github settings: %w", err)
	}

	if settings.IsConfigured && settings.WebhookSecret != "" {
		if !domain.VerifyGitHubHMAC(payload, settings.WebhookSecret, signatureHeader) {
			return nil, errors.New("invalid webhook signature")
		}
	}

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

	matchedApps, err := s.appFinder.ListByGitRepoAndBranch(ctx, targetRepo, targetBranch)
	if err != nil {
		return nil, fmt.Errorf("failed querying applications for repo: %w", err)
	}

	if len(matchedApps) == 0 && event.Repository.CloneURL != "" {
		matchedApps, _ = s.appFinder.ListByGitRepoAndBranch(ctx, event.Repository.CloneURL, targetBranch)
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
		dep, err := s.deployer.DeployWithDetails(ctx, app.ID, commitHash, commitMsg)
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
