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

func TestParseWrangler_DurableObjectsAndMigrations_JSON(t *testing.T) {
	t.Run("Given wrangler.json with durable_objects and migrations", func(t *testing.T) {
		rawJSON := []byte(`{
			"name": "do-worker",
			"main": "src/index.ts",
			"durable_objects": {
				"bindings": [
					{
						"name": "MY_COUNTER",
						"class_name": "CounterDO"
					},
					{
						"binding": "REMOTE_DO",
						"class_name": "SharedStateDO",
						"script_name": "auth-service",
						"environment": "production"
					}
				]
			},
			"migrations": [
				{
					"tag": "v1",
					"new_classes": ["CounterDO"]
				},
				{
					"tag": "v2",
					"renamed_classes": [
						{ "from": "CounterDO", "to": "AdvancedCounterDO" }
					],
					"deleted_classes": ["DeprecatedDO"],
					"steps": [
						{
							"new_classes": ["SharedStateDO"]
						}
					]
				}
			]
		}`)

		t.Run("When parsed and applied to application", func(t *testing.T) {
			cfg, format, err := wrangler.Parse(rawJSON, "auto")
			if err != nil {
				t.Fatalf("unexpected parse error: %v", err)
			}
			if format != "json" {
				t.Errorf("expected format 'json', got '%s'", format)
			}

			app, _ := domain.NewApplicationWithSource("app-do", "do-worker", domain.SourceTypeInline, "", "", "", nil, nil)
			summary, err := wrangler.ApplyToApplication(app, cfg, "", "json")

			t.Run("Then Durable Object bindings are populated on the application", func(t *testing.T) {
				if err != nil {
					t.Fatalf("unexpected apply error: %v", err)
				}
				if summary.ImportedDurableObjectsCount != 2 {
					t.Errorf("expected 2 imported DOs, got %d", summary.ImportedDurableObjectsCount)
				}
				if summary.ImportedBindingsCount != 2 {
					t.Errorf("expected 2 total imported bindings, got %d", summary.ImportedBindingsCount)
				}

				doMap := make(map[string]domain.ResourceBinding)
				for _, b := range app.Bindings {
					if b.Type == domain.BindingTypeDurableObject {
						doMap[b.Name] = b
					}
				}

				if len(doMap) != 2 {
					t.Fatalf("expected 2 DO bindings in app.Bindings, got %d", len(doMap))
				}

				counter, ok := doMap["MY_COUNTER"]
				if !ok {
					t.Fatalf("missing MY_COUNTER binding: %+v", app.Bindings)
				}
				if counter.ClassName != "CounterDO" || counter.ResourceID != "CounterDO" {
					t.Errorf("unexpected counter DO: %+v", counter)
				}

				remote, ok := doMap["REMOTE_DO"]
				if !ok {
					t.Fatalf("missing REMOTE_DO binding: %+v", app.Bindings)
				}
				if remote.ClassName != "SharedStateDO" || remote.ScriptName != "auth-service" || remote.Environment != "production" {
					t.Errorf("unexpected remote DO: %+v", remote)
				}
			})

			t.Run("Then migration history is recorded on the application", func(t *testing.T) {
				if summary.ImportedMigrationsCount != 2 {
					t.Errorf("expected 2 imported migrations, got %d", summary.ImportedMigrationsCount)
				}
				if len(app.Migrations) != 2 {
					t.Fatalf("expected 2 migrations in app.Migrations, got %d", len(app.Migrations))
				}

				v1 := app.Migrations[0]
				if v1.Tag != "v1" || len(v1.NewClasses) != 1 || v1.NewClasses[0] != "CounterDO" {
					t.Errorf("unexpected migration v1: %+v", v1)
				}

				v2 := app.Migrations[1]
				if v2.Tag != "v2" {
					t.Errorf("expected tag v2, got %s", v2.Tag)
				}
				if len(v2.RenamedClasses) != 1 || v2.RenamedClasses[0].From != "CounterDO" || v2.RenamedClasses[0].To != "AdvancedCounterDO" {
					t.Errorf("unexpected renamed classes in v2: %+v", v2.RenamedClasses)
				}
				if len(v2.DeletedClasses) != 1 || v2.DeletedClasses[0] != "DeprecatedDO" {
					t.Errorf("unexpected deleted classes in v2: %+v", v2.DeletedClasses)
				}
				if len(v2.NewClasses) != 1 || v2.NewClasses[0] != "SharedStateDO" {
					t.Errorf("unexpected multi-step new classes in v2: %+v", v2.NewClasses)
				}
			})
		})
	})
}

