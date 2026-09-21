package usecase_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"

	"github.com/ishaf/cubit/internal/domain"
	"github.com/ishaf/cubit/internal/usecase"
)

type mockGitHubRepo struct {
	settings *domain.GitHubAppSettings
}

func (m *mockGitHubRepo) GetSettings(ctx context.Context) (*domain.GitHubAppSettings, error) {
	if m.settings == nil {
		return &domain.GitHubAppSettings{IsConfigured: false}, nil
	}
	return m.settings, nil
}

func (m *mockGitHubRepo) SaveSettings(ctx context.Context, s *domain.GitHubAppSettings) error {
	m.settings = s
	return nil
}

func (m *mockGitHubRepo) ClearSettings(ctx context.Context) error {
	m.settings = nil
	return nil
}

type mockAppRepoWithGit struct {
	apps []*domain.Application
}

func (m *mockAppRepoWithGit) Save(ctx context.Context, app *domain.Application) error {
	m.apps = append(m.apps, app)
	return nil
}

func (m *mockAppRepoWithGit) GetByID(ctx context.Context, id string) (*domain.Application, error) {
	for _, a := range m.apps {
		if a.ID == id {
			return a, nil
		}
	}
	return nil, domain.NewNotFoundError("not found")
}

func (m *mockAppRepoWithGit) GetBySubdomain(ctx context.Context, subdomain string) (*domain.Application, error) {
	for _, a := range m.apps {
		if a.Subdomain == subdomain {
			return a, nil
		}
	}
	return nil, domain.NewNotFoundError("not found")
}

func (m *mockAppRepoWithGit) List(ctx context.Context) ([]*domain.Application, error) {
	return m.apps, nil
}

func (m *mockAppRepoWithGit) Delete(ctx context.Context, id string) error {
	return nil
}

func (m *mockAppRepoWithGit) Update(ctx context.Context, app *domain.Application) error {
	for i, a := range m.apps {
		if a.ID == app.ID {
			m.apps[i] = app
			return nil
		}
	}
	return nil
}

func (m *mockAppRepoWithGit) ListByGitRepoAndBranch(ctx context.Context, repo, branch string) ([]*domain.Application, error) {
	var matched []*domain.Application
	for _, a := range m.apps {
		if a.SourceType == domain.SourceTypeGit && a.GitRepo == repo && a.Branch == branch && a.AutoDeploy {
			matched = append(matched, a)
		}
	}
	return matched, nil
}

type mockDepRepoSimple struct {
	deps []*domain.Deployment
	logs map[string][]domain.DeploymentLog
}

func (m *mockDepRepoSimple) Save(ctx context.Context, dep *domain.Deployment) error {
	m.deps = append(m.deps, dep)
	return nil
}

func (m *mockDepRepoSimple) GetByID(ctx context.Context, id string) (*domain.Deployment, error) {
	for _, d := range m.deps {
		if d.ID == id {
			return d, nil
		}
	}
	return nil, domain.NewNotFoundError("not found")
}

func (m *mockDepRepoSimple) GetLatestBuildVersion(ctx context.Context, appID string) (int, error) {
	max := 0
	for _, d := range m.deps {
		if d.ApplicationID == appID && d.BuildVersion > max {
			max = d.BuildVersion
		}
	}
	return max, nil
}

func (m *mockDepRepoSimple) ListByAppID(ctx context.Context, appID string) ([]*domain.Deployment, error) {
	var res []*domain.Deployment
	for _, d := range m.deps {
		if d.ApplicationID == appID {
			res = append(res, d)
		}
	}
	return res, nil
}

func (m *mockDepRepoSimple) Update(ctx context.Context, dep *domain.Deployment) error {
	for i, d := range m.deps {
		if d.ID == dep.ID {
			m.deps[i] = dep
			return nil
		}
	}
	return nil
}

func (m *mockDepRepoSimple) AppendLog(ctx context.Context, depID string, entry domain.DeploymentLog) error {
	if m.logs == nil {
		m.logs = make(map[string][]domain.DeploymentLog)
	}
	m.logs[depID] = append(m.logs[depID], entry)
	return nil
}

func (m *mockDepRepoSimple) GetLogs(ctx context.Context, depID string) ([]domain.DeploymentLog, error) {
	return m.logs[depID], nil
}

