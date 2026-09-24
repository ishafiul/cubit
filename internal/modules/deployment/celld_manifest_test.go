package deployment_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ishaf/cubit/internal/domain"
	"github.com/ishaf/cubit/internal/modules/deployment"
	"github.com/ishaf/cubit/internal/modules/wrangler"
)

func TestComputeSHA256(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected string
	}{
		{
			name:     "empty content",
			input:    []byte(""),
			expected: "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		},
		{
			name:     "sample js bundle",
			input:    []byte("console.log('hello world');"),
			expected: "sha256:fdbfc7bd27940b250bd17955d592b49f00e4d7c3b606933949c8fe759d707886",
		},
	}

	for _, tc := range tests {
		t.Run("Given "+tc.name, func(t *testing.T) {
			digest := deployment.ComputeSHA256(tc.input)
			if digest != tc.expected {
				t.Errorf("expected %s, got %s", tc.expected, digest)
			}
		})
	}
}

func TestBuildCelldDeployment(t *testing.T) {
	t.Run("Given an application and deployment", func(t *testing.T) {
		app := &domain.Application{
			ID:         "app-export-1",
			Name:       "worker-export",
			InlineCode: "export default { fetch() { return new Response('exported'); } };",
		}
		dep, err := domain.NewDeploymentWithVersion("dep-export-1", app.ID, "hash", "msg", 1)
		if err != nil {
			t.Fatalf("failed to create deployment: %v", err)
		}

		t.Run("When calling BuildCelldDeployment", func(t *testing.T) {
			data, err := deployment.BuildCelldDeployment(context.Background(), app, dep)

			t.Run("Then returns valid JSON manifest with schema version and script name", func(t *testing.T) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				var m deployment.CelldManifest
				if err := json.Unmarshal(data, &m); err != nil {
					t.Fatalf("failed to parse JSON manifest: %v", err)
				}
				if m.SchemaVersion != 1 {
					t.Errorf("expected schema version 1, got %d", m.SchemaVersion)
				}
				if m.ScriptName != "worker-export" {
					t.Errorf("expected script_name worker-export, got %s", m.ScriptName)
				}
			})
		})
	})
}

func TestBuildCelldManifest(t *testing.T) {
	t.Run("Given an application with DO bindings, migrations, crons and worker bundle", func(t *testing.T) {
		app := &domain.Application{
			ID:         "app-test-1",
			Name:       "worker-chat",
			SourceType: domain.SourceTypeInline,
			Bindings: []domain.ResourceBinding{
				{
					Type:      domain.BindingTypeDurableObject,
					Name:      "CHAT_ROOM",
					ClassName: "ChatRoom",
				},
				{
					Type:       domain.BindingTypeD1,
					Name:       "DB",
					ResourceID: "db-123",
				},
				{
					Type:       domain.BindingTypeKV,
					Name:       "CACHE",
					ResourceID: "kv-123",
				},
				{
					Type:       domain.BindingTypeR2,
					Name:       "BUCKET",
					ResourceID: "bucket-123",
				},
			},
			Migrations: []domain.MigrationStep{
				{
					Tag:        "v1",
					NewClasses: []string{"ChatRoom", "UserSession"},
				},
			},
			CompatibilityDate: "2024-11-20",
		}

		dep, err := domain.NewDeploymentWithVersion("dep-1", app.ID, "commit123", "deploy feat", 1)
		if err != nil {
			t.Fatalf("failed to create deployment: %v", err)
		}

		bundle := []byte("export class ChatRoom {}; export default { fetch() { return new Response('ok'); } };")

		cfg := &wrangler.WranglerConfig{
			Name: "worker-chat",
			Triggers: &wrangler.TriggerConfig{
				Crons: []string{"0 * * * *", "*/15 * * * *"},
			},
			Assets: &wrangler.AssetsConfig{
				Directory: "./dist",
				Binding:   "ASSETS",
			},
		}

		t.Run("When building celld deployment manifest", func(t *testing.T) {
			manifest, err := deployment.BuildCelldManifest(app, dep, bundle, cfg)

			t.Run("Then manifest contains correct schema version, script name, digest, DO classes and crons", func(t *testing.T) {
				if err != nil {
					t.Fatalf("expected no error, got: %v", err)
				}
				if manifest.SchemaVersion != 1 {
					t.Errorf("expected schema version 1, got %d", manifest.SchemaVersion)
				}
				if manifest.Version != dep.ID {
					t.Errorf("expected version %s, got %s", dep.ID, manifest.Version)
				}
				if manifest.ScriptName != "worker-chat" {
					t.Errorf("expected script_name worker-chat, got %s", manifest.ScriptName)
				}
				if manifest.MainModule != "worker.js" {
					t.Errorf("expected main_module worker.js, got %s", manifest.MainModule)
				}
				if len(manifest.Modules) != 1 {
					t.Fatalf("expected 1 module, got %d", len(manifest.Modules))
				}
				mod := manifest.Modules[0]
				if mod.Name != "worker.js" || mod.Type != "esm" {
					t.Errorf("unexpected module spec: %+v", mod)
				}
				if mod.Size != int64(len(bundle)) {
					t.Errorf("expected module size %d, got %d", len(bundle), mod.Size)
				}
				if !strings.HasPrefix(mod.Digest, "sha256:") {
					t.Errorf("expected sha256 prefix on digest, got %s", mod.Digest)
				}

				// Check DO and SQLite classes
				expectedDO := map[string]bool{"ChatRoom": true, "UserSession": true}
				for _, c := range manifest.DOClasses {
					delete(expectedDO, c)
				}
				if len(expectedDO) != 0 {
					t.Errorf("missing expected DO classes: %v", expectedDO)
				}

				// Check crons
				if len(manifest.Crons) != 2 || manifest.Crons[0] != "0 * * * *" {
					t.Errorf("unexpected crons: %v", manifest.Crons)
				}

				// Check assets
				if manifest.Assets == nil || manifest.Assets.Directory != "./dist" || manifest.Assets.Binding != "ASSETS" {
					t.Errorf("unexpected assets configuration: %+v", manifest.Assets)
				}

				// Check required features
				featMap := make(map[string]bool)
				for _, f := range manifest.RequiredFeatures {
					featMap[f] = true
				}
				if !featMap["d1-v1"] || !featMap["kv-v1"] || !featMap["r2-v1"] || !featMap["durable-objects"] {
					t.Errorf("missing expected required features: %v", manifest.RequiredFeatures)
				}
			})
		})
	})

	t.Run("Given nil application or deployment", func(t *testing.T) {
		t.Run("When building manifest with nil app", func(t *testing.T) {
			_, err := deployment.BuildCelldManifest(nil, nil, []byte("code"), nil)
			t.Run("Then returns validation error", func(t *testing.T) {
				if err == nil {
					t.Fatalf("expected validation error, got nil")
				}
			})
		})
	})
}