func TestParseWrangler_DurableObjectsAndMigrations_TOML(t *testing.T) {
	t.Run("Given wrangler.toml with durable_objects and migrations", func(t *testing.T) {
		rawTOML := []byte(`
name = "toml-do-worker"
main = "src/index.js"

[[durable_objects.bindings]]
name = "SESSION_DO"
class_name = "SessionDO"

[[migrations]]
tag = "v1"
new_classes = ["SessionDO"]
`)

		t.Run("When parsed and applied to application", func(t *testing.T) {
			cfg, format, err := wrangler.Parse(rawTOML, "auto")
			if err != nil {
				t.Fatalf("unexpected error parsing TOML: %v", err)
			}
			if format != "toml" {
				t.Errorf("expected format 'toml', got '%s'", format)
			}

			app, _ := domain.NewApplicationWithSource("app-toml-do", "toml-do-worker", domain.SourceTypeInline, "", "", "", nil, nil)
			summary, err := wrangler.ApplyToApplication(app, cfg, "", "toml")

			t.Run("Then Durable Object binding and migration step are created", func(t *testing.T) {
				if err != nil {
					t.Fatalf("unexpected apply error: %v", err)
				}
				if summary.ImportedDurableObjectsCount != 1 {
					t.Errorf("expected 1 imported DO, got %d", summary.ImportedDurableObjectsCount)
				}
				if len(app.Bindings) != 1 || app.Bindings[0].Type != domain.BindingTypeDurableObject || app.Bindings[0].Name != "SESSION_DO" {
					t.Errorf("unexpected DO binding: %+v", app.Bindings)
				}
				if len(app.Migrations) != 1 || app.Migrations[0].Tag != "v1" || app.Migrations[0].NewClasses[0] != "SessionDO" {
					t.Errorf("unexpected migration: %+v", app.Migrations)
				}
			})
		})
	})
}

func TestApplyToApplication_MigrationsIdempotence(t *testing.T) {
	t.Run("Given an application with existing migration history", func(t *testing.T) {
		app, _ := domain.NewApplicationWithSource("app-mig-idemp", "mig-app", domain.SourceTypeInline, "", "", "", nil, nil)
		app.Migrations = []domain.MigrationStep{
			{
				Tag:        "v1",
				NewClasses: []string{"OldCounterDO"},
			},
		}

		cfg := &wrangler.WranglerConfig{
			Migrations: []wrangler.MigrationConfig{
				{
					Tag:        "v1",
					NewClasses: []string{"UpdatedCounterDO"},
				},
				{
					Tag:        "v2",
					NewClasses: []string{"NewClassDO"},
				},
			},
		}

		t.Run("When applying configuration", func(t *testing.T) {
			summary, err := wrangler.ApplyToApplication(app, cfg, "", "json")

			t.Run("Then existing migration tag is updated without duplicating and new tag is appended", func(t *testing.T) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if summary.ImportedMigrationsCount != 2 {
					t.Errorf("expected 2 imported migrations, got %d", summary.ImportedMigrationsCount)
				}
				if len(app.Migrations) != 2 {
					t.Fatalf("expected total 2 migrations, got %d: %+v", len(app.Migrations), app.Migrations)
				}
				if app.Migrations[0].Tag != "v1" || app.Migrations[0].NewClasses[0] != "UpdatedCounterDO" {
					t.Errorf("expected v1 to be updated: %+v", app.Migrations[0])
				}
				if app.Migrations[1].Tag != "v2" || app.Migrations[1].NewClasses[0] != "NewClassDO" {
					t.Errorf("expected v2 to be appended: %+v", app.Migrations[1])
				}
			})
		})
	})
}

