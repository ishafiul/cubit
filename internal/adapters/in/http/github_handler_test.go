package http_test

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	http_adapter "github.com/ishaf/cubit/internal/adapters/in/http"
	"github.com/ishaf/cubit/internal/domain"
	"github.com/ishaf/cubit/internal/usecase"
)

type mockGitHubRepoForHandler struct {
	settings *domain.GitHubAppSettings
}

func (m *mockGitHubRepoForHandler) GetSettings(ctx context.Context) (*domain.GitHubAppSettings, error) {
	if m.settings == nil {
		return &domain.GitHubAppSettings{IsConfigured: false}, nil
	}
	return m.settings, nil
}

func (m *mockGitHubRepoForHandler) SaveSettings(ctx context.Context, s *domain.GitHubAppSettings) error {
	m.settings = s
	return nil
}

func (m *mockGitHubRepoForHandler) ClearSettings(ctx context.Context) error {
	m.settings = nil
	return nil
}

type mockAppRepoForGHHandler struct {
	apps []*domain.Application
}

func (m *mockAppRepoForGHHandler) Save(ctx context.Context, app *domain.Application) error {
	m.apps = append(m.apps, app)
	return nil
}

func (m *mockAppRepoForGHHandler) GetByID(ctx context.Context, id string) (*domain.Application, error) {
	for _, a := range m.apps {
		if a.ID == id {
			return a, nil
		}
	}
	return nil, domain.NewNotFoundError("not found")
}

func (m *mockAppRepoForGHHandler) GetBySubdomain(ctx context.Context, subdomain string) (*domain.Application, error) {
	for _, a := range m.apps {
		if a.Subdomain == subdomain {
			return a, nil
		}
	}
	return nil, domain.NewNotFoundError("not found")
}

func (m *mockAppRepoForGHHandler) List(ctx context.Context) ([]*domain.Application, error) {
	return m.apps, nil
}

func (m *mockAppRepoForGHHandler) Delete(ctx context.Context, id string) error {
	return nil
}

func (m *mockAppRepoForGHHandler) Update(ctx context.Context, app *domain.Application) error {
	for i, a := range m.apps {
		if a.ID == app.ID {
			m.apps[i] = app
			return nil
		}
	}
	return nil
}

func (m *mockAppRepoForGHHandler) ListByGitRepoAndBranch(ctx context.Context, repo, branch string) ([]*domain.Application, error) {
	var matched []*domain.Application
	for _, a := range m.apps {
		if a.SourceType == domain.SourceTypeGit && a.GitRepo == repo && a.Branch == branch && a.AutoDeploy {
			matched = append(matched, a)
		}
	}
	return matched, nil
}

type mockDepRepoForGHHandler struct {
	deps []*domain.Deployment
	logs map[string][]domain.DeploymentLog
}

func (m *mockDepRepoForGHHandler) Save(ctx context.Context, dep *domain.Deployment) error {
	m.deps = append(m.deps, dep)
	return nil
}

func (m *mockDepRepoForGHHandler) GetByID(ctx context.Context, id string) (*domain.Deployment, error) {
	for _, d := range m.deps {
		if d.ID == id {
			return d, nil
		}
	}
	return nil, domain.NewNotFoundError("not found")
}

func (m *mockDepRepoForGHHandler) GetLatestBuildVersion(ctx context.Context, appID string) (int, error) {
	max := 0
	for _, d := range m.deps {
		if d.ApplicationID == appID && d.BuildVersion > max {
			max = d.BuildVersion
		}
	}
	return max, nil
}

func (m *mockDepRepoForGHHandler) ListByAppID(ctx context.Context, appID string) ([]*domain.Deployment, error) {
	return m.deps, nil
}

func (m *mockDepRepoForGHHandler) Update(ctx context.Context, dep *domain.Deployment) error {
	for i, d := range m.deps {
		if d.ID == dep.ID {
			m.deps[i] = dep
			return nil
		}
	}
	return nil
}

func (m *mockDepRepoForGHHandler) AppendLog(ctx context.Context, depID string, entry domain.DeploymentLog) error {
	return nil
}

func (m *mockDepRepoForGHHandler) GetLogs(ctx context.Context, depID string) ([]domain.DeploymentLog, error) {
	return nil, nil
}