func TestFormatCurrentPointer(t *testing.T) {
	t.Run("Given a valid CelldManifest and paths", func(t *testing.T) {
		manifest := &deployment.CelldManifest{
			SchemaVersion: 1,
			Version:       "dep-42",
			ScriptName:    "my-worker",
			MainModule:    "worker.js",
			Modules: []deployment.CelldModule{
				{
					Name:   "worker.js",
					Type:   "esm",
					Digest: "sha256:abcdef123456",
					Size:   100,
				},
			},
		}

		manifestPath := "deployments/my-worker/dep-42/manifest.json"
		bundlePath := "deployments/my-worker/dep-42/bundle.js"

		t.Run("When formatting current pointer JSON", func(t *testing.T) {
			data, err := deployment.FormatCurrentPointer(manifest, manifestPath, bundlePath)

			t.Run("Then generates valid JSON with version, script_name, manifest_path and bundle_path", func(t *testing.T) {
				if err != nil {
					t.Fatalf("expected no error, got: %v", err)
				}

				var ptr deployment.CelldCurrentPointer
				if err := json.Unmarshal(data, &ptr); err != nil {
					t.Fatalf("failed to unmarshal pointer JSON: %v", err)
				}

				if ptr.Version != "dep-42" {
					t.Errorf("expected version dep-42, got %s", ptr.Version)
				}
				if ptr.ScriptName != "my-worker" {
					t.Errorf("expected script_name my-worker, got %s", ptr.ScriptName)
				}
				if ptr.ManifestPath != manifestPath {
					t.Errorf("expected manifest_path %s, got %s", manifestPath, ptr.ManifestPath)
				}
				if ptr.BundlePath != bundlePath {
					t.Errorf("expected bundle_path %s, got %s", bundlePath, ptr.BundlePath)
				}
				if ptr.Digest != "sha256:abcdef123456" {
					t.Errorf("expected digest sha256:abcdef123456, got %s", ptr.Digest)
				}
			})
		})
	})
}

func TestTriggerCelldReload(t *testing.T) {
	t.Run("Given a healthy celld daemon internal listener returning 200 OK", func(t *testing.T) {
		receivedMethod := ""
		receivedPath := ""

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			receivedMethod = r.Method
			receivedPath = r.URL.Path
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ok","reloaded_version":"dep-123"}`))
		}))
		defer server.Close()

		t.Run("When TriggerCelldReloadURL is invoked", func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			err := deployment.TriggerCelldReloadURL(ctx, server.Client(), server.URL)

			t.Run("Then sends POST /reload and returns nil error", func(t *testing.T) {
				if err != nil {
					t.Fatalf("expected no error, got: %v", err)
				}
				if receivedMethod != http.MethodPost {
					t.Errorf("expected POST method, got %s", receivedMethod)
				}
				if receivedPath != "/reload" {
					t.Errorf("expected /reload path, got %s", receivedPath)
				}
			})
		})

		t.Run("When TriggerCelldReload is invoked with internalPort", func(t *testing.T) {
			parts := strings.Split(strings.TrimPrefix(server.URL, "http://"), ":")
			if len(parts) > 1 {
				var port int
				fmt.Sscanf(parts[1], "%d", &port)
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				defer cancel()
				_ = deployment.TriggerCelldReload(ctx, port)
			}
		})
	})

	t.Run("Given celld daemon returning 500 Internal Server Error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`failed to compile isolate bundle`))
		}))
		defer server.Close()

		t.Run("When TriggerCelldReloadURL is invoked", func(t *testing.T) {
			err := deployment.TriggerCelldReloadURL(context.Background(), server.Client(), server.URL+"/reload")

			t.Run("Then returns error containing status code and error body", func(t *testing.T) {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if !strings.Contains(err.Error(), "500") || !strings.Contains(err.Error(), "failed to compile isolate bundle") {
					t.Errorf("expected status 500 and failure message in error, got: %v", err)
				}
			})
		})
	})

	t.Run("Given empty target URL", func(t *testing.T) {
		t.Run("When TriggerCelldReloadURL is invoked with empty URL", func(t *testing.T) {
			err := deployment.TriggerCelldReloadURL(context.Background(), nil, "")

			t.Run("Then returns validation error", func(t *testing.T) {
				if err == nil {
					t.Fatalf("expected error for empty target URL")
				}
			})
		})
	})
}