func TestApplyToApplication_DurableObjectsEnvironmentOverride(t *testing.T) {
	t.Run("Given wrangler config with environment override for durable_objects", func(t *testing.T) {
		app, _ := domain.NewApplicationWithSource("app-env-do", "env-do-app", domain.SourceTypeInline, "", "", "", nil, nil)

		cfg := &wrangler.WranglerConfig{
			DurableObjects: &wrangler.DurableObjectsConfig{
				Bindings: []wrangler.DurableObjectBinding{
					{Name: "DO_STORE", ClassName: "LocalStore"},
				},
			},
			Env: map[string]wrangler.WranglerConfig{
				"production": {
					DurableObjects: &wrangler.DurableObjectsConfig{
						Bindings: []wrangler.DurableObjectBinding{
							{Name: "DO_STORE", ClassName: "ProductionStore"},
						},
					},
				},
			},
		}

		t.Run("When applying with production environment override", func(t *testing.T) {
			summary, err := wrangler.ApplyToApplication(app, cfg, "production", "json")

			t.Run("Then production Durable Object binding overrides base", func(t *testing.T) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if summary.ImportedDurableObjectsCount != 1 {
					t.Errorf("expected 1 imported DO, got %d", summary.ImportedDurableObjectsCount)
				}
				if len(app.Bindings) != 1 {
					t.Fatalf("expected 1 binding, got %d", len(app.Bindings))
				}
				if app.Bindings[0].ClassName != "ProductionStore" {
					t.Errorf("expected ProductionStore, got %s", app.Bindings[0].ClassName)
				}
			})
		})
	})
}

func TestParseWrangler_WorkflowsAndContainers_JSON(t *testing.T) {
	t.Run("Given wrangler.json with workflows and containers", func(t *testing.T) {
		rawJSON := []byte(`{
			"name": "workflow-app",
			"workflows": [
				{
					"name": "signup-workflow",
					"binding": "SIGNUP_WF",
					"class_name": "SignupWorkflow"
				}
			],
			"containers": [
				{
					"name": "redis-sidecar",
					"image": "redis:7-alpine",
					"port": 6379,
					"env_vars": {
						"CACHE_SIZE": "512mb"
					}
				}
			]
		}`)

		t.Run("When parsed and applied to application", func(t *testing.T) {
			cfg, format, err := wrangler.Parse(rawJSON, "json")
			if err != nil {
				t.Fatalf("unexpected parse error: %v", err)
			}
			if format != "json" {
				t.Errorf("expected format 'json', got '%s'", format)
			}

			app, _ := domain.NewApplicationWithSource("app-wf-ct", "wf-ct-app", domain.SourceTypeInline, "", "", "", nil, nil)
			summary, err := wrangler.ApplyToApplication(app, cfg, "", "json")

			t.Run("Then workflow and container bindings are properly created on application", func(t *testing.T) {
				if err != nil {
					t.Fatalf("unexpected apply error: %v", err)
				}
				if summary.ImportedWorkflowsCount != 1 {
					t.Errorf("expected 1 imported workflow, got %d", summary.ImportedWorkflowsCount)
				}
				if summary.ImportedContainersCount != 1 {
					t.Errorf("expected 1 imported container, got %d", summary.ImportedContainersCount)
				}
				if len(app.Bindings) != 2 {
					t.Fatalf("expected 2 bindings, got %d: %+v", len(app.Bindings), app.Bindings)
				}

				var wfBinding, ctBinding *domain.ResourceBinding
				for i := range app.Bindings {
					if app.Bindings[i].Type == domain.BindingTypeWorkflow {
						wfBinding = &app.Bindings[i]
					}
					if app.Bindings[i].Type == domain.BindingTypeContainer {
						ctBinding = &app.Bindings[i]
					}
				}

				if wfBinding == nil {
					t.Fatal("expected workflow binding to exist")
				}
				if wfBinding.Name != "SIGNUP_WF" {
					t.Errorf("expected workflow name 'SIGNUP_WF', got '%s'", wfBinding.Name)
				}
				if wfBinding.ClassName != "SignupWorkflow" {
					t.Errorf("expected workflow ClassName 'SignupWorkflow', got '%s'", wfBinding.ClassName)
				}
				if wfBinding.WorkflowName != "signup-workflow" {
					t.Errorf("expected workflow WorkflowName 'signup-workflow', got '%s'", wfBinding.WorkflowName)
				}
				if wfBinding.ResourceID != "signup-workflow" {
					t.Errorf("expected workflow ResourceID 'signup-workflow', got '%s'", wfBinding.ResourceID)
				}

				if ctBinding == nil {
					t.Fatal("expected container binding to exist")
				}
				if ctBinding.Name != "redis-sidecar" {
					t.Errorf("expected container name 'redis-sidecar', got '%s'", ctBinding.Name)
				}
				if ctBinding.ContainerName != "redis-sidecar" {
					t.Errorf("expected container ContainerName 'redis-sidecar', got '%s'", ctBinding.ContainerName)
				}
				if ctBinding.Image != "redis:7-alpine" {
					t.Errorf("expected container Image 'redis:7-alpine', got '%s'", ctBinding.Image)
				}
				if ctBinding.Port != 6379 {
					t.Errorf("expected container Port 6379, got %d", ctBinding.Port)
				}
				if ctBinding.ResourceID != "redis:7-alpine" {
					t.Errorf("expected container ResourceID 'redis:7-alpine', got '%s'", ctBinding.ResourceID)
				}
				if ctBinding.ContainerEnvVars == nil || ctBinding.ContainerEnvVars["CACHE_SIZE"] != "512mb" {
					t.Errorf("expected container env var CACHE_SIZE=512mb, got %+v", ctBinding.ContainerEnvVars)
				}
			})
		})
	})
}

