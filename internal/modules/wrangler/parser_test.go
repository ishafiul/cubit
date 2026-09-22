package wrangler_test

import (
	"testing"

	"github.com/ishaf/cubit/internal/domain"
	"github.com/ishaf/cubit/internal/modules/wrangler"
)

func TestParseWranglerJSON(t *testing.T) {
	// Given
	rawJSON := []byte(`{
		"name": "my-worker",
		"main": "src/index.ts",
		"compatibility_date": "2024-09-23",
		"compatibility_flags": ["nodejs_compat"],
		"vars": {
			"API_HOST": "https://api.cubit.dev",
			"RETRY_COUNT": 3,
			"ENABLE_FEATURE": true
		},
		"kv_namespaces": [
			{ "binding": "CONFIG_KV", "id": "kv-uuid-123" }
		],
		"d1_databases": [
			{ "binding": "MAIN_DB", "database_name": "cubit-db", "database_id": "db-uuid-456" }
		],
		"r2_buckets": [
			{ "binding": "ASSETS", "bucket_name": "worker-assets" }
		],
		"services": [
			{ "binding": "AUTH_SVC", "service": "auth-service" }
		],
		"queues": {
			"producers": [
				{ "binding": "EVENT_QUEUE", "queue": "user-events" }
			]
		},
		"triggers": {
			"crons": ["*/5 * * * *"]
		}
	}`)

	// When
	cfg, format, err := wrangler.Parse(rawJSON, "auto")

	// Then
	if err != nil {
		t.Fatalf("unexpected error parsing wrangler.json: %v", err)
	}
	if format != "json" {
		t.Errorf("expected format 'json', got '%s'", format)
	}
	if cfg.Name != "my-worker" {
		t.Errorf("expected name 'my-worker', got '%s'", cfg.Name)
	}
	if cfg.Main != "src/index.ts" {
		t.Errorf("expected main 'src/index.ts', got '%s'", cfg.Main)
	}
	if cfg.CompatibilityDate != "2024-09-23" {
		t.Errorf("expected compat date '2024-09-23', got '%s'", cfg.CompatibilityDate)
	}
	if len(cfg.CompatibilityFlags) != 1 || cfg.CompatibilityFlags[0] != "nodejs_compat" {
		t.Errorf("unexpected compat flags: %v", cfg.CompatibilityFlags)
	}
	if len(cfg.Vars) != 3 {
		t.Errorf("expected 3 vars, got %d", len(cfg.Vars))
	}
	if len(cfg.KVNamespaces) != 1 || cfg.KVNamespaces[0].Binding != "CONFIG_KV" {
		t.Errorf("unexpected KV namespaces: %v", cfg.KVNamespaces)
	}
	if len(cfg.D1Databases) != 1 || cfg.D1Databases[0].DatabaseName != "cubit-db" {
		t.Errorf("unexpected D1 databases: %v", cfg.D1Databases)
	}
	if len(cfg.R2Buckets) != 1 || cfg.R2Buckets[0].BucketName != "worker-assets" {
		t.Errorf("unexpected R2 buckets: %v", cfg.R2Buckets)
	}
	if len(cfg.Services) != 1 || cfg.Services[0].Service != "auth-service" {
		t.Errorf("unexpected Services: %v", cfg.Services)
	}
	if cfg.Queues == nil || len(cfg.Queues.Producers) != 1 || cfg.Queues.Producers[0].Queue != "user-events" {
		t.Errorf("unexpected Queues: %v", cfg.Queues)
	}
	if cfg.Triggers == nil || len(cfg.Triggers.Crons) != 1 || cfg.Triggers.Crons[0] != "*/5 * * * *" {
		t.Errorf("unexpected Triggers: %v", cfg.Triggers)
	}
}

func TestParseWranglerJSONC_WithCommentsAndTrailingCommas(t *testing.T) {
	// Given
	rawJSONC := []byte(`{
		// Worker identity
		"name": "commented-worker",
		/* Entry point configuration
		   with multi-line notes */
		"main": "src/worker.js",
		"compatibility_date": "2024-05-15",
		"vars": {
			"BASE_URL": "https://example.com/api//v1", // URL containing slashes
			"DEBUG": "true",
		},
	}`)

	// When
	cfg, format, err := wrangler.Parse(rawJSONC, "auto")

	// Then
	if err != nil {
		t.Fatalf("unexpected error parsing wrangler.jsonc: %v", err)
	}
	if format != "json" {
		t.Errorf("expected format 'json', got '%s'", format)
	}
	if cfg.Name != "commented-worker" {
		t.Errorf("expected name 'commented-worker', got '%s'", cfg.Name)
	}
	if cfg.Main != "src/worker.js" {
		t.Errorf("expected main 'src/worker.js', got '%s'", cfg.Main)
	}
	if cfg.Vars["BASE_URL"] != "https://example.com/api//v1" {
		t.Errorf("URL with comment chars inside string should be preserved, got: %v", cfg.Vars["BASE_URL"])
	}
}