func TestGitHubUsecase(t *testing.T) {
	ctx := context.Background()

	t.Run("Given a configured GitHub App with a webhook secret", func(t *testing.T) {
		webhookSecret := "gh-secret-xyz"
		ghRepo := &mockGitHubRepo{
			settings: &domain.GitHubAppSettings{
				AppID:         "12345",
				AppName:       "cubit-app",
				WebhookSecret: webhookSecret,
				IsConfigured:  true,
				UpdatedAt:     time.Now().UTC(),
			},
		}

		app, _ := domain.NewApplicationWithSource(
			"app-git-1",
			"my-api-worker",
			domain.SourceTypeGit,
			"ishaf/my-api-worker",
			"main",
			"",
			nil,
			nil,
		)
		app.AutoDeploy = true

		appRepo := &mockAppRepoWithGit{apps: []*domain.Application{app}}
		depRepo := &mockDepRepoSimple{}

		appUc := usecase.NewAppUsecase(appRepo, depRepo, nil, nil, nil, nil, "cubit-bundles")
		ghUc := usecase.NewGitHubUsecase(ghRepo, appRepo, appUc)

		t.Run("When receiving a GitHub push webhook with valid signature", func(t *testing.T) {
			payload := []byte(`{
				"ref": "refs/heads/main",
				"after": "9f8a7b6c5d4e3f2a1b0c9d8e7f6a5b4c3d2e1f0a",
				"repository": {
					"id": 888,
					"name": "my-api-worker",
					"full_name": "ishaf/my-api-worker",
					"clone_url": "https://github.com/ishaf/my-api-worker.git"
				},
				"head_commit": {
					"id": "9f8a7b6c5d4e3f2a1b0c9d8e7f6a5b4c3d2e1f0a",
					"message": "fix: update router endpoint",
					"author": { "name": "Alice" }
				},
				"pusher": { "name": "Alice" }
			}`)

			mac := hmac.New(sha256.New, []byte(webhookSecret))
			mac.Write(payload)
			sig := "sha256=" + hex.EncodeToString(mac.Sum(nil))

			result, err := ghUc.HandlePushWebhook(ctx, "push", sig, payload)

			t.Run("Then it successfully triggers automated deployment for the application", func(t *testing.T) {
				if err != nil {
					t.Fatalf("unexpected webhook error: %v", err)
				}
				if len(result.MatchedApplications) != 1 || result.MatchedApplications[0] != "app-git-1" {
					t.Fatalf("expected 1 matched application 'app-git-1', got %v", result.MatchedApplications)
				}
				if len(result.TriggeredDeployments) != 1 {
					t.Fatalf("expected 1 triggered deployment, got %d", len(result.TriggeredDeployments))
				}

				// Check deployment commit hash and message
				depID := result.TriggeredDeployments[0]
				dep, err := depRepo.GetByID(ctx, depID)
				if err != nil {
					t.Fatalf("failed retrieving triggered deployment: %v", err)
				}
				if dep.CommitHash != "9f8a7b6c5d4e3f2a1b0c9d8e7f6a5b4c3d2e1f0a" {
					t.Errorf("expected commit hash from push, got %s", dep.CommitHash)
				}
				if dep.CommitMessage != "fix: update router endpoint (pushed by Alice)" {
					t.Errorf("expected commit message with author, got %s", dep.CommitMessage)
				}
				if dep.Status != domain.DeploymentStatusActive {
					t.Errorf("expected deployment status to be active, got %s", dep.Status)
				}
			})
		})

		t.Run("When receiving a GitHub push webhook with invalid HMAC signature", func(t *testing.T) {
			payload := []byte(`{"ref":"refs/heads/main"}`)
			badSig := "sha256=0000000000000000000000000000000000000000000000000000000000000000"

			_, err := ghUc.HandlePushWebhook(ctx, "push", badSig, payload)

			t.Run("Then it rejects the webhook with an authorization error", func(t *testing.T) {
				if err == nil {
					t.Fatalf("expected error for invalid HMAC signature, got nil")
				}
			})
		})

		t.Run("When receiving a ping event", func(t *testing.T) {
			payload := []byte(`{"zen":"Design for failure."}`)
			mac := hmac.New(sha256.New, []byte(webhookSecret))
			mac.Write(payload)
			sig := "sha256=" + hex.EncodeToString(mac.Sum(nil))

			res, err := ghUc.HandlePushWebhook(ctx, "ping", sig, payload)

			t.Run("Then it responds with pong message", func(t *testing.T) {
				if err != nil {
					t.Fatalf("unexpected ping error: %v", err)
				}
				if res.Message != "pong" {
					t.Errorf("expected pong message, got %s", res.Message)
				}
			})
		})

		t.Run("When generating a GitHub App manifest", func(t *testing.T) {
			manifest, err := ghUc.GenerateManifest(ctx, "https://cubit.example.com")

			t.Run("Then it returns a valid manifest configured with webhook URL and push events", func(t *testing.T) {
				if err != nil {
					t.Fatalf("unexpected error generating manifest: %v", err)
				}
				if manifest["url"] != "https://cubit.example.com" {
					t.Errorf("expected url https://cubit.example.com, got %v", manifest["url"])
				}
				hookAttrs, ok := manifest["hook_attributes"].(map[string]any)
				if !ok || hookAttrs["url"] != "https://cubit.example.com/api/v1/github/webhook" {
					t.Errorf("expected webhook url in hook_attributes, got %v", hookAttrs)
				}
			})
		})
	})
}