func TestParseWrangler_WorkflowsAndContainers_TOML(t *testing.T) {
	t.Run("Given wrangler.toml with [[workflows]] and [[containers]]", func(t *testing.T) {
		rawTOML := []byte(`
name = "toml-wf-ct"
main = "src/index.js"

[[workflows]]
name = "order-processing"
binding = "ORDER_WORKFLOW"
class_name = "OrderWorkflow"

[[containers]]
name = "db-sidecar"
image = "postgres:16-alpine"
port = 5432
`)

		t.Run("When parsed and applied to application", func(t *testing.T) {
			cfg, format, err := wrangler.Parse(rawTOML, "auto")
			if err != nil {
				t.Fatalf("unexpected parse error: %v", err)
			}
			if format != "toml" {
				t.Errorf("expected format 'toml', got '%s'", format)
			}

			app, _ := domain.NewApplicationWithSource("app-toml-wf", "toml-wf-app", domain.SourceTypeInline, "", "", "", nil, nil)
			summary, err := wrangler.ApplyToApplication(app, cfg, "", "toml")

			t.Run("Then workflow and container bindings are populated from TOML", func(t *testing.T) {
				if err != nil {
					t.Fatalf("unexpected apply error: %v", err)
				}
				if summary.ImportedWorkflowsCount != 1 {
					t.Errorf("expected 1 imported workflow, got %d", summary.ImportedWorkflowsCount)
				}
				if summary.ImportedContainersCount != 1 {
					t.Errorf("expected 1 imported container, got %d", summary.ImportedContainersCount)
				}

				var wfFound, ctFound bool
				for _, b := range app.Bindings {
					if b.Type == domain.BindingTypeWorkflow && b.Name == "ORDER_WORKFLOW" && b.ClassName == "OrderWorkflow" && b.WorkflowName == "order-processing" {
						wfFound = true
					}
					if b.Type == domain.BindingTypeContainer && b.Name == "db-sidecar" && b.Image == "postgres:16-alpine" && b.Port == 5432 {
						ctFound = true
					}
				}
				if !wfFound {
					t.Errorf("expected workflow binding not found in: %+v", app.Bindings)
				}
				if !ctFound {
					t.Errorf("expected container binding not found in: %+v", app.Bindings)
				}
			})
		})
	})
}

func TestApplyToApplication_WorkflowsAndContainers_EnvironmentOverride(t *testing.T) {
	t.Run("Given wrangler config with environment overrides for workflows and containers", func(t *testing.T) {
		app, _ := domain.NewApplicationWithSource("app-env-wf", "env-wf-app", domain.SourceTypeInline, "", "", "", nil, nil)

		cfg := &wrangler.WranglerConfig{
			Workflows: []wrangler.WorkflowBinding{
				{Name: "base-wf", Binding: "WORKFLOW", ClassName: "BaseWF"},
			},
			Containers: []wrangler.ContainerBinding{
				{Name: "cache", Image: "redis:6-alpine", Port: 6379},
			},
			Env: map[string]wrangler.WranglerConfig{
				"production": {
					Workflows: []wrangler.WorkflowBinding{
						{Name: "prod-wf", Binding: "WORKFLOW", ClassName: "ProdWF"},
					},
					Containers: []wrangler.ContainerBinding{
						{Name: "cache", Image: "redis:7-alpine", Port: 6380},
					},
				},
			},
		}

		t.Run("When applied with production environment override", func(t *testing.T) {
			summary, err := wrangler.ApplyToApplication(app, cfg, "production", "json")

			t.Run("Then production workflow and container override base configuration", func(t *testing.T) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if summary.ImportedWorkflowsCount != 1 {
					t.Errorf("expected 1 imported workflow, got %d", summary.ImportedWorkflowsCount)
				}
				if summary.ImportedContainersCount != 1 {
					t.Errorf("expected 1 imported container, got %d", summary.ImportedContainersCount)
				}

				for _, b := range app.Bindings {
					if b.Type == domain.BindingTypeWorkflow {
						if b.ClassName != "ProdWF" || b.WorkflowName != "prod-wf" {
							t.Errorf("expected production workflow, got: %+v", b)
						}
					}
					if b.Type == domain.BindingTypeContainer {
						if b.Image != "redis:7-alpine" || b.Port != 6380 {
							t.Errorf("expected production container, got: %+v", b)
						}
					}
				}
			})
		})
	})
}

