package traefik_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ishaf/cubit/internal/adapters/out/traefik"
	"github.com/ishaf/cubit/internal/usecase"
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
		rules := []usecase.RouteRule{
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
}