func TestParseWranglerTOML(t *testing.T) {
	// Given
	rawTOML := []byte(`
name = "toml-worker"
main = "src/index.js"
compatibility_date = "2024-08-01"
compatibility_flags = ["nodejs_compat", "streams_enable_constructors"]

[vars]
ENVIRONMENT = "staging"
PORT = 8080
RATE_LIMIT = 500

[[kv_namespaces]]
binding = "CACHE_KV"
id = "kv-staging-id"

[[d1_databases]]
binding = "DB"
database_name = "staging-db"

[[r2_buckets]]
binding = "STORAGE"
bucket_name = "staging-bucket"

[[services]]
binding = "INTERNAL_RPC"
service = "payment-gateway"

[triggers]
crons = ["0 0 * * *"]
`)

	// When
	cfg, format, err := wrangler.Parse(rawTOML, "auto")

	// Then
	if err != nil {
		t.Fatalf("unexpected error parsing wrangler.toml: %v", err)
	}
	if format != "toml" {
		t.Errorf("expected format 'toml', got '%s'", format)
	}
	if cfg.Name != "toml-worker" {
		t.Errorf("expected name 'toml-worker', got '%s'", cfg.Name)
	}
	if cfg.CompatibilityDate != "2024-08-01" {
		t.Errorf("expected compat date '2024-08-01', got '%s'", cfg.CompatibilityDate)
	}
	if len(cfg.CompatibilityFlags) != 2 {
		t.Errorf("expected 2 compat flags, got %d", len(cfg.CompatibilityFlags))
	}
	if len(cfg.Vars) != 3 {
		t.Errorf("expected 3 vars, got %d", len(cfg.Vars))
	}
	if len(cfg.KVNamespaces) != 1 || cfg.KVNamespaces[0].Binding != "CACHE_KV" {
		t.Errorf("unexpected KV: %v", cfg.KVNamespaces)
	}
	if len(cfg.D1Databases) != 1 || cfg.D1Databases[0].DatabaseName != "staging-db" {
		t.Errorf("unexpected D1: %v", cfg.D1Databases)
	}
}

func TestApplyToApplication_PreservesExistingSecrets(t *testing.T) {
	// Given
	app, err := domain.NewApplicationWithSource(
		"app-1",
		"my-app",
		domain.SourceTypeInline,
		"",
		"main",
		"export default { fetch() {} }",
		[]domain.EnvironmentVariable{
			{Key: "EXISTING_PUBLIC", Value: "old-val", IsSecret: false},
			{Key: "SUPER_SECRET_KEY", Value: "sensitive-token-12345", IsSecret: true},
			{Key: "MANUAL_SECRET", Value: "ui-added-secret", IsSecret: true},
		},
		[]domain.ResourceBinding{},
	)
	if err != nil {
		t.Fatalf("failed to create test app: %v", err)
	}

	cfg := &wrangler.WranglerConfig{
		CompatibilityDate:  "2024-11-15",
		CompatibilityFlags: []string{"nodejs_compat"},
		Vars: map[string]any{
			"EXISTING_PUBLIC":  "new-updated-val",
			"SUPER_SECRET_KEY": "dummy-wrangler-placeholder",
			"NEW_WRANGLER_VAR": "fresh-value",
		},
		KVNamespaces: []wrangler.KVNamespaceBinding{
			{Binding: "NEW_KV", ID: "kv-new-id"},
		},
	}

	// When
	summary, err := wrangler.ApplyToApplication(app, cfg, "", "json")

	// Then
	if err != nil {
		t.Fatalf("unexpected error applying to app: %v", err)
	}
	if summary.ImportedVarsCount != 3 {
		t.Errorf("expected 3 imported vars, got %d", summary.ImportedVarsCount)
	}
	if summary.PreservedSecretsCount != 1 {
		t.Errorf("expected 1 preserved secret count, got %d", summary.PreservedSecretsCount)
	}
	if app.CompatibilityDate != "2024-11-15" {
		t.Errorf("expected compat date 2024-11-15, got %s", app.CompatibilityDate)
	}

	// Check final environment variables
	varMap := make(map[string]domain.EnvironmentVariable)
	for _, ev := range app.EnvVars {
		varMap[ev.Key] = ev
	}

	// 1. Existing public updated
	if ev, ok := varMap["EXISTING_PUBLIC"]; !ok || ev.Value != "new-updated-val" || ev.IsSecret {
		t.Errorf("EXISTING_PUBLIC not updated properly: %+v", ev)
	}

	// 2. Secret preserved with IsSecret = true
	if ev, ok := varMap["SUPER_SECRET_KEY"]; !ok || !ev.IsSecret {
		t.Errorf("SUPER_SECRET_KEY should maintain IsSecret=true: %+v", ev)
	}

	// 3. UI-added secret not in wrangler should be fully retained
	if ev, ok := varMap["MANUAL_SECRET"]; !ok || !ev.IsSecret || ev.Value != "ui-added-secret" {
		t.Errorf("MANUAL_SECRET should remain untouched: %+v", ev)
	}

	// 4. New wrangler var added as plaintext
	if ev, ok := varMap["NEW_WRANGLER_VAR"]; !ok || ev.Value != "fresh-value" || ev.IsSecret {
		t.Errorf("NEW_WRANGLER_VAR should be plaintext: %+v", ev)
	}

	// 5. Binding added
	if len(app.Bindings) != 1 || app.Bindings[0].Name != "NEW_KV" {
		t.Errorf("expected 1 binding NEW_KV, got: %v", app.Bindings)
	}
}

