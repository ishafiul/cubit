package github_test

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ishaf/cubit/internal/domain"
	"github.com/ishaf/cubit/internal/infrastructure/db"
	ghModule "github.com/ishaf/cubit/internal/modules/github"
)

func setupTestGitHubGin(t *testing.T) (*gin.Engine, *ghModule.Service, *mockDeployTrigger) {
	gin.SetMode(gin.TestMode)
	tempDir, err := os.MkdirTemp("", "cubit-gh-handler-test-*")
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
				ID:         "app-gh-test",
				Name:       "gh-app",
				GitRepo:    "org/repo",
				Branch:     "main",
				AutoDeploy: true,
			},
		},
	}
	trigger := &mockDeployTrigger{}
	svc := ghModule.NewService(repo, finder, trigger)
	handler := ghModule.NewHandler(svc)

	r := gin.New()
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)

	return r, svc, trigger
}

func TestGitHubHandler(t *testing.T) {
	router, svc, trigger := setupTestGitHubGin(t)

	t.Run("Given GitHub settings endpoints", func(t *testing.T) {
		t.Run("When fetching initial settings", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/github/settings", nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			t.Run("Then returns 200 OK with unconfigured state", func(t *testing.T) {
				if rec.Code != http.StatusOK {
					t.Fatalf("expected 200 OK, got %d", rec.Code)
				}
				var s map[string]any
				_ = json.NewDecoder(rec.Body).Decode(&s)
				if s["isConfigured"] != false {
					t.Errorf("expected isConfigured false, got %v", s["isConfigured"])
				}
			})
		})

		t.Run("When saving GitHub settings via POST", func(t *testing.T) {
			body := []byte(`{
				"appId": "999",
				"appName": "TestApp",
				"webhookSecret": "wh-secret-123"
			}`)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/github/settings", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			t.Run("Then returns 200 OK with configured state", func(t *testing.T) {
				if rec.Code != http.StatusOK {
					t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
				}
				var s map[string]any
				_ = json.NewDecoder(rec.Body).Decode(&s)
				if s["appName"] != "TestApp" {
					t.Errorf("expected TestApp, got %v", s["appName"])
				}
				if s["isConfigured"] != true {
					t.Errorf("expected isConfigured true, got %v", s["isConfigured"])
				}
			})
		})
	})

	t.Run("Given Manifest generation endpoint", func(t *testing.T) {
		t.Run("When requesting manifest JSON", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/github/manifest?baseUrl=https://fleet.example.com", nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			t.Run("Then returns 200 OK with valid manifest structure and webhook attributes", func(t *testing.T) {
				if rec.Code != http.StatusOK {
					t.Fatalf("expected 200 OK, got %d", rec.Code)
				}
				var manifest map[string]any
				_ = json.NewDecoder(rec.Body).Decode(&manifest)
				if manifest["url"] != "https://fleet.example.com" {
					t.Errorf("expected manifest url https://fleet.example.com, got %v", manifest["url"])
				}
				if manifest["hook_attributes"] == nil {
					t.Errorf("expected hook_attributes for public baseURL")
				}
			})
		})

		t.Run("When requesting manifest for localhost without webhook", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/github/manifest?baseUrl=http://localhost:8000", nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			t.Run("Then omits hook_attributes to satisfy GitHub public Internet requirement", func(t *testing.T) {
				if rec.Code != http.StatusOK {
					t.Fatalf("expected 200 OK, got %d", rec.Code)
				}
				var manifest map[string]any
				_ = json.NewDecoder(rec.Body).Decode(&manifest)
				if manifest["hook_attributes"] != nil {
					t.Errorf("expected hook_attributes to be omitted for localhost, got %v", manifest["hook_attributes"])
				}
				if manifest["default_events"] != nil {
					t.Errorf("expected default_events to be omitted when hook_attributes is absent, got %v", manifest["default_events"])
				}
			})
		})

		t.Run("When requesting manifest for localhost with public tunnel webhook", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/github/manifest?baseUrl=http://localhost:8000&webhookUrl=https://tunnel.trycloudflare.com/webhook", nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			t.Run("Then includes hook_attributes pointing to public tunnel", func(t *testing.T) {
				if rec.Code != http.StatusOK {
					t.Fatalf("expected 200 OK, got %d", rec.Code)
				}
				var manifest map[string]any
				_ = json.NewDecoder(rec.Body).Decode(&manifest)
				hook, ok := manifest["hook_attributes"].(map[string]any)
				if !ok || hook["url"] != "https://tunnel.trycloudflare.com/webhook" {
					t.Errorf("expected tunnel webhook url, got %v", manifest["hook_attributes"])
				}
			})
		})

		t.Run("When handling manifest callback redirect", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/github/manifest/callback", nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			t.Run("Then redirects to /github UI", func(t *testing.T) {
				if rec.Code != http.StatusTemporaryRedirect {
					t.Fatalf("expected 307 redirect, got %d", rec.Code)
				}
				if rec.Header().Get("Location") != "/github" {
					t.Errorf("expected redirect to /github, got %s", rec.Header().Get("Location"))
				}
			})
		})
	})

	t.Run("Given Webhook endpoint with configured secret", func(t *testing.T) {
		// Ensure secret is configured
		_ = svc.SaveSettings(context.Background(), &domain.GitHubAppSettings{
			AppID:         "999",
			AppName:       "TestApp",
			WebhookSecret: "secret-key",
			IsConfigured:  true,
			UpdatedAt:     time.Now().UTC(),
		})

		payload := []byte(`{
			"ref": "refs/heads/main",
			"after": "abc12345",
			"repository": {
				"name": "repo",
				"full_name": "org/repo"
			}
		}`)

		mac := hmac.New(sha256.New, []byte("secret-key"))
		mac.Write(payload)
		sig := "sha256=" + hex.EncodeToString(mac.Sum(nil))

		t.Run("When push webhook is delivered with valid signature", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/github/webhook", bytes.NewReader(payload))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-GitHub-Event", "push")
			req.Header.Set("X-Hub-Signature-256", sig)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			t.Run("Then returns 200 OK and triggers deployment", func(t *testing.T) {
				if rec.Code != http.StatusOK {
					t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
				}
				if len(trigger.deployedApps) != 1 || trigger.deployedApps[0] != "app-gh-test" {
					t.Errorf("expected deploy for app-gh-test, got %v", trigger.deployedApps)
				}
			})
		})

		t.Run("When push webhook is delivered with INVALID signature", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/github/webhook", bytes.NewReader(payload))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-GitHub-Event", "push")
			req.Header.Set("X-Hub-Signature-256", "sha256=invalidhash")
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			t.Run("Then returns 401 Unauthorized", func(t *testing.T) {
				if rec.Code != http.StatusUnauthorized {
					t.Errorf("expected 401 Unauthorized, got %d", rec.Code)
				}
			})
		})
	})
}