func TestGitHubHandler(t *testing.T) {
	webhookSecret := "test-secret-456"
	ghRepo := &mockGitHubRepoForHandler{
		settings: &domain.GitHubAppSettings{
			AppID:         "98765",
			AppName:       "cubit-ci-app",
			WebhookSecret: webhookSecret,
			IsConfigured:  true,
			UpdatedAt:     time.Now().UTC(),
		},
	}

	app, _ := domain.NewApplicationWithSource(
		"app-hook-1",
		"backend-api",
		domain.SourceTypeGit,
		"org/backend-api",
		"main",
		"",
		nil,
		nil,
	)
	app.AutoDeploy = true

	appRepo := &mockAppRepoForGHHandler{apps: []*domain.Application{app}}
	depRepo := &mockDepRepoForGHHandler{}

	appUc := usecase.NewAppUsecase(appRepo, depRepo, nil, nil, nil, nil, "cubit-bundles")
	ghUc := usecase.NewGitHubUsecase(ghRepo, appRepo, appUc)
	ghHandler := http_adapter.NewGitHubHandler(ghUc)

	r := chi.NewRouter()
	r.Route("/api/v1", func(sub chi.Router) {
		ghHandler.RegisterRoutes(sub)
	})

	t.Run("Given a configured GitHub Handler", func(t *testing.T) {
		t.Run("When requesting GET /api/v1/github/settings", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/github/settings", nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			t.Run("Then it returns status 200 with masked credentials", func(t *testing.T) {
				if w.Code != http.StatusOK {
					t.Fatalf("expected 200 OK, got %d", w.Code)
				}
				var res map[string]any
				if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
					t.Fatalf("failed decoding response: %v", err)
				}
				if res["appName"] != "cubit-ci-app" {
					t.Errorf("expected appName cubit-ci-app, got %v", res["appName"])
				}
				if res["isConfigured"] != true {
					t.Errorf("expected isConfigured true, got %v", res["isConfigured"])
				}
				if _, ok := res["webhookSecret"]; ok {
					t.Errorf("webhookSecret should be masked from response")
				}
			})
		})

		t.Run("When sending a push webhook with valid signature", func(t *testing.T) {
			payload := []byte(`{
				"ref": "refs/heads/main",
				"after": "1234567890abcdef1234567890abcdef12345678",
				"repository": {
					"id": 111,
					"name": "backend-api",
					"full_name": "org/backend-api",
					"clone_url": "https://github.com/org/backend-api.git"
				},
				"head_commit": {
					"id": "1234567890abcdef1234567890abcdef12345678",
					"message": "fix: update db connection pool",
					"author": { "name": "Bob" }
				},
				"pusher": { "name": "Bob" }
			}`)

			mac := hmac.New(sha256.New, []byte(webhookSecret))
			mac.Write(payload)
			sig := "sha256=" + hex.EncodeToString(mac.Sum(nil))

			req := httptest.NewRequest(http.MethodPost, "/api/v1/github/webhook", bytes.NewReader(payload))
			req.Header.Set("X-GitHub-Event", "push")
			req.Header.Set("X-Hub-Signature-256", sig)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			t.Run("Then it returns status 200 and reports triggered deployment", func(t *testing.T) {
				if w.Code != http.StatusOK {
					t.Fatalf("expected 200 OK, got %d (body: %s)", w.Code, w.Body.String())
				}
				var res usecase.WebhookResult
				if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
					t.Fatalf("failed decoding webhook result: %v", err)
				}
				if len(res.MatchedApplications) != 1 || res.MatchedApplications[0] != "app-hook-1" {
					t.Errorf("expected matched application 'app-hook-1', got %v", res.MatchedApplications)
				}
				if len(res.TriggeredDeployments) != 1 {
					t.Errorf("expected 1 triggered deployment, got %v", res.TriggeredDeployments)
				}
			})
		})

		t.Run("When sending a push webhook with invalid HMAC signature", func(t *testing.T) {
			payload := []byte(`{"ref":"refs/heads/main"}`)
			badSig := "sha256=invalidhex00000000000000000000000000000000000000000000000000000000"

			req := httptest.NewRequest(http.MethodPost, "/api/v1/github/webhook", bytes.NewReader(payload))
			req.Header.Set("X-GitHub-Event", "push")
			req.Header.Set("X-Hub-Signature-256", badSig)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			t.Run("Then it returns 401 Unauthorized", func(t *testing.T) {
				if w.Code != http.StatusUnauthorized {
					t.Fatalf("expected 401 Unauthorized, got %d", w.Code)
				}
			})
		})

		t.Run("When sending a ping webhook event", func(t *testing.T) {
			payload := []byte(`{"zen":"Practicality beats purity."}`)
			mac := hmac.New(sha256.New, []byte(webhookSecret))
			mac.Write(payload)
			sig := "sha256=" + hex.EncodeToString(mac.Sum(nil))

			req := httptest.NewRequest(http.MethodPost, "/api/v1/github/webhook", bytes.NewReader(payload))
			req.Header.Set("X-GitHub-Event", "ping")
			req.Header.Set("X-Hub-Signature-256", sig)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			t.Run("Then it returns 200 OK with pong", func(t *testing.T) {
				if w.Code != http.StatusOK {
					t.Fatalf("expected 200 OK, got %d", w.Code)
				}
				var res usecase.WebhookResult
				_ = json.Unmarshal(w.Body.Bytes(), &res)
				if res.Message != "pong" {
					t.Errorf("expected pong, got %s", res.Message)
				}
			})
		})
	})
}