func TestApplyToApplication_EnvironmentOverride(t *testing.T) {
	// Given
	app, _ := domain.NewApplicationWithSource(
		"app-env",
		"env-app",
		domain.SourceTypeGit,
		"org/repo",
		"staging",
		"",
		nil,
		nil,
	)

	cfg := &wrangler.WranglerConfig{
		CompatibilityDate: "2024-01-01",
		Vars: map[string]any{
			"BASE_URL": "https://default.example.com",
			"DEBUG":    false,
		},
		Env: map[string]wrangler.WranglerConfig{
			"staging": {
				CompatibilityDate: "2024-06-01",
				Vars: map[string]any{
					"BASE_URL": "https://staging.example.com",
				},
				KVNamespaces: []wrangler.KVNamespaceBinding{
					{Binding: "STAGE_KV", ID: "kv-stage-1"},
				},
			},
		},
	}

	// When
	summary, err := wrangler.ApplyToApplication(app, cfg, "staging", "json")

	// Then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if summary.CompatibilityDate != "2024-06-01" {
		t.Errorf("expected staging compat date, got: %s", summary.CompatibilityDate)
	}
	if app.CompatibilityDate != "2024-06-01" {
		t.Errorf("expected app compat date updated to 2024-06-01, got: %s", app.CompatibilityDate)
	}

	varMap := make(map[string]domain.EnvironmentVariable)
	for _, ev := range app.EnvVars {
		varMap[ev.Key] = ev
	}
	if varMap["BASE_URL"].Value != "https://staging.example.com" {
		t.Errorf("expected staging BASE_URL, got: %s", varMap["BASE_URL"].Value)
	}
	if varMap["DEBUG"].Value != "false" {
		t.Errorf("expected base DEBUG false preserved, got: %s", varMap["DEBUG"].Value)
	}
}

func TestParseWranglerJSON_WithAssets(t *testing.T) {
	// Given
	rawJSON := []byte(`{
		"name": "fullstack-remix-app",
		"main": "build/server/index.js",
		"assets": {
			"directory": "./build/client",
			"binding": "STATIC_ASSETS"
		}
	}`)

	// When
	cfg, _, err := wrangler.Parse(rawJSON, "auto")
	if err != nil {
		t.Fatalf("unexpected error parsing assets: %v", err)
	}

	app, _ := domain.NewApplicationWithSource("app-fullstack", "remix-app", domain.SourceTypeInline, "", "", "", nil, nil)
	summary, err := wrangler.ApplyToApplication(app, cfg, "", "json")

	// Then
	if err != nil {
		t.Fatalf("unexpected error applying assets: %v", err)
	}
	if summary.ImportedBindingsCount != 1 {
		t.Errorf("expected 1 imported binding, got %d", summary.ImportedBindingsCount)
	}

	var foundAssetBinding bool
	for _, b := range app.Bindings {
		if b.Type == domain.BindingTypeAssets && b.Name == "STATIC_ASSETS" && b.ResourceID == "./build/client" {
			foundAssetBinding = true
			break
		}
	}
	if !foundAssetBinding {
		t.Errorf("expected STATIC_ASSETS binding with ./build/client, got: %+v", app.Bindings)
	}
}