func TestApplyToApplication_Workflow_FallbackNameToBinding(t *testing.T) {
	t.Run("Given a workflow binding with only name and class_name", func(t *testing.T) {
		app, _ := domain.NewApplicationWithSource("app-wf-fb", "wf-fb-app", domain.SourceTypeInline, "", "", "", nil, nil)
		cfg := &wrangler.WranglerConfig{
			Workflows: []wrangler.WorkflowBinding{
				{Name: "auth-pipeline", ClassName: "AuthPipeline"},
			},
		}

		t.Run("When applied to application", func(t *testing.T) {
			_, err := wrangler.ApplyToApplication(app, cfg, "", "json")

			t.Run("Then binding Name falls back to Name and ResourceID is preserved", func(t *testing.T) {
				if err != nil {
					t.Fatalf("unexpected apply error: %v", err)
				}
				if len(app.Bindings) != 1 {
					t.Fatalf("expected 1 binding, got %d", len(app.Bindings))
				}
				b := app.Bindings[0]
				if b.Type != domain.BindingTypeWorkflow || b.Name != "auth-pipeline" || b.ClassName != "AuthPipeline" || b.WorkflowName != "auth-pipeline" {
					t.Errorf("unexpected binding: %+v", b)
				}
			})
		})
	})

	t.Run("Given a container binding with only binding and image", func(t *testing.T) {
		app, _ := domain.NewApplicationWithSource("app-ct-fb", "ct-fb-app", domain.SourceTypeInline, "", "", "", nil, nil)
		cfg := &wrangler.WranglerConfig{
			Containers: []wrangler.ContainerBinding{
				{Binding: "CACHE_SERVICE", Image: "memcached:alpine"},
			},
		}

		t.Run("When applied to application", func(t *testing.T) {
			_, err := wrangler.ApplyToApplication(app, cfg, "", "json")

			t.Run("Then container Name falls back to Binding and Image is mapped", func(t *testing.T) {
				if err != nil {
					t.Fatalf("unexpected apply error: %v", err)
				}
				if len(app.Bindings) != 1 {
					t.Fatalf("expected 1 binding, got %d", len(app.Bindings))
				}
				b := app.Bindings[0]
				if b.Type != domain.BindingTypeContainer || b.Name != "CACHE_SERVICE" || b.Image != "memcached:alpine" {
					t.Errorf("unexpected binding: %+v", b)
				}
			})
		})
	})

	t.Run("Given a container binding with both name and binding", func(t *testing.T) {
		app, _ := domain.NewApplicationWithSource("app-ct-both", "ct-both-app", domain.SourceTypeInline, "", "", "", nil, nil)
		cfg := &wrangler.WranglerConfig{
			Containers: []wrangler.ContainerBinding{
				{Name: "redis-workload", Binding: "REDIS_CLIENT", Image: "redis:alpine"},
			},
		}

		t.Run("When applied to application", func(t *testing.T) {
			_, err := wrangler.ApplyToApplication(app, cfg, "", "json")

			t.Run("Then Name is the JS binding and ContainerName is the workload name", func(t *testing.T) {
				if err != nil {
					t.Fatalf("unexpected apply error: %v", err)
				}
				if len(app.Bindings) != 1 {
					t.Fatalf("expected 1 binding, got %d", len(app.Bindings))
				}
				b := app.Bindings[0]
				if b.Type != domain.BindingTypeContainer || b.Name != "REDIS_CLIENT" || b.ContainerName != "redis-workload" || b.Image != "redis:alpine" {
					t.Errorf("unexpected binding: %+v", b)
				}
			})
		})
	})
}


