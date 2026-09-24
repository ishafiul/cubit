package traefik_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ishaf/cubit/internal/adapters/out/traefik"
)

func TestFileProvider(t *testing.T) {
	ctx := context.Background()
	tmpDir, err := os.MkdirTemp("", "traefik-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	outFile := filepath.Join(tmpDir, "cubit.yaml")
	provider := traefik.NewFileProvider(outFile, "letsencrypt")

	t.Run("Given dynamic routing rules for an application", func(t *testing.T) {
		rules := []traefik.RouteRule{
			{
				AppName:    "my-worker",
				Hostname:   "api.example.com",
				PathPrefix: "/v1",
				TargetURLs: []string{"http://10.0.0.1:8080", "http://10.0.0.2:8080"},
				EnableTLS:  true,
			},
		}

		t.Run("When synchronizing routes then Traefik configuration file is written", func(t *testing.T) {
			if err := provider.SyncRoutes(ctx, rules); err != nil {
				t.Fatalf("expected no error syncing routes, got: %v", err)
			}

			content, err := os.ReadFile(outFile)
			if err != nil {
				t.Fatalf("expected config file to exist: %v", err)
			}
			yamlStr := string(content)

			t.Run("Then Host and PathPrefix rules are present", func(t *testing.T) {
				if !strings.Contains(yamlStr, "Host(`api.example.com`)") {
					t.Errorf("expected Host rule, got:\n%s", yamlStr)
				}
				if !strings.Contains(yamlStr, "PathPrefix(`/v1`)") {
					t.Errorf("expected PathPrefix rule, got:\n%s", yamlStr)
				}
			})

			t.Run("Then TLS and CertResolver are configured", func(t *testing.T) {
				if !strings.Contains(yamlStr, "certResolver: letsencrypt") {
					t.Errorf("expected letsencrypt resolver, got:\n%s", yamlStr)
				}
			})

			t.Run("Then upstream server URLs are defined in service load balancer", func(t *testing.T) {
				if !strings.Contains(yamlStr, "http://10.0.0.1:8080") || !strings.Contains(yamlStr, "http://10.0.0.2:8080") {
					t.Errorf("expected load balancer targets, got:\n%s", yamlStr)
				}
			})
		})
	})

	t.Run("Given wildcard and multi-path routing rules", func(t *testing.T) {
		rules := []traefik.RouteRule{
			{
				AppName:    "my-worker",
				Hostname:   "*.example.com",
				PathPrefix: "/",
				TargetURLs: []string{"http://10.0.0.1:8080"},
				EnableTLS:  false,
			},
			{
				AppName:    "my-worker",
				Hostname:   "api.example.com",
				PathPrefix: "/v1",
				TargetURLs: []string{"http://10.0.0.1:8080"},
				EnableTLS:  false,
			},
			{
				AppName:    "my-worker",
				Hostname:   "api.example.com",
				PathPrefix: "/v2",
				TargetURLs: []string{"http://10.0.0.1:8080"},
				EnableTLS:  false,
			},
		}

		t.Run("When routes are synchronized", func(t *testing.T) {
			if err := provider.SyncRoutes(ctx, rules); err != nil {
				t.Fatalf("expected no error syncing routes, got: %v", err)
			}

			content, err := os.ReadFile(outFile)
			if err != nil {
				t.Fatalf("expected config file to exist: %v", err)
			}
			yamlStr := string(content)

			t.Run("Then wildcard host rule is generated", func(t *testing.T) {
				if !strings.Contains(yamlStr, "Host(`*.example.com`)") {
					t.Errorf("expected wildcard Host rule, got:\n%s", yamlStr)
				}
			})

			t.Run("Then router names do not contain raw asterisks", func(t *testing.T) {
				if strings.Contains(yamlStr, "rt-my-worker-*-") {
					t.Errorf("router name must not contain raw asterisks, got:\n%s", yamlStr)
				}
			})

			t.Run("Then both path prefixes exist without collision", func(t *testing.T) {
				if !strings.Contains(yamlStr, "PathPrefix(`/v1`)") || !strings.Contains(yamlStr, "PathPrefix(`/v2`)") {
					t.Errorf("expected both /v1 and /v2 path prefixes preserved without collision, got:\n%s", yamlStr)
				}
			})
		})
	})
}
