package github_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ishaf/cubit/internal/domain"
	"github.com/ishaf/cubit/internal/infrastructure/db"
	ghModule "github.com/ishaf/cubit/internal/modules/github"
)

type mockAppFinder struct {
	apps []*domain.Application
}

func (m *mockAppFinder) ListByGitRepoAndBranch(ctx context.Context, repo, branch string) ([]*domain.Application, error) {
	var matched []*domain.Application
	for _, a := range m.apps {
		if a.GitRepo == repo && a.Branch == branch {
			matched = append(matched, a)
		}
	}
	return matched, nil
}

type mockDeployTrigger struct {
	deployedApps []string
}

func (m *mockDeployTrigger) DeployWithDetails(ctx context.Context, appID, commitHash, commitMessage string) (*domain.Deployment, error) {
	m.deployedApps = append(m.deployedApps, appID)
	return domain.NewDeploymentWithVersion(
		fmt.Sprintf("dep-%d", len(m.deployedApps)),
		appID,
		commitHash,
		commitMessage,
		len(m.deployedApps),
	)
}

func TestGitHubService(t *testing.T) {
	ctx := context.Background()
	tempDir, err := os.MkdirTemp("", "cubit-gh-test-*")
	if err != nil {
		t.Fatalf("failed creating temp dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(tempDir) })

	dbPath := filepath.Join(tempDir, "test.db")
	database, err := db.OpenSQLite(dbPath)
	if err != nil {
		t.Fatalf("failed opening sqlite: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	repo := ghModule.NewRepository(database)
	finder := &mockAppFinder{
		apps: []*domain.Application{
			{
				ID:         "app-webhook-1",
				Name:       "web-api",
				GitRepo:    "octocat/Hello-World",
				Branch:     "main",
				AutoDeploy: true,
			},
		},
	}
	trigger := &mockDeployTrigger{}
	svc := ghModule.NewService(repo, finder, trigger)

	t.Run("Given unconfigured settings", func(t *testing.T) {
		t.Run("When getting initial settings", func(t *testing.T) {
			s, err := svc.GetSettings(ctx)
			if err != nil {
				t.Fatalf("failed getting settings: %v", err)
			}

			t.Run("Then settings are reported as not configured", func(t *testing.T) {
				if s.IsConfigured {
					t.Errorf("expected IsConfigured to be false")
				}
			})
		})

		t.Run("When saving valid GitHub App credentials", func(t *testing.T) {
			err := svc.SaveSettings(ctx, &domain.GitHubAppSettings{
				AppID:         "12345",
				AppName:       "MyCubitApp",
				ClientID:      "Iv1.abc",
				ClientSecret:  "secret-token",
				WebhookSecret: "super-secret-hmac",
				IsConfigured:  true,
				UpdatedAt:     time.Now().UTC(),
			})
			if err != nil {
				t.Fatalf("failed saving settings: %v", err)
			}

			t.Run("Then settings are persisted and configured", func(t *testing.T) {
				s, err := svc.GetSettings(ctx)
				if err != nil {
					t.Fatalf("failed getting settings: %v", err)
				}
				if !s.IsConfigured {
					t.Errorf("expected IsConfigured true")
				}
				if s.AppName != "MyCubitApp" {
					t.Errorf("expected MyCubitApp, got %s", s.AppName)
				}
			})
		})
	})

	t.Run("Given a configured webhook secret and matching application", func(t *testing.T) {
		payload := []byte(`{
			"ref": "refs/heads/main",
			"after": "6dcb09b5b57875f334f61aebed695e2e4193db5e",
			"repository": {
				"name": "Hello-World",
				"full_name": "octocat/Hello-World",
				"clone_url": "https://github.com/octocat/Hello-World.git"
			},
			"head_commit": {
				"id": "6dcb09b5b57875f334f61aebed695e2e4193db5e",
				"message": "Deploy via webhook"
			},
			"pusher": {
				"name": "octocat"
			}
		}`)

		mac := hmac.New(sha256.New, []byte("super-secret-hmac"))
		mac.Write(payload)
		sig := "sha256=" + hex.EncodeToString(mac.Sum(nil))

		t.Run("When a valid GitHub push webhook is received", func(t *testing.T) {
			res, err := svc.HandlePushWebhook(ctx, "push", sig, payload)
			if err != nil {
				t.Fatalf("unexpected webhook error: %v", err)
			}

			t.Run("Then matching application triggers a deployment", func(t *testing.T) {
				if len(res.MatchedApplications) != 1 || res.MatchedApplications[0] != "app-webhook-1" {
					t.Errorf("expected matched app app-webhook-1, got %v", res.MatchedApplications)
				}
				if len(trigger.deployedApps) != 1 || trigger.deployedApps[0] != "app-webhook-1" {
					t.Errorf("expected triggered deploy for app-webhook-1, got %v", trigger.deployedApps)
				}
			})
		})
	})
}
